package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sakib412/ekman/db"
	"github.com/sakib412/ekman/ekmantest"
)

// kvFake is Cloudflare's Workers KV REST API for one namespace, recording every write in order.
type kvFake struct {
	mu     sync.Mutex
	vals   map[string]string
	writes []string // "key ttl"
	down   bool
}

const kvValues = "/accounts/acc/storage/kv/namespaces/ns/values/"

func (f *kvFake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.down || !strings.HasPrefix(r.URL.Path, kvValues) || r.Header.Get("Authorization") != "Bearer kv-token" {
		http.Error(w, `{"success":false}`, http.StatusInternalServerError)
		return
	}
	key := strings.TrimPrefix(r.URL.Path, kvValues)
	switch r.Method {
	case http.MethodPut:
		b, _ := io.ReadAll(r.Body)
		f.vals[key] = string(b)
		f.writes = append(f.writes, strings.TrimSpace(key+" "+r.URL.Query().Get("expiration_ttl")))
	case http.MethodGet:
		v, ok := f.vals[key]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, v)
	}
}

func (f *kvFake) takeWrites() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	w := f.writes
	f.writes = nil
	return w
}

func (f *kvFake) setDown(down bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.down = down
}

// dispatchFake is GitHub's repository_dispatch endpoint.
type dispatchFake struct {
	mu   sync.Mutex
	sent []map[string]any // client_payload of each dispatch
}

func (f *dispatchFake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		EventType     string         `json:"event_type"`
		ClientPayload map[string]any `json:"client_payload"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.URL.Path == "/repos/BinarySUST/binarysust-web/dispatches" && body.EventType == "publish" {
		f.sent = append(f.sent, body.ClientPayload)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (f *dispatchFake) take() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.sent
	f.sent = nil
	return s
}

type publishRig struct {
	srv      *httptest.Server
	admin    *http.Client
	server   *Server
	kv       *kvFake
	dispatch *dispatchFake
	stop     func() // stops the publisher, as SIGTERM does
}

// newPublishRig boots the real wiring against Postgres, with KV and GitHub faked, an admin
// signed in, and the publisher running (its boot publish already done).
func newPublishRig(t *testing.T, kv *kvFake) *publishRig { return boot(t, kv, nil) }

// newPublishRigOn boots a second API process over prev's database and KV (a restart).
func newPublishRigOn(t *testing.T, kv *kvFake, prev *publishRig) *publishRig {
	return boot(t, kv, prev.server.Svc.Sources.DB)
}

func boot(t *testing.T, kv *kvFake, d *db.DB) *publishRig {
	t.Helper()
	if d == nil {
		d = ekmantest.DB(t, "pgx")
	}
	if kv == nil {
		kv = &kvFake{vals: map[string]string{}}
	}
	gh := &dispatchFake{}
	kvSrv, ghSrv := httptest.NewServer(kv), httptest.NewServer(gh)
	t.Cleanup(kvSrv.Close)
	t.Cleanup(ghSrv.Close)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	cfg := LoadConfig(env(map[string]string{
		"DEV_ADMIN_EMAIL":          "admin@binarysust.local",
		"GITHUB_DISPATCH_TOKEN":    "gh-token",
		"GITHUB_REPOSITORY":        "BinarySUST/binarysust-web",
		"CLOUDFLARE_ACCOUNT_ID":    "acc",
		"SNAPSHOT_KV_NAMESPACE_ID": "ns",
		"SNAPSHOT_KV_TOKEN":        "kv-token",
	}))
	cfg.PublicURL = "http://" + ln.Addr().String()
	cfg.cloudflareAPI, cfg.githubAPI = kvSrv.URL, ghSrv.URL
	s, err := NewApp(cfg, d, quiet)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.App.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	h, err := s.App.Handler()
	if err != nil {
		t.Fatal(err)
	}
	srv := &httptest.Server{Listener: ln, Config: &http.Server{Handler: h}}
	srv.Start()
	t.Cleanup(srv.Close)

	for _, q := range []string{
		`DELETE FROM role_grants`, `DELETE FROM persons WHERE handle = 'dev-admin'`,
		`INSERT INTO persons (handle, name, email, listed) VALUES ('dev-admin', 'Dev Admin', 'admin@binarysust.local', false)`,
		`INSERT INTO role_grants (person_id, role) SELECT id, 'admin' FROM persons WHERE handle = 'dev-admin'`,
	} {
		if _, err := d.SQL().Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	jar, _ := cookiejar.New(nil)
	admin := &http.Client{Jar: jar, CheckRedirect: func(r *http.Request, _ []*http.Request) error {
		if strings.HasSuffix(r.URL.Path, "/dashboard") {
			return http.ErrUseLastResponse
		}
		return nil
	}}
	resp, err := admin.Get(srv.URL + "/api/v1/auth/dev/start?next=/dashboard")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	ctx, cancel := context.WithCancel(context.Background())
	s.Publisher.Start(ctx)
	stop := func() {
		cancel()
		sctx, scancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer scancel()
		s.Publisher.Stop(sctx)
	}
	t.Cleanup(stop)
	r := &publishRig{srv: srv, admin: admin, server: s, kv: kv, dispatch: gh, stop: stop}
	r.idle(t)
	return r
}

func (r *publishRig) idle(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := r.server.Publisher.Wait(ctx); err != nil {
		t.Fatalf("publisher never went idle: %v", err)
	}
}

func (r *publishRig) do(t *testing.T, client *http.Client, method, path string, body any) int {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, r.srv.URL+path, rd)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// previewHash is the hash the next build would publish, from the dashboard's preview endpoint.
func (r *publishRig) previewHash(t *testing.T) string {
	t.Helper()
	resp, err := r.admin.Get(r.srv.URL + "/api/v1/site/preview")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var snap struct {
		Version struct {
			Hash string `json:"hash"`
		} `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil || snap.Version.Hash == "" {
		t.Fatalf("preview: %v", err)
	}
	return snap.Version.Hash
}

