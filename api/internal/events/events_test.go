package events_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/BinarySUST/binarysust-web/api/internal/events"
	"github.com/BinarySUST/binarysust-web/api/internal/testapp"
)

func newEvent(t *testing.T, a *testapp.App, slug string, capacity *int) {
	t.Helper()
	now := time.Now().UTC()
	in := events.EventInput{Slug: slug, Title: "Go workshop", Kind: "workshop", Status: events.Published,
		Summary: "s", BodyMarkdown: "**hi**", StartsAt: now.Add(48 * time.Hour), EndsAt: now.Add(50 * time.Hour),
		Venue:        ptr("Lab 3"),
		Registration: &events.Registration{OpensAt: now.Add(-time.Hour), ClosesAt: now.Add(24 * time.Hour), Capacity: capacity}}
	r := a.As("manager", events.Manage).Do("POST", "/api/v1/events", in)
	if r.Status() != 201 {
		t.Fatalf("create event: %d %s", r.Status(), r.Body())
	}
}

func ptr[T any](v T) *T { return &v }

func TestRSVPWaitlistAndPromotion(t *testing.T) {
	a := testapp.New(t)
	newEvent(t, a, "go-workshop", ptr(2))
	ann, bob, cat := a.Account(t, "ann@example.com"), a.Account(t, "bob@example.com"), a.Account(t, "cat@example.com")

	rsvp := func(user string) events.MyRSVP {
		t.Helper()
		r := a.As(user).Do("POST", "/api/v1/events/go-workshop/rsvp", nil)
		if r.Status() != 200 {
			t.Fatalf("rsvp %s: %d %s", user, r.Status(), r.Body())
		}
		var m events.MyRSVP
		r.JSON(&m)
		return m
	}
	first := rsvp(ann)
	if first.Status != events.Confirmed || first.Ticket == nil {
		t.Fatalf("ann: %+v", first)
	}
	if again := rsvp(ann); *again.Ticket != *first.Ticket {
		t.Fatal("RSVP is not idempotent")
	}
	rsvp(bob)
	third := rsvp(cat)
	if third.Status != events.Waitlisted || third.Position == nil || *third.Position != 1 || third.Ticket != nil {
		t.Fatalf("cat should be waitlisted #1: %+v", third)
	}

	r := a.Do("GET", "/api/v1/events/go-workshop/availability", nil)
	var av events.Availability
	r.JSON(&av)
	if r.Header().Get("Cache-Control") != "no-store" || av.Confirmed != 2 || av.Waitlisted != 1 || av.Mine != nil || av.State != events.WindowFull {
		t.Fatalf("availability: %+v %v", av, r.Header())
	}
	if r := a.Do("POST", "/api/v1/events/go-workshop/rsvp", nil); r.Status() != 401 {
		t.Fatalf("anonymous rsvp: %d", r.Status())
	}

	// Ann cancels: Cat is promoted and told so.
	if r := a.As(ann).Do("DELETE", "/api/v1/events/go-workshop/rsvp", nil); r.Status() != 204 {
		t.Fatalf("cancel: %d %s", r.Status(), r.Body())
	}
	if r := a.As(ann).Do("DELETE", "/api/v1/events/go-workshop/rsvp", nil); r.Status() != 204 {
		t.Fatalf("cancel twice: %d", r.Status())
	}
	a.As(cat).Do("GET", "/api/v1/events/go-workshop/availability", nil).JSON(&av)
	if av.Mine == nil || av.Mine.Status != events.Confirmed || av.Mine.Ticket == nil || av.Confirmed != 2 || av.Waitlisted != 0 {
		t.Fatalf("after promotion: %+v mine=%+v", av, av.Mine)
	}
	promoted := false
	for _, m := range a.Mail.Sent() {
		if m.To[0] == "cat@example.com" && strings.HasPrefix(m.Subject, "A seat opened up") {
			promoted = true
		}
	}
	if !promoted {
		t.Fatalf("no promotion mail: %+v", a.Mail.Sent())
	}

	// Ann comes back: the event is full again, so she joins the end of the waitlist.
	if m := rsvp(ann); m.Status != events.Waitlisted {
		t.Fatalf("re-register: %+v", m)
	}

	// Check-in.
	vol := a.As("volunteer", events.CheckIn)
	in := map[string]string{"ticket": strings.ToLower(*av.Mine.Ticket)}
	var res events.CheckInResult
	r = vol.Do("POST", "/api/v1/events/go-workshop/checkin", in)
	r.JSON(&res)
	if r.Status() != 200 || res.AlreadyCheckedIn || res.Email != "cat@example.com" {
		t.Fatalf("check-in: %d %+v", r.Status(), res)
	}
	vol.Do("POST", "/api/v1/events/go-workshop/checkin", in).JSON(&res)
	if !res.AlreadyCheckedIn {
		t.Fatal("second scan should report the first check-in")
	}
	if r := vol.Do("POST", "/api/v1/events/go-workshop/checkin", map[string]string{"ticket": "NOPE"}); r.Status() != 404 {
		t.Fatalf("unknown ticket: %d", r.Status())
	}
	if r := a.As(cat).Do("POST", "/api/v1/events/go-workshop/checkin", in); r.Status() != 403 {
		t.Fatalf("check-in needs events.checkin: %d", r.Status())
	}

	// Registrants export and my RSVPs.
	csv := a.As("manager", events.Manage).Do("GET", "/api/v1/events/go-workshop/registrants.csv", nil)
	lines := strings.Split(strings.TrimSpace(string(csv.Body())), "\n")
	if csv.Status() != 200 || len(lines) != 4 || !strings.HasPrefix(lines[0], "name,email") || !strings.Contains(lines[3], "ann@example.com,,waitlisted") {
		t.Fatalf("csv: %d\n%s", csv.Status(), csv.Body())
	}
	var mine []events.RSVPItem
	a.As(cat).Do("GET", "/api/v1/me/rsvps", nil).JSON(&mine)
	if len(mine) != 1 || mine[0].EventSlug != "go-workshop" || mine[0].Status != events.Confirmed {
		t.Fatalf("my rsvps: %+v", mine)
	}

	// Raising capacity promotes the waitlist.
	var ev []events.Event
	a.As("manager", events.Manage).Do("GET", "/api/v1/events", nil).JSON(&ev)
	e := ev[0]
	up := events.EventInput{Slug: e.Slug, Title: e.Title, Kind: e.Kind, Status: e.Status, Summary: e.Summary,
		BodyMarkdown: e.BodyMarkdown, StartsAt: e.StartsAt, EndsAt: e.EndsAt, Venue: e.Venue, Registration: e.Registration}
	up.Registration.Capacity = ptr(3)
	if r := a.As("manager", events.Manage).Do("PUT", "/api/v1/events/"+e.ID, up); r.Status() != 200 {
		t.Fatalf("update: %d %s", r.Status(), r.Body())
	}
	a.As(ann).Do("GET", "/api/v1/events/go-workshop/availability", nil).JSON(&av)
	if av.Mine == nil || av.Mine.Status != events.Confirmed {
		t.Fatalf("capacity raise should promote ann: %+v", av.Mine)
	}
}

