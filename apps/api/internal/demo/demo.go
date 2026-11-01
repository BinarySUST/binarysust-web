// Package demo is the demo dataset behind `make seed`: two committee terms on the
// constitution's positions (about 25 executive seats in the current term, plus the advisory
// committee), the default position -> role mapping, people across every status, events past and upcoming (one with registration open),
// posts in every review state, talks and sponsors. Seed is idempotent: rows are upserted by
// their natural keys (handle, slug, name), and event dates move with today.
package demo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type person struct {
	handle, name, affiliation, dept, headline string
	batch                                     int
	memberSince, graduatedOn                  string
	workplace, jobTitle, bio                  string
	skills                                    []string
	socials                                   string // jsonb
	listed                                    bool
}

var persons = []person{
	{handle: "tahmid", name: "Tahmid Rahman", affiliation: "student", dept: "CSE", batch: 2021, memberSince: "2021-11-01",
		headline: "Competitive programmer who ended up liking backends", skills: []string{"Go", "C++", "PostgreSQL", "Algorithms", "Linux"},
		bio:     "Fourth-year CSE student. Two-time **ICPC Dhaka Regional** finalist, and the person who keeps the club's servers alive.",
		socials: `[{"kind":"github","url":"https://github.com/"},{"kind":"codeforces","url":"https://codeforces.com/"}]`, listed: true},
	{handle: "nusrat", name: "Nusrat Jahan", affiliation: "student", dept: "CSE", batch: 2021, memberSince: "2021-12-01",
		headline: "Organizer, writer, occasional designer", skills: []string{"TypeScript", "React", "Writing"},
		bio: "Runs the club's paperwork so everyone else can run contests.", socials: `[{"kind":"linkedin","url":"https://www.linkedin.com/"}]`, listed: true},
	{handle: "arif", name: "Arif Hossain", affiliation: "student", dept: "EEE", batch: 2022, memberSince: "2022-11-01",
		headline: "Hardware tinkerer", skills: []string{"Embedded C", "KiCad"}, bio: "Builds things that blink.", socials: `[]`, listed: true},
	{handle: "sadia", name: "Sadia Afrin", affiliation: "student", dept: "CSE", batch: 2022, memberSince: "2022-11-15",
		headline: "Data and dashboards", skills: []string{"Python", "SQL", "pandas"}, bio: "Keeps the budget honest.", socials: `[]`, listed: true},
	{handle: "rakib", name: "Rakibul Islam", affiliation: "student", dept: "CSE", batch: 2022, memberSince: "2022-11-01",
		headline: "Gopher and tooling nerd", skills: []string{"Go", "Docker", "CI"}, bio: "Teaches the Go workshop every semester.", socials: `[{"kind":"github","url":"https://github.com/"}]`, listed: true},
	{handle: "farhan", name: "Farhan Kabir", affiliation: "student", dept: "SWE", batch: 2023, memberSince: "2023-11-01",
		headline: "Events and logistics", skills: []string{"Planning"}, bio: "Finds rooms, chairs and pizza.", socials: `[]`, listed: true},
	{handle: "mkarim", name: "Dr. Mahbub Karim", affiliation: "faculty", dept: "CSE",
		headline: "Associate Professor, CSE", skills: []string{"Compilers", "Performance"}, bio: "Faculty advisor since the club began.", socials: `[]`, listed: true},
	{handle: "shirin", name: "Dr. Shirin Akter", affiliation: "faculty", dept: "CSE",
		headline: "Assistant Professor, CSE", skills: []string{"Machine learning"}, bio: "Advises the club's ML study group.", socials: `[]`, listed: true},
	{handle: "tanvir", name: "Tanvir Ahmed", affiliation: "student", dept: "CSE", batch: 2023, memberSince: "2024-01-10",
		headline: "Learning everything at once", skills: []string{"Java", "Kotlin"}, bio: "Android apps and contest problems.", socials: `[]`, listed: true},
	{handle: "ishrat", name: "Ishrat Jahan", affiliation: "student", dept: "Math", batch: 2023, memberSince: "2024-02-01",
		headline: "Math major, competitive programmer", skills: []string{"C++", "Number theory"}, bio: "Sets the hard problems.", socials: `[]`, listed: true},
	{handle: "lamia", name: "Lamia Chowdhury", affiliation: "student", dept: "CSE", batch: 2019, memberSince: "2019-11-01", graduatedOn: "2025-06-30",
		headline: "Software engineer", workplace: "Surma Cloud", jobTitle: "Backend engineer", skills: []string{"Go", "Kubernetes"},
		bio: "Former president. Now ships backends in Dhaka.", socials: `[{"kind":"linkedin","url":"https://www.linkedin.com/"}]`, listed: true},
	{handle: "nafis", name: "Nafis Iqbal", affiliation: "student", dept: "CSE", batch: 2019, memberSince: "2019-12-01", graduatedOn: "2025-06-30",
		headline: "ML engineer", workplace: "Berlin AI Labs", jobTitle: "ML engineer", skills: []string{"Python", "PyTorch"},
		bio: "Former Director of Training.", socials: `[]`, listed: true},
	{handle: "mehedi", name: "Mehedi Hasan", affiliation: "student", dept: "CSE", batch: 2018, memberSince: "2018-11-01", graduatedOn: "2023-01-31",
		headline: "Engineering manager", workplace: "Lion City Tech", jobTitle: "Engineering manager", skills: []string{"Leadership"},
		bio: "Alumni mentor.", socials: `[]`, listed: true},
	{handle: "sabbir", name: "Sabbir Hossain", affiliation: "guest", dept: "",
		headline: "Security researcher", skills: []string{"Security"}, bio: "Guest speaker on web security.", socials: `[]`, listed: true},
	// The rest of the demo committee: made-up names, filled in only enough for the committee page.
	member("mahin", "Mahin Chowdhury", "CSE", 2022), member("rifat", "Rifat Hasan", "SWE", 2023),
	member("sumaiya", "Sumaiya Islam", "CSE", 2022), member("nabil", "Nabil Ahsan", "EEE", 2023),
	member("tasnim", "Tasnim Ara", "CSE", 2023), member("anika", "Anika Tabassum", "CSE", 2022),
	member("fahim", "Fahim Shahriar", "SWE", 2023), member("shuvo", "Shuvo Debnath", "CSE", 2022),
	member("raisa", "Raisa Noor", "CSE", 2023), member("zarif", "Zarif Mahmud", "IPE", 2022),
	member("maliha", "Maliha Rahman", "CSE", 2022), member("imran", "Imran Kabir", "CSE", 2023),
	member("priya", "Priya Das", "Math", 2022), member("jubayer", "Jubayer Alam", "CSE", 2023),
	member("rumana", "Rumana Akter", "CSE", 2022), member("sohel", "Sohel Rana", "EEE", 2022),
	member("afia", "Afia Nawar", "CSE", 2024), member("tamim", "Tamim Iqbal Khan", "SWE", 2024),
	member("nafisa", "Nafisa Haque", "CSE", 2024),
	faculty("ahossain", "Prof. Dr. Anwar Hossain", "Vice-Chancellor"),
	faculty("fmehjabin", "Dr. Farzana Mehjabin", "Professor, CSE"),
	faculty("ashahriar", "Dr. Asif Shahriar", "Associate Professor, CSE"),
	faculty("ntabassum", "Dr. Nazia Tabassum", "Assistant Professor, SWE"),
	faculty("rkabir", "Dr. Rashed Kabir", "Associate Professor, EEE"),
}

