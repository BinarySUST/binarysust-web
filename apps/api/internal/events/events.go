// Package events holds events, registration (RSVP with a waitlist), QR check-in, the
// registrants export and the recap gallery. Registration state is computed by one pure
// function (WindowAt / Admit) and served live; the static event page only shows facts.
package events

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/sakib412/ekman"
	"github.com/sakib412/ekman/db"
	"github.com/sakib412/ekman/mail"
	"github.com/sakib412/ekman/rbac"

	"github.com/BinarySUST/binarysust-web/apps/api/internal/md"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/people"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/pg"
	"github.com/BinarySUST/binarysust-web/apps/api/internal/republish"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Permissions.
const (
	Manage  rbac.Permission = "events.manage"
	CheckIn rbac.Permission = "events.checkin"
)

// Kind is workshop, contest, hackathon, seminar or meetup.
type Kind string

// Enum lists the values.
func (Kind) Enum() []string { return []string{"workshop", "contest", "hackathon", "seminar", "meetup"} }

// Photo is a recap gallery image.
type Photo struct {
	URL     string `json:"url"`
	Caption string `json:"caption"`
}

// Service is the events module.
type Service struct {
	db      *db.DB
	people  *people.Service
	mail    mail.Sender
	siteURL string
	log     *slog.Logger
	now     func() time.Time
}

// New returns the events service. m may be nil (no confirmation emails).
func New(d *db.DB, p *people.Service, m mail.Sender, siteURL string, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{db: d, people: p, mail: m, siteURL: strings.TrimSuffix(siteURL, "/"), log: log, now: time.Now}
}

