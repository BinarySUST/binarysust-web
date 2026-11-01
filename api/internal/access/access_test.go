package access_test

import (
	"context"
	"slices"
	"testing"

	"github.com/BinarySUST/binarysust-web/api/internal/access"
	"github.com/BinarySUST/binarysust-web/api/internal/people"
	"github.com/BinarySUST/binarysust-web/api/internal/testapp"
)

// committee sets up two terms, makes the President position (reference data from the people
// migrations) confer admin, and returns ids.
func committee(t *testing.T, a *testapp.App) (oldTerm, newTerm, president, ann, bob string) {
	t.Helper()
	ann = a.QueryString(t, `INSERT INTO persons (handle, name, email) VALUES ('ann', 'Ann', 'ann@example.com') RETURNING id`)
	bob = a.QueryString(t, `INSERT INTO persons (handle, name, email) VALUES ('bob', 'Bob', 'bob@example.com') RETURNING id`)
	oldTerm = a.QueryString(t, `INSERT INTO terms (slug, label, starts_on) VALUES ('2025-26', '2025–26', '2025-10-01') RETURNING id`)
	newTerm = a.QueryString(t, `INSERT INTO terms (slug, label, starts_on) VALUES ('2026-27', '2026–27', '2026-10-01') RETURNING id`)
	president = a.QueryString(t, `SELECT id FROM positions WHERE slug = 'president'`)
	a.Exec(t, `INSERT INTO position_roles (position_id, role) VALUES ($1, 'admin')`, president)
	a.Exec(t, `INSERT INTO committee_seats (term_id, position_id, person_id) VALUES ($1, $2, $3)`, oldTerm, president, ann)
	a.Exec(t, `INSERT INTO committee_seats (term_id, position_id, person_id) VALUES ($1, $2, $3)`, newTerm, president, bob)
	return
}