func sponsor(name string) map[string]any {
	return map[string]any{"name": name, "tier": "gold", "logoUrl": "https://example.com/logo.png", "url": "https://example.com", "since": 2024}
}

func TestWriteStoresSnapshotThenDispatches(t *testing.T) {
	r := newPublishRig(t, nil)
	r.kv.takeWrites()
	r.dispatch.take()

	if code := r.do(t, r.admin, "POST", "/api/v1/sponsors", sponsor("Acme")); code != http.StatusCreated {
		t.Fatalf("create sponsor: %d", code)
	}
	r.idle(t)

	hash := r.previewHash(t)
	writes := r.kv.takeWrites()
	if len(writes) != 2 || writes[0] != "snapshot:"+hash+" 2592000" || writes[1] != "latest" {
		t.Fatalf("writes %v, want snapshot:%s with a 30-day TTL, then latest", writes, hash)
	}
	var stored struct {
		Version  struct{ Hash string }   `json:"version"`
		Sponsors []struct{ Name string } `json:"sponsors"`
	}
	if err := json.Unmarshal([]byte(r.kv.vals["snapshot:"+hash]), &stored); err != nil ||
		stored.Version.Hash != hash || len(stored.Sponsors) != 1 || stored.Sponsors[0].Name != "Acme" {
		t.Fatalf("stored snapshot: %+v %v", stored, err)
	}
	var latest struct {
		Hash    string    `json:"hash"`
		BuiltAt time.Time `json:"builtAt"`
	}
	if err := json.Unmarshal([]byte(r.kv.vals["latest"]), &latest); err != nil || latest.Hash != hash || latest.BuiltAt.IsZero() {
		t.Fatalf("latest: %s", r.kv.vals["latest"])
	}
	sent := r.dispatch.take()
	if len(sent) != 1 || sent[0]["hash"] != hash || sent[0]["reason"] != "POST /api/v1/sponsors" {
		t.Fatalf("dispatches %v", sent)
	}
}

func TestBootPublishesOnlyWhenContentDiffersFromKV(t *testing.T) {
	kv := &kvFake{vals: map[string]string{}}
	r := newPublishRig(t, kv)
	hash := r.previewHash(t)
	if w := kv.takeWrites(); len(w) != 2 || w[0] != "snapshot:"+hash+" 2592000" {
		t.Fatalf("first boot writes %v", w)
	}
	if sent := r.dispatch.take(); len(sent) != 1 || sent[0]["reason"] != "boot" || sent[0]["hash"] != hash {
		t.Fatalf("first boot dispatches %v", sent)
	}

	// A restart with KV already holding this content: nothing to do.
	r.stop()
	again := newPublishRigOn(t, kv, r)
	if w := kv.takeWrites(); len(w) != 0 {
		t.Fatalf("restart writes %v", w)
	}
	if sent := again.dispatch.take(); len(sent) != 0 {
		t.Fatalf("restart dispatches %v", sent)
	}
}

func TestBurstOfWritesPublishesOnce(t *testing.T) {
	r := newPublishRig(t, nil)
	r.kv.takeWrites()
	r.dispatch.take()
	// Hold the worker in its first run so the rest of the burst queues behind it.
	r.kv.mu.Lock()
	var wg sync.WaitGroup
	for _, name := range []string{"A", "B", "C", "D"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if code := r.do(t, r.admin, "POST", "/api/v1/sponsors", sponsor(name)); code != http.StatusCreated {
				t.Errorf("create %s: %d", name, code)
			}
		}()
	}
	wg.Wait() // every write answered while KV was blocked: publishing never delays the editor
	r.kv.mu.Unlock()
	r.idle(t)
	hash := r.previewHash(t)
	sent := r.dispatch.take()
	if len(sent) == 0 || len(sent) > 2 || sent[len(sent)-1]["hash"] != hash {
		t.Fatalf("a burst of 4 writes dispatched %v; want at most 2 (the run in progress, then one for the rest)", sent)
	}
}