func TestAdmissionIsSerialized(t *testing.T) {
	a := testapp.New(t)
	newEvent(t, a, "tiny", ptr(3))
	users := make([]string, 12)
	for i := range users {
		users[i] = a.Account(t, "u"+string(rune('a'+i))+"@example.com")
	}
	var wg sync.WaitGroup
	for _, u := range users {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.As(u).Do("POST", "/api/v1/events/tiny/rsvp", nil)
		}()
	}
	wg.Wait()
	var av events.Availability
	a.Do("GET", "/api/v1/events/tiny/availability", nil).JSON(&av)
	if av.Confirmed != 3 || av.Waitlisted != 9 {
		t.Fatalf("capacity 3 under concurrency: %+v", av)
	}
}

func TestRegistrationRules(t *testing.T) {
	a := testapp.New(t)
	user := a.As(a.Account(t, "ann@example.com"))
	now := time.Now().UTC()
	mgr := a.As("manager", events.Manage)
	draft := events.EventInput{Slug: "draft-event", Title: "Draft", Kind: "meetup", StartsAt: now.Add(time.Hour), EndsAt: now.Add(2 * time.Hour),
		OnlineURL: ptr("https://meet.example.com/x")}
	if r := mgr.Do("POST", "/api/v1/events", draft); r.Status() != 201 {
		t.Fatalf("draft: %d %s", r.Status(), r.Body())
	}
	if r := a.Do("GET", "/api/v1/events/draft-event/availability", nil); r.Status() != 404 {
		t.Fatalf("drafts are invisible: %d", r.Status())
	}
	noReg := draft
	noReg.Slug, noReg.Status = "meetup", events.Published
	mgr.Do("POST", "/api/v1/events", noReg)
	if r := user.Do("POST", "/api/v1/events/meetup/rsvp", nil); r.Status() != 409 || r.ErrorCode() != "no_registration" {
		t.Fatalf("no registration: %d %s", r.Status(), r.ErrorCode())
	}
	bad := draft
	bad.Slug, bad.Venue, bad.OnlineURL = "bad", nil, nil
	bad.Registration = &events.Registration{OpensAt: now, ClosesAt: now.Add(-time.Hour)}
	if r := mgr.Do("POST", "/api/v1/events", bad); r.Status() != 422 {
		t.Fatalf("invalid event: %d %s", r.Status(), r.Body())
	}
	var list []events.Event
	mgr.Do("GET", "/api/v1/events", nil).JSON(&list)
	if len(list) != 2 {
		t.Fatalf("manager list includes drafts: %d", len(list))
	}
	pub, err := a.Svc.Events.Published(t.Context())
	if err != nil || len(pub) != 1 || pub[0].Slug != "meetup" {
		t.Fatalf("published: %v %+v", err, pub)
	}
}