func member(handle, name, dept string, batch int) person {
	return person{handle: handle, name: name, affiliation: "student", dept: dept, batch: batch,
		memberSince: fmt.Sprintf("%d-11-01", batch), headline: dept + " student", skills: []string{}, socials: `[]`, listed: true}
}

func faculty(handle, name, headline string) person {
	return person{handle: handle, name: name, affiliation: "faculty", headline: headline, skills: []string{}, socials: `[]`, listed: true}
}

// PositionRoles is the default position -> role mapping (api/README.md). Positions themselves
// are reference data (people/migrations/0002_positions.sql); which roles they carry is data a
// roles manager edits in the dashboard, so only the seed fills these defaults.
var PositionRoles = map[string][]string{
	"president":                            {"admin"},
	"general-secretary":                    {"admin"},
	"director-of-training":                 {"event_manager"},
	"event-management-secretary":           {"event_manager"},
	"assistant-event-management-secretary": {"event_manager"},
	"publication-secretary":                {"blog_editor"},
	"assistant-publication-secretary":      {"blog_editor"},
	"it-design-secretary":                  {"sponsor_manager", "talks_manager"},
	"office-secretary":                     {"people_manager"},
	"internal-affairs-secretary":           {"people_manager"},
	"organizing-secretary":                 {"sponsor_manager"},
	"senior-executive-member":              {"checkin_volunteer"},
	"executive-member":                     {"checkin_volunteer"},
}

