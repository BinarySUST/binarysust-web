package republish_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BinarySUST/binarysust-web/apps/api/internal/republish"
)

// fakeKV is Cloudflare's Workers KV REST API for one namespace, in memory.
type fakeKV struct {
	vals  map[string]string
	ttls  map[string]string
	auths []string
	paths []string
	down  bool
}

const kvPrefix = "/accounts/acc/storage/kv/namespaces/ns/values/"

func newFakeKV() *fakeKV { return &fakeKV{vals: map[string]string{}, ttls: map[string]string{}} }

func (f *fakeKV) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.auths = append(f.auths, r.Header.Get("Authorization"))
	f.paths = append(f.paths, r.Method+" "+r.URL.EscapedPath())
	if f.down {
		http.Error(w, `{"success":false}`, http.StatusInternalServerError)
		return
	}
	if len(r.URL.Path) <= len(kvPrefix) || r.URL.Path[:len(kvPrefix)] != kvPrefix {
		http.NotFound(w, r)
		return
	}
	key := r.URL.Path[len(kvPrefix):]
	switch r.Method {
	case http.MethodPut:
		b, _ := io.ReadAll(r.Body)
		f.vals[key] = string(b)
		f.ttls[key] = r.URL.Query().Get("expiration_ttl")
		_, _ = w.Write([]byte(`{"success":true}`))
	case http.MethodGet:
		v, ok := f.vals[key]
		if !ok {
			http.Error(w, `{"success":false,"errors":[{"code":10009,"message":"key not found"}]}`, http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(v))
	}
}

func TestKV(t *testing.T) {
	f := newFakeKV()
	srv := httptest.NewServer(f)
	defer srv.Close()
	kv := &republish.KV{AccountID: "acc", NamespaceID: "ns", Token: "t0ken", BaseURL: srv.URL}
	ctx := context.Background()

	if err := kv.Put(ctx, "snapshot:abc", []byte(`{"a":1}`), 30*24*time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := kv.Put(ctx, "latest", []byte(`{"hash":"abc"}`), 0); err != nil {
		t.Fatal(err)
	}
	if f.vals["snapshot:abc"] != `{"a":1}` || f.ttls["snapshot:abc"] != "2592000" {
		t.Fatalf("stored %q with ttl %q", f.vals["snapshot:abc"], f.ttls["snapshot:abc"])
	}
	if f.ttls["latest"] != "" {
		t.Fatalf("latest must not expire, ttl %q", f.ttls["latest"])
	}
	got, err := kv.Get(ctx, "latest")
	if err != nil || string(got) != `{"hash":"abc"}` {
		t.Fatalf("get: %q %v", got, err)
	}
	for _, a := range f.auths {
		if a != "Bearer t0ken" {
			t.Fatalf("auth header %q", a)
		}
	}

	if _, err := kv.Get(ctx, "missing"); !errors.Is(err, republish.ErrNotFound) {
		t.Fatalf("missing key: %v", err)
	}
	f.down = true
	if err := kv.Put(ctx, "latest", []byte("x"), 0); err == nil || errors.Is(err, republish.ErrNotFound) {
		t.Fatalf("server error must be an error, got %v", err)
	}
}

func TestKVTimeout(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-block }))
	defer srv.Close()
	defer close(block)
	kv := &republish.KV{AccountID: "acc", NamespaceID: "ns", Token: "t", BaseURL: srv.URL,
		Client: &http.Client{Timeout: 50 * time.Millisecond}}
	if err := kv.Put(context.Background(), "k", []byte("v"), 0); err == nil {
		t.Fatal("expected a timeout")
	}
}

func TestNextRollover(t *testing.T) {
	dhaka := time.FixedZone("Asia/Dhaka", 6*60*60)
	at := func(s string) time.Time {
		v, err := time.ParseInLocation("2006-01-02 15:04", s, dhaka)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	cases := map[string]string{
		"2026-11-20 00:04": "2026-11-20 00:05",
		"2026-11-20 00:05": "2026-11-21 00:05", // strictly after: a run at 00:05 waits a day
		"2026-11-20 13:00": "2026-11-21 00:05",
		"2026-12-31 23:59": "2027-01-01 00:05",
	}
	for now, want := range cases {
		// The VPS clock is UTC: the club's day still turns over in Dhaka.
		if got := republish.NextRollover(at(now).UTC(), dhaka); !got.Equal(at(want)) {
			t.Errorf("after %s: got %s, want %s", now, got.In(dhaka).Format("2006-01-02 15:04"), want)
		}
	}
}
