// Package sponsors is the smallest complete module, and the template to copy for a new
// content type (see api/README.md, "Add a module"):
//
//  1. migrations/0001_<name>.sql: the module's own tables (embedded below).
//  2. A permission constant, declared in Module().Permissions.
//  3. Public: the snapshot shape (docs/snapshot.md); Sponsor: the dashboard shape.
//  4. Module(): every route states its access in the route table.
//  5. Published(): the read side site.Build calls. Add a field to site.Snapshot for it.
//
// Writes need no publishing code: the republish middleware rebuilds the site after every
// successful write unless the route opts out with republish.Skip().
package sponsors

import (
	"context"
	"embed"
	"strings"
	"time"

	"github.com/sakib412/ekman"
	"github.com/sakib412/ekman/db"
	"github.com/sakib412/ekman/rbac"

	"github.com/BinarySUST/binarysust-web/apps/api/internal/pg"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Manage lets a role edit sponsors.
const Manage rbac.Permission = "sponsors.manage"

// Tier is platinum, gold, silver or partner.
type Tier string

// Enum lists the values, highest first. A string type with an Enum method becomes a named
// enum in the OpenAPI document (internal/apidoc), so the web's types narrow to these values.
func (Tier) Enum() []string { return []string{"platinum", "gold", "silver", "partner"} }

// Public is a sponsor on the public site.
type Public struct {
	Name    string `json:"name"`
	Tier    Tier   `json:"tier"`
	LogoURL string `json:"logoUrl"`
	URL     string `json:"url"`
	Since   int    `json:"since"`
	Until   *int   `json:"until"`
}

// Sponsor is the dashboard view.
type Sponsor struct {
	ID string `json:"id"`
	Public
	UpdatedAt time.Time `json:"updatedAt"`
}

// Service is the sponsors module.
type Service struct{ db *db.DB }

// New returns the sponsors service.
func New(d *db.DB) *Service { return &Service{db: d} }

// Module wires sponsors into the app.
func (s *Service) Module() ekman.Module {
	return ekman.Module{
		Name:        "sponsors",
		Migrations:  migrations,
		Permissions: []rbac.Permission{Manage},
		Routes: func(r *ekman.Router) {
			ekman.Handle(r, "GET /sponsors", ekman.Requires(Manage), s.list)
			ekman.Handle(r, "POST /sponsors", ekman.Requires(Manage), s.create)
			ekman.Handle(r, "GET /sponsors/{id}", ekman.Requires(Manage), s.get)
			ekman.Handle(r, "PUT /sponsors/{id}", ekman.Requires(Manage), s.update)
			ekman.Handle(r, "DELETE /sponsors/{id}", ekman.Requires(Manage), s.remove)
		},
	}
}

// In is the body of create and update.
type In struct {
	Name    string `json:"name"`
	Tier    Tier   `json:"tier"`
	LogoURL string `json:"logoUrl"`
	URL     string `json:"url"`
	Since   int    `json:"since"`
	Until   *int   `json:"until"`
}

// Validate implements ekman.Validator.
func (in *In) Validate(v *ekman.Validation) {
	in.Name = strings.TrimSpace(in.Name)
	v.Required("name", in.Name)
	v.MaxLen("name", in.Name, 120)
	v.Required("tier", string(in.Tier))
	v.OneOf("tier", string(in.Tier), Tier("").Enum()...)
	v.Required("logoUrl", in.LogoURL)
	v.URL("logoUrl", in.LogoURL)
	v.URL("url", in.URL)
	v.Check(in.Since >= 2000 && in.Since <= 2100, "since", "must be a year")
	v.Check(in.Until == nil || *in.Until >= in.Since, "until", "must be on or after since")
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

const cols = `id, name, tier, logo_url, url, since, until, updated_at`

func scan(sc interface{ Scan(...any) error }) (Sponsor, error) {
	var x Sponsor
	err := sc.Scan(&x.ID, &x.Name, &x.Tier, &x.LogoURL, &x.URL, &x.Since, &x.Until, &x.UpdatedAt)
	x.UpdatedAt = x.UpdatedAt.UTC()
	return x, err
}

func (s *Service) list(ctx context.Context, _ ekman.Caller, _ *struct{}) ([]Sponsor, error) {
	rows, err := s.db.Q(ctx).QueryContext(ctx, `SELECT `+cols+` FROM sponsors ORDER BY since DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Sponsor
	for rows.Next() {
		x, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Service) get(ctx context.Context, _ ekman.Caller, in *IDIn) (Sponsor, error) {
	if !pg.ValidID(in.ID) {
		return Sponsor{}, pg.NotFound("sponsor")
	}
	return scan(s.db.Q(ctx).QueryRowContext(ctx, `SELECT `+cols+` FROM sponsors WHERE id = $1`, in.ID))
}

func (s *Service) create(ctx context.Context, _ ekman.Caller, in *In) (Sponsor, error) {
	return scan(s.db.Q(ctx).QueryRowContext(ctx, `
		INSERT INTO sponsors (name, tier, logo_url, url, since, until) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+cols, in.Name, in.Tier, in.LogoURL, in.URL, in.Since, in.Until))
}

func (s *Service) update(ctx context.Context, _ ekman.Caller, in *UpdateIn) (Sponsor, error) {
	if !pg.ValidID(in.ID) {
		return Sponsor{}, pg.NotFound("sponsor")
	}
	return scan(s.db.Q(ctx).QueryRowContext(ctx, `
		UPDATE sponsors SET name = $2, tier = $3, logo_url = $4, url = $5, since = $6, until = $7, updated_at = now()
		 WHERE id = $1 RETURNING `+cols, in.ID, in.Name, in.Tier, in.LogoURL, in.URL, in.Since, in.Until))
}

func (s *Service) remove(ctx context.Context, _ ekman.Caller, in *IDIn) (ekman.NoContent, error) {
	if !pg.ValidID(in.ID) {
		return ekman.NoContent{}, pg.NotFound("sponsor")
	}
	_, err := s.db.Q(ctx).ExecContext(ctx, `DELETE FROM sponsors WHERE id = $1`, in.ID)
	return ekman.NoContent{}, err
}

// Published is the read side for site.Build: every sponsor, past and present (the site decides
// "active" at build time), highest tier first.
func (s *Service) Published(ctx context.Context) ([]Public, error) {
	rows, err := s.db.Q(ctx).QueryContext(ctx, `
		SELECT name, tier, logo_url, url, since, until FROM sponsors
		 ORDER BY array_position(ARRAY['platinum', 'gold', 'silver', 'partner'], tier), since, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Public{}
	for rows.Next() {
		var p Public
		if err := rows.Scan(&p.Name, &p.Tier, &p.LogoURL, &p.URL, &p.Since, &p.Until); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