// legacyPositions were demo-only positions before the constitution's list; seeding removes them.
var legacyPositions = []string{"treasurer", "tech-lead", "event-secretary", "chief-advisor", "faculty-advisor"}

// seats: term slug -> position slug -> handles. Several people may share a position.
var seats = map[string]map[string][]string{
	"2024-25": {
		"president": {"lamia"}, "general-secretary": {"tahmid"}, "director-of-training": {"nafis"},
		"organizing-secretary": {"arif"}, "event-management-secretary": {"nusrat"}, "publication-secretary": {"sadia"},
		"it-design-secretary": {"rakib"}, "assistant-it-design-secretary": {"shuvo"}, "finance-secretary": {"maliha"},
		"executive-member": {"farhan", "anika", "sumaiya"},
		"chief-patron":     {"ahossain"}, "chief-adviser": {"mkarim"}, "adviser": {"shirin", "fmehjabin", "ashahriar"},
	},
	"2025-26": {
		"president": {"tahmid"}, "general-secretary": {"nusrat"}, "director-of-training": {"rakib"},
		"project-coordinator": {"sadia"}, "assistant-general-secretary": {"mahin", "rifat"},
		"organizing-secretary": {"arif"}, "assistant-organizing-secretary": {"farhan", "nabil"},
		"event-management-secretary": {"sumaiya"}, "assistant-event-management-secretary": {"tasnim", "raisa"},
		"publication-secretary": {"anika"}, "assistant-publication-secretary": {"fahim"},
		"it-design-secretary": {"shuvo"}, "assistant-it-design-secretary": {"jubayer"},
		"finance-secretary": {"zarif"}, "office-secretary": {"maliha"}, "assistant-office-secretary": {"imran"},
		"internal-affairs-secretary": {"priya"}, "assistant-internal-affairs-secretary": {"rumana"},
		"senior-executive-member": {"sohel", "afia"}, "executive-member": {"tamim", "nafisa"},
		"chief-patron": {"ahossain"}, "chief-adviser": {"mkarim"},
		"adviser": {"shirin", "fmehjabin", "ashahriar", "ntabassum", "rkabir"},
	},
}

