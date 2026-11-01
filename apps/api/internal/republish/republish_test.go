package republish_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/sakib412/ekman"
	"github.com/sakib412/ekman/auth"

	"github.com/BinarySUST/binarysust-web/apps/api/internal/republish"
)

func TestShouldPublish(t *testing.T) {
	plain := ekman.RouteInfo{Module: "events", Pattern: "/api/v1/events/{id}"}
	skipped := ekman.RouteInfo{Module: "events", Tags: []string{"republish:skip"}}
	framework := ekman.RouteInfo{Module: "ekman", Pattern: "/api/v1/auth/logout"}
	cases := []struct {
		method string
		status int
		ri     ekman.RouteInfo
		ok     bool
		want   bool
	}{
		{"PUT", 200, plain, true, true},
		{"POST", 201, plain, true, true},
		{"DELETE", 204, plain, true, true},
		{"GET", 200, plain, true, false},
		{"HEAD", 200, plain, true, false},
		{"PUT", 422, plain, true, false},
		{"PUT", 500, plain, true, false},
		{"POST", 302, plain, true, false},
		{"POST", 201, skipped, true, false},
		{"POST", 204, framework, true, false},
		{"POST", 404, ekman.RouteInfo{}, false, false},
	}
	for _, c := range cases {
		if got := republish.ShouldPublish(c.method, c.status, c.ri, c.ok); got != c.want {
			t.Errorf("%s %d %+v: got %v", c.method, c.status, c.ri, got)
		}
	}
}

type fake struct {
	mu       sync.Mutex
	touched  int
	notified []string
	err      error
}

func (f *fake) TouchContent(context.Context) (time.Time, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.touched++
	return time.Now(), nil
}

func (f *fake) Notify(_ context.Context, reason string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.notified = append(f.notified, reason)
	return f.err
}

func TestMiddleware(t *testing.T) {
	f := &fake{err: errors.New("github is down")}
	app, err := ekman.New(ekman.Options{Auth: &auth.Config{Store: auth.MemoryStore()}, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	app.Use(republish.Middleware(f, f, slog.New(slog.NewTextHandler(io.Discard, nil))))
	type in struct {
		ID string `path:"id"`
	}
	ok := func(context.Context, ekman.Caller, *in) (ekman.NoContent, error) { return ekman.NoContent{}, nil }
	fail := func(context.Context, ekman.Caller, *in) (ekman.NoContent, error) {
		return ekman.NoContent{}, ekman.Conflict("nope", "nope")
	}
	app.Register(ekman.Module{Name: "things", Routes: func(r *ekman.Router) {
		ekman.Handle(r, "PUT /things/{id}", ekman.Public(), ok)
		ekman.Handle(r, "POST /things/{id}/rsvp", ekman.Public(), ok, republish.Skip())
		ekman.Handle(r, "DELETE /things/{id}", ekman.Public(), fail)
		ekman.Handle(r, "GET /things/{id}", ekman.Public(), ok)
	}})
	h, err := app.Handler()
	if err != nil {
		t.Fatal(err)
	}
	do := func(method, path string) int {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
		return rec.Code
	}
	if code := do("PUT", "/api/v1/things/1"); code != 204 {
		t.Fatalf("a failing dispatch must not fail the write: %d", code)
	}
	do("POST", "/api/v1/things/1/rsvp")
	do("DELETE", "/api/v1/things/1")
	do("GET", "/api/v1/things/1")
	do("POST", "/api/v1/auth/logout")
	if f.touched != 1 || len(f.notified) != 1 || f.notified[0] != "PUT /api/v1/things/{id}" {
		t.Fatalf("touched %d, notified %v", f.touched, f.notified)
	}
}

func TestGitHubDispatch(t *testing.T) {
	var got struct {
		path, auth string
		body       map[string]any
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path, got.auth = r.URL.Path, r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&got.body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	g := &republish.GitHub{Token: "t0ken", Repository: "BinarySUST/binarysust-web", BaseURL: srv.URL}
	if err := g.Notify(context.Background(), "PUT /api/v1/events/{id}"); err != nil {
		t.Fatal(err)
	}
	if got.path != "/repos/BinarySUST/binarysust-web/dispatches" || got.auth != "Bearer t0ken" || got.body["event_type"] != "publish" {
		t.Fatalf("dispatch: %+v", got)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Bad credentials"}`, http.StatusUnauthorized)
	}))
	defer bad.Close()
	g.BaseURL = bad.URL
	if err := g.Notify(context.Background(), "x"); err == nil {
		t.Fatal("expected an error for 401")
	}
}
