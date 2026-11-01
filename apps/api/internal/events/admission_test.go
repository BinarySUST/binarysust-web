package events

import (
	"errors"
	"testing"
	"time"

	"github.com/sakib412/ekman"
)

func TestAdmit(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	two := 2
	open := &Registration{OpensAt: now.Add(-time.Hour), ClosesAt: now.Add(time.Hour), Capacity: &two}
	unlimited := &Registration{OpensAt: now.Add(-time.Hour), ClosesAt: now.Add(time.Hour)}
	cases := []struct {
		name      string
		st        Status
		reg       *Registration
		confirmed int
		want      RSVPStatus
		code      string
	}{
		{"seat available", Published, open, 1, Confirmed, ""},
		{"full goes to waitlist", Published, open, 2, Waitlisted, ""},
		{"unlimited", Published, unlimited, 5000, Confirmed, ""},
		{"no registration", Published, nil, 0, "", "no_registration"},
		{"draft", Draft, open, 0, "", "no_registration"},
		{"cancelled", Cancelled, open, 0, "", "event_cancelled"},
		{"not open", Published, &Registration{OpensAt: now.Add(time.Minute), ClosesAt: now.Add(time.Hour)}, 0, "", "registration_not_open"},
		{"closes exactly now", Published, &Registration{OpensAt: now.Add(-time.Hour), ClosesAt: now}, 0, "", "registration_closed"},
	}
	for _, c := range cases {
		got, err := Admit(c.st, c.reg, c.confirmed, now)
		code := ""
		var e *ekman.Error
		if errors.As(err, &e) {
			code = e.Code
		}
		if got != c.want || code != c.code {
			t.Errorf("%s: got (%q, %q), want (%q, %q)", c.name, got, code, c.want, c.code)
		}
	}
}

func TestPromotable(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	start := now.Add(24 * time.Hour)
	three := 3
	reg := &Registration{OpensAt: now.Add(-time.Hour), ClosesAt: now.Add(time.Hour), Capacity: &three}
	cases := []struct {
		name                  string
		reg                   *Registration
		start                 time.Time
		confirmed, waitlisted int
		want                  int
	}{
		{"one seat freed", reg, start, 2, 5, 1},
		{"capacity raised", reg, start, 0, 2, 2},
		{"still full", reg, start, 3, 4, 0},
		{"empty waitlist", reg, start, 1, 0, 0},
		{"event started", reg, now, 1, 4, 0},
		{"after the window closes, before the event", &Registration{OpensAt: now.Add(-2 * time.Hour), ClosesAt: now.Add(-time.Hour), Capacity: &three}, start, 2, 1, 1},
		{"unlimited", &Registration{OpensAt: now, ClosesAt: start}, start, 10, 2, 2},
	}
	for _, c := range cases {
		if got := Promotable(Published, c.reg, c.start, c.confirmed, c.waitlisted, now); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
	if got := Promotable(Cancelled, reg, start, 0, 3, now); got != 0 {
		t.Errorf("cancelled event promoted %d", got)
	}
}