// Execer is what Seed needs from a database handle.
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Seed upserts the demo data. q is usually a transaction (db.DB.Q inside db.DB.Tx).
func Seed(ctx context.Context, q Execer, now time.Time, adminEmail string) error {
	ids := map[string]string{}
	for _, p := range persons {
		var batch any
		if p.batch != 0 {
			batch = p.batch
		}
		var id string
		err := q.QueryRowContext(ctx, `
			INSERT INTO persons (handle, name, email, affiliation, listed, headline, department, batch, member_since,
			                     graduated_on, workplace, job_title, bio_md, skills, socials)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
			ON CONFLICT (handle) DO UPDATE SET name = EXCLUDED.name, email = EXCLUDED.email, affiliation = EXCLUDED.affiliation,
			   listed = EXCLUDED.listed, headline = EXCLUDED.headline, department = EXCLUDED.department, batch = EXCLUDED.batch,
			   member_since = EXCLUDED.member_since, graduated_on = EXCLUDED.graduated_on, workplace = EXCLUDED.workplace,
			   job_title = EXCLUDED.job_title, bio_md = EXCLUDED.bio_md, skills = EXCLUDED.skills, socials = EXCLUDED.socials,
			   updated_at = now()
			RETURNING id`,
			p.handle, p.name, p.handle+"@binarysust.local", p.affiliation, p.listed, p.headline, p.dept, batch,
			nullable(p.memberSince), nullable(p.graduatedOn), p.workplace, p.jobTitle, p.bio, p.skills, p.socials).Scan(&id)
		if err != nil {
			return fmt.Errorf("person %s: %w", p.handle, err)
		}
		ids[p.handle] = id
	}
	// Projects and achievements for one fully filled-in portfolio.
	if _, err := q.ExecContext(ctx, `UPDATE persons SET projects = $2, achievements = $3 WHERE handle = $1`, "tahmid",
		`[{"title":"ekman","summary":"An API-only Go framework where every route declares its access rule.","url":null,"repoUrl":"https://github.com/sakib412/ekman","imageUrl":null,"tags":["go","framework","open source"]}]`,
		`[{"title":"ICPC Asia Dhaka Regional, 14th place","date":"2025-12-06","detail":"Team SUST_Binary_Bandits."}]`); err != nil {
		return err
	}
	// An old handle that now redirects.
	if _, err := q.ExecContext(ctx, `INSERT INTO person_handles (handle, person_id) VALUES ('tahmid-r', $1)
		ON CONFLICT (handle) DO UPDATE SET person_id = EXCLUDED.person_id`, ids["tahmid"]); err != nil {
		return err
	}

	terms := map[string]string{}
	for _, t := range []struct{ slug, label, starts string }{{"2024-25", "2024–25", "2024-10-01"}, {"2025-26", "2025–26", "2025-10-01"}} {
		var id string
		if err := q.QueryRowContext(ctx, `
			INSERT INTO terms (slug, label, starts_on) VALUES ($1, $2, $3)
			ON CONFLICT (slug) DO UPDATE SET label = EXCLUDED.label, starts_on = EXCLUDED.starts_on RETURNING id`,
			t.slug, t.label, t.starts).Scan(&id); err != nil {
			return fmt.Errorf("term %s: %w", t.slug, err)
		}
		terms[t.slug] = id
	}
	// Positions come from the migration; drop the old demo-only ones and their seats.
	if _, err := q.ExecContext(ctx, `DELETE FROM committee_seats WHERE position_id IN (SELECT id FROM positions WHERE slug = ANY($1))`, legacyPositions); err != nil {
		return err
	}
	if _, err := q.ExecContext(ctx, `DELETE FROM positions WHERE slug = ANY($1)`, legacyPositions); err != nil {
		return err
	}
	posIDs := map[string]string{}
	posID := func(slug string) (string, error) {
		if id, ok := posIDs[slug]; ok {
			return id, nil
		}
		var id string
		if err := q.QueryRowContext(ctx, `SELECT id FROM positions WHERE slug = $1`, slug).Scan(&id); err != nil {
			return "", fmt.Errorf("position %s (from migrations/0002_positions.sql): %w", slug, err)
		}
		posIDs[slug] = id
		return id, nil
	}
	for slug, roles := range PositionRoles {
		id, err := posID(slug)
		if err != nil {
			return err
		}
		for _, r := range roles {
			if _, err := q.ExecContext(ctx, `INSERT INTO position_roles (position_id, role) VALUES ($1, $2) ON CONFLICT DO NOTHING`, id, r); err != nil {
				return err
			}
		}
	}
	// The demo terms' seats are replaced wholesale, so re-seeding matches this file exactly.
	for term, byPos := range seats {
		if _, err := q.ExecContext(ctx, `DELETE FROM committee_seats WHERE term_id = $1`, terms[term]); err != nil {
			return err
		}
		for pos, handles := range byPos {
			id, err := posID(pos)
			if err != nil {
				return err
			}
			for _, h := range handles {
				pid, ok := ids[h]
				if !ok {
					return fmt.Errorf("seat %s %s: no demo person %q", term, pos, h)
				}
				if _, err := q.ExecContext(ctx, `INSERT INTO committee_seats (term_id, position_id, person_id) VALUES ($1, $2, $3)`,
					terms[term], id, pid); err != nil {
					return fmt.Errorf("seat %s %s %s: %w", term, pos, h, err)
				}
			}
		}
	}
	if _, err := q.ExecContext(ctx, `UPDATE site_settings SET current_term_id = $1, content_changed_at = now()`, terms["2025-26"]); err != nil {
		return err
	}

	if adminEmail != "" {
		var id string
		if err := q.QueryRowContext(ctx, `
			INSERT INTO persons (handle, name, email, affiliation, listed) VALUES ('dev-admin', 'Dev Admin', $1, 'guest', false)
			ON CONFLICT (handle) DO UPDATE SET email = EXCLUDED.email RETURNING id`, strings.ToLower(strings.TrimSpace(adminEmail))).Scan(&id); err != nil {
			return fmt.Errorf("dev admin: %w", err)
		}
		if _, err := q.ExecContext(ctx, `
			INSERT INTO role_grants (person_id, role) VALUES ($1, 'admin')
			ON CONFLICT (person_id, role, coalesce(term_id, '00000000-0000-0000-0000-000000000000')) DO NOTHING`, id); err != nil {
			return fmt.Errorf("dev admin grant: %w", err)
		}
	}
	// Re-link accounts to persons by email (same rule as sign-in).
	if _, err := q.ExecContext(ctx, `UPDATE accounts a SET person_id = p.id FROM persons p
		WHERE p.email = a.email AND a.person_id IS DISTINCT FROM p.id`); err != nil {
		return err
	}

	if err := seedEvents(ctx, q, now, ids); err != nil {
		return err
	}
	if err := seedPosts(ctx, q, now, ids); err != nil {
		return err
	}
	if err := seedTalks(ctx, q, now, ids); err != nil {
		return err
	}
	return seedSponsors(ctx, q)
}

