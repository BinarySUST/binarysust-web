package sponsors_test

import (
	"testing"

	"github.com/BinarySUST/binarysust-web/apps/api/internal/sponsors"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/testapp"
)

// The template test for a module: access rules, validation, CRUD and the public read side.
func TestSponsors(t *testing.T) {
	a := testapp.New(t)
	mgr := a.As("sm", sponsors.Manage)
	in := sponsors.In{Name: "Surma Cloud", Tier: "gold", LogoURL: "https://media.test/logos/x.png", URL: "https://example.com", Since: 2024}

	if r := a.Do("POST", "/api/v1/sponsors", in); r.Status() != 401 {
		t.Fatalf("anonymous: %d", r.Status())
	}
	if r := a.As("member").Do("POST", "/api/v1/sponsors", in); r.Status() != 403 {
		t.Fatalf("without sponsors.manage: %d", r.Status())
	}
	bad := in
	bad.Tier, bad.Until = "diamond", new(int)
	if r := mgr.Do("POST", "/api/v1/sponsors", bad); r.Status() != 422 {
		t.Fatalf("invalid: %d %s", r.Status(), r.Body())
	}
	r := mgr.Do("POST", "/api/v1/sponsors", in)
	if r.Status() != 201 {
		t.Fatalf("create: %d %s", r.Status(), r.Body())
	}
	var s sponsors.Sponsor
	r.JSON(&s)

	in.Tier = "platinum"
	if r := mgr.Do("PUT", "/api/v1/sponsors/"+s.ID, in); r.Status() != 200 {
		t.Fatalf("update: %d %s", r.Status(), r.Body())
	}
	pub, err := a.Svc.Sponsors.Published(t.Context())
	if err != nil || len(pub) != 1 || pub[0].Tier != "platinum" || pub[0].Until != nil {
		t.Fatalf("published: %v %+v", err, pub)
	}
	if r := mgr.Do("DELETE", "/api/v1/sponsors/"+s.ID, nil); r.Status() != 204 {
		t.Fatalf("delete: %d", r.Status())
	}
	if r := mgr.Do("GET", "/api/v1/sponsors/"+s.ID, nil); r.Status() != 404 {
		t.Fatalf("gone: %d", r.Status())
	}
	if r := mgr.Do("GET", "/api/v1/sponsors/not-an-id", nil); r.Status() != 404 {
		t.Fatalf("bad id: %d", r.Status())
	}
}
