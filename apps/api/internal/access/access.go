// Package access owns sign-in identities (accounts) and who holds which role.
//
// Roles follow the committee. A person's roles are, at read time:
//
//	roles of the positions they hold a seat in, in the CURRENT term (position_roles)
//	∪ grants that are permanent or scoped to the current term (role_grants)
//
// site_settings.current_term_id is the only pointer to the current term, so "make 2026-27
// current" is the whole handover: last year's committee loses dashboard access on its next
// request. Only roles.manage edits position_roles, grants and the pointer.
//
// Service implements ekman's rbac.GrantSource (RolesOf) and auth.Directory (UserFor).
package access

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/sakib412/ekman"
	"github.com/sakib412/ekman/auth"
	"github.com/sakib412/ekman/db"
	"github.com/sakib412/ekman/rbac"

	"github.com/BinarySUST/binarysust-web/apps/api/internal/pg"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/republish"
)

//go:embed migrations/*.sql
var migrations embed.FS

// ManageRoles edits position roles, grants and the current term.
const ManageRoles rbac.Permission = "roles.manage"

// ErrNotMember is returned when a signed-in account has no club profile.
var ErrNotMember = &ekman.Error{Status: http.StatusForbidden, Code: "not_a_member",
	Message: "your account is not linked to a club profile; ask a people manager to add your email"}

// Role names a role defined in code (club.Roles). Its OpenAPI enum is the role table.
type Role string

// Service is the access module.
type Service struct {
	db       *db.DB
	roles    []rbac.Role
	declared []rbac.Permission // every permission a module declares; expands rbac.All
}

// New returns the access service. roles is club.Roles: grants and position roles must name one.
func New(d *db.DB, roles []rbac.Role) *Service { return &Service{db: d, roles: roles} }

// SetDeclared records every permission the modules declare, so GET /roles can spell out
// rbac.All. club.New calls it once the module list exists.
func (s *Service) SetDeclared(perms []rbac.Permission) { s.declared = slices.Clone(perms) }

// RoleNames lists the roles in table order.
func RoleNames(roles []rbac.Role) []string {
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		out = append(out, r.Name)
	}
	return out
}

var (
	_ rbac.GrantSource = (*Service)(nil)
	_ auth.Directory   = (*Service)(nil)
)

// Module wires access into the app.
func (s *Service) Module() ekman.Module {
	return ekman.Module{
		Name:        "access",
		Migrations:  migrations,
		Permissions: []rbac.Permission{ManageRoles},
		Routes: func(r *ekman.Router) {
			ekman.Handle(r, "GET /me", ekman.SignedIn(), s.me, ekman.Summary("The signed-in account and its linked person"))
			ekman.Handle(r, "GET /roles", ekman.Requires(ManageRoles), s.listRoles, ekman.Summary("Roles defined in code"))
			ekman.Handle(r, "GET /position-roles", ekman.Requires(ManageRoles), s.listPositionRoles)
			ekman.Handle(r, "PUT /positions/{id}/roles", ekman.Requires(ManageRoles), s.setPositionRoles, republish.Skip())
			ekman.Handle(r, "GET /grants", ekman.Requires(ManageRoles), s.listGrants)
			ekman.Handle(r, "POST /grants", ekman.Requires(ManageRoles), s.createGrant, republish.Skip())
			ekman.Handle(r, "DELETE /grants/{id}", ekman.Requires(ManageRoles), s.deleteGrant, republish.Skip())
			// Changes statuses and the current committee on the public site: republishes.
			ekman.Handle(r, "POST /terms/{id}/make-current", ekman.Requires(ManageRoles), s.makeCurrent,
				ekman.Status(http.StatusOK), ekman.Summary("Hand over: make this term current"))
		},
	}
}

