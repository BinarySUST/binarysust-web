package events

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sakib412/ekman"
	"github.com/sakib412/ekman/db"
	"github.com/sakib412/ekman/mail"

	"github.com/BinarySUST/binarysust-web/api/internal/pg"
)

// MyRSVP is the caller's registration.
type MyRSVP struct {
	Status   RSVPStatus `json:"status"`   // confirmed or waitlisted
	Ticket   *string    `json:"ticket"`   // QR ticket, confirmed only
	Position *int       `json:"position"` // place on the waitlist, waitlisted only
}

// Availability is the live payload behind the RSVP button. Served with Cache-Control: no-store.
type Availability struct {
	State      Window     `json:"state"`
	Open       bool       `json:"open"` // a new RSVP would be accepted (confirmed or waitlisted)
	OpensAt    *time.Time `json:"opensAt"`
	ClosesAt   *time.Time `json:"closesAt"`
	Capacity   *int       `json:"capacity"`
	Confirmed  int        `json:"confirmed"`
	Waitlisted int        `json:"waitlisted"`
	Mine       *MyRSVP    `json:"mine"` // null when anonymous or not registered
}

// SlugIn is a path slug.
type SlugIn struct {
	Slug string `path:"slug"`
}

// public event facts needed for admission, optionally locking the row.
type regFacts struct {
	id, title  string
	status     Status
	reg        *Registration
	startsAt   time.Time
	confirmed  int
	waitlisted int
}

func (s *Service) facts(ctx context.Context, slug string, lock bool) (regFacts, error) {
	var f regFacts
	var opens, closes *time.Time
	var capacity *int
	q := `SELECT id, title, status, reg_opens_at, reg_closes_at, capacity, starts_at
	        FROM events WHERE slug = $1 AND status <> 'draft'`
	if lock {
		q += ` FOR UPDATE`
	}
	err := s.db.Q(ctx).QueryRowContext(ctx, q, slug).Scan(&f.id, &f.title, &f.status, &opens, &closes, &capacity, &f.startsAt)
	if errors.Is(err, sql.ErrNoRows) {
		return f, pg.NotFound("event")
	}
	if err != nil {
		return f, err
	}
	if opens != nil && closes != nil {
		f.reg = &Registration{OpensAt: opens.UTC(), ClosesAt: closes.UTC(), Capacity: capacity}
	}
	err = s.db.Q(ctx).QueryRowContext(ctx, `
		SELECT count(*) FILTER (WHERE status = 'confirmed'), count(*) FILTER (WHERE status = 'waitlisted')
		  FROM rsvps WHERE event_id = $1`, f.id).Scan(&f.confirmed, &f.waitlisted)
	return f, err
}

func (s *Service) mine(ctx context.Context, eventID, accountID string) (*MyRSVP, error) {
	if !pg.ValidID(accountID) {
		return nil, nil
	}
	var m MyRSVP
	var ticket string
	var pos int
	err := s.db.Q(ctx).QueryRowContext(ctx, `
		SELECT r.status, r.ticket,
		       (SELECT count(*) FROM rsvps w WHERE w.event_id = r.event_id AND w.status = 'waitlisted'
		           AND (w.created_at, w.id) <= (r.created_at, r.id))
		  FROM rsvps r WHERE r.event_id = $1 AND r.account_id = $2 AND r.status <> 'cancelled'`,
		eventID, accountID).Scan(&m.Status, &ticket, &pos)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if m.Status == Confirmed {
		m.Ticket = &ticket
	} else {
		m.Position = &pos
	}
	return &m, nil
}

func (s *Service) availability(ctx context.Context, c ekman.Caller, in *SlugIn) (Availability, error) {
	f, err := s.facts(ctx, in.Slug, false)
	if err != nil {
		return Availability{}, err
	}
	w := WindowAt(f.status, f.reg, f.confirmed, s.now())
	a := Availability{State: w, Open: w == WindowOpen || w == WindowFull, Confirmed: f.confirmed, Waitlisted: f.waitlisted}
	if f.reg != nil {
		a.OpensAt, a.ClosesAt, a.Capacity = &f.reg.OpensAt, &f.reg.ClosesAt, f.reg.Capacity
	}
	if !c.Anonymous() {
		a.Mine, err = s.mine(ctx, f.id, c.UserID())
	}
	return a, err
}

