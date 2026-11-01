package people_test

import (
	"strings"
	"testing"

	"github.com/BinarySUST/binarysust-web/api/internal/people"
	"github.com/BinarySUST/binarysust-web/api/internal/testapp"
)

func TestConstitutionPositions(t *testing.T) {
	a := testapp.New(t)
	pm := a.As("pm", people.Manage)
	var ps []people.Position
	pm.Do("GET", "/api/v1/positions", nil).JSON(&ps)
	bySlug := map[string]people.Position{}
	for _, p := range ps {
		bySlug[p.Slug] = p
	}
	want := map[string]people.Group{"president": people.GroupLeadership, "director-of-training": people.GroupLeadership,
		"assistant-general-secretary": people.GroupAssistant, "it-design-secretary": people.GroupSecretary,
		"executive-member": people.GroupMember, "chief-patron": people.GroupPatron, "adviser": people.GroupAdviser}
	for slug, g := range want {
		if bySlug[slug].Group != g {
			t.Errorf("%s: group %q, want %q", slug, bySlug[slug].Group, g)
		}
	}
	if len(ps) != 24 || ps[0].Slug != "president" || !bySlug["general-secretary"].Lifetime || bySlug["vice-president"].Lifetime {
		t.Fatalf("positions: %d, first %s", len(ps), ps[0].Slug)
	}
	// A new position defaults its group from its category.
	var np people.Position
	r := pm.Do("POST", "/api/v1/positions", people.PositionIn{Slug: "patron-emeritus", Title: "Patron emeritus", Category: people.AdvisorCat, Rank: 40})
	r.JSON(&np)
	if r.Status() != 201 || np.Group != people.GroupAdviser {
		t.Fatalf("create position: %d %s", r.Status(), r.Body())
	}
	if r := pm.Do("POST", "/api/v1/positions", people.PositionIn{Slug: "x-y", Title: "X", Category: "executive", Group: "boss"}); r.Status() != 422 {
		t.Fatalf("unknown group: %d", r.Status())
	}
}

func TestSeatsGroupsLifetimeAndDelete(t *testing.T) {
	a := testapp.New(t)
	pm := a.As("pm", people.Manage)
	term := a.QueryString(t, `INSERT INTO terms (slug, label, starts_on) VALUES ('2023-24', '2023–24', '2023-10-01') RETURNING id`)
	person := func(handle string) string {
		return a.QueryString(t, `INSERT INTO persons (handle, name, member_since, graduated_on) VALUES ($1, $1, '2019-11-01', '2024-06-30') RETURNING id`, handle)
	}
	ann, bob, cat := person("ann"), person("bob"), person("cat")
	// Two people may hold the same position in one term.
	for _, seat := range [][2]string{{ann, "director-of-training"}, {bob, "executive-member"}, {cat, "executive-member"}} {
		pos := a.QueryString(t, `SELECT id FROM positions WHERE slug = $1`, seat[1])
		if r := pm.Do("POST", "/api/v1/terms/"+term+"/seats", people.SeatIn{PersonID: seat[0], PositionID: pos}); r.Status() != 201 {
			t.Fatalf("seat %s: %d %s", seat[1], r.Status(), r.Body())
		}
	}
	pub, err := a.Svc.People.Published(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pub.People {
		if p.Status != people.StatusAlumni || p.LifetimeMember != (p.Handle == "ann") {
			t.Errorf("%s: status %s lifetime %v", p.Handle, p.Status, p.LifetimeMember)
		}
	}
	if len(pub.Committees) != 1 || len(pub.Committees[0].Seats) != 3 || pub.Committees[0].Seats[0].Group != people.GroupLeadership ||
		pub.Committees[0].Seats[2].Group != people.GroupMember {
		t.Fatalf("committee: %+v", pub.Committees)
	}

	// Deleting: a seat holder is a 409 with a readable message; someone with no history goes.
	r := pm.Do("DELETE", "/api/v1/people/"+ann, nil)
	if r.Status() != 409 || !strings.Contains(string(r.Body()), "committee seat") {
		t.Fatalf("delete seat holder: %d %s", r.Status(), r.Body())
	}
	writer := a.QueryString(t, `INSERT INTO persons (handle, name) VALUES ('writer', 'Writer') RETURNING id`)
	a.Exec(t, `INSERT INTO posts (slug, author_id, title) VALUES ('hello', $1, 'Hello')`, writer)
	if r := pm.Do("DELETE", "/api/v1/people/"+writer, nil); r.Status() != 409 || !strings.Contains(string(r.Body()), "blog posts") {
		t.Fatalf("delete author: %d %s", r.Status(), r.Body())
	}
	loner := a.QueryString(t, `INSERT INTO persons (handle, name) VALUES ('loner', 'Loner') RETURNING id`)
	if r := pm.Do("DELETE", "/api/v1/people/"+loner, nil); r.Status() != 204 {
		t.Fatalf("delete: %d %s", r.Status(), r.Body())
	}
	if r := pm.Do("DELETE", "/api/v1/people/"+loner, nil); r.Status() != 404 {
		t.Fatalf("delete twice: %d", r.Status())
	}
}

func TestLookup(t *testing.T) {
	a := testapp.New(t)
	a.Exec(t, `INSERT INTO persons (handle, name) VALUES ('tahmid', 'Tahmid Rahman'), ('rakib', 'Rakibul Islam'), ('ab', 'Under_Score')`)
	a.Exec(t, `INSERT INTO persons (handle, name, listed) VALUES ('hidden', 'Hidden Rahman', false)`)
	find := func(perms []string, q string) []string {
		t.Helper()
		c := a.As("someone")
		if perms != nil {
			c = a.As("pm", people.Manage)
		}
		r := c.Do("GET", "/api/v1/people/lookup?q="+q, nil)
		if r.Status() != 200 {
			t.Fatalf("lookup %q: %d %s", q, r.Status(), r.Body())
		}
		var refs []people.PersonRef
		r.JSON(&refs)
		var out []string
		for _, p := range refs {
			out = append(out, p.Handle)
		}
		return out
	}
	if got := find(nil, "rahman"); len(got) != 1 || got[0] != "tahmid" {
		t.Fatalf("by name, listed only: %v", got)
	}
	if got := find([]string{"pm"}, "rahman"); len(got) != 2 {
		t.Fatalf("people managers see unlisted people: %v", got)
	}
	if got := find(nil, "%40rak"); len(got) != 1 || got[0] != "rakib" {
		t.Fatalf("by @handle: %v", got)
	}
	if got := find(nil, "_"); len(got) != 1 || got[0] != "ab" {
		t.Fatalf("LIKE wildcards are literal: %v", got)
	}
	if r := a.Do("GET", "/api/v1/people/lookup?q=x", nil); r.Status() != 401 {
		t.Fatalf("anonymous: %d", r.Status())
	}
}
