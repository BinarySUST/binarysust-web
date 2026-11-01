package site_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sakib412/ekman"
	"github.com/sakib412/ekman/ekmantest"

	"github.com/BinarySUST/binarysust-web/api/internal/demo"
	"github.com/BinarySUST/binarysust-web/api/internal/site"
	"github.com/BinarySUST/binarysust-web/api/internal/testapp"
)

func TestStateOf(t *testing.T) {
	changed := time.Date(2026, 11, 2, 10, 0, 0, 123456789, time.UTC)
	if site.StateOf(changed, nil) != "publishing" {
		t.Fatal("unknown live version is publishing")
	}
	if site.StateOf(changed, &site.LiveVersion{ContentChangedAt: changed.Add(-time.Second)}) != "publishing" {
		t.Fatal("older deploy is publishing")
	}
	if site.StateOf(changed, &site.LiveVersion{ContentChangedAt: changed.Truncate(time.Microsecond)}) != "live" {
		t.Fatal("same change is live")
	}
}

// keys collects, for every JSON path, the union of object keys seen there.
func keys(v any, path string, out map[string][]string) {
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			if !slices.Contains(out[path], k) {
				out[path] = append(out[path], k)
			}
			keys(child, path+"."+k, out)
		}
	case []any:
		for _, child := range x {
			keys(child, path+"[]", out)
		}
	}
}

func TestSnapshotMatchesContract(t *testing.T) {
	a := testapp.New(t)
	ctx := context.Background()
	for range 2 { // the seed is idempotent
		if err := a.DB.Tx(ctx, func(ctx context.Context) error {
			return demo.Seed(ctx, a.DB.Q(ctx), time.Now().UTC().Truncate(time.Hour), "admin@binarysust.local")
		}); err != nil {
			t.Fatal(err)
		}
	}
	snap, err := site.Build(ctx, a.Svc.Sources, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if snap.CurrentTerm == nil || snap.CurrentTerm.Slug != "2025-26" || len(snap.Committees) != 2 || len(snap.Terms) != 2 {
		t.Fatalf("committee data: %+v %d %d", snap.CurrentTerm, len(snap.Committees), len(snap.Terms))
	}
	executive, groups := 0, map[string]int{}
	for _, st := range snap.Committees[0].Seats {
		groups[string(st.Group)]++
		if st.Category == "executive" {
			executive++
		}
	}
	if executive < 20 || groups["patron"] != 1 || groups["adviser"] < 5 || groups["assistant"] == 0 {
		t.Fatalf("current committee: %d executive seats, groups %v", executive, groups)
	}
	statuses := map[string]int{}
	for _, p := range snap.People {
		statuses[string(p.Status)]++
		if p.Handle == "dev-admin" {
			t.Fatal("unlisted people must not be public")
		}
	}
	for _, s := range []string{"committee", "advisor", "member", "alumni", "guest"} {
		if statuses[s] == 0 {
			t.Errorf("no %s in the demo data: %v", s, statuses)
		}
	}
	for _, e := range snap.Events {
		if e.Status == "draft" {
			t.Fatal("drafts must not be public")
		}
	}
	if len(snap.Posts) != 3 || len(snap.Talks) != 3 || len(snap.Sponsors) != 4 || len(snap.Redirects) != 1 {
		t.Fatalf("counts: posts %d talks %d sponsors %d redirects %d", len(snap.Posts), len(snap.Talks), len(snap.Sponsors), len(snap.Redirects))
	}
	again, _ := site.Build(ctx, a.Svc.Sources, time.Now().Add(time.Hour))
	if again.Version.Hash != snap.Version.Hash || len(snap.Version.Hash) != 64 {
		t.Fatalf("hash must depend on content only: %s vs %s", snap.Version.Hash, again.Version.Hash)
	}

	raw, _ := json.Marshal(snap)
	var got any
	_ = json.Unmarshal(raw, &got)
	gotKeys := map[string][]string{}
	keys(got, "", gotKeys)

	fixture, err := os.ReadFile("../../../web/.snapshot/fixture.json")
	if err != nil {
		t.Skipf("no web fixture to compare against: %v", err)
	}
	var want any
	if err := json.Unmarshal(fixture, &want); err != nil {
		t.Fatal(err)
	}
	wantKeys := map[string][]string{}
	keys(want, "", wantKeys)
	// docs/snapshot.md is the contract; where the fixture is thinner than the doc, the doc wins.
	if !slices.Contains(wantKeys[".posts[]"], "coverUrl") { // older fixtures omit PublicPost.coverUrl
		wantKeys[".posts[]"] = append(wantKeys[".posts[]"], "coverUrl")
	}
	if _, ok := wantKeys[".events[].photos[]"]; !ok { // fixture has no photos at all
		wantKeys[".events[].photos[]"] = []string{"url", "caption"}
	}
	for path, wk := range wantKeys {
		gk, ok := gotKeys[path]
		if !ok {
			continue // e.g. an empty list in our data
		}
		sort.Strings(wk)
		sort.Strings(gk)
		if !slices.Equal(wk, gk) {
			t.Errorf("keys at %q differ:\n  fixture %s\n  api     %s", path, strings.Join(wk, ","), strings.Join(gk, ","))
		}
	}
	for path := range gotKeys {
		if _, ok := wantKeys[path]; !ok {
			t.Errorf("api has %q, the fixture does not", path)
		}
	}
}

func TestStatusReadsDeployedVersion(t *testing.T) {
	a := testapp.New(t)
	changed, err := a.Svc.Access.TouchContent(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var deployed atomic.Value
	deployed.Store(changed.Add(-time.Minute))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/version.json" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"contentChangedAt": deployed.Load(), "builtAt": deployed.Load(), "hash": "x"})
	}))
	defer srv.Close()

	svc := site.New(a.Svc.Sources, srv.URL)
	st := status(t, svc)
	if st.State != "publishing" || st.Live == nil {
		t.Fatalf("older deploy: %+v", st)
	}
	deployed.Store(changed)
	if st := status(t, svc); st.State != "live" {
		t.Fatalf("deployed the latest change: %+v", st)
	}
	if st := status(t, site.New(a.Svc.Sources, "http://127.0.0.1:1")); st.Live != nil || st.State != "publishing" {
		t.Fatalf("unreachable site: %+v", st)
	}
}

func status(t *testing.T, svc *site.Service) site.Status {
	t.Helper()
	r := ekmantest.New(t, ekman.Options{}, svc.Module()).As("someone").Do("GET", "/api/v1/site/status", nil)
	if r.Status() != 200 {
		t.Fatalf("status: %d %s", r.Status(), r.Body())
	}
	var st site.Status
	r.JSON(&st)
	return st
}
