package people

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/sakib412/ekman"

	"github.com/BinarySUST/binarysust-web/api/internal/pg"
)

// SocialKind is one of github, linkedin, x, website, codeforces, leetcode, other.
type SocialKind string

// Social is a link on a profile.
type Social struct {
	Kind SocialKind `json:"kind"`
	URL  string     `json:"url"`
}

// Project is a portfolio project.
type Project struct {
	Title    string   `json:"title"`
	Summary  string   `json:"summary"`
	URL      *string  `json:"url"`
	RepoURL  *string  `json:"repoUrl"`
	ImageURL *string  `json:"imageUrl"`
	Tags     []string `json:"tags"`
}

// Achievement is a portfolio entry such as a contest result.
type Achievement struct {
	Title  string  `json:"title"`
	Date   *string `json:"date"` // YYYY-MM-DD
	Detail string  `json:"detail"`
}

// Portfolio is what a member edits about themselves. Identity fields (handle, name, email,
// batch, graduation) belong to people managers.
type Portfolio struct {
	Headline     string        `json:"headline"`
	BioMarkdown  string        `json:"bioMarkdown"`
	Workplace    string        `json:"workplace"`
	JobTitle     string        `json:"jobTitle"`
	Skills       []string      `json:"skills"`
	Socials      []Social      `json:"socials"`
	CVURL        *string       `json:"cvUrl"`
	Projects     []Project     `json:"projects"`
	Achievements []Achievement `json:"achievements"`
	// AvatarURL is optional on PUT: omitted or null keeps the current picture, "" removes it.
	AvatarURL *string `json:"avatarUrl,omitempty"`
}

var socialKinds = []string{"github", "linkedin", "x", "website", "codeforces", "leetcode", "other"}

// Enum lists the values.
func (SocialKind) Enum() []string { return socialKinds }

// Validate implements ekman.Validator.
func (p *Portfolio) Validate(v *ekman.Validation) {
	v.MaxLen("headline", p.Headline, 160)
	v.MaxLen("bioMarkdown", p.BioMarkdown, 20_000)
	v.MaxLen("workplace", p.Workplace, 120)
	v.MaxLen("jobTitle", p.JobTitle, 120)
	v.Check(len(p.Skills) <= 30, "skills", "at most 30 skills")
	for i, s := range p.Skills {
		p.Skills[i] = strings.TrimSpace(s)
		v.MaxLen("skills", s, 40)
	}
	v.Check(len(p.Socials) <= 12, "socials", "at most 12 links")
	for i, s := range p.Socials {
		f := "socials." + strconv.Itoa(i)
		v.OneOf(f+".kind", string(s.Kind), socialKinds...)
		v.Required(f+".kind", string(s.Kind))
		v.Required(f+".url", s.URL)
		v.URL(f+".url", s.URL)
	}
	optURL(v, "cvUrl", p.CVURL)
	if p.AvatarURL != nil && *p.AvatarURL != "" {
		v.URL("avatarUrl", *p.AvatarURL)
	}
	v.Check(len(p.Projects) <= 30, "projects", "at most 30 projects")
	for i, pr := range p.Projects {
		f := "projects." + strconv.Itoa(i)
		v.Required(f+".title", pr.Title)
		v.MaxLen(f+".title", pr.Title, 120)
		v.MaxLen(f+".summary", pr.Summary, 600)
		optURL(v, f+".url", pr.URL)
		optURL(v, f+".repoUrl", pr.RepoURL)
		optURL(v, f+".imageUrl", pr.ImageURL)
		v.Check(len(pr.Tags) <= 10, f+".tags", "at most 10 tags")
	}
	v.Check(len(p.Achievements) <= 50, "achievements", "at most 50 achievements")
	for i, a := range p.Achievements {
		f := "achievements." + strconv.Itoa(i)
		v.Required(f+".title", a.Title)
		v.MaxLen(f+".title", a.Title, 160)
		v.MaxLen(f+".detail", a.Detail, 600)
		optDate(v, f+".date", a.Date)
	}
}

func (s *Service) getPortfolio(ctx context.Context, c ekman.Caller, _ *struct{}) (Portfolio, error) {
	pid, err := s.acc.PersonOf(ctx, c.UserID())
	if err != nil {
		return Portfolio{}, err
	}
	return s.portfolio(ctx, pid)
}

func (s *Service) portfolio(ctx context.Context, personID string) (Portfolio, error) {
	var p Portfolio
	var socials, projects, achievements []byte
	err := s.db.Q(ctx).QueryRowContext(ctx, `
		SELECT headline, bio_md, workplace, job_title, skills, socials, cv_url, projects, achievements, avatar_url
		  FROM persons WHERE id = $1`, personID).Scan(&p.Headline, &p.BioMarkdown, &p.Workplace, &p.JobTitle,
		pg.Array(&p.Skills), &socials, &p.CVURL, &projects, &achievements, &p.AvatarURL)
	if err != nil {
		return p, err
	}
	if p.Skills == nil {
		p.Skills = []string{}
	}
	p.Socials = pg.JSONList[Social](socials)
	p.Projects = pg.JSONList[Project](projects)
	p.Achievements = pg.JSONList[Achievement](achievements)
	return p, nil
}

func (s *Service) putPortfolio(ctx context.Context, c ekman.Caller, in *Portfolio) (Portfolio, error) {
	pid, err := s.acc.PersonOf(ctx, c.UserID())
	if err != nil {
		return Portfolio{}, err
	}
	skills := in.Skills
	if skills == nil {
		skills = []string{}
	}
	for i := range in.Projects {
		if in.Projects[i].Tags == nil {
			in.Projects[i].Tags = []string{}
		}
	}
	socials, _ := json.Marshal(nonNil(in.Socials))
	projects, _ := json.Marshal(nonNil(in.Projects))
	achievements, _ := json.Marshal(nonNil(in.Achievements))
	_, err = s.db.Q(ctx).ExecContext(ctx, `
		UPDATE persons SET headline = $2, bio_md = $3, workplace = $4, job_title = $5, skills = $6,
		       socials = $7, cv_url = $8, projects = $9, achievements = $10,
		       avatar_url = CASE WHEN $11::text IS NULL THEN avatar_url ELSE nullif($11::text, '') END,
		       updated_at = now()
		 WHERE id = $1`, pid, in.Headline, in.BioMarkdown, in.Workplace, in.JobTitle, skills,
		string(socials), in.CVURL, string(projects), string(achievements), in.AvatarURL)
	if err != nil {
		return Portfolio{}, err
	}
	return s.portfolio(ctx, pid)
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
