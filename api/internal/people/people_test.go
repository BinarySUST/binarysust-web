package people_test

import (
	"testing"

	"github.com/BinarySUST/binarysust-web/api/internal/people"
	"github.com/BinarySUST/binarysust-web/api/internal/testapp"
)

func TestPortfolioAndPeople(t *testing.T) {
	a := testapp.New(t)
	pm := a.As("pm", people.Manage)
	r := pm.Do("POST", "/api/v1/people", people.PersonIn{Handle: "Tahmid-R", Name: "Tahmid Rahman", Email: ptr("Tahmid@Example.com"),
		Listed: true, MemberSince: ptr("2021-11-01"), Batch: ptr(2021)})
	if r.Status() != 201 {
		t.Fatalf("create person: %d %s", r.Status(), r.Body())
	}
	var p people.Person
	r.JSON(&p)
	if p.Handle != "tahmid-r" || *p.Email != "tahmid@example.com" || p.Status != people.StatusMember || *p.MemberSince != "2021-11-01" {
		t.Fatalf("person: %+v", p)
	}
	if r := pm.Do("POST", "/api/v1/people", people.PersonIn{Handle: "tahmid-r", Name: "Dup"}); r.Status() != 409 {
		t.Fatalf("duplicate handle: %d", r.Status())
	}
	if r := pm.Do("POST", "/api/v1/people", people.PersonIn{Handle: "x", Name: ""}); r.Status() != 422 {
		t.Fatalf("invalid: %d", r.Status())
	}

	// The account links by email, whenever it was created.
	me := a.As(a.Account(t, "tahmid@example.com"), people.EditOwn)
	put := people.Portfolio{Headline: "Gopher", BioMarkdown: "Hi *there*", Skills: []string{"Go"},
		Socials:  []people.Social{{Kind: "github", URL: "https://github.com/x"}},
		Projects: []people.Project{{Title: "ekman", Summary: "framework", RepoURL: ptr("https://github.com/sakib412/ekman")}}}
	r = me.Do("PUT", "/api/v1/me/portfolio", put)
	if r.Status() != 200 {
		t.Fatalf("put portfolio: %d %s", r.Status(), r.Body())
	}
	var got people.Portfolio
	me.Do("GET", "/api/v1/me/portfolio", nil).JSON(&got)
	if got.Headline != "Gopher" || len(got.Projects) != 1 || got.Projects[0].Tags == nil || got.Achievements == nil {
		t.Fatalf("portfolio: %+v", got)
	}
	bad := put
	bad.Socials = []people.Social{{Kind: "myspace", URL: "ftp://x"}}
	if r := me.Do("PUT", "/api/v1/me/portfolio", bad); r.Status() != 422 {
		t.Fatalf("bad socials: %d", r.Status())
	}
	if r := a.As(a.Account(t, "nobody@example.com"), people.EditOwn).Do("GET", "/api/v1/me/portfolio", nil); r.Status() != 403 || r.ErrorCode() != "not_a_member" {
		t.Fatalf("no profile: %d %s", r.Status(), r.ErrorCode())
	}

	// Renaming a handle leaves a redirect behind.
	upd := people.PersonIn{Handle: "tahmid", Name: "Tahmid Rahman", Email: ptr("tahmid@example.com"), Listed: true, MemberSince: ptr("2021-11-01")}
	if r := pm.Do("PUT", "/api/v1/people/"+p.ID, upd); r.Status() != 200 {
		t.Fatalf("rename: %d %s", r.Status(), r.Body())
	}
	pub, err := a.Svc.People.Published(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(pub.People) != 1 || pub.People[0].Handle != "tahmid" || pub.People[0].BioHTML != "<p>Hi <em>there</em></p>" {
		t.Fatalf("public people: %+v", pub.People)
	}
	if len(pub.Redirects) != 1 || pub.Redirects[0].From != "/people/tahmid-r" || pub.Redirects[0].To != "/people/tahmid" {
		t.Fatalf("redirects: %+v", pub.Redirects)
	}
}

func ptr[T any](v T) *T { return &v }
