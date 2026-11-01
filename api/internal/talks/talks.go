// Package talks holds advisor and guest talks: a speaker (a person), a date, a summary, and
// video and slide links.
package talks

import (
	"context"
	"embed"
	"strings"
	"time"

	"github.com/sakib412/ekman"
	"github.com/sakib412/ekman/db"
	"github.com/sakib412/ekman/rbac"

	"github.com/BinarySUST/binarysust-web/api/internal/md"
	"github.com/BinarySUST/binarysust-web/api/internal/people"
	"github.com/BinarySUST/binarysust-web/api/internal/pg"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Manage lets a role edit talks.
const Manage rbac.Permission = "talks.manage"

// Public is a talk on the public site.
type Public struct {
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Speaker     string    `json:"speaker"` // person handle
	Date        time.Time `json:"date"`
	SummaryHTML string    `json:"summaryHtml"`
	VideoURL    *string   `json:"videoUrl"`
	SlidesURL   *string   `json:"slidesUrl"`
	Featured    bool      `json:"featured"`
}

// Talk is the dashboard view.
type Talk struct {
	ID              string    `json:"id"`
	Slug            string    `json:"slug"`
	Title           string    `json:"title"`
	Speaker         string    `json:"speaker"` // person handle
	SpeakerName     string    `json:"speakerName"`
	Date            time.Time `json:"date"`
	SummaryMarkdown string    `json:"summaryMarkdown"`
	VideoURL        *string   `json:"videoUrl"`
	SlidesURL       *string   `json:"slidesUrl"`
	Featured        bool      `json:"featured"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// Service is the talks module.
type Service struct {
	db     *db.DB
	people *people.Service
}

// New returns the talks service.
func New(d *db.DB, p *people.Service) *Service { return &Service{db: d, people: p} }

// Module wires talks into the app.
func (s *Service) Module() ekman.Module {
	return ekman.Module{
		Name:        "talks",
		Migrations:  migrations,
		Permissions: []rbac.Permission{Manage},
		Routes: func(r *ekman.Router) {
			ekman.Handle(r, "GET /talks", ekman.Requires(Manage), s.list)
			ekman.Handle(r, "POST /talks", ekman.Requires(Manage), s.create)
			ekman.Handle(r, "GET /talks/{id}", ekman.Requires(Manage), s.get)
			ekman.Handle(r, "PUT /talks/{id}", ekman.Requires(Manage), s.update)
			ekman.Handle(r, "DELETE /talks/{id}", ekman.Requires(Manage), s.remove)
		},
	}
}

// In is the body of create and update.
type In struct {
	Slug            string    `json:"slug"`
	Title           string    `json:"title"`
	Speaker         string    `json:"speaker"` // person handle
	Date            time.Time `json:"date"`
	SummaryMarkdown string    `json:"summaryMarkdown"`
	VideoURL        *string   `json:"videoUrl"`
	SlidesURL       *string   `json:"slidesUrl"`
	Featured        bool      `json:"featured"`
}

// Validate implements ekman.Validator.
func (in *In) Validate(v *ekman.Validation) {
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	v.Required("slug", in.Slug)
	v.MaxLen("slug", in.Slug, 80)
	v.Required("title", in.Title)
	v.MaxLen("title", in.Title, 200)
	v.Required("speaker", in.Speaker)
	v.Check(!in.Date.IsZero(), "date", "required")
	v.MaxLen("summaryMarkdown", in.SummaryMarkdown, 20_000)
	if in.VideoURL != nil {
		v.URL("videoUrl", *in.VideoURL)
	}
	if in.SlidesURL != nil {
		v.URL("slidesUrl", *in.SlidesURL)
	}
}

// IDIn is a path id.
type IDIn struct {
	ID string `path:"id"`
}

// UpdateIn is In for an existing id.
type UpdateIn struct {
	ID string `path:"id"`
	In
}

const cols = `t.id, t.slug, t.title, p.handle, p.name, t.given_at, t.summary_md, t.video_url, t.slides_url, t.featured, t.updated_at`
const from = ` FROM talks t JOIN persons p ON p.id = t.speaker_id`

func scan(sc interface{ Scan(...any) error }) (Talk, error) {
	var t Talk
	err := sc.Scan(&t.ID, &t.Slug, &t.Title, &t.Speaker, &t.SpeakerName, &t.Date, &t.SummaryMarkdown,
		&t.VideoURL, &t.SlidesURL, &t.Featured, &t.UpdatedAt)
	t.Date, t.UpdatedAt = t.Date.UTC(), t.UpdatedAt.UTC()
	return t, err
}

func (s *Service) list(ctx context.Context, _ ekman.Caller, _ *struct{}) ([]Talk, error) {
	rows, err := s.db.Q(ctx).QueryContext(ctx, `SELECT `+cols+from+` ORDER BY t.given_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Talk
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Service) one(ctx context.Context, id string) (Talk, error) {
	if !pg.ValidID(id) {
		return Talk{}, pg.NotFound("talk")
	}
	return scan(s.db.Q(ctx).QueryRowContext(ctx, `SELECT `+cols+from+` WHERE t.id = $1`, id))
}

func (s *Service) get(ctx context.Context, _ ekman.Caller, in *IDIn) (Talk, error) {
	return s.one(ctx, in.ID)
}

func (s *Service) create(ctx context.Context, _ ekman.Caller, in *In) (Talk, error) {
	ids, err := s.people.IDsByHandle(ctx, "speaker", []string{in.Speaker})
	if err != nil {
		return Talk{}, err
	}
	var id string
	err = s.db.Q(ctx).QueryRowContext(ctx, `
		INSERT INTO talks (slug, title, speaker_id, given_at, summary_md, video_url, slides_url, featured)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		in.Slug, in.Title, ids[0], in.Date, in.SummaryMarkdown, in.VideoURL, in.SlidesURL, in.Featured).Scan(&id)
	if err != nil {
		return Talk{}, pg.Friendly(err, "a talk with that slug exists")
	}
	return s.one(ctx, id)
}

func (s *Service) update(ctx context.Context, _ ekman.Caller, in *UpdateIn) (Talk, error) {
	if !pg.ValidID(in.ID) {
		return Talk{}, pg.NotFound("talk")
	}
	ids, err := s.people.IDsByHandle(ctx, "speaker", []string{in.Speaker})
	if err != nil {
		return Talk{}, err
	}
	res, err := s.db.Q(ctx).ExecContext(ctx, `
		UPDATE talks SET slug = $2, title = $3, speaker_id = $4, given_at = $5, summary_md = $6,
		       video_url = $7, slides_url = $8, featured = $9, updated_at = now()
		 WHERE id = $1`,
		in.ID, in.Slug, in.Title, ids[0], in.Date, in.SummaryMarkdown, in.VideoURL, in.SlidesURL, in.Featured)
	if err != nil {
		return Talk{}, pg.Friendly(err, "a talk with that slug exists")
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Talk{}, pg.NotFound("talk")
	}
	return s.one(ctx, in.ID)
}

func (s *Service) remove(ctx context.Context, _ ekman.Caller, in *IDIn) (ekman.NoContent, error) {
	if !pg.ValidID(in.ID) {
		return ekman.NoContent{}, pg.NotFound("talk")
	}
	_, err := s.db.Q(ctx).ExecContext(ctx, `DELETE FROM talks WHERE id = $1`, in.ID)
	return ekman.NoContent{}, err
}

// Published is the read side for site.Build, newest first.
func (s *Service) Published(ctx context.Context) ([]Public, error) {
	rows, err := s.db.Q(ctx).QueryContext(ctx, `SELECT `+cols+from+` ORDER BY t.given_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Public{}
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, Public{Slug: t.Slug, Title: t.Title, Speaker: t.Speaker, Date: t.Date,
			SummaryHTML: md.Render(t.SummaryMarkdown), VideoURL: t.VideoURL, SlidesURL: t.SlidesURL, Featured: t.Featured})
	}
	return out, rows.Err()
}
