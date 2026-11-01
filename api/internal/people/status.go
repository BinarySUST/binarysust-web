package people

import "time"

// Status is derived, never stored, so it cannot drift from the facts it summarizes.
type Status string

// Statuses, in the order the directory lists them.
const (
	StatusCommittee Status = "committee"
	StatusAdvisor   Status = "advisor"
	StatusMember    Status = "member"
	StatusAlumni    Status = "alumni"
	StatusGuest     Status = "guest"
)

// Enum lists the values (the OpenAPI document and the web's types are generated from it).
func (Status) Enum() []string { return []string{"committee", "advisor", "member", "alumni", "guest"} }

// Affiliation is what a person is to SUST.
type Affiliation string

// Affiliations.
const (
	Student Affiliation = "student"
	Faculty Affiliation = "faculty"
	Visitor Affiliation = "guest"
)

// Enum lists the values.
func (Affiliation) Enum() []string { return []string{"student", "faculty", "guest"} }

// Category groups committee positions.
type Category string

// Categories.
const (
	Executive  Category = "executive"
	AdvisorCat Category = "advisor"
)

// Enum lists the values.
func (Category) Enum() []string { return []string{"executive", "advisor"} }

// Group is how the public committee page groups positions: the advisory committee (patron,
// adviser), then leadership (President, Vice President, General Secretary, Director of
// Training, Project Co-ordinator), secretaries, their assistants and executive members.
type Group string

// Groups, in display order.
const (
	GroupPatron     Group = "patron"
	GroupAdviser    Group = "adviser"
	GroupLeadership Group = "leadership"
	GroupSecretary  Group = "secretary"
	GroupAssistant  Group = "assistant"
	GroupMember     Group = "member"
)

// Enum lists the values in display order.
func (Group) Enum() []string {
	return []string{"patron", "adviser", "leadership", "secretary", "assistant", "member"}
}

// Facts are the stored inputs to StatusOf.
type Facts struct {
	Affiliation Affiliation
	MemberSince *time.Time
	GraduatedOn *time.Time
	// CurrentSeat is the category of the person's seat in the current term, nil when unseated.
	// An executive seat wins when a person holds both kinds.
	CurrentSeat *Category
	// EverAdvisor: held an advisor seat in any term.
	EverAdvisor bool
}

// StatusOf is the single definition of a person's public status:
//
//   - a seat in the current term: advisor for an advisor position, else committee;
//   - faculty: advisor if they ever held an advisor seat, else guest;
//   - guests: guest;
//   - students: alumni once graduated_on <= today, member once member_since <= today, else guest.
func StatusOf(f Facts, today time.Time) Status {
	if f.CurrentSeat != nil {
		if *f.CurrentSeat == AdvisorCat {
			return StatusAdvisor
		}
		return StatusCommittee
	}
	switch f.Affiliation {
	case Faculty:
		if f.EverAdvisor {
			return StatusAdvisor
		}
		return StatusGuest
	case Visitor:
		return StatusGuest
	}
	day := dateOnly(today)
	if f.GraduatedOn != nil && !dateOnly(*f.GraduatedOn).After(day) {
		return StatusAlumni
	}
	if f.MemberSince != nil && !dateOnly(*f.MemberSince).After(day) {
		return StatusMember
	}
	return StatusGuest
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
