// Package site owns the public contract: Snapshot, the single value the static site is built
// from (docs/snapshot.md). Build reads every module through its own Published function inside
// one read-only, repeatable-read transaction, so a snapshot is a consistent cut of the database.
// cmd/snapshot (CI) and GET /site/preview (dashboard) run the same Build.
package site

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sakib412/ekman"
	"github.com/sakib412/ekman/db"

	"github.com/BinarySUST/binarysust-web/api/internal/access"
	"github.com/BinarySUST/binarysust-web/api/internal/blog"
	"github.com/BinarySUST/binarysust-web/api/internal/events"
	"github.com/BinarySUST/binarysust-web/api/internal/people"
	"github.com/BinarySUST/binarysust-web/api/internal/sponsors"
	"github.com/BinarySUST/binarysust-web/api/internal/talks"
)

// Snapshot is everything public. Adding a content type = one field here and one line in Build.
type Snapshot struct {
	Version     Version               `json:"version"`
	Club        Club                  `json:"club"`
	CurrentTerm *people.TermRef       `json:"currentTerm"`
	Terms       []people.Term         `json:"terms"`      // newest first
	People      []people.PublicPerson `json:"people"`     // listed only
	Committees  []people.Committee    `json:"committees"` // every term, newest first
	Events      []events.PublicEvent  `json:"events"`     // published and cancelled, newest first
	Posts       []blog.PublicPost     `json:"posts"`      // published, newest first
	Talks       []talks.Public        `json:"talks"`      // newest first
	Sponsors    []sponsors.Public     `json:"sponsors"`   // highest tier first
	Redirects   []people.Redirect     `json:"redirects"`  // old handles
}

// Version says what a snapshot was built from. The deployed site serves it as /version.json.
type Version struct {
	ContentChangedAt time.Time `json:"contentChangedAt"` // site_settings.content_changed_at
	BuiltAt          time.Time `json:"builtAt"`
	Hash             string    `json:"hash"` // sha256 of everything except Version: unchanged hash, skip the deploy
}

// Club is the club's own description.
type Club struct {
	Name    string       `json:"name"`
	Tagline string       `json:"tagline"`
	About   string       `json:"about"`
	Email   string       `json:"email"`
	Socials []ClubSocial `json:"socials"`
	Stats   []Stat       `json:"stats"`
}

// ClubSocial is a club social link.
type ClubSocial struct {
	Kind string `json:"kind"`
	URL  string `json:"url"`
}

// Stat is a landing-page number. Label already agrees with Value ("1 member", "2 members").
type Stat struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

// club is edited here, in code review, rather than in the dashboard: it changes once a decade.
var club = Club{
	Name:    "Binary SUST",
	Tagline: "Technology for real life.",
	About: "Aiming to boost digital skills, foster teamwork, and inspire creativity among members. " +
		"Binary SUST is the student-run technology club of Shahjalal University of Science & Technology, Sylhet: " +
		"workshops, contests, career talks and live sessions, run by an elected executive committee every year.",
	Email: "binary.sust@gmail.com",
	Socials: []ClubSocial{
		{Kind: "facebook", URL: "https://www.facebook.com/binary.sust"},
		{Kind: "linkedin", URL: "https://www.linkedin.com/company/binary-sust"},
		{Kind: "github", URL: "https://github.com/BinarySUST"},
	},
}

// Sources are the modules Build reads, each through its own Published function.
type Sources struct {
	DB       *db.DB
	Access   *access.Service
	People   *people.Service
	Events   *events.Service
	Blog     *blog.Service
	Talks    *talks.Service
	Sponsors *sponsors.Service
}