func (s *Service) rsvp(ctx context.Context, c ekman.Caller, in *SlugIn) (MyRSVP, error) {
	var out *MyRSVP
	var title string
	newlyAdded := false
	err := s.db.Tx(ctx, func(ctx context.Context) error {
		f, err := s.facts(ctx, in.Slug, true) // the event row lock serializes admissions
		if err != nil {
			return err
		}
		title = f.title
		if out, err = s.mine(ctx, f.id, c.UserID()); err != nil || out != nil {
			return err // already registered: same answer as the first time
		}
		st, err := Admit(f.status, f.reg, f.confirmed, s.now())
		if err != nil {
			return err
		}
		if _, err := s.db.Q(ctx).ExecContext(ctx, `
			INSERT INTO rsvps (event_id, account_id, status, ticket) VALUES ($1, $2, $3, $4)
			ON CONFLICT (event_id, account_id) DO UPDATE
			   SET status = EXCLUDED.status, created_at = now(), updated_at = now(), checked_in_at = NULL`,
			f.id, c.UserID(), st, newTicket()); err != nil {
			return err
		}
		newlyAdded = true
		out, err = s.mine(ctx, f.id, c.UserID())
		return err
	})
	if err != nil {
		return MyRSVP{}, err
	}
	if out == nil {
		return MyRSVP{}, errors.New("rsvp: registration vanished")
	}
	if newlyAdded {
		s.mailRSVP(ctx, c.UserID(), in.Slug, title, *out, false)
	}
	return *out, nil
}

func (s *Service) cancel(ctx context.Context, c ekman.Caller, in *SlugIn) (ekman.NoContent, error) {
	var promoted []promotion
	err := s.db.Tx(ctx, func(ctx context.Context) error {
		f, err := s.facts(ctx, in.Slug, true)
		if err != nil {
			return err
		}
		res, err := s.db.Q(ctx).ExecContext(ctx, `
			UPDATE rsvps SET status = 'cancelled', updated_at = now()
			 WHERE event_id = $1 AND account_id = $2 AND status <> 'cancelled'`, f.id, c.UserID())
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return nil // nothing to cancel: cancelling twice is fine
		}
		promoted, err = s.promote(ctx, f.id)
		return err
	})
	if err == nil {
		s.notifyPromoted(ctx, promoted)
	}
	return ekman.NoContent{}, err
}

type promotion struct{ accountID, slug, title string }