func TestUnchangedContentPublishesNothing(t *testing.T) {
	r := newPublishRig(t, nil)
	r.kv.takeWrites()
	r.dispatch.take()
	// A write that changes nothing public (say, an unlisted person's profile) signals the publisher
	// exactly as the middleware does; the snapshot hash is unchanged.
	_ = r.server.Publisher.Notify(context.Background(), "PUT /api/v1/people/{handle}")
	r.idle(t)
	if w, s := r.kv.takeWrites(), r.dispatch.take(); len(w) != 0 || len(s) != 0 {
		t.Fatalf("unchanged content: writes %v, dispatches %v", w, s)
	}
}

func TestKVFailureNeverFailsTheWriteAndRetriesNextTime(t *testing.T) {
	r := newPublishRig(t, nil)
	r.kv.takeWrites()
	r.dispatch.take()
	r.kv.setDown(true)
	if code := r.do(t, r.admin, "POST", "/api/v1/sponsors", sponsor("Acme")); code != http.StatusCreated {
		t.Fatalf("a KV outage must not fail the write: %d", code)
	}
	r.idle(t)
	if s := r.dispatch.take(); len(s) != 0 {
		t.Fatalf("dispatched without a stored snapshot: %v", s)
	}
	// KV is back; the next trigger (here the nightly run) publishes the same content.
	r.kv.setDown(false)
	_ = r.server.Publisher.Notify(context.Background(), "nightly")
	r.idle(t)
	if s := r.dispatch.take(); len(s) != 1 || s[0]["hash"] != r.previewHash(t) {
		t.Fatalf("retry dispatches %v", s)
	}
}

// main turns each republish.Daily tick (00:05 Asia/Dhaka, see TestNextRollover) into this Notify.
func TestNightlyRunRepublishesOnlyWhenContentChanged(t *testing.T) {
	r := newPublishRig(t, nil)
	r.kv.takeWrites()
	r.dispatch.take()
	nightly := func() { _ = r.server.Publisher.Notify(context.Background(), "nightly") }
	// Nothing changed overnight: the run costs nothing.
	nightly()
	r.idle(t)
	if w, s := r.kv.takeWrites(), r.dispatch.take(); len(w) != 0 || len(s) != 0 {
		t.Fatalf("idle night: writes %v, dispatches %v", w, s)
	}
	// Content changed (here directly in the database, as a date rollover would change the build).
	if _, err := r.server.Svc.Sources.DB.SQL().Exec(`INSERT INTO sponsors (name, tier, logo_url, url, since) VALUES ('Night', 'gold', 'https://example.com/l.png', '', 2024)`); err != nil {
		t.Fatal(err)
	}
	nightly()
	r.idle(t)
	if s := r.dispatch.take(); len(s) != 1 || s[0]["reason"] != "nightly" || s[0]["hash"] != r.previewHash(t) {
		t.Fatalf("nightly dispatches %v", s)
	}
}

func TestRepublishNowIsAdminOnlyAndAlwaysDispatches(t *testing.T) {
	r := newPublishRig(t, nil)
	r.kv.takeWrites()
	r.dispatch.take()
	anon := &http.Client{}
	if code := r.do(t, anon, "POST", "/api/v1/site/republish", nil); code != http.StatusUnauthorized {
		t.Fatalf("anonymous republish: %d", code)
	}
	if code := r.do(t, r.admin, "POST", "/api/v1/site/republish", nil); code != http.StatusAccepted {
		t.Fatalf("admin republish: %d", code)
	}
	r.idle(t)
	hash := r.previewHash(t)
	if s := r.dispatch.take(); len(s) != 1 || s[0]["hash"] != hash || s[0]["reason"] != "POST /api/v1/site/republish" {
		t.Fatalf("republish dispatches %v (content unchanged, so only a forced run dispatches)", s)
	}
}

func TestWithoutKVWritesDispatchAsBefore(t *testing.T) {
	gh := &dispatchFake{}
	ghSrv := httptest.NewServer(gh)
	t.Cleanup(ghSrv.Close)
	d := ekmantest.DB(t, "pgx")
	cfg := LoadConfig(env(map[string]string{"GITHUB_DISPATCH_TOKEN": "gh-token", "GITHUB_REPOSITORY": "BinarySUST/binarysust-web"}))
	cfg.githubAPI = ghSrv.URL
	s, err := NewApp(cfg, d, quiet)
	if err != nil {
		t.Fatal(err)
	}
	if s.Publisher != nil {
		t.Fatal("no SNAPSHOT_KV_*: there must be no publisher")
	}
}