// Build reads every module in one consistent, read-only transaction.
func Build(ctx context.Context, src Sources, now time.Time) (Snapshot, error) {
	var s Snapshot
	err := src.DB.Tx(ctx, func(ctx context.Context) error {
		// ekman's Tx begins with default options; this must be the first statement.
		if _, err := src.DB.Q(ctx).ExecContext(ctx, `SET TRANSACTION ISOLATION LEVEL REPEATABLE READ, READ ONLY`); err != nil {
			return err
		}
		changed, err := src.Access.ContentChangedAt(ctx)
		if err != nil {
			return err
		}
		pub, err := src.People.Published(ctx)
		if err != nil {
			return err
		}
		if s.Events, err = src.Events.Published(ctx); err != nil {
			return err
		}
		if s.Posts, err = src.Blog.Published(ctx); err != nil {
			return err
		}
		if s.Talks, err = src.Talks.Published(ctx); err != nil {
			return err
		}
		if s.Sponsors, err = src.Sponsors.Published(ctx); err != nil {
			return err
		}
		pastEvents, err := src.Events.PastCount(ctx)
		if err != nil {
			return err
		}
		s.Club = club
		s.Club.Stats = []Stat{
			stat(pub.Counts[people.StatusMember]+pub.Counts[people.StatusCommittee], "member", "members"),
			stat(pastEvents, "event run", "events run"),
			stat(len(s.Talks), "talk", "talks"),
			stat(pub.Counts[people.StatusAlumni], "alumnus", "alumni"),
		}
		s.CurrentTerm, s.Terms, s.People, s.Committees, s.Redirects =
			pub.CurrentTerm, pub.Terms, pub.People, pub.Committees, pub.Redirects
		s.Version.ContentChangedAt = changed
		return nil
	})
	if err != nil {
		return Snapshot{}, err
	}
	s.Version.BuiltAt = now.UTC().Truncate(time.Second)
	s.Version.Hash, err = contentHash(s)
	return s, err
}

// stat labels n with the form that reads right after it: "1 event run", "4 events run".
func stat(n int, one, many string) Stat {
	if n == 1 {
		return Stat{Label: one, Value: n}
	}
	return Stat{Label: many, Value: n}
}

func contentHash(s Snapshot) (string, error) {
	s.Version = Version{}
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// Service serves the dashboard's site endpoints.
type Service struct {
	src       Sources
	publicURL string
	client    *http.Client
	now       func() time.Time
}

// New returns the site service. publicURL is where the static site (and its /version.json) lives.
func New(src Sources, publicURL string) *Service {
	return &Service{src: src, publicURL: strings.TrimSuffix(publicURL, "/"),
		client: &http.Client{Timeout: 3 * time.Second}, now: time.Now}
}

// Module wires the site endpoints into the app. It owns no tables.
func (s *Service) Module() ekman.Module {
	return ekman.Module{
		Name: "site",
		Routes: func(r *ekman.Router) {
			ekman.Handle(r, "GET /site/status", ekman.SignedIn(), s.status, ekman.Summary("Is my change live?"))
			ekman.Handle(r, "GET /site/preview", ekman.SignedIn(), s.preview,
				ekman.Summary("The snapshot the next build would publish"))
		},
	}
}

func (s *Service) preview(ctx context.Context, _ ekman.Caller, _ *struct{}) (Snapshot, error) {
	return Build(ctx, s.src, s.now())
}

// LiveVersion is the deployed site's /version.json, as far as status needs it.
type LiveVersion struct {
	ContentChangedAt time.Time `json:"contentChangedAt"`
}

// PublishState is "live" when the deployed site includes the latest change, else "publishing".
type PublishState string

// Enum lists the values.
func (PublishState) Enum() []string { return []string{"live", "publishing"} }

// Status answers "is my change live?" by comparing timestamps; it never builds anything.
type Status struct {
	ContentChangedAt time.Time    `json:"contentChangedAt"`
	Live             *LiveVersion `json:"live"` // null when /version.json could not be read
	State            PublishState `json:"state"`
}

func (s *Service) status(ctx context.Context, _ ekman.Caller, _ *struct{}) (Status, error) {
	changed, err := s.src.Access.ContentChangedAt(ctx)
	if err != nil {
		return Status{}, err
	}
	st := Status{ContentChangedAt: changed, Live: s.live(ctx)}
	st.State = StateOf(changed, st.Live)
	return st, nil
}

// StateOf compares the latest change with what is deployed.
func StateOf(changed time.Time, live *LiveVersion) PublishState {
	if live != nil && !live.ContentChangedAt.Before(changed.Truncate(time.Microsecond)) {
		return "live"
	}
	return "publishing"
}

// live fetches PUBLIC_URL/version.json; any failure is just "unknown".
func (s *Service) live(ctx context.Context) *LiveVersion {
	if s.publicURL == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.publicURL+"/version.json", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Cache-Control", "no-cache")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var v LiveVersion
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&v); err != nil || v.ContentChangedAt.IsZero() {
		return nil
	}
	v.ContentChangedAt = v.ContentChangedAt.UTC()
	return &v
}