// promote confirms the oldest waitlisted RSVPs while seats are free. The caller holds the event
// row lock (or has just updated the row, which takes the same lock).
func (s *Service) promote(ctx context.Context, eventID string) ([]promotion, error) {
	var slug string
	if err := s.db.Q(ctx).QueryRowContext(ctx, `SELECT slug FROM events WHERE id = $1 FOR UPDATE`, eventID).Scan(&slug); err != nil {
		return nil, err
	}
	f, err := s.facts(ctx, slug, false)
	if errors.Is(err, sql.ErrNoRows) || isNotFound(err) {
		return nil, nil // drafts have no registrations to promote
	}
	if err != nil {
		return nil, err
	}
	n := Promotable(f.status, f.reg, f.startsAt, f.confirmed, f.waitlisted, s.now())
	if n == 0 {
		return nil, nil
	}
	rows, err := s.db.Q(ctx).QueryContext(ctx, `
		UPDATE rsvps SET status = 'confirmed', updated_at = now()
		 WHERE id IN (SELECT id FROM rsvps WHERE event_id = $1 AND status = 'waitlisted'
		               ORDER BY created_at, id LIMIT $2)
		RETURNING account_id`, eventID, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []promotion
	for rows.Next() {
		var p promotion
		if err := rows.Scan(&p.accountID); err != nil {
			return nil, err
		}
		p.slug, p.title = slug, f.title
		out = append(out, p)
	}
	return out, rows.Err()
}

func isNotFound(err error) bool {
	var e *ekman.Error
	return errors.As(err, &e) && e.Status == http.StatusNotFound
}

// CheckInIn is a scanned ticket.
type CheckInIn struct {
	Slug   string `path:"slug"`
	Ticket string `json:"ticket"`
}

// Validate implements ekman.Validator.
func (in *CheckInIn) Validate(v *ekman.Validation) {
	in.Ticket = strings.ToUpper(strings.TrimSpace(in.Ticket))
	v.Required("ticket", in.Ticket)
}

// CheckInResult is what the volunteer's scanner shows.
type CheckInResult struct {
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	CheckedInAt      time.Time `json:"checkedInAt"`
	AlreadyCheckedIn bool      `json:"alreadyCheckedIn"` // scanned before: CheckedInAt is the first scan
}

func (s *Service) checkIn(ctx context.Context, _ ekman.Caller, in *CheckInIn) (CheckInResult, error) {
	var r CheckInResult
	err := s.db.Tx(ctx, func(ctx context.Context) error {
		var id string
		var status RSVPStatus
		var at *time.Time
		err := s.db.Q(ctx).QueryRowContext(ctx, `
			SELECT r.id, r.status, r.checked_in_at, coalesce(nullif(a.name, ''), a.email), a.email
			  FROM rsvps r JOIN events e ON e.id = r.event_id JOIN accounts a ON a.id = r.account_id
			 WHERE e.slug = $1 AND r.ticket = $2 FOR UPDATE OF r`, in.Slug, in.Ticket).Scan(&id, &status, &at, &r.Name, &r.Email)
		if errors.Is(err, sql.ErrNoRows) {
			return ekman.NotFound("no such ticket for this event")
		}
		if err != nil {
			return err
		}
		if status != Confirmed {
			return ekman.Conflict("not_confirmed", "this registration is "+string(status)+", not confirmed")
		}
		if at != nil {
			r.CheckedInAt, r.AlreadyCheckedIn = at.UTC(), true
			return nil
		}
		err = s.db.Q(ctx).QueryRowContext(ctx,
			`UPDATE rsvps SET checked_in_at = now(), updated_at = now() WHERE id = $1 RETURNING checked_in_at`, id).Scan(&r.CheckedInAt)
		r.CheckedInAt = r.CheckedInAt.UTC()
		return err
	})
	return r, err
}

// RSVPItem is one of the caller's registrations.
type RSVPItem struct {
	MyRSVP
	EventSlug  string    `json:"eventSlug"`
	EventTitle string    `json:"eventTitle"`
	StartsAt   time.Time `json:"startsAt"`
}

func (s *Service) myRSVPs(ctx context.Context, c ekman.Caller, _ *struct{}) ([]RSVPItem, error) {
	if !pg.ValidID(c.UserID()) {
		return nil, nil
	}
	rows, err := s.db.Q(ctx).QueryContext(ctx, `
		SELECT e.id, e.slug, e.title, e.starts_at FROM rsvps r JOIN events e ON e.id = r.event_id
		 WHERE r.account_id = $1 AND r.status <> 'cancelled' AND e.status <> 'draft'
		 ORDER BY e.starts_at DESC`, c.UserID())
	if err != nil {
		return nil, err
	}
	type ev struct {
		id   string
		item RSVPItem
	}
	var evs []ev
	for rows.Next() {
		var e ev
		if err := rows.Scan(&e.id, &e.item.EventSlug, &e.item.EventTitle, &e.item.StartsAt); err != nil {
			rows.Close()
			return nil, err
		}
		e.item.StartsAt = e.item.StartsAt.UTC()
		evs = append(evs, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]RSVPItem, 0, len(evs))
	for _, e := range evs {
		m, err := s.mine(ctx, e.id, c.UserID())
		if err != nil {
			return nil, err
		}
		if m != nil {
			e.item.MyRSVP = *m
			out = append(out, e.item)
		}
	}
	return out, nil
}

// Registrant is one registration, for the dashboard and the CSV export.
type Registrant struct {
	Name         string     `json:"name"` // the account's name, or its email when it has none
	Email        string     `json:"email"`
	Handle       *string    `json:"handle"` // the linked person, if any
	Status       RSVPStatus `json:"status"`
	Ticket       string     `json:"ticket"`
	RegisteredAt time.Time  `json:"registeredAt"`
	CheckedInAt  *time.Time `json:"checkedInAt"`
}

// listRegistrants: confirmed first, then the waitlist in order, then cancellations. cond picks
// the event by $1 (e.slug or e.id).
func (s *Service) listRegistrants(ctx context.Context, cond, arg string) ([]Registrant, error) {
	rows, err := s.db.Q(ctx).QueryContext(ctx, `
		SELECT coalesce(nullif(a.name, ''), a.email), a.email, p.handle, r.status, r.ticket, r.created_at, r.checked_in_at
		  FROM rsvps r JOIN events e ON e.id = r.event_id
		  JOIN accounts a ON a.id = r.account_id LEFT JOIN persons p ON p.id = a.person_id
		 WHERE `+cond+`
		 ORDER BY array_position(ARRAY['confirmed', 'waitlisted', 'cancelled'], r.status), r.created_at, r.id`, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Registrant{}
	for rows.Next() {
		var x Registrant
		if err := rows.Scan(&x.Name, &x.Email, &x.Handle, &x.Status, &x.Ticket, &x.RegisteredAt, &x.CheckedInAt); err != nil {
			return nil, err
		}
		x.RegisteredAt = x.RegisteredAt.UTC()
		if x.CheckedInAt != nil {
			t := x.CheckedInAt.UTC()
			x.CheckedInAt = &t
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Service) registrants(ctx context.Context, _ ekman.Caller, in *IDIn) ([]Registrant, error) {
	if _, err := s.one(ctx, in.ID); err != nil {
		return nil, err
	}
	return s.listRegistrants(ctx, `e.id = $1`, in.ID)
}

// exportCSV writes the same list as GET /events/{id}/registrants, for spreadsheets.
func (s *Service) exportCSV(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slug := r.PathValue("slug")
	list, err := s.listRegistrants(ctx, `e.slug = $1`, slug)
	if err != nil {
		ekman.WriteError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-registrants.csv"`, slug))
	w.Header().Set("Cache-Control", "no-store")
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"name", "email", "handle", "status", "ticket", "registered_at", "checked_in_at"})
	for _, x := range list {
		handle, in := "", ""
		if x.Handle != nil {
			handle = *x.Handle
		}
		if x.CheckedInAt != nil {
			in = x.CheckedInAt.Format(time.RFC3339)
		}
		_ = cw.Write([]string{csvSafe(x.Name), x.Email, handle, string(x.Status), x.Ticket, x.RegisteredAt.Format(time.RFC3339), in})
	}
	cw.Flush()
}

// csvSafe stops spreadsheet formula injection from user-chosen names.
func csvSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func newTicket() string {
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
}

// ---- mail ---------------------------------------------------------------------------------

func (s *Service) accountEmail(ctx context.Context, accountID string) (email, name string, err error) {
	err = s.db.Q(ctx).QueryRowContext(ctx, `SELECT email, name FROM accounts WHERE id = $1`, accountID).Scan(&email, &name)
	return
}

func (s *Service) mailRSVP(ctx context.Context, accountID, slug, title string, m MyRSVP, promoted bool) {
	if s.mail == nil {
		return
	}
	// Mail goes out after the commit; a failure never undoes the registration.
	db.AfterCommit(ctx, func(ctx context.Context) {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		email, name, err := s.accountEmail(ctx, accountID)
		if err != nil {
			s.log.ErrorContext(ctx, "events: rsvp mail lookup", "err", err.Error())
			return
		}
		link := s.siteURL + "/events/" + slug
		var subject, body string
		switch {
		case m.Status == Confirmed && promoted:
			subject = "A seat opened up: " + title
			body = fmt.Sprintf("Hi %s,\n\nGood news: a seat opened up and you are now confirmed for %s.\nYour ticket: %s\nShow the QR code from your dashboard at the door: %s/dashboard/rsvps\n\n%s\n", name, title, *m.Ticket, s.siteURL, link)
		case m.Status == Confirmed:
			subject = "You're registered: " + title
			body = fmt.Sprintf("Hi %s,\n\nYour seat for %s is confirmed.\nYour ticket: %s\nShow the QR code from your dashboard at the door: %s/dashboard/rsvps\n\n%s\n", name, title, *m.Ticket, s.siteURL, link)
		default:
			subject = "You're on the waitlist: " + title
			body = fmt.Sprintf("Hi %s,\n\n%s is full, so you are number %d on the waitlist. We will email you if a seat opens up.\n\n%s\n", name, title, *m.Position, link)
		}
		if err := s.mail.Send(ctx, mail.Message{To: []string{email}, Subject: subject, Text: body}); err != nil {
			s.log.ErrorContext(ctx, "events: rsvp mail", "err", err.Error())
		}
	})
}

func (s *Service) notifyPromoted(ctx context.Context, ps []promotion) {
	for _, p := range ps {
		ticket := ""
		_ = s.db.Q(ctx).QueryRowContext(ctx, `SELECT r.ticket FROM rsvps r JOIN events e ON e.id = r.event_id
			WHERE e.slug = $1 AND r.account_id = $2`, p.slug, p.accountID).Scan(&ticket)
		s.mailRSVP(ctx, p.accountID, p.slug, p.title, MyRSVP{Status: Confirmed, Ticket: &ticket}, true)
	}
}
