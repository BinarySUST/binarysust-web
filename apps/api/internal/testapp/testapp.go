// Package testapp runs the whole API against a throwaway Postgres schema for integration tests.
// Tests skip unless EKMAN_TEST_DATABASE_URL is set, e.g.
//
//	EKMAN_TEST_DATABASE_URL=postgres://binary:binary@localhost:5433/binarysust?sslmode=disable go test ./...
//
// Each test gets its own schema (ekmantest.DB), so the dev data in the same database is untouched.
package testapp

import (
	"context"
	"sync"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/sakib412/ekman"
	"github.com/sakib412/ekman/auth"
	"github.com/sakib412/ekman/db"
	"github.com/sakib412/ekman/ekmantest"
	"github.com/sakib412/ekman/mail"
	"github.com/sakib412/ekman/rbac"

	"github.com/BinarySUST/binarysust-web/apps/api/internal/club"
)

// App is a migrated test app.
type App struct {
	*ekmantest.Client
	Svc  *club.Services
	DB   *db.DB
	Mail *Outbox
}

// New migrates a fresh schema and serves every module.
func New(t testing.TB) *App {
	t.Helper()
	d := ekmantest.DB(t, "pgx")
	box := &Outbox{}
	svc := club.New(d, club.Deps{Mail: box, PublicURL: "http://site.test"})
	c := ekmantest.New(t, ekman.Options{DB: d, Roles: club.Roles, Baseline: club.Baseline, Grants: svc.Access}, svc.Modules()...)
	if err := c.App().Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return &App{Client: c, Svc: svc, DB: d, Mail: box}
}

// Account returns the account (user) id for email, creating it directly and linking it to the
// person with that email, if any. It bypasses the Directory, which refuses uninvited emails, so
// tests can have guests; SignIn goes through the real Directory.
func (a *App) Account(t testing.TB, email string) string {
	t.Helper()
	id := a.QueryString(t, `
		INSERT INTO accounts (email, name) VALUES (lower($1), $1)
		ON CONFLICT (email) DO UPDATE SET last_login_at = now()
		RETURNING id`, email)
	if err := a.Svc.Access.LinkPersons(context.Background()); err != nil {
		t.Fatal(err)
	}
	return id
}

// SignIn signs email in through the real Directory, as Google sign-in does.
func (a *App) SignIn(email string) (string, error) {
	return a.Svc.Access.UserFor(context.Background(), auth.Identity{Provider: "test", Subject: email, Email: email, EmailVerified: true, Name: email})
}

// AsUser runs as an account with permissions resolved the production way: baseline plus the
// roles RolesOf returns (seats in the current term and grants), plus extra.
func (a *App) AsUser(t testing.TB, userID string, extra ...rbac.Permission) *ekmantest.Client {
	t.Helper()
	roles, err := a.Svc.Access.RolesOf(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	perms := extra
	for _, name := range roles {
		for _, r := range club.Roles {
			if r.Name == name {
				perms = append(perms, r.Permissions...)
			}
		}
	}
	return a.As(userID, perms...)
}

// Exec runs SQL in the test schema.
func (a *App) Exec(t testing.TB, q string, args ...any) {
	t.Helper()
	if _, err := a.DB.SQL().ExecContext(context.Background(), q, args...); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}

// QueryString returns one text value.
func (a *App) QueryString(t testing.TB, q string, args ...any) string {
	t.Helper()
	var s string
	if err := a.DB.SQL().QueryRowContext(context.Background(), q, args...).Scan(&s); err != nil {
		t.Fatalf("query %q: %v", q, err)
	}
	return s
}

// Outbox records sent mail.
type Outbox struct {
	mu   sync.Mutex
	sent []mail.Message
}

// Send implements mail.Sender.
func (o *Outbox) Send(_ context.Context, m mail.Message) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.sent = append(o.sent, m)
	return nil
}

// Sent returns a copy of the sent messages.
func (o *Outbox) Sent() []mail.Message {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]mail.Message(nil), o.sent...)
}