// RolesOf implements rbac.GrantSource with one query over seats and grants.
func (s *Service) RolesOf(ctx context.Context, userID string) ([]string, error) {
	if !pg.ValidID(userID) {
		return nil, nil
	}
	rows, err := s.db.Q(ctx).QueryContext(ctx, `
		SELECT pr.role
		  FROM accounts a
		  JOIN committee_seats cs ON cs.person_id = a.person_id
		  JOIN site_settings ss   ON ss.current_term_id = cs.term_id
		  JOIN position_roles pr  ON pr.position_id = cs.position_id
		 WHERE a.id = $1
		UNION
		SELECT g.role
		  FROM accounts a
		  JOIN role_grants g ON g.person_id = a.person_id
		 CROSS JOIN site_settings ss
		 WHERE a.id = $1 AND (g.term_id IS NULL OR g.term_id = ss.current_term_id)
		 ORDER BY 1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// UserFor implements auth.Directory: any verified email gets an account, linked to the person
// with the same email when there is one.
func (s *Service) UserFor(ctx context.Context, id auth.Identity) (string, error) {
	if !id.EmailVerified || id.Email == "" {
		return "", ekman.Forbidden("sign in with an account whose email address is verified")
	}
	var accountID string
	err := s.db.Tx(ctx, func(ctx context.Context) error {
		var avatar *string
		if id.AvatarURL != "" {
			avatar = &id.AvatarURL
		}
		if err := s.db.Q(ctx).QueryRowContext(ctx, `
			INSERT INTO accounts (email, name, avatar_url) VALUES ($1, $2, $3)
			ON CONFLICT (email) DO UPDATE
			   SET name = CASE WHEN EXCLUDED.name <> '' THEN EXCLUDED.name ELSE accounts.name END,
			       avatar_url = coalesce(EXCLUDED.avatar_url, accounts.avatar_url),
			       last_login_at = now()
			RETURNING id`, strings.ToLower(strings.TrimSpace(id.Email)), id.Name, avatar).Scan(&accountID); err != nil {
			return err
		}
		return s.LinkPersons(ctx)
	})
	return accountID, err
}

// LinkPersons points every account at the person with the same email (or at none). The people
// module calls it after changing emails; sign-in calls it too.
func (s *Service) LinkPersons(ctx context.Context) error {
	return s.db.Tx(ctx, func(ctx context.Context) error {
		if _, err := s.db.Q(ctx).ExecContext(ctx, `
			UPDATE accounts a SET person_id = NULL
			 WHERE person_id IS NOT NULL
			   AND NOT EXISTS (SELECT 1 FROM persons p WHERE p.id = a.person_id AND p.email = a.email)`); err != nil {
			return err
		}
		_, err := s.db.Q(ctx).ExecContext(ctx, `
			UPDATE accounts a SET person_id = p.id
			  FROM persons p
			 WHERE p.email = a.email AND a.person_id IS DISTINCT FROM p.id`)
		return err
	})
}

// PersonOf returns the person id linked to an account, or ErrNotMember.
func (s *Service) PersonOf(ctx context.Context, accountID string) (string, error) {
	if !pg.ValidID(accountID) {
		return "", ErrNotMember
	}
	var pid sql.NullString
	err := s.db.Q(ctx).QueryRowContext(ctx, `SELECT person_id FROM accounts WHERE id = $1`, accountID).Scan(&pid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !pid.Valid) {
		return "", ErrNotMember
	}
	return pid.String, err
}

// ConfersRoles reports whether holding positionID in the current term hands out any role.
func (s *Service) ConfersRoles(ctx context.Context, positionID string) (bool, error) {
	if !pg.ValidID(positionID) {
		return false, nil
	}
	var ok bool
	err := s.db.Q(ctx).QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM position_roles WHERE position_id = $1)`, positionID).Scan(&ok)
	return ok, err
}

// CurrentTermID returns the current term's id, or "" when none is set.
func (s *Service) CurrentTermID(ctx context.Context) (string, error) {
	var id sql.NullString
	err := s.db.Q(ctx).QueryRowContext(ctx, `SELECT current_term_id FROM site_settings`).Scan(&id)
	return id.String, err
}

// TouchContent records that public content changed now (republish.Recorder).
func (s *Service) TouchContent(ctx context.Context) (time.Time, error) {
	var t time.Time
	err := s.db.Q(ctx).QueryRowContext(ctx,
		`UPDATE site_settings SET content_changed_at = now() RETURNING content_changed_at`).Scan(&t)
	return t.UTC(), err
}

// ContentChangedAt is when public content last changed.
func (s *Service) ContentChangedAt(ctx context.Context) (time.Time, error) {
	var t time.Time
	err := s.db.Q(ctx).QueryRowContext(ctx, `SELECT content_changed_at FROM site_settings`).Scan(&t)
	return t.UTC(), err
}