type event struct {
	slug, title, kind, status, summary, body, venue, recap string
	starts, ends                                           time.Time
	opens, closes                                          *time.Time
	capacity                                               *int
	speakers                                               []string
	photos                                                 [][2]string
}

func ptr[T any](v T) *T { return &v }

// dhaka is the club's clock. Seeded events happen at local hours people would hold them, stored
// as UTC like everything else (a time built from "now" plus whole days lands at whatever hour the
// seed ran, often midnight in Sylhet).
var dhaka = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Dhaka"); err == nil {
		return loc
	}
	return time.FixedZone("Asia/Dhaka", 6*60*60) // no tzdata on this machine; Bangladesh has no DST
}()

// at is days after now's date in Dhaka, at hour:minute Dhaka time, in UTC.
func at(now time.Time, days, hour, minute int) time.Time {
	d := now.In(dhaka).AddDate(0, 0, days)
	return time.Date(d.Year(), d.Month(), d.Day(), hour, minute, 0, 0, dhaka).UTC()
}

func seedEvents(ctx context.Context, q Execer, now time.Time, ids map[string]string) error {
	list := []event{
		{slug: "intro-to-go-workshop", title: "Intro to Go: build a JSON API in an afternoon", kind: "workshop", status: "published",
			summary: "Bring a laptop. Leave with a working API, tests and a Dockerfile.",
			body:    "## What you'll build\n\nA small **JSON API** with Go's standard library, then the same API on ekman.\n\n- Routing with `net/http`\n- Tests with `httptest`\n- A Dockerfile that ships a 10 MB image\n\nInstall Go 1.25 before you come.",
			venue:   "Lab 3, IICT Building, SUST", starts: at(now, 12, 10, 0), ends: at(now, 12, 13, 0),
			opens: ptr(at(now, -2, 10, 0)), closes: ptr(at(now, 9, 23, 59)), capacity: ptr(40), speakers: []string{"rakib", "tahmid"}},
		{slug: "sust-hackathon-2026", title: "SUST Hackathon 2026", kind: "hackathon", status: "published",
			summary: "36 hours, teams of three, problems from local startups.",
			body:    "Registration opens soon. Teams of up to three; mentors on site all night.",
			venue:   "Central Auditorium, SUST", starts: at(now, 40, 9, 0), ends: at(now, 41, 21, 0),
			opens: ptr(at(now, 10, 10, 0)), closes: ptr(at(now, 30, 23, 59)), capacity: ptr(120), speakers: []string{"lamia"}},
		{slug: "career-meetup-2026", title: "Alumni career meetup", kind: "meetup", status: "published",
			summary: "Alumni from Dhaka, Berlin and Singapore on first jobs, interviews and moving abroad.",
			body:    "An evening of short talks and long questions. No registration: just turn up.",
			venue:   "Central Auditorium, SUST", starts: at(now, 20, 18, 0), ends: at(now, 20, 21, 0),
			speakers: []string{"lamia", "nafis", "mehedi"}},
		{slug: "icpc-prep-contest", title: "ICPC preparation contest", kind: "contest", status: "published",
			summary: "Five hours, twelve problems, one leaderboard.",
			body:    "A full-length mock contest in ICPC format.", venue: "Lab 1 and Lab 2, IICT Building, SUST",
			starts: at(now, -30, 9, 0), ends: at(now, -30, 14, 0),
			opens: ptr(at(now, -50, 10, 0)), closes: ptr(at(now, -31, 23, 59)), capacity: ptr(120),
			recap:    "**94 teams** took part. Congratulations to *SUST_Binary_Bandits* for solving 10 of 12.",
			photos:   [][2]string{{"https://picsum.photos/seed/icpc1/1200/800", "Teams at work"}, {"https://picsum.photos/seed/icpc2/1200/800", "The final standings"}},
			speakers: []string{"ishrat"}},
		{slug: "web-security-seminar", title: "Web security for builders", kind: "seminar", status: "cancelled",
			summary: "Postponed; we will announce a new date.", body: "This seminar was cancelled.",
			venue: "Room 101, Academic Building A", starts: at(now, 5, 15, 0), ends: at(now, 5, 17, 0), speakers: []string{"sabbir"}},
		{slug: "ml-study-group-kickoff", title: "ML study group kickoff", kind: "seminar", status: "draft",
			summary: "Draft: not public yet.", body: "Agenda to be decided.", venue: "Room 204, Academic Building B",
			starts: at(now, 25, 15, 0), ends: at(now, 25, 17, 0), speakers: []string{"shirin"}},
	}
	for _, e := range list {
		var id string
		err := q.QueryRowContext(ctx, `
			INSERT INTO events (slug, title, kind, status, summary, body_md, venue, starts_at, ends_at, reg_opens_at,
			                    reg_closes_at, capacity, recap_md)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, kind = EXCLUDED.kind, status = EXCLUDED.status,
			   summary = EXCLUDED.summary, body_md = EXCLUDED.body_md, venue = EXCLUDED.venue, starts_at = EXCLUDED.starts_at,
			   ends_at = EXCLUDED.ends_at, reg_opens_at = EXCLUDED.reg_opens_at, reg_closes_at = EXCLUDED.reg_closes_at,
			   capacity = EXCLUDED.capacity, recap_md = EXCLUDED.recap_md, updated_at = now()
			RETURNING id`,
			e.slug, e.title, e.kind, e.status, e.summary, e.body, e.venue, e.starts, e.ends, e.opens, e.closes, e.capacity, e.recap).Scan(&id)
		if err != nil {
			return fmt.Errorf("event %s: %w", e.slug, err)
		}
		if _, err := q.ExecContext(ctx, `DELETE FROM event_speakers WHERE event_id = $1`, id); err != nil {
			return err
		}
		for i, h := range e.speakers {
			if _, err := q.ExecContext(ctx, `INSERT INTO event_speakers (event_id, person_id, sort) VALUES ($1, $2, $3)`, id, ids[h], i); err != nil {
				return err
			}
		}
		if _, err := q.ExecContext(ctx, `DELETE FROM event_photos WHERE event_id = $1`, id); err != nil {
			return err
		}
		for i, ph := range e.photos {
			if _, err := q.ExecContext(ctx, `INSERT INTO event_photos (event_id, url, caption, sort) VALUES ($1, $2, $3, $4)`, id, ph[0], ph[1], i); err != nil {
				return err
			}
		}
	}
	return nil
}