func roles(t *testing.T, a *testapp.App, user string) []string {
	t.Helper()
	r, err := a.Svc.Access.RolesOf(context.Background(), user)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRolesFollowCommittee(t *testing.T) {
	a := testapp.New(t)
	oldTerm, newTerm, _, annID, _ := committee(t, a)
	ann := a.Account(t, "Ann@Example.com") // case-insensitive link to the person
	bob := a.Account(t, "bob@example.com")
	guest := a.Account(t, "guest@example.com")

	if r := roles(t, a, ann); len(r) != 0 {
		t.Fatalf("no current term yet, got %v", r)
	}
	if err := a.Svc.Access.SetCurrentTerm(context.Background(), oldTerm); err != nil {
		t.Fatal(err)
	}
	if r := roles(t, a, ann); !slices.Equal(r, []string{"admin"}) {
		t.Fatalf("ann in the current term: %v", r)
	}
	if r := roles(t, a, bob); len(r) != 0 {
		t.Fatalf("bob is seated next term only: %v", r)
	}

	// A term-scoped grant counts only while its term is current; a permanent one always.
	a.Exec(t, `INSERT INTO role_grants (person_id, role, term_id) VALUES ($1, 'blog_editor', $2)`, annID, oldTerm)
	a.Exec(t, `INSERT INTO role_grants (person_id, role) VALUES ($1, 'checkin_volunteer')`, annID)
	if r := roles(t, a, ann); !slices.Equal(r, []string{"admin", "blog_editor", "checkin_volunteer"}) {
		t.Fatalf("with grants: %v", r)
	}

	// Handover is one write: last year's committee loses its roles.
	if err := a.Svc.Access.SetCurrentTerm(context.Background(), newTerm); err != nil {
		t.Fatal(err)
	}
	if r := roles(t, a, ann); !slices.Equal(r, []string{"checkin_volunteer"}) {
		t.Fatalf("after handover ann keeps only the permanent grant: %v", r)
	}
	if r := roles(t, a, bob); !slices.Equal(r, []string{"admin"}) {
		t.Fatalf("after handover bob is admin: %v", r)
	}
	if r := roles(t, a, guest); len(r) != 0 {
		t.Fatalf("guest: %v", r)
	}
	if r := roles(t, a, "not-a-uuid"); r != nil {
		t.Fatalf("bad id: %v", r)
	}
}

func TestSeatEditsCannotEscalate(t *testing.T) {
	a := testapp.New(t)
	oldTerm, newTerm, president, _, _ := committee(t, a)
	if err := a.Svc.Access.SetCurrentTerm(context.Background(), oldTerm); err != nil {
		t.Fatal(err)
	}
	carol := a.QueryString(t, `INSERT INTO persons (handle, name) VALUES ('carol', 'Carol') RETURNING id`)
	body := map[string]string{"personId": carol, "positionId": president}

	pm := a.As(a.Account(t, "pm@example.com"), people.Manage)
	if r := pm.Do("POST", "/api/v1/terms/"+oldTerm+"/seats", body); r.Status() != 403 {
		t.Fatalf("people manager seating an admin position in the current term: %d %s", r.Status(), r.Body())
	}
	if r := pm.Do("POST", "/api/v1/terms/"+newTerm+"/seats", body); r.Status() != 201 {
		t.Fatalf("preparing next term's roster: %d %s", r.Status(), r.Body())
	}
	if r := pm.Do("POST", "/api/v1/terms/"+newTerm+"/make-current", nil); r.Status() != 403 {
		t.Fatalf("make-current needs roles.manage: %d", r.Status())
	}
	admin := a.As(a.Account(t, "root@example.com"), people.Manage, access.ManageRoles)
	if r := admin.Do("POST", "/api/v1/terms/"+oldTerm+"/seats", body); r.Status() != 201 {
		t.Fatalf("roles manager: %d %s", r.Status(), r.Body())
	}
	if r := admin.Do("POST", "/api/v1/terms/"+newTerm+"/make-current", nil); r.Status() != 200 {
		t.Fatalf("make-current: %d %s", r.Status(), r.Body())
	}
	var terms []people.TermRow
	pm.Do("GET", "/api/v1/terms", nil).JSON(&terms)
	if len(terms) != 2 || terms[0].Slug != "2026-27" || !terms[0].Current || terms[0].Seats != 2 {
		t.Fatalf("terms: %+v", terms)
	}
}

func TestGrantsAPI(t *testing.T) {
	a := testapp.New(t)
	_, _, president, annID, _ := committee(t, a)
	admin := a.As(a.Account(t, "root@example.com"), access.ManageRoles)

	if r := admin.Do("POST", "/api/v1/grants", map[string]any{"personId": annID, "role": "wizard"}); r.Status() != 422 {
		t.Fatalf("unknown role: %d", r.Status())
	}
	r := admin.Do("POST", "/api/v1/grants", map[string]any{"personId": annID, "role": "blog_editor"})
	if r.Status() != 201 {
		t.Fatalf("grant: %d %s", r.Status(), r.Body())
	}
	var g access.Grant
	r.JSON(&g)
	if g.PersonHandle != "ann" || g.TermID != nil {
		t.Fatalf("grant: %+v", g)
	}
	if r := admin.Do("POST", "/api/v1/grants", map[string]any{"personId": annID, "role": "blog_editor"}); r.Status() != 409 {
		t.Fatalf("duplicate grant: %d", r.Status())
	}
	if r := admin.Do("PUT", "/api/v1/positions/"+president+"/roles", map[string]any{"roles": []string{"admin", "event_manager"}}); r.Status() != 200 {
		t.Fatalf("position roles: %d %s", r.Status(), r.Body())
	}
	if r := admin.Do("DELETE", "/api/v1/grants/"+g.ID, nil); r.Status() != 204 {
		t.Fatalf("revoke: %d", r.Status())
	}
	// Roles never show rbac.All ("*"): admin is flagged and lists every declared permission.
	var infos []access.RoleInfo
	admin.Do("GET", "/api/v1/roles", nil).JSON(&infos)
	for _, ri := range infos {
		if slices.Contains(ri.Permissions, "*") {
			t.Fatalf("role %s leaks *: %v", ri.Name, ri.Permissions)
		}
		if ri.Name == "admin" && (!ri.AllPermissions || !slices.Contains(ri.Permissions, access.ManageRoles)) {
			t.Fatalf("admin: %+v", ri)
		}
		if ri.Name == "blog_editor" && ri.AllPermissions {
			t.Fatalf("blog_editor: %+v", ri)
		}
	}
	if r := a.As(a.Account(t, "x@example.com")).Do("GET", "/api/v1/grants", nil); r.Status() != 403 {
		t.Fatalf("grants need roles.manage: %d", r.Status())
	}

	me := a.As(a.Account(t, "ann@example.com"))
	var acct access.Account
	me.Do("GET", "/api/v1/me", nil).JSON(&acct)
	if acct.PersonHandle == nil || *acct.PersonHandle != "ann" {
		t.Fatalf("me: %+v", acct)
	}
}