// SetCurrentTerm moves the current-term pointer (the yearly handover).
func (s *Service) SetCurrentTerm(ctx context.Context, termID string) error {
	if !pg.ValidID(termID) {
		return pg.NotFound("term")
	}
	res, err := s.db.Q(ctx).ExecContext(ctx,
		`UPDATE site_settings SET current_term_id = t.id FROM terms t WHERE t.id = $1`, termID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return pg.NotFound("term")
	}
	return nil
}

// Account is a signed-in identity.
type Account struct {
	ID           string  `json:"id"`
	Email        string  `json:"email"`
	Name         string  `json:"name"`
	AvatarURL    *string `json:"avatarUrl"`
	PersonID     *string `json:"personId"`
	PersonHandle *string `json:"personHandle"`
}

func (s *Service) me(ctx context.Context, c ekman.Caller, _ *struct{}) (Account, error) {
	var a Account
	if !pg.ValidID(c.UserID()) {
		return a, pg.NotFound("account")
	}
	err := s.db.Q(ctx).QueryRowContext(ctx, `
		SELECT a.id, a.email, a.name, a.avatar_url, p.id, p.handle
		  FROM accounts a LEFT JOIN persons p ON p.id = a.person_id
		 WHERE a.id = $1`, c.UserID()).Scan(&a.ID, &a.Email, &a.Name, &a.AvatarURL, &a.PersonID, &a.PersonHandle)
	return a, err
}

// RoleInfo describes a role defined in code. A role holding every permission (admin) has
// AllPermissions set, and Permissions lists them all, so clients never see rbac.All ("*").
type RoleInfo struct {
	Name           Role              `json:"name"`
	Title          string            `json:"title"`
	Permissions    []rbac.Permission `json:"permissions"`
	AllPermissions bool              `json:"allPermissions"`
}

func (s *Service) listRoles(_ context.Context, _ ekman.Caller, _ *struct{}) ([]RoleInfo, error) {
	out := make([]RoleInfo, 0, len(s.roles))
	for _, r := range s.roles {
		info := RoleInfo{Name: Role(r.Name), Title: r.Title, Permissions: []rbac.Permission{}}
		if slices.Contains(r.Permissions, rbac.All) {
			info.AllPermissions = true
			info.Permissions = slices.Sorted(slices.Values(s.declared))
		} else if r.Permissions != nil {
			info.Permissions = slices.Clone(r.Permissions)
		}
		info.Permissions = slices.Compact(info.Permissions)
		out = append(out, info)
	}
	return out, nil
}

func (s *Service) knownRole(name Role) bool {
	return slices.ContainsFunc(s.roles, func(r rbac.Role) bool { return r.Name == string(name) })
}

// PositionRoles lists the roles a position confers.
type PositionRoles struct {
	PositionID string `json:"positionId"`
	Roles      []Role `json:"roles"`
}

