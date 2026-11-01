package blog

import (
	"slices"

	"github.com/sakib412/ekman"
)

// State is a post's place in the review workflow:
//
//	draft → in_review → published ⇄ unpublished
type State string

// States.
const (
	Draft       State = "draft"
	InReview    State = "in_review"
	Published   State = "published"
	Unpublished State = "unpublished"
)

// Enum lists the values (OpenAPI and the web's types are generated from it).
func (State) Enum() []string { return []string{"draft", "in_review", "published", "unpublished"} }

// Action moves a post between states.
type Action string

// Actions.
const (
	Submit         Action = "submit"          // author: draft|unpublished → in_review
	Withdraw       Action = "withdraw"        // author: in_review → draft
	RequestChanges Action = "request_changes" // reviewer: in_review → draft (with a note)
	Publish        Action = "publish"         // reviewer: in_review|unpublished → published
	Unpublish      Action = "unpublish"       // reviewer: published → unpublished
)

// Enum lists the values.
func (Action) Enum() []string { return Actions() }

// Actor is the caller's relationship to a post, computed by the handler so the workflow stays pure.
type Actor struct {
	IsAuthor  bool
	CanReview bool // holds blog.review
}

type rule struct {
	from     []State
	to       State
	reviewer bool // false: the author's action
}

var rules = map[Action]rule{
	Submit:         {from: []State{Draft, Unpublished}, to: InReview},
	Withdraw:       {from: []State{InReview}, to: Draft},
	RequestChanges: {from: []State{InReview}, to: Draft, reviewer: true},
	Publish:        {from: []State{InReview, Unpublished}, to: Published, reviewer: true},
	Unpublish:      {from: []State{Published}, to: Unpublished, reviewer: true},
}

// Actions lists every action, for validation messages.
func Actions() []string {
	return []string{string(Submit), string(Withdraw), string(RequestChanges), string(Publish), string(Unpublish)}
}

// Transition is the whole workflow. It is pure: the service loads the post, calls Transition,
// and saves the result. An action whose target is already the current state is a no-op, so a
// retried "publish" on a published post succeeds without changing anything.
func Transition(from State, a Action, who Actor) (State, error) {
	r, ok := rules[a]
	if !ok {
		return from, ekman.Unprocessable("invalid_action", "unknown action "+string(a))
	}
	if r.reviewer && !who.CanReview {
		return from, ekman.Forbidden("only a blog reviewer can " + string(a) + " a post")
	}
	if !r.reviewer && !who.IsAuthor {
		return from, ekman.Forbidden("only the author can " + string(a) + " a post")
	}
	if from == r.to {
		return from, nil
	}
	if !slices.Contains(r.from, from) {
		return from, ekman.Conflict("invalid_transition", "cannot "+string(a)+" a post that is "+string(from))
	}
	return r.to, nil
}

// CanEdit: authors edit their drafts (and unpublished posts, before resubmitting); reviewers
// edit anything.
func CanEdit(s State, who Actor) bool {
	return who.CanReview || (who.IsAuthor && (s == Draft || s == Unpublished))
}