func seedPosts(ctx context.Context, q Execer, now time.Time, ids map[string]string) error {
	list := []struct {
		slug, title, author, state, excerpt, body string
		published                                 *time.Time
		tags                                      []string
	}{
		{"a-static-site-from-one-snapshot", "How this site is built: one snapshot, zero servers", "tahmid", "published",
			"The public site never talks to our API. Here is why a sleeping free-tier server doesn't matter.",
			"Our API runs on a free tier that sleeps after fifteen idle minutes.\n\n## The trick\n\nEvery page is **pre-rendered** from one JSON snapshot of the database. When someone edits content, the API asks GitHub to rebuild the site.\n\n```go\nekman.Handle(r, \"POST /events/{slug}/rsvp\", ekman.SignedIn(), s.rsvp, republish.Skip())\n```\n\nSo visitors never wait for the server to wake up.",
			ptr(at(now, -13, 19, 30)), []string{"engineering", "go", "web"}},
		{"icpc-dhaka-2025-recap", "ICPC Dhaka 2025: what we learned", "ishrat", "published",
			"Three teams, one long day, and a lot of number theory.",
			"We sent three teams to the **ICPC Asia Dhaka Regional**. Here is what worked, what didn't, and what we are practising next.\n\n1. Read every problem in the first 20 minutes.\n2. One keyboard, three brains.\n3. Sleep the night before.",
			ptr(at(now, -40, 20, 15)), []string{"icpc", "contests"}},
		{"go-workshop-notes", "Notes from the Go workshop", "rakib", "published",
			"Slides, code and the questions you asked.",
			"Thanks to everyone who came. The code is on GitHub, and the most-asked question was *why no framework?*. Short answer: start with `net/http`.",
			ptr(at(now, -70, 11, 0)), []string{"go", "workshops"}},
		{"joining-binary-sust", "How to join Binary SUST", "nusrat", "in_review",
			"Short version: turn up.", "Draft awaiting review.", nil, []string{"club"}},
		{"my-first-hackathon", "My first hackathon", "tanvir", "draft", "", "Still writing this one.", nil, nil},
	}
	for _, p := range list {
		var id string
		err := q.QueryRowContext(ctx, `
			INSERT INTO posts (slug, author_id, title, excerpt, body_md, state, first_published_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (slug) DO UPDATE SET author_id = EXCLUDED.author_id, title = EXCLUDED.title, excerpt = EXCLUDED.excerpt,
			   body_md = EXCLUDED.body_md, state = EXCLUDED.state, first_published_at = EXCLUDED.first_published_at, updated_at = now()
			RETURNING id`, p.slug, ids[p.author], p.title, p.excerpt, p.body, p.state, p.published).Scan(&id)
		if err != nil {
			return fmt.Errorf("post %s: %w", p.slug, err)
		}
		if _, err := q.ExecContext(ctx, `DELETE FROM post_tags WHERE post_id = $1`, id); err != nil {
			return err
		}
		for _, t := range p.tags {
			if _, err := q.ExecContext(ctx, `INSERT INTO post_tags (post_id, tag) VALUES ($1, $2)`, id, strings.ReplaceAll(t, " ", "-")); err != nil {
				return err
			}
		}
	}
	return nil
}

