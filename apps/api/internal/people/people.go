// Package people owns persons (one row per human, for life), their portfolios, terms,
// committee positions and seats. A person's status is derived (StatusOf), never stored.
package people

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sakib412/ekman"
	"github.com/sakib412/ekman/db"
	"github.com/sakib412/ekman/rbac"

	"github.com/BinarySUST/binarysust-web/apps/api/internal/access"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/pg"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Permissions.
const (
	EditOwn rbac.Permission = "profile.edit_own" // baseline: edit your own portfolio
	Manage  rbac.Permission = "people.manage"    // persons, terms, positions, seats
)

// Dhaka is the club's clock for "today" (alumni from graduation day, and so on).
var Dhaka = time.FixedZone("Asia/Dhaka", 6*60*60)

// Today is the current date in Dhaka, as midnight UTC.
func Today(now time.Time) time.Time {
	y, m, d := now.In(Dhaka).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// Service is the people module.
type Service struct {
	db  *db.DB
	acc *access.Service
	now func() time.Time
}

// New returns the people service.
func New(d *db.DB, acc *access.Service) *Service {
	return &Service{db: d, acc: acc, now: time.Now}
}

// Module wires people into the app.
func (s *Service) Module() ekman.Module {
	return ekman.Module{
		Name:        "people",
		Migrations:  migrations,
		Permissions: []rbac.Permission{EditOwn, Manage},
		Routes: func(r *ekman.Router) {
			ekman.Handle(r, "GET /me/portfolio", ekman.Requires(EditOwn), s.getPortfolio, ekman.Summary("Your own portfolio"))
			ekman.Handle(r, "PUT /me/portfolio", ekman.Requires(EditOwn), s.putPortfolio, ekman.Summary("Replace your own portfolio"))

			ekman.Handle(r, "GET /people", ekman.Requires(Manage), s.listPeople)
			ekman.Handle(r, "GET /people/lookup", ekman.SignedIn(), s.lookup,
				ekman.Summary("Find people by name or handle (speaker and author pickers)"))
			ekman.Handle(r, "POST /people", ekman.Requires(Manage), s.createPerson)
			ekman.Handle(r, "GET /people/{id}", ekman.Requires(Manage), s.getPerson)
			ekman.Handle(r, "PUT /people/{id}", ekman.Requires(Manage), s.updatePerson)
			ekman.Handle(r, "DELETE /people/{id}", ekman.Requires(Manage), s.deletePerson,
				ekman.Summary("Delete a person with no committee seats, posts or talks (409 otherwise)"))

			ekman.Handle(r, "GET /terms", ekman.Requires(Manage), s.listTerms)
			ekman.Handle(r, "POST /terms", ekman.Requires(Manage), s.createTerm)
			ekman.Handle(r, "PUT /terms/{id}", ekman.Requires(Manage), s.updateTerm)

			ekman.Handle(r, "GET /positions", ekman.Requires(Manage), s.listPositions)
			ekman.Handle(r, "POST /positions", ekman.Requires(Manage), s.createPosition)
			ekman.Handle(r, "PUT /positions/{id}", ekman.Requires(Manage), s.updatePosition)

			// Seats in the current term confer roles; see requireSeatEdit.
			ekman.Handle(r, "GET /terms/{id}/seats", ekman.Requires(Manage), s.listSeats)
			ekman.Handle(r, "POST /terms/{id}/seats", ekman.Requires(Manage), s.createSeat)
			ekman.Handle(r, "DELETE /seats/{id}", ekman.Requires(Manage), s.deleteSeat)
		},
	}
}

// ---- persons (people managers) ------------------------------------------------------------

// PersonRow is one line of the dashboard's people list.
type PersonRow struct {
	ID          string      `json:"id"`
	Handle      string      `json:"handle"`
	Name        string      `json:"name"`
	Email       *string     `json:"email"`
	Affiliation Affiliation `json:"affiliation"`
	Status      Status      `json:"status"`
	Listed      bool        `json:"listed"`
}

// Person is the manager's view of a person (identity fields; the portfolio is the owner's).
type Person struct {
	ID          string      `json:"id"`
	Handle      string      `json:"handle"`
	Name        string      `json:"name"`
	Email       *string     `json:"email"`
	Affiliation Affiliation `json:"affiliation"`
	Status      Status      `json:"status"`
	Listed      bool        `json:"listed"`
	AvatarURL   *string     `json:"avatarUrl"`
	Headline    string      `json:"headline"`
	Department  string      `json:"department"`
	Batch       *int        `json:"batch"`
	MemberSince *string     `json:"memberSince"` // YYYY-MM-DD
	GraduatedOn *string     `json:"graduatedOn"` // YYYY-MM-DD
	UpdatedAt   time.Time   `json:"updatedAt"`
}

// factsJoin computes the stored inputs of StatusOf next to each person. $1 is the current term id.
const factsCols = `,
	(SELECT CASE WHEN bool_or(pos.category = 'executive') THEN 'executive'
	             WHEN count(*) > 0 THEN 'advisor' END
	   FROM committee_seats cs JOIN positions pos ON pos.id = cs.position_id
	  WHERE cs.person_id = p.id AND cs.term_id = $1::uuid),
	EXISTS (SELECT 1 FROM committee_seats cs JOIN positions pos ON pos.id = cs.position_id
	         WHERE cs.person_id = p.id AND pos.category = 'advisor')`

const personCols = `p.id, p.handle, p.name, p.email, p.affiliation, p.listed, p.avatar_url, p.headline,
	p.department, p.batch, p.member_since, p.graduated_on, p.updated_at`

func (s *Service) currentTerm(ctx context.Context) (*string, error) {
	id, err := s.acc.CurrentTermID(ctx)
	if err != nil || id == "" {
		return nil, err
	}
	return &id, nil
}

func scanPerson(sc interface{ Scan(...any) error }, today time.Time) (Person, error) {
	var p Person
	var member, grad *time.Time
	var seat *string
	var everAdvisor bool
	err := sc.Scan(&p.ID, &p.Handle, &p.Name, &p.Email, &p.Affiliation, &p.Listed, &p.AvatarURL, &p.Headline,
		&p.Department, &p.Batch, &member, &grad, &p.UpdatedAt, &seat, &everAdvisor)
	if err != nil {
		return p, err
	}
	p.MemberSince, p.GraduatedOn = fmtDate(member), fmtDate(grad)
	p.UpdatedAt = p.UpdatedAt.UTC()
	p.Status = StatusOf(Facts{Affiliation: p.Affiliation, MemberSince: member, GraduatedOn: grad,
		CurrentSeat: (*Category)(seat), EverAdvisor: everAdvisor}, today)
	return p, nil
}

func (s *Service) queryPersons(ctx context.Context, where string, args ...any) ([]Person, error) {
	term, err := s.currentTerm(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Q(ctx).QueryContext(ctx,
		`SELECT `+personCols+factsCols+` FROM persons p `+where, append([]any{term}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	today := Today(s.now())
	var out []Person
	for rows.Next() {
		p, err := scanPerson(rows, today)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Service) person(ctx context.Context, id string) (Person, error) {
	if !pg.ValidID(id) {
		return Person{}, pg.NotFound("person")
	}
	ps, err := s.queryPersons(ctx, `WHERE p.id = $2`, id)
	if err != nil {
		return Person{}, err
	}
	if len(ps) == 0 {
		return Person{}, pg.NotFound("person")
	}
	return ps[0], nil
}

func (s *Service) listPeople(ctx context.Context, _ ekman.Caller, _ *struct{}) ([]PersonRow, error) {
	ps, err := s.queryPersons(ctx, `ORDER BY p.name`)
	out := make([]PersonRow, 0, len(ps))
	for _, p := range ps {
		out = append(out, PersonRow{ID: p.ID, Handle: p.Handle, Name: p.Name, Email: p.Email,
			Affiliation: p.Affiliation, Status: p.Status, Listed: p.Listed})
	}
	return out, err
}

// PersonIn creates or updates a person's identity fields.
type PersonIn struct {
	Handle      string      `json:"handle"`
	Name        string      `json:"name"`
	Email       *string     `json:"email"`
	Affiliation Affiliation `json:"affiliation"`
	Listed      bool        `json:"listed"`
	AvatarURL   *string     `json:"avatarUrl"`
	Headline    string      `json:"headline"`
	Department  string      `json:"department"`
	Batch       *int        `json:"batch"`
	MemberSince *string     `json:"memberSince"`
	GraduatedOn *string     `json:"graduatedOn"`
}

// Validate implements ekman.Validator.
func (in *PersonIn) Validate(v *ekman.Validation) {
	in.Handle = strings.ToLower(strings.TrimSpace(in.Handle))
	v.Required("handle", in.Handle)
	v.Check(handleOK(in.Handle), "handle", "use 2-40 lowercase letters, digits and dashes")
	v.Required("name", strings.TrimSpace(in.Name))
	v.MaxLen("name", in.Name, 120)
	if in.Email != nil {
		*in.Email = strings.ToLower(strings.TrimSpace(*in.Email))
		if *in.Email == "" {
			in.Email = nil
		} else {
			v.Email("email", *in.Email)
		}
	}
	v.OneOf("affiliation", string(in.Affiliation), string(Student), string(Faculty), string(Visitor))
	if in.Affiliation == "" {
		in.Affiliation = Student
	}
	optURL(v, "avatarUrl", in.AvatarURL)
	v.MaxLen("headline", in.Headline, 160)
	v.MaxLen("department", in.Department, 80)
	v.Check(in.Batch == nil || (*in.Batch >= 1990 && *in.Batch <= 2100), "batch", "must be a year")
	optDate(v, "memberSince", in.MemberSince)
	optDate(v, "graduatedOn", in.GraduatedOn)
}

func (s *Service) createPerson(ctx context.Context, _ ekman.Caller, in *PersonIn) (Person, error) {
	var id string
	err := s.db.Tx(ctx, func(ctx context.Context) error {
		err := s.db.Q(ctx).QueryRowContext(ctx, `
			INSERT INTO persons (handle, name, email, affiliation, listed, avatar_url, headline, department,
			                     batch, member_since, graduated_on)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id`,
			in.Handle, in.Name, in.Email, in.Affiliation, in.Listed, in.AvatarURL, in.Headline, in.Department,
			in.Batch, in.MemberSince, in.GraduatedOn).Scan(&id)
		if err != nil {
			return pg.Friendly(err, "that handle or email is already used")
		}
		if _, err := s.db.Q(ctx).ExecContext(ctx, `DELETE FROM person_handles WHERE handle = $1`, in.Handle); err != nil {
			return err
		}
		return s.acc.LinkPersons(ctx)
	})
	if err != nil {
		return Person{}, err
	}
	return s.person(ctx, id)
}

// PersonUpdateIn is PersonIn for an existing id.
type PersonUpdateIn struct {
	ID string `path:"id"`
	PersonIn
}

func (s *Service) getPerson(ctx context.Context, _ ekman.Caller, in *access.IDIn) (Person, error) {
	return s.person(ctx, in.ID)
}

func (s *Service) updatePerson(ctx context.Context, _ ekman.Caller, in *PersonUpdateIn) (Person, error) {
	if !pg.ValidID(in.ID) {
		return Person{}, pg.NotFound("person")
	}
	err := s.db.Tx(ctx, func(ctx context.Context) error {
		var old string
		if err := s.db.Q(ctx).QueryRowContext(ctx, `SELECT handle FROM persons WHERE id = $1 FOR UPDATE`, in.ID).Scan(&old); err != nil {
			return err
		}
		_, err := s.db.Q(ctx).ExecContext(ctx, `
			UPDATE persons SET handle = $2, name = $3, email = $4, affiliation = $5, listed = $6, avatar_url = $7,
			       headline = $8, department = $9, batch = $10, member_since = $11, graduated_on = $12, updated_at = now()
			 WHERE id = $1`,
			in.ID, in.Handle, in.Name, in.Email, in.Affiliation, in.Listed, in.AvatarURL, in.Headline, in.Department,
			in.Batch, in.MemberSince, in.GraduatedOn)
		if err != nil {
			return pg.Friendly(err, "that handle or email is already used")
		}
		if old != in.Handle {
			// The old URL keeps working through a redirect; a reclaimed handle stops redirecting.
			if _, err := s.db.Q(ctx).ExecContext(ctx, `
				INSERT INTO person_handles (handle, person_id) VALUES ($1, $2)
				ON CONFLICT (handle) DO UPDATE SET person_id = EXCLUDED.person_id`, old, in.ID); err != nil {
				return err
			}
			if _, err := s.db.Q(ctx).ExecContext(ctx, `DELETE FROM person_handles WHERE handle = $1`, in.Handle); err != nil {
				return err
			}
		}
		return s.acc.LinkPersons(ctx)
	})
	if err != nil {
		return Person{}, err
	}
	return s.person(ctx, in.ID)
}

// deletePerson removes a person who left no trace in the club's history. Seats are history
// (the public committee pages), and posts and talks would lose their author or speaker, so a
// person with any of them is a 409: unlist them instead.
func (s *Service) deletePerson(ctx context.Context, _ ekman.Caller, in *access.IDIn) (ekman.NoContent, error) {
	if !pg.ValidID(in.ID) {
		return ekman.NoContent{}, pg.NotFound("person")
	}
	return ekman.NoContent{}, s.db.Tx(ctx, func(ctx context.Context) error {
		var name string
		var seats int
		err := s.db.Q(ctx).QueryRowContext(ctx, `
			SELECT p.name, (SELECT count(*) FROM committee_seats cs WHERE cs.person_id = p.id)
			  FROM persons p WHERE p.id = $1 FOR UPDATE`, in.ID).Scan(&name, &seats)
		if errors.Is(err, sql.ErrNoRows) {
			return pg.NotFound("person")
		}
		if err != nil {
			return err
		}
		if seats > 0 {
			return ekman.Conflict("person_in_use", fmt.Sprintf("%s holds %d committee %s; committee history is kept, so unlist them instead (or remove the seats first)",
				name, seats, plural(seats, "seat", "seats")))
		}
		_, err = s.db.Q(ctx).ExecContext(ctx, `DELETE FROM persons WHERE id = $1`, in.ID)
		// posts.author_id and talks.speaker_id belong to other modules; their foreign keys refuse.
		switch c := pg.Constraint(err); {
		case pg.ForeignKey(err) && strings.HasPrefix(c, "posts_"):
			return ekman.Conflict("person_in_use", name+" wrote blog posts; delete or reassign them first, or unlist the person instead")
		case pg.ForeignKey(err) && strings.HasPrefix(c, "talks_"):
			return ekman.Conflict("person_in_use", name+" is the speaker of a talk; delete the talk first, or unlist the person instead")
		}
		return pg.Friendly(err, "")
	})
}

// PersonRef is a person in a picker: enough to show and to reference by handle.
type PersonRef struct {
	Handle    string  `json:"handle"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatarUrl"`
}

// LookupIn searches by name or handle.
type LookupIn struct {
	Q string `query:"q"`
}

// lookup serves pickers to anyone signed in (event managers choose speakers, editors see
// authors). Unlisted people stay hidden unless the caller manages people.
func (s *Service) lookup(ctx context.Context, c ekman.Caller, in *LookupIn) ([]PersonRef, error) {
	q := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(in.Q), "@")))
	like := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(q) + "%"
	rows, err := s.db.Q(ctx).QueryContext(ctx, `
		SELECT handle, name, avatar_url FROM persons
		 WHERE (listed OR $3) AND ($1 = '' OR handle LIKE $2 OR lower(name) LIKE $2)
		 ORDER BY (handle = $1) DESC, (handle LIKE $1 || '%') DESC, name, handle
		 LIMIT 20`, q, like, c.Can(Manage))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PersonRef{}
	for rows.Next() {
		var p PersonRef
		if err := rows.Scan(&p.Handle, &p.Name, &p.AvatarURL); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// IDsByHandle resolves handles to person ids, in order. Unknown handles are a 422 on field.
func (s *Service) IDsByHandle(ctx context.Context, field string, handles []string) ([]string, error) {
	ids := make([]string, 0, len(handles))
	for _, h := range handles {
		var id string
		err := s.db.Q(ctx).QueryRowContext(ctx, `SELECT id FROM persons WHERE handle = $1`, strings.TrimSpace(h)).Scan(&id)
		if err == sql.ErrNoRows {
			return nil, &ekman.Error{Status: 422, Code: "invalid_input", Message: "unknown person",
				Fields: map[string]string{field: "no person with handle " + h}}
		}
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// ---- helpers ------------------------------------------------------------------------------

func handleOK(h string) bool {
	if len(h) < 2 || len(h) > 40 || h[0] == '-' {
		return false
	}
	for _, c := range h {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return true
}

func fmtDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.DateOnly)
	return &s
}

func optDate(v *ekman.Validation, field string, s *string) {
	if s == nil {
		return
	}
	_, err := time.Parse(time.DateOnly, *s)
	v.Check(err == nil, field, "must be a date like 2025-10-01")
}

func optURL(v *ekman.Validation, field string, s *string) {
	if s != nil {
		v.URL(field, *s)
	}
}