func TestDeleteAndRegistrants(t *testing.T) {
	a := testapp.New(t)
	m := a.As("manager", events.Manage)
	newEvent(t, a, "busy", nil)
	newEvent(t, a, "quiet", nil)
	var list []events.Event
	m.Do("GET", "/api/v1/events", nil).JSON(&list)
	ids := map[string]string{}
	for _, e := range list {
		ids[e.Slug] = e.ID
	}
	ann := a.Account(t, "ann@example.com")
	if r := a.As(ann).Do("POST", "/api/v1/events/busy/rsvp", nil); r.Status() != 200 {
		t.Fatalf("rsvp: %d %s", r.Status(), r.Body())
	}
	r := m.Do("GET", "/api/v1/events/"+ids["busy"]+"/registrants", nil)
	var regs []events.Registrant
	r.JSON(&regs)
	if r.Status() != 200 || len(regs) != 1 || regs[0].Email != "ann@example.com" || regs[0].Status != events.Confirmed || regs[0].Handle != nil {
		t.Fatalf("registrants: %d %s", r.Status(), r.Body())
	}
	if r := m.Do("DELETE", "/api/v1/events/"+ids["busy"], nil); r.Status() != 409 || r.ErrorCode() != "event_has_registrations" {
		t.Fatalf("delete with RSVPs: %d %s", r.Status(), r.Body())
	}
	if r := m.Do("DELETE", "/api/v1/events/"+ids["quiet"], nil); r.Status() != 204 {
		t.Fatalf("delete without RSVPs: %d %s", r.Status(), r.Body())
	}
	if r := m.Do("GET", "/api/v1/events/"+ids["quiet"]+"/registrants", nil); r.Status() != 404 {
		t.Fatalf("registrants of a deleted event: %d", r.Status())
	}
	// Once everyone cancels, the event can go.
	a.As(ann).Do("DELETE", "/api/v1/events/busy/rsvp", nil)
	if r := m.Do("DELETE", "/api/v1/events/"+ids["busy"], nil); r.Status() != 204 {
		t.Fatalf("delete after cancellations: %d %s", r.Status(), r.Body())
	}
}
