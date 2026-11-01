package blog

import (
	"errors"
	"testing"

	"github.com/sakib412/ekman"
)

func TestTransition(t *testing.T) {
	author := Actor{IsAuthor: true}
	editor := Actor{CanReview: true}
	cases := []struct {
		from   State
		action Action
		who    Actor
		want   State
		status int // 0: success
	}{
		{Draft, Submit, author, InReview, 0},
		{Unpublished, Submit, author, InReview, 0},
		{InReview, Submit, author, InReview, 0}, // retry: no-op
		{Published, Submit, author, Published, 409},
		{Draft, Submit, editor, Draft, 403}, // reviewers don't submit other people's drafts
		{InReview, Withdraw, author, Draft, 0},
		{InReview, RequestChanges, editor, Draft, 0},
		{InReview, RequestChanges, author, InReview, 403},
		{InReview, Publish, editor, Published, 0},
		{Published, Publish, editor, Published, 0}, // retry: no-op
		{Unpublished, Publish, editor, Published, 0},
		{Draft, Publish, editor, Draft, 409},
		{InReview, Publish, author, InReview, 403},
		{Published, Unpublish, editor, Unpublished, 0},
		{Unpublished, Unpublish, editor, Unpublished, 0},
		{Draft, Unpublish, editor, Draft, 409},
		{Draft, "delete", editor, Draft, 422},
	}
	for _, c := range cases {
		got, err := Transition(c.from, c.action, c.who)
		status := 0
		var e *ekman.Error
		if errors.As(err, &e) {
			status = e.Status
		} else if err != nil {
			t.Fatalf("%s %s: unexpected error %v", c.from, c.action, err)
		}
		if got != c.want || status != c.status {
			t.Errorf("%s --%s--> got (%s, %d), want (%s, %d)", c.from, c.action, got, status, c.want, c.status)
		}
	}
}

func TestCanEdit(t *testing.T) {
	author := Actor{IsAuthor: true}
	if !CanEdit(Draft, author) || CanEdit(InReview, author) || CanEdit(Published, author) {
		t.Fatal("authors edit drafts only")
	}
	if !CanEdit(Published, Actor{CanReview: true}) {
		t.Fatal("reviewers edit anything")
	}
	if CanEdit(Draft, Actor{}) {
		t.Fatal("strangers edit nothing")
	}
}

func TestSlugify(t *testing.T) {
	for in, want := range map[string]string{
		"How this site is built: one snapshot!": "how-this-site-is-built-one-snapshot",
		"  ICPC 2026 -- Dhaka  ":                "icpc-2026-dhaka",
		"বাংলা":                                 "",
	} {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
