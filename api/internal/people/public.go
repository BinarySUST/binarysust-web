package people

import (
	"context"
	"time"

	"github.com/BinarySUST/binarysust-web/api/internal/md"
	"github.com/BinarySUST/binarysust-web/api/internal/pg"
)

// PublicPerson is a person on the public site (docs/snapshot.md).
type PublicPerson struct {
	Handle    string  `json:"handle"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatarUrl"`
	Headline  string  `json:"headline"`
	Status    Status  `json:"status"`
	// LifetimeMember: held a lifetime position (President, General Secretary, Director of
	// Training) in any term. Independent of status: a lifetime member can also be alumni.
	LifetimeMember bool           `json:"lifetimeMember"`
	Department     string         `json:"department"`
	Batch          *int           `json:"batch"`
	GraduatedOn    *string        `json:"graduatedOn"`
	Workplace      string         `json:"workplace"`
	JobTitle       string         `json:"jobTitle"`
	BioHTML        string         `json:"bioHtml"`
	Skills         []string       `json:"skills"`
	Socials        []Social       `json:"socials"`
	CVURL          *string        `json:"cvUrl"`
	Projects       []Project      `json:"projects"`
	Achievements   []Achievement  `json:"achievements"`
	Positions      []PositionHeld `json:"positions"` // committee history, newest first
}

// PositionHeld is one seat in a person's committee history.
type PositionHeld struct {
	Term  string `json:"term"`
	Title string `json:"title"`
}

// Seat is a seat on a public committee page.
type Seat struct {
	Handle    string   `json:"handle"`
	Name      string   `json:"name"`
	AvatarURL *string  `json:"avatarUrl"`
	Position  string   `json:"position"`
	Category  Category `json:"category"`
	Group     Group    `json:"group"`
	Rank      int      `json:"rank"`
}

// Committee is one term's committee.
type Committee struct {
	Term  string `json:"term"`
	Seats []Seat `json:"seats"`
}

// Term is a public term.
type Term struct {
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	StartsOn string `json:"startsOn"`
}

// TermRef names the current term.
type TermRef struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

// Redirect is an old public path that moved.
type Redirect struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Public is everything public this module owns.
type Public struct {
	People      []PublicPerson
	Committees  []Committee
	Terms       []Term
	CurrentTerm *TermRef
	Redirects   []Redirect
	Counts      map[Status]int // listed people by status
}

