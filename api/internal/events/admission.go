package events

import (
	"time"

	"github.com/sakib412/ekman"
)

// Status is an event's publication status.
type Status string

// Event statuses. Drafts never reach the public site.
const (
	Draft     Status = "draft"
	Published Status = "published"
	Cancelled Status = "cancelled"
)

// Enum lists the values (OpenAPI and the web's types are generated from it).
func (Status) Enum() []string { return []string{"draft", "published", "cancelled"} }

// RSVPStatus is the state of one registration.
type RSVPStatus string

// RSVP statuses.
const (
	Confirmed  RSVPStatus = "confirmed"
	Waitlisted RSVPStatus = "waitlisted"
	Withdrawn  RSVPStatus = "cancelled"
)

// Enum lists the values.
func (RSVPStatus) Enum() []string { return []string{"confirmed", "waitlisted", "cancelled"} }

// Registration is nil for events without RSVP. The window and capacity live in one optional
// struct, so "a capacity without a window" cannot be expressed (the table has the same CHECK).
type Registration struct {
	OpensAt  time.Time `json:"opensAt"`
	ClosesAt time.Time `json:"closesAt"`
	Capacity *int      `json:"capacity"` // nil: unlimited
}

// Window is where registration stands at a moment, before counting seats.
type Window string

// Windows.
const (
	WindowNone      Window = "none"      // no registration for this event
	WindowNotOpen   Window = "not_open"  // opens later
	WindowOpen      Window = "open"      // seats or waitlist available
	WindowFull      Window = "full"      // open, but new RSVPs go to the waitlist
	WindowClosed    Window = "closed"    // the window has passed
	WindowCancelled Window = "cancelled" // the event was cancelled
)

// Enum lists the values.
func (Window) Enum() []string {
	return []string{"none", "not_open", "open", "full", "closed", "cancelled"}
}

// WindowAt is the single definition of registration state.
func WindowAt(st Status, reg *Registration, confirmed int, now time.Time) Window {
	switch {
	case st == Cancelled:
		return WindowCancelled
	case st != Published || reg == nil:
		return WindowNone
	case now.Before(reg.OpensAt):
		return WindowNotOpen
	case !now.Before(reg.ClosesAt):
		return WindowClosed
	case reg.Capacity != nil && confirmed >= *reg.Capacity:
		return WindowFull
	}
	return WindowOpen
}

// Admit decides what a new RSVP becomes: confirmed while seats remain, waitlisted once full,
// or an error explaining why registration is not possible. The caller holds the event row
// lock (SELECT ... FOR UPDATE), so confirmed is exact.
func Admit(st Status, reg *Registration, confirmed int, now time.Time) (RSVPStatus, error) {
	switch WindowAt(st, reg, confirmed, now) {
	case WindowOpen:
		return Confirmed, nil
	case WindowFull:
		return Waitlisted, nil
	case WindowCancelled:
		return "", ekman.Conflict("event_cancelled", "this event was cancelled")
	case WindowNone:
		return "", ekman.Conflict("no_registration", "this event does not take registrations")
	case WindowNotOpen:
		return "", ekman.Conflict("registration_not_open", "registration has not opened yet")
	}
	return "", ekman.Conflict("registration_closed", "registration has closed")
}

// Promotable is how many waitlisted RSVPs to confirm now that seats may have freed up (after a
// cancellation or a capacity increase). Nobody is promoted once the event has started.
func Promotable(st Status, reg *Registration, startsAt time.Time, confirmed, waitlisted int, now time.Time) int {
	if st != Published || reg == nil || waitlisted == 0 || !now.Before(startsAt) {
		return 0
	}
	if reg.Capacity == nil {
		return waitlisted
	}
	free := *reg.Capacity - confirmed
	if free <= 0 {
		return 0
	}
	return min(free, waitlisted)
}