// Module wires events into the app.
func (s *Service) Module() ekman.Module {
	return ekman.Module{
		Name:        "events",
		Migrations:  migrations,
		Permissions: []rbac.Permission{Manage, CheckIn},
		Routes: func(r *ekman.Router) {
			// Dashboard: every event, drafts included.
			ekman.Handle(r, "GET /events", ekman.Requires(Manage), s.list)
			ekman.Handle(r, "POST /events", ekman.Requires(Manage), s.create)
			ekman.Handle(r, "GET /events/{id}", ekman.Requires(Manage), s.get)
			ekman.Handle(r, "PUT /events/{id}", ekman.Requires(Manage), s.update)
			ekman.Handle(r, "DELETE /events/{id}", ekman.Requires(Manage), s.remove,
				ekman.Summary("Delete a draft, or an event nobody registered for (409 otherwise)"))
			ekman.Handle(r, "GET /events/{id}/registrants", ekman.Requires(Manage), s.registrants,
				ekman.Use(noStore), ekman.Summary("Registrants: confirmed, then the waitlist in order, then cancellations"))
			r.Raw("GET /events/{slug}/registrants.csv", ekman.Requires(Manage), http.HandlerFunc(s.exportCSV),
				ekman.Summary("Registrants as CSV"))

			// Live registration: never cached, never republishes.
			ekman.Handle(r, "GET /events/{slug}/availability", ekman.Public(), s.availability,
				ekman.Use(noStore), ekman.Summary("Live seats and the caller's RSVP"))
			ekman.Handle(r, "POST /events/{slug}/rsvp", ekman.SignedIn(), s.rsvp,
				ekman.Use(noStore), ekman.Status(http.StatusOK), republish.Skip(), ekman.Summary("Reserve a seat (idempotent)"))
			ekman.Handle(r, "DELETE /events/{slug}/rsvp", ekman.SignedIn(), s.cancel,
				ekman.Use(noStore), republish.Skip(), ekman.Summary("Cancel; promotes the waitlist"))
			ekman.Handle(r, "POST /events/{slug}/checkin", ekman.Requires(CheckIn), s.checkIn,
				ekman.Status(http.StatusOK), republish.Skip(), ekman.Summary("Check in a ticket"))
			ekman.Handle(r, "GET /me/rsvps", ekman.SignedIn(), s.myRSVPs, ekman.Use(noStore))
		},
	}
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// ---- dashboard ----------------------------------------------------------------------------

// EventInput is what an event manager edits.
type EventInput struct {
	Slug         string        `json:"slug"`
	Title        string        `json:"title"`
	Kind         Kind          `json:"kind"`
	Status       Status        `json:"status"`
	Summary      string        `json:"summary"`
	BodyMarkdown string        `json:"bodyMarkdown"`
	StartsAt     time.Time     `json:"startsAt"`
	EndsAt       time.Time     `json:"endsAt"`
	Venue        *string       `json:"venue"`
	OnlineURL    *string       `json:"onlineUrl"`
	Registration *Registration `json:"registration"` // window and capacity together, or null
	// Optional on the wire. They are replaced on every save, so an editor sends them back.
	CoverURL      *string  `json:"coverUrl,omitempty"`
	RecapMarkdown string   `json:"recapMarkdown,omitempty"`
	Speakers      []string `json:"speakers,omitempty"` // person handles
	Photos        []Photo  `json:"photos,omitempty"`
}

// Validate implements ekman.Validator.
func (in *EventInput) Validate(v *ekman.Validation) {
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	v.Required("slug", in.Slug)
	v.Check(slugOK(in.Slug), "slug", "use lowercase letters, digits and dashes")
	in.Title = strings.TrimSpace(in.Title)
	v.Required("title", in.Title)
	v.MaxLen("title", in.Title, 200)
	v.Required("kind", string(in.Kind))
	v.OneOf("kind", string(in.Kind), Kind("").Enum()...)
	if in.Status == "" {
		in.Status = Draft
	}
	v.OneOf("status", string(in.Status), string(Draft), string(Published), string(Cancelled))
	v.MaxLen("summary", in.Summary, 400)
	v.MaxLen("bodyMarkdown", in.BodyMarkdown, 50_000)
	v.MaxLen("recapMarkdown", in.RecapMarkdown, 50_000)
	v.Check(!in.StartsAt.IsZero(), "startsAt", "required")
	v.Check(in.EndsAt.After(in.StartsAt), "endsAt", "must be after the start")
	in.Venue, in.OnlineURL = blankToNil(in.Venue), blankToNil(in.OnlineURL)
	v.Check(in.Venue != nil || in.OnlineURL != nil, "venue", "give a venue or an online link")
	if in.OnlineURL != nil {
		v.URL("onlineUrl", *in.OnlineURL)
	}
	if in.CoverURL != nil {
		v.URL("coverUrl", *in.CoverURL)
	}
	if r := in.Registration; r != nil {
		v.Check(!r.OpensAt.IsZero() && !r.ClosesAt.IsZero(), "registration", "give both opensAt and closesAt")
		v.Check(r.ClosesAt.After(r.OpensAt), "registration.closesAt", "must be after opensAt")
		v.Check(r.Capacity == nil || *r.Capacity > 0, "registration.capacity", "must be positive or null")
	}
	v.Check(len(in.Photos) <= 200, "photos", "at most 200 photos")
	for _, p := range in.Photos {
		v.URL("photos", p.URL)
	}
}

// Event is the dashboard view of an event.
type Event struct {
	ID            string        `json:"id"`
	Slug          string        `json:"slug"`
	Title         string        `json:"title"`
	Kind          Kind          `json:"kind"`
	Status        Status        `json:"status"`
	Summary       string        `json:"summary"`
	BodyMarkdown  string        `json:"bodyMarkdown"`
	CoverURL      *string       `json:"coverUrl"`
	StartsAt      time.Time     `json:"startsAt"`
	EndsAt        time.Time     `json:"endsAt"`
	Venue         *string       `json:"venue"`
	OnlineURL     *string       `json:"onlineUrl"`
	Registration  *Registration `json:"registration"`
	RecapMarkdown string        `json:"recapMarkdown"`
	Speakers      []string      `json:"speakers"`
	Photos        []Photo       `json:"photos"`
	UpdatedAt     time.Time     `json:"updatedAt"`
	RSVPCount     int           `json:"rsvpCount"` // confirmed + waitlisted
	Confirmed     int           `json:"confirmed"`
	Waitlisted    int           `json:"waitlisted"`
	CheckedIn     int           `json:"checkedIn"`
}

const eventCols = `e.id, e.slug, e.title, e.kind, e.status, e.summary, e.body_md, e.cover_url, e.starts_at, e.ends_at,
	e.venue, e.online_url, e.reg_opens_at, e.reg_closes_at, e.capacity, e.recap_md, e.updated_at,
	(SELECT count(*) FILTER (WHERE status = 'confirmed') FROM rsvps WHERE event_id = e.id),
	(SELECT count(*) FILTER (WHERE status = 'waitlisted') FROM rsvps WHERE event_id = e.id),
	(SELECT count(*) FILTER (WHERE checked_in_at IS NOT NULL) FROM rsvps WHERE event_id = e.id),
	coalesce((SELECT array_agg(p.handle::text ORDER BY sp.sort) FROM event_speakers sp JOIN persons p ON p.id = sp.person_id
	           WHERE sp.event_id = e.id), '{}'),
	coalesce((SELECT json_agg(json_build_object('url', ph.url, 'caption', ph.caption) ORDER BY ph.sort)
	           FROM event_photos ph WHERE ph.event_id = e.id), '[]')`

func scanEvent(sc interface{ Scan(...any) error }) (Event, error) {
	var e Event
	var opens, closes *time.Time
	var capacity *int
	var photos []byte
	err := sc.Scan(&e.ID, &e.Slug, &e.Title, &e.Kind, &e.Status, &e.Summary, &e.BodyMarkdown, &e.CoverURL,
		&e.StartsAt, &e.EndsAt, &e.Venue, &e.OnlineURL, &opens, &closes, &capacity, &e.RecapMarkdown, &e.UpdatedAt,
		&e.Confirmed, &e.Waitlisted, &e.CheckedIn, pg.Array(&e.Speakers), &photos)
	if err != nil {
		return e, err
	}
	e.StartsAt, e.EndsAt, e.UpdatedAt = e.StartsAt.UTC(), e.EndsAt.UTC(), e.UpdatedAt.UTC()
	if opens != nil && closes != nil {
		e.Registration = &Registration{OpensAt: opens.UTC(), ClosesAt: closes.UTC(), Capacity: capacity}
	}
	e.RSVPCount = e.Confirmed + e.Waitlisted
	if e.Speakers == nil {
		e.Speakers = []string{}
	}
	e.Photos = pg.JSONList[Photo](photos)
	return e, nil
}

func (s *Service) events(ctx context.Context, where string, args ...any) ([]Event, error) {
	rows, err := s.db.Q(ctx).QueryContext(ctx, `SELECT `+eventCols+` FROM events e `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Service) one(ctx context.Context, id string) (Event, error) {
	if !pg.ValidID(id) {
		return Event{}, pg.NotFound("event")
	}
	es, err := s.events(ctx, `WHERE e.id = $1`, id)
	if err != nil {
		return Event{}, err
	}
	if len(es) == 0 {
		return Event{}, pg.NotFound("event")
	}
	return es[0], nil
}

func (s *Service) list(ctx context.Context, _ ekman.Caller, _ *struct{}) ([]Event, error) {
	return s.events(ctx, `ORDER BY e.starts_at DESC`)
}

// IDIn is a path id.
type IDIn struct {
	ID string `path:"id"`
}

func (s *Service) get(ctx context.Context, _ ekman.Caller, in *IDIn) (Event, error) {
	return s.one(ctx, in.ID)
}

func (s *Service) create(ctx context.Context, _ ekman.Caller, in *EventInput) (Event, error) {
	var id string
	err := s.db.Tx(ctx, func(ctx context.Context) error {
		opens, closes, capacity := regCols(in.Registration)
		err := s.db.Q(ctx).QueryRowContext(ctx, `
			INSERT INTO events (slug, title, kind, status, summary, body_md, cover_url, starts_at, ends_at, venue,
			                    online_url, reg_opens_at, reg_closes_at, capacity, recap_md)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) RETURNING id`,
			in.Slug, in.Title, in.Kind, in.Status, in.Summary, in.BodyMarkdown, in.CoverURL, in.StartsAt, in.EndsAt,
			in.Venue, in.OnlineURL, opens, closes, capacity, in.RecapMarkdown).Scan(&id)
		if err != nil {
			return pg.Friendly(err, "an event with that slug exists")
		}
		return s.setChildren(ctx, id, in)
	})
	if err != nil {
		return Event{}, err
	}
	return s.one(ctx, id)
}

// UpdateIn edits an event.
type UpdateIn struct {
	ID string `path:"id"`
	EventInput
}

func (s *Service) update(ctx context.Context, _ ekman.Caller, in *UpdateIn) (Event, error) {
	if !pg.ValidID(in.ID) {
		return Event{}, pg.NotFound("event")
	}
	var promoted []promotion
	err := s.db.Tx(ctx, func(ctx context.Context) error {
		opens, closes, capacity := regCols(in.Registration)
		res, err := s.db.Q(ctx).ExecContext(ctx, `
			UPDATE events SET slug = $2, title = $3, kind = $4, status = $5, summary = $6, body_md = $7, cover_url = $8,
			       starts_at = $9, ends_at = $10, venue = $11, online_url = $12, reg_opens_at = $13, reg_closes_at = $14,
			       capacity = $15, recap_md = $16, updated_at = now()
			 WHERE id = $1`,
			in.ID, in.Slug, in.Title, in.Kind, in.Status, in.Summary, in.BodyMarkdown, in.CoverURL, in.StartsAt, in.EndsAt,
			in.Venue, in.OnlineURL, opens, closes, capacity, in.RecapMarkdown)
		if err != nil {
			return pg.Friendly(err, "an event with that slug exists")
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return pg.NotFound("event")
		}
		if err := s.setChildren(ctx, in.ID, &in.EventInput); err != nil {
			return err
		}
		// More seats (or no limit any more) may confirm people on the waitlist.
		promoted, err = s.promote(ctx, in.ID)
		return err
	})
	if err != nil {
		return Event{}, err
	}
	s.notifyPromoted(ctx, promoted)
	return s.one(ctx, in.ID)
}

func (s *Service) setChildren(ctx context.Context, id string, in *EventInput) error {
	ids, err := s.people.IDsByHandle(ctx, "speakers", in.Speakers)
	if err != nil {
		return err
	}
	q := s.db.Q(ctx)
	if _, err := q.ExecContext(ctx, `DELETE FROM event_speakers WHERE event_id = $1`, id); err != nil {
		return err
	}
	for i, pid := range ids {
		if _, err := q.ExecContext(ctx, `INSERT INTO event_speakers (event_id, person_id, sort) VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING`, id, pid, i); err != nil {
			return err
		}
	}
	if _, err := q.ExecContext(ctx, `DELETE FROM event_photos WHERE event_id = $1`, id); err != nil {
		return err
	}
	for i, p := range in.Photos {
		if _, err := q.ExecContext(ctx, `INSERT INTO event_photos (event_id, url, caption, sort) VALUES ($1, $2, $3, $4)`,
			id, p.URL, p.Caption, i); err != nil {
			return err
		}
	}
	return nil
}

// remove deletes a draft, or a published event nobody holds a registration for. An event
// people registered for is history (and their tickets): cancel it instead.
func (s *Service) remove(ctx context.Context, _ ekman.Caller, in *IDIn) (ekman.NoContent, error) {
	if !pg.ValidID(in.ID) {
		return ekman.NoContent{}, pg.NotFound("event")
	}
	return ekman.NoContent{}, s.db.Tx(ctx, func(ctx context.Context) error {
		var status Status
		var active int
		err := s.db.Q(ctx).QueryRowContext(ctx, `
			SELECT e.status, (SELECT count(*) FROM rsvps r WHERE r.event_id = e.id AND r.status <> 'cancelled')
			  FROM events e WHERE e.id = $1 FOR UPDATE`, in.ID).Scan(&status, &active)
		if errors.Is(err, sql.ErrNoRows) {
			return pg.NotFound("event")
		}
		if err != nil {
			return err
		}
		if status != Draft && active > 0 {
			return ekman.Conflict("event_has_registrations", fmt.Sprintf(
				"%d %s registered for this event; set its status to Cancelled instead of deleting it", active, plural(active, "person has", "people have")))
		}
		_, err = s.db.Q(ctx).ExecContext(ctx, `DELETE FROM events WHERE id = $1`, in.ID)
		return err
	})
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// ---- public -------------------------------------------------------------------------------

// PublicEvent is an event on the public site.
type PublicEvent struct {
	Slug         string        `json:"slug"`
	Title        string        `json:"title"`
	Kind         Kind          `json:"kind"`
	Status       Status        `json:"status"` // published or cancelled
	Summary      string        `json:"summary"`
	BodyHTML     string        `json:"bodyHtml"`
	CoverURL     *string       `json:"coverUrl"`
	StartsAt     time.Time     `json:"startsAt"`
	EndsAt       time.Time     `json:"endsAt"`
	Venue        *string       `json:"venue"`
	OnlineURL    *string       `json:"onlineUrl"`
	Registration *Registration `json:"registration"`
	RecapHTML    string        `json:"recapHtml"`
	Photos       []Photo       `json:"photos"`
	Speakers     []string      `json:"speakers"` // person handles
}

// Published is the read side for site.Build: published and cancelled events, newest first.
func (s *Service) Published(ctx context.Context) ([]PublicEvent, error) {
	es, err := s.events(ctx, `WHERE e.status IN ('published', 'cancelled') ORDER BY e.starts_at DESC, e.slug`)
	if err != nil {
		return nil, err
	}
	out := make([]PublicEvent, 0, len(es))
	for _, e := range es {
		out = append(out, PublicEvent{Slug: e.Slug, Title: e.Title, Kind: e.Kind, Status: e.Status, Summary: e.Summary,
			BodyHTML: md.Render(e.BodyMarkdown), CoverURL: e.CoverURL, StartsAt: e.StartsAt, EndsAt: e.EndsAt,
			Venue: e.Venue, OnlineURL: e.OnlineURL, Registration: e.Registration, RecapHTML: md.Render(e.RecapMarkdown),
			Photos: e.Photos, Speakers: e.Speakers})
	}
	return out, nil
}

// PastCount is how many published events have ended (a landing-page stat).
func (s *Service) PastCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.Q(ctx).QueryRowContext(ctx, `SELECT count(*) FROM events WHERE status = 'published' AND ends_at < now()`).Scan(&n)
	return n, err
}

// ---- helpers ------------------------------------------------------------------------------

func regCols(r *Registration) (opens, closes *time.Time, capacity *int) {
	if r == nil {
		return nil, nil, nil
	}
	return &r.OpensAt, &r.ClosesAt, r.Capacity
}

func blankToNil(s *string) *string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	t := strings.TrimSpace(*s)
	return &t
}

func slugOK(s string) bool {
	if len(s) < 2 || len(s) > 80 || s[0] == '-' {
		return false
	}
	for _, c := range s {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
			return false
		}
	}
	return true
}
