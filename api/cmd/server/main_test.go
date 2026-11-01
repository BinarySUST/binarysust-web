package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/sakib412/ekman"
	"github.com/sakib412/ekman/db"
	"github.com/sakib412/ekman/ekmantest"

	"github.com/BinarySUST/binarysust-web/api/internal/club"
	"github.com/BinarySUST/binarysust-web/api/internal/republish"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

// serve boots the real wiring. The database is opened lazily, so routes that never touch it
// work without one.
func serve(t *testing.T, cfg Config, d *db.DB) *httptest.Server {
	t.Helper()
	if d == nil {
		var err error
		if d, err = db.Open("pgx", "postgres://nobody@127.0.0.1:1/none"); err != nil {
			t.Fatal(err)
		}
	}
	app, _, err := NewApp(cfg, d, quiet)
	if err != nil {
		t.Fatal(err)
	}
	h, err := app.Handler()
	if err != nil {
		t.Fatalf("boot: %v", err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

func noRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

func TestDevSignInOnlyOnLocalOrigins(t *testing.T) {
	cases := []struct {
		publicURL, email string
		want             int
	}{
		{"http://binarysust.localhost", "admin@binarysust.local", http.StatusFound},
		{"http://localhost:8080", "admin@binarysust.local", http.StatusFound},
		{"https://binarysust.org", "admin@binarysust.local", http.StatusNotFound},
		{"http://binarysust.localhost.evil.com", "admin@binarysust.local", http.StatusNotFound},
		{"", "admin@binarysust.local", http.StatusNotFound},
		{"http://binarysust.localhost", "", http.StatusNotFound},
	}
	for _, c := range cases {
		cfg := LoadConfig(env(map[string]string{"PUBLIC_URL": c.publicURL, "DEV_ADMIN_EMAIL": c.email}))
		srv := serve(t, cfg, nil)
		client := &http.Client{CheckRedirect: noRedirect}
		resp, err := client.Get(srv.URL + "/api/v1/auth/dev/start")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != c.want {
			t.Errorf("PUBLIC_URL=%q DEV_ADMIN_EMAIL=%q: got %d, want %d", c.publicURL, c.email, resp.StatusCode, c.want)
		}
		if c.want == http.StatusFound && !strings.Contains(resp.Header.Get("Location"), "/api/v1/auth/dev/callback?") {
			t.Errorf("redirect: %s", resp.Header.Get("Location"))
		}
	}
}

func TestBootAndOpenAPI(t *testing.T) {
	lazy, _ := db.Open("pgx", "postgres://nobody@127.0.0.1:1/none")
	app, _, err := NewApp(LoadConfig(env(nil)), lazy, quiet)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := club.OpenAPI(app)
	if err != nil {
		t.Fatalf("openapi: %v", err)
	}
	var spec struct {
		Paths      map[string]any `json:"paths"`
		Components struct {
			Schemas map[string]json.RawMessage `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(doc, &spec); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"/api/v1/site/preview", "/api/v1/site/status", "/api/v1/events/{slug}/availability", "/api/v1/me/portfolio", "/api/v1/media"} {
		if _, ok := spec.Paths[p]; !ok {
			t.Errorf("missing path %s", p)
		}
	}
	for _, s := range []string{"site.Snapshot", "people.PublicPerson", "events.PublicEvent", "blog.PublicPost", "talks.Public", "sponsors.Public"} {
		if _, ok := spec.Components.Schemas[s]; !ok {
			t.Errorf("missing schema %s", s)
		}
	}
	var perms struct {
		Enum []string `json:"enum"`
	}
	_ = json.Unmarshal(spec.Components.Schemas["Permission"], &perms)
	want := []string{"blog.review", "blog.write", "events.checkin", "events.manage", "media.upload", "people.manage",
		"profile.edit_own", "roles.manage", "sponsors.manage", "talks.manage"}
	if !slices.Equal(perms.Enum, want) {
		t.Errorf("permissions: %v", perms.Enum)
	}
	// String types with an Enum method are named enums, and fields point at them.
	for name, first := range map[string]string{"people.Group": "patron", "people.Status": "committee", "events.Kind": "workshop",
		"blog.State": "draft", "sponsors.Tier": "platinum", "access.Role": "admin", "media.PurposeName": "avatar"} {
		var e struct {
			Enum []string `json:"enum"`
		}
		_ = json.Unmarshal(spec.Components.Schemas[name], &e)
		if len(e.Enum) == 0 || e.Enum[0] != first {
			t.Errorf("enum %s: %v", name, e.Enum)
		}
	}
	if !strings.Contains(string(spec.Components.Schemas["people.Seat"]), `"#/components/schemas/people.Group"`) {
		t.Errorf("people.Seat.group is not the enum: %s", spec.Components.Schemas["people.Seat"])
	}
	// The editor's live preview writes nothing, so it must never republish.
	for _, ri := range app.Routes() {
		if ri.Method == "POST" && ri.Pattern == "/api/v1/posts/preview" && republish.ShouldPublish("POST", 200, ri, true) {
			t.Error("POST /posts/preview republishes")
		}
	}
}

// TestDevSignInFlow signs in through the real ekman session flow against Postgres and checks
// that the seeded permanent grant makes the dev admin an admin.
func TestDevSignInFlow(t *testing.T) {
	d := ekmantest.DB(t, "pgx")
	cfg := LoadConfig(env(map[string]string{"DEV_ADMIN_EMAIL": "Admin@BinarySUST.local"}))
	// PUBLIC_URL must be the server's own origin for the redirects to land.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	cfg.PublicURL = "http://" + ln.Addr().String()
	app, _, err := NewApp(cfg, d, quiet)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	h, err := app.Handler()
	if err != nil {
		t.Fatal(err)
	}
	srv := &httptest.Server{Listener: ln, Config: &http.Server{Handler: h}}
	srv.Start()
	t.Cleanup(srv.Close)
	if _, err := d.SQL().Exec(`INSERT INTO persons (handle, name, email, listed) VALUES ('dev-admin', 'Dev Admin', 'admin@binarysust.local', false)`); err != nil {
		t.Fatal(err)
	}
	if _, err := d.SQL().Exec(`INSERT INTO role_grants (person_id, role) SELECT id, 'admin' FROM persons WHERE handle = 'dev-admin'`); err != nil {
		t.Fatal(err)
	}

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, CheckRedirect: func(r *http.Request, _ []*http.Request) error {
		if strings.HasSuffix(r.URL.Path, "/dashboard") {
			return http.ErrUseLastResponse // the web app is not running here
		}
		return nil
	}}
	resp, err := client.Get(srv.URL + "/api/v1/auth/dev/start?next=/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != srv.URL+"/dashboard" {
		t.Fatalf("callback: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp, err = client.Get(srv.URL + "/api/v1/auth/me")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var me ekman.Me
	_ = json.NewDecoder(resp.Body).Decode(&me)
	if me.UserID == "" || !slices.Contains(me.Roles, "admin") || !slices.Contains(me.Permissions, "roles.manage") {
		t.Fatalf("me: %+v", me)
	}
}
