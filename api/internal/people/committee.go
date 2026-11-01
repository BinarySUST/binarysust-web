package people

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/sakib412/ekman"

	"github.com/BinarySUST/binarysust-web/api/internal/access"
	"github.com/BinarySUST/binarysust-web/api/internal/pg"
)

// ---- terms --------------------------------------------------------------------------------

// TermRow is a term in the dashboard.
type TermRow struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	StartsOn string `json:"startsOn"`
	Current  bool   `json:"current"`
	Seats    int    `json:"seats"`
}

func (s *Service) terms(ctx context.Context, where string, args ...any) ([]TermRow, error) {
	term, err := s.currentTerm(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Q(ctx).QueryContext(ctx, `
		SELECT t.id, t.slug, t.label, t.starts_on, t.id = $1::uuid,
		       (SELECT count(*) FROM committee_seats cs WHERE cs.term_id = t.id)
		  FROM terms t `+where+` ORDER BY t.starts_on DESC`, append([]any{term}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TermRow
	for rows.Next() {
		var t TermRow
		var starts time.Time
		var current *bool
		if err := rows.Scan(&t.ID, &t.Slug, &t.Label, &starts, &current, &t.Seats); err != nil {
			return nil, err
		}
		t.StartsOn = starts.Format(time.DateOnly)
		t.Current = current != nil && *current
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Service) listTerms(ctx context.Context, _ ekman.Caller, _ *struct{}) ([]TermRow, error) {
	return s.terms(ctx, "")
}

// TermIn creates or edits a term. Making it current is access's job (roles.manage).
type TermIn struct {
	Slug     string `json:"slug"`     // 2026-27
	Label    string `json:"label"`    // 2026–27 (defaults to the slug with an en dash)
	StartsOn string `json:"startsOn"` // YYYY-MM-DD
}

// Validate implements ekman.Validator.
func (in *TermIn) Validate(v *ekman.Validation) {
	in.Slug = strings.TrimSpace(in.Slug)
	v.Required("slug", in.Slug)
	v.Check(len(in.Slug) == 7 && in.Slug[4] == '-', "slug", "use the form 2026-27")
	if strings.TrimSpace(in.Label) == "" {
		in.Label = strings.Replace(in.Slug, "-", "–", 1)
	}
	v.MaxLen("label", in.Label, 40)
	v.Required("startsOn", in.StartsOn)
	optDate(v, "startsOn", &in.StartsOn)
}

func (s *Service) oneTerm(ctx context.Context, id string) (TermRow, error) {
	ts, err := s.terms(ctx, `WHERE t.id = $2`, id)
	if err != nil {
		return TermRow{}, err
	}
	if len(ts) == 0 {
		return TermRow{}, pg.NotFound("term")
	}
	return ts[0], nil
}

func (s *Service) createTerm(ctx context.Context, _ ekman.Caller, in *TermIn) (TermRow, error) {
	var id string
	err := s.db.Q(ctx).QueryRowContext(ctx,
		`INSERT INTO terms (slug, label, starts_on) VALUES ($1, $2, $3) RETURNING id`,
		in.Slug, in.Label, in.StartsOn).Scan(&id)
	if err != nil {
		return TermRow{}, pg.Friendly(err, "a term with that slug exists")
	}
	return s.oneTerm(ctx, id)
}

// TermUpdateIn edits a term.
type TermUpdateIn struct {
	ID string `path:"id"`
	TermIn
}

func (s *Service) updateTerm(ctx context.Context, _ ekman.Caller, in *TermUpdateIn) (TermRow, error) {
	if !pg.ValidID(in.ID) {
		return TermRow{}, pg.NotFound("term")
	}
	res, err := s.db.Q(ctx).ExecContext(ctx,
		`UPDATE terms SET slug = $2, label = $3, starts_on = $4 WHERE id = $1`, in.ID, in.Slug, in.Label, in.StartsOn)
	if err != nil {
		return TermRow{}, pg.Friendly(err, "a term with that slug exists")
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return TermRow{}, pg.NotFound("term")
	}
	return s.oneTerm(ctx, in.ID)
}

// ---- positions ----------------------------------------------------------------------------

// Position is a committee position, reused across terms. The constitution's positions are
// reference data (migrations/0002_positions.sql); managers may add more.
type Position struct {
	ID       string   `json:"id"`
	Slug     string   `json:"slug"`
	Title    string   `json:"title"`
	Category Category `json:"category"`
	Group    Group    `json:"group"`
	Rank     int      `json:"rank"`
	// Lifetime: holding this position in any term makes the person a lifetime member.
	Lifetime bool `json:"lifetime"`
}

const positionCols = `id, slug, title, category, display_group, rank, lifetime`

func scanPosition(sc interface{ Scan(...any) error }) (Position, error) {
	var p Position
	err := sc.Scan(&p.ID, &p.Slug, &p.Title, &p.Category, &p.Group, &p.Rank, &p.Lifetime)
	return p, err
}

func (s *Service) listPositions(ctx context.Context, _ ekman.Caller, _ *struct{}) ([]Position, error) {
	rows, err := s.db.Q(ctx).QueryContext(ctx, `SELECT `+positionCols+` FROM positions ORDER BY category DESC, rank, title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Position
	for rows.Next() {
		p, err := scanPosition(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// PositionIn creates or edits a position.
type PositionIn struct {
	Slug     string   `json:"slug"`
	Title    string   `json:"title"`
	Category Category `json:"category"`
	// Group defaults to adviser for advisor positions and member otherwise.
	Group    Group `json:"group,omitempty"`
	Rank     int   `json:"rank"`
	Lifetime bool  `json:"lifetime,omitempty"`
}

// Validate implements ekman.Validator.
func (in *PositionIn) Validate(v *ekman.Validation) {
	v.Check(handleOK(in.Slug), "slug", "use 2-40 lowercase letters, digits and dashes")
	v.Required("title", in.Title)
	v.MaxLen("title", in.Title, 80)
	v.Required("category", string(in.Category))
	v.OneOf("category", string(in.Category), Category("").Enum()...)
	if in.Group == "" {
		in.Group = GroupMember
		if in.Category == AdvisorCat {
			in.Group = GroupAdviser
		}
	}
	v.OneOf("group", string(in.Group), Group("").Enum()...)
}

func (s *Service) createPosition(ctx context.Context, _ ekman.Caller, in *PositionIn) (Position, error) {
	p, err := scanPosition(s.db.Q(ctx).QueryRowContext(ctx, `
		INSERT INTO positions (slug, title, category, display_group, rank, lifetime) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+positionCols, in.Slug, in.Title, in.Category, in.Group, in.Rank, in.Lifetime))
	return p, pg.Friendly(err, "a position with that slug exists")
}

// PositionUpdateIn edits a position.
type PositionUpdateIn struct {
	ID string `path:"id"`
	PositionIn
}

func (s *Service) updatePosition(ctx context.Context, c ekman.Caller, in *PositionUpdateIn) (Position, error) {
	if !pg.ValidID(in.ID) {
		return Position{}, pg.NotFound("position")
	}
	// Turning an executive post with roles into an advisor post (or back) changes nobody's
	// roles, so a people manager may do it.
	p, err := scanPosition(s.db.Q(ctx).QueryRowContext(ctx, `
		UPDATE positions SET slug = $2, title = $3, category = $4, display_group = $5, rank = $6, lifetime = $7
		 WHERE id = $1 RETURNING `+positionCols,
		in.ID, in.Slug, in.Title, in.Category, in.Group, in.Rank, in.Lifetime))
	if errors.Is(err, sql.ErrNoRows) {
		return Position{}, pg.NotFound("position")
	}
	return p, pg.Friendly(err, "a position with that slug exists")
}

// ---- seats --------------------------------------------------------------------------------

// SeatRow is a seat in the dashboard.
type SeatRow struct {
	ID            string   `json:"id"`
	TermID        string   `json:"termId"`
	PositionID    string   `json:"positionId"`
	PositionTitle string   `json:"positionTitle"`
	Category      Category `json:"category"`
	Group         Group    `json:"group"`
	Rank          int      `json:"rank"`
	PersonID      string   `json:"personId"`
	Handle        string   `json:"handle"`
	Name          string   `json:"name"`
}

const seatSelect = `
	SELECT cs.id, cs.term_id, pos.id, pos.title, pos.category, pos.display_group, pos.rank, p.id, p.handle, p.name
	  FROM committee_seats cs
	  JOIN positions pos ON pos.id = cs.position_id
	  JOIN persons p ON p.id = cs.person_id`

func (s *Service) seats(ctx context.Context, where string, args ...any) ([]SeatRow, error) {
	rows, err := s.db.Q(ctx).QueryContext(ctx, seatSelect+" "+where+` ORDER BY pos.category DESC, pos.rank, p.name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SeatRow
	for rows.Next() {
		var r SeatRow
		if err := rows.Scan(&r.ID, &r.TermID, &r.PositionID, &r.PositionTitle, &r.Category, &r.Group, &r.Rank, &r.PersonID, &r.Handle, &r.Name); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) listSeats(ctx context.Context, _ ekman.Caller, in *access.IDIn) ([]SeatRow, error) {
	if !pg.ValidID(in.ID) {
		return nil, pg.NotFound("term")
	}
	return s.seats(ctx, `WHERE cs.term_id = $1`, in.ID)
}

// SeatIn seats a person in a position for a term.
type SeatIn struct {
	TermID     string `path:"id"`
	PersonID   string `json:"personId"`
	PositionID string `json:"positionId"`
}

// Validate implements ekman.Validator.
func (in *SeatIn) Validate(v *ekman.Validation) {
	v.Check(pg.ValidID(in.PersonID), "personId", "must be a person id")
	v.Check(pg.ValidID(in.PositionID), "positionId", "must be a position id")
}

// requireSeatEdit keeps "editing a roster cannot escalate privileges" true: a seat in the
// current term whose position carries roles hands out those roles, so changing one needs
// roles.manage as well as people.manage.
func (s *Service) requireSeatEdit(ctx context.Context, c ekman.Caller, termID, positionID string) error {
	current, err := s.acc.CurrentTermID(ctx)
	if err != nil || current != termID {
		return err
	}
	confers, err := s.acc.ConfersRoles(ctx, positionID)
	if err != nil || !confers {
		return err
	}
	return c.Require(access.ManageRoles)
}

func (s *Service) createSeat(ctx context.Context, c ekman.Caller, in *SeatIn) (SeatRow, error) {
	if !pg.ValidID(in.TermID) {
		return SeatRow{}, pg.NotFound("term")
	}
	if err := s.requireSeatEdit(ctx, c, in.TermID, in.PositionID); err != nil {
		return SeatRow{}, err
	}
	var id string
	err := s.db.Q(ctx).QueryRowContext(ctx,
		`INSERT INTO committee_seats (term_id, position_id, person_id) VALUES ($1, $2, $3) RETURNING id`,
		in.TermID, in.PositionID, in.PersonID).Scan(&id)
	if err != nil {
		return SeatRow{}, pg.Friendly(err, "that person already holds this position in this term")
	}
	rows, err := s.seats(ctx, `WHERE cs.id = $1`, id)
	if err != nil || len(rows) == 0 {
		return SeatRow{}, err
	}
	return rows[0], nil
}

func (s *Service) deleteSeat(ctx context.Context, c ekman.Caller, in *access.IDIn) (ekman.NoContent, error) {
	if !pg.ValidID(in.ID) {
		return ekman.NoContent{}, pg.NotFound("seat")
	}
	rows, err := s.seats(ctx, `WHERE cs.id = $1`, in.ID)
	if err != nil {
		return ekman.NoContent{}, err
	}
	if len(rows) == 0 {
		return ekman.NoContent{}, pg.NotFound("seat")
	}
	if err := s.requireSeatEdit(ctx, c, rows[0].TermID, rows[0].PositionID); err != nil {
		return ekman.NoContent{}, err
	}
	_, err = s.db.Q(ctx).ExecContext(ctx, `DELETE FROM committee_seats WHERE id = $1`, in.ID)
	return ekman.NoContent{}, err
}