// Published reads the public view. Unlisted people are left out everywhere, including seats.
func (s *Service) Published(ctx context.Context) (Public, error) {
	var out Public
	term, err := s.currentTerm(ctx)
	if err != nil {
		return out, err
	}
	today := Today(s.now())

	// Committee history per person, and who ever held a lifetime position.
	history := map[string][]PositionHeld{}
	lifetime := map[string]bool{}
	rows, err := s.db.Q(ctx).QueryContext(ctx, `
		SELECT cs.person_id, t.slug, pos.title, pos.lifetime
		  FROM committee_seats cs JOIN terms t ON t.id = cs.term_id JOIN positions pos ON pos.id = cs.position_id
		 ORDER BY t.starts_on DESC, pos.category DESC, pos.rank, pos.title`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var pid string
		var h PositionHeld
		var life bool
		if err := rows.Scan(&pid, &h.Term, &h.Title, &life); err != nil {
			rows.Close()
			return out, err
		}
		history[pid] = append(history[pid], h)
		lifetime[pid] = lifetime[pid] || life
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}

	rows, err = s.db.Q(ctx).QueryContext(ctx, `
		SELECT p.id, p.handle, p.name, p.affiliation, p.avatar_url, p.headline, p.department, p.batch,
		       p.member_since, p.graduated_on, p.workplace, p.job_title, p.bio_md, p.skills, p.socials,
		       p.cv_url, p.projects, p.achievements`+factsCols+`
		  FROM persons p WHERE p.listed ORDER BY p.name, p.handle`, term)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	out.Counts = map[Status]int{}
	for rows.Next() {
		var p PublicPerson
		var id string
		var aff Affiliation
		var member, grad *time.Time
		var bio string
		var socials, projects, achievements []byte
		var seat *string
		var everAdvisor bool
		if err := rows.Scan(&id, &p.Handle, &p.Name, &aff, &p.AvatarURL, &p.Headline, &p.Department, &p.Batch,
			&member, &grad, &p.Workplace, &p.JobTitle, &bio, pg.Array(&p.Skills), &socials,
			&p.CVURL, &projects, &achievements, &seat, &everAdvisor); err != nil {
			return out, err
		}
		p.Status = StatusOf(Facts{Affiliation: aff, MemberSince: member, GraduatedOn: grad,
			CurrentSeat: (*Category)(seat), EverAdvisor: everAdvisor}, today)
		p.GraduatedOn = fmtDate(grad)
		p.BioHTML = md.Render(bio)
		p.Skills = nonNil(p.Skills)
		p.Socials = pg.JSONList[Social](socials)
		p.Projects = pg.JSONList[Project](projects)
		for i := range p.Projects {
			p.Projects[i].Tags = nonNil(p.Projects[i].Tags)
		}
		p.Achievements = pg.JSONList[Achievement](achievements)
		p.Positions = nonNil(history[id])
		p.LifetimeMember = lifetime[id]
		out.People = append(out.People, p)
		out.Counts[p.Status]++
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	rows.Close()

	if out.Terms, out.CurrentTerm, err = s.publicTerms(ctx, term); err != nil {
		return out, err
	}
	if out.Committees, err = s.publicCommittees(ctx, out.Terms); err != nil {
		return out, err
	}
	out.Redirects, err = s.redirects(ctx)
	out.People = nonNil(out.People)
	return out, err
}

func (s *Service) publicTerms(ctx context.Context, current *string) ([]Term, *TermRef, error) {
	rows, err := s.db.Q(ctx).QueryContext(ctx,
		`SELECT slug, label, starts_on, id = $1::uuid FROM terms ORDER BY starts_on DESC`, current)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	terms := []Term{}
	var ref *TermRef
	for rows.Next() {
		var t Term
		var starts time.Time
		var isCurrent *bool
		if err := rows.Scan(&t.Slug, &t.Label, &starts, &isCurrent); err != nil {
			return nil, nil, err
		}
		t.StartsOn = starts.Format(time.DateOnly)
		if isCurrent != nil && *isCurrent {
			ref = &TermRef{Slug: t.Slug, Label: t.Label}
		}
		terms = append(terms, t)
	}
	return terms, ref, rows.Err()
}

func (s *Service) publicCommittees(ctx context.Context, terms []Term) ([]Committee, error) {
	rows, err := s.db.Q(ctx).QueryContext(ctx, `
		SELECT t.slug, p.handle, p.name, p.avatar_url, pos.title, pos.category, pos.display_group, pos.rank
		  FROM committee_seats cs
		  JOIN terms t ON t.id = cs.term_id
		  JOIN positions pos ON pos.id = cs.position_id
		  JOIN persons p ON p.id = cs.person_id
		 WHERE p.listed
		 ORDER BY t.starts_on DESC, pos.category DESC, pos.rank, p.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byTerm := map[string][]Seat{}
	for rows.Next() {
		var term string
		var st Seat
		if err := rows.Scan(&term, &st.Handle, &st.Name, &st.AvatarURL, &st.Position, &st.Category, &st.Group, &st.Rank); err != nil {
			return nil, err
		}
		byTerm[term] = append(byTerm[term], st)
	}
	out := []Committee{}
	for _, t := range terms { // newest first, like terms
		if seats, ok := byTerm[t.Slug]; ok {
			out = append(out, Committee{Term: t.Slug, Seats: seats})
		}
	}
	return out, rows.Err()
}

func (s *Service) redirects(ctx context.Context) ([]Redirect, error) {
	rows, err := s.db.Q(ctx).QueryContext(ctx, `
		SELECT h.handle, p.handle FROM person_handles h JOIN persons p ON p.id = h.person_id
		 WHERE p.listed ORDER BY h.handle`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Redirect{}
	for rows.Next() {
		var from, to string
		if err := rows.Scan(&from, &to); err != nil {
			return nil, err
		}
		out = append(out, Redirect{From: "/people/" + from, To: "/people/" + to})
	}
	return out, rows.Err()
}