func seedTalks(ctx context.Context, q Execer, now time.Time, ids map[string]string) error {
	list := []struct {
		slug, title, speaker, summary string
		at                            time.Time
		video                         *string
		featured                      bool
	}{
		{"what-compilers-teach-about-performance", "What your compiler course should have taught you about performance", "mkarim",
			"Why the fastest loop is often the one the compiler can **see through**.", at(now, -60, 15, 0), ptr("https://www.youtube.com/"), true},
		{"ml-without-the-hype", "Machine learning without the hype", "shirin",
			"What a linear model can still do in 2026.", at(now, -120, 15, 0), nil, false},
		{"breaking-your-own-web-app", "Breaking your own web app", "sabbir",
			"A live tour of the five bugs every student project has.", at(now, -200, 15, 0), nil, false},
	}
	for _, t := range list {
		if _, err := q.ExecContext(ctx, `
			INSERT INTO talks (slug, title, speaker_id, given_at, summary_md, video_url, featured)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, speaker_id = EXCLUDED.speaker_id, given_at = EXCLUDED.given_at,
			   summary_md = EXCLUDED.summary_md, video_url = EXCLUDED.video_url, featured = EXCLUDED.featured, updated_at = now()`,
			t.slug, t.title, ids[t.speaker], t.at, t.summary, t.video, t.featured); err != nil {
			return fmt.Errorf("talk %s: %w", t.slug, err)
		}
	}
	return nil
}

func seedSponsors(ctx context.Context, q Execer) error {
	logo := func(name string) string {
		return "https://placehold.co/240x80/png?text=" + strings.ReplaceAll(name, " ", "+")
	}
	list := []struct {
		name, tier, url string
		since           int
		until           *int
	}{
		{"Surma Cloud", "platinum", "https://example.com/surma-cloud", 2024, nil},
		{"Sylhet Software Ltd", "gold", "https://example.com/sylhet-software", 2025, nil},
		{"Tea Garden Labs", "silver", "https://example.com/tea-garden-labs", 2023, nil},
		{"Old Print House", "partner", "https://example.com/old-print-house", 2021, ptr(2023)},
	}
	for _, s := range list {
		res, err := q.ExecContext(ctx, `UPDATE sponsors SET tier = $2, logo_url = $3, url = $4, since = $5, until = $6, updated_at = now()
			WHERE name = $1`, s.name, s.tier, logo(s.name), s.url, s.since, s.until)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			continue
		}
		if _, err := q.ExecContext(ctx, `INSERT INTO sponsors (name, tier, logo_url, url, since, until) VALUES ($1, $2, $3, $4, $5, $6)`,
			s.name, s.tier, logo(s.name), s.url, s.since, s.until); err != nil {
			return fmt.Errorf("sponsor %s: %w", s.name, err)
		}
	}
	return nil
}