func (s *Service) listPositionRoles(ctx context.Context, _ ekman.Caller, _ *struct{}) ([]PositionRoles, error) {
	rows, err := s.db.Q(ctx).QueryContext(ctx, `
		SELECT p.id, coalesce(array_agg(pr.role ORDER BY pr.role) FILTER (WHERE pr.role IS NOT NULL), '{}')
		  FROM positions p LEFT JOIN position_roles pr ON pr.position_id = p.id
		 GROUP BY p.id, p.rank, p.title ORDER BY p.rank, p.title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PositionRoles
	for rows.Next() {
		var p PositionRoles
		var roles []string
		if err := rows.Scan(&p.PositionID, pg.Array(&roles)); err != nil {
			return nil, err
		}
		p.Roles = make([]Role, 0, len(roles))
		for _, r := range roles {
			p.Roles = append(p.Roles, Role(r))
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SetPositionRolesIn replaces a position's roles.
type SetPositionRolesIn struct {
	ID    string `path:"id"`
	Roles []Role `json:"roles"`
}

func (s *Service) setPositionRoles(ctx context.Context, _ ekman.Caller, in *SetPositionRolesIn) (PositionRoles, error) {
	if !pg.ValidID(in.ID) {
		return PositionRoles{}, pg.NotFound("position")
	}
	var v ekman.Validation
	for _, r := range in.Roles {
		v.Check(s.knownRole(r), "roles", "unknown role "+string(r))
	}
	if err := v.Err(); err != nil {
		return PositionRoles{}, err
	}
	roles := slices.Compact(slices.Sorted(slices.Values(in.Roles)))
	err := s.db.Tx(ctx, func(ctx context.Context) error {
		var exists bool
		if err := s.db.Q(ctx).QueryRowContext(ctx, `SELECT true FROM positions WHERE id = $1`, in.ID).Scan(&exists); err != nil {
			return err
		}
		if _, err := s.db.Q(ctx).ExecContext(ctx, `DELETE FROM position_roles WHERE position_id = $1`, in.ID); err != nil {
			return err
		}
		for _, r := range roles {
			if _, err := s.db.Q(ctx).ExecContext(ctx, `INSERT INTO position_roles (position_id, role) VALUES ($1, $2)`, in.ID, r); err != nil {
				return err
			}
		}
		return nil
	})
	if roles == nil {
		roles = []Role{}
	}
	return PositionRoles{PositionID: in.ID, Roles: roles}, err
}

// Grant is an explicit role grant. TermID nil means permanent.
type Grant struct {
	ID           string    `json:"id"`
	PersonID     string    `json:"personId"`
	PersonHandle string    `json:"personHandle"`
	PersonName   string    `json:"personName"`
	Role         Role      `json:"role"`
	TermID       *string   `json:"termId"`
	TermSlug     *string   `json:"termSlug"`
	CreatedAt    time.Time `json:"createdAt"`
}

const grantSelect = `
	SELECT g.id, p.id, p.handle, p.name, g.role, t.id, t.slug, g.created_at
	  FROM role_grants g JOIN persons p ON p.id = g.person_id LEFT JOIN terms t ON t.id = g.term_id`

func scanGrant(sc interface{ Scan(...any) error }) (Grant, error) {
	var g Grant
	err := sc.Scan(&g.ID, &g.PersonID, &g.PersonHandle, &g.PersonName, &g.Role, &g.TermID, &g.TermSlug, &g.CreatedAt)
	g.CreatedAt = g.CreatedAt.UTC()
	return g, err
}

func (s *Service) listGrants(ctx context.Context, _ ekman.Caller, _ *struct{}) ([]Grant, error) {
	rows, err := s.db.Q(ctx).QueryContext(ctx, grantSelect+` ORDER BY p.name, g.role`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Grant
	for rows.Next() {
		g, err := scanGrant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// GrantIn creates a grant.
type GrantIn struct {
	PersonID string  `json:"personId"`
	Role     Role    `json:"role"`
	TermID   *string `json:"termId"` // null: permanent
}

// Validate implements ekman.Validator.
func (in *GrantIn) Validate(v *ekman.Validation) {
	v.Check(pg.ValidID(in.PersonID), "personId", "must be a person id")
	v.Check(in.TermID == nil || pg.ValidID(*in.TermID), "termId", "must be a term id or null")
}

func (s *Service) createGrant(ctx context.Context, c ekman.Caller, in *GrantIn) (Grant, error) {
	if !s.knownRole(in.Role) {
		return Grant{}, &ekman.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_input",
			Message: "unknown role", Fields: map[string]string{"role": "unknown role " + string(in.Role)}}
	}
	var by *string
	if pg.ValidID(c.UserID()) {
		u := c.UserID()
		by = &u
	}
	var id string
	err := s.db.Q(ctx).QueryRowContext(ctx, `
		INSERT INTO role_grants (person_id, role, term_id, granted_by) VALUES ($1, $2, $3, $4)
		RETURNING id`, in.PersonID, in.Role, in.TermID, by).Scan(&id)
	if err != nil {
		return Grant{}, pg.Friendly(err, "this person already holds that grant")
	}
	return scanGrant(s.db.Q(ctx).QueryRowContext(ctx, grantSelect+` WHERE g.id = $1`, id))
}

// IDIn is a path id.
type IDIn struct {
	ID string `path:"id"`
}

func (s *Service) deleteGrant(ctx context.Context, _ ekman.Caller, in *IDIn) (ekman.NoContent, error) {
	if !pg.ValidID(in.ID) {
		return ekman.NoContent{}, pg.NotFound("grant")
	}
	_, err := s.db.Q(ctx).ExecContext(ctx, `DELETE FROM role_grants WHERE id = $1`, in.ID)
	return ekman.NoContent{}, err // deleting twice is fine
}

// CurrentTerm is the result of a handover.
type CurrentTerm struct {
	TermID string `json:"termId"`
}

func (s *Service) makeCurrent(ctx context.Context, _ ekman.Caller, in *IDIn) (CurrentTerm, error) {
	return CurrentTerm{TermID: in.ID}, s.SetCurrentTerm(ctx, in.ID)
}
