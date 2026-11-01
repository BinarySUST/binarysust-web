package people

import (
	"testing"
	"time"
)

func TestStatusOf(t *testing.T) {
	today := time.Date(2026, 11, 2, 15, 0, 0, 0, time.UTC)
	day := func(s string) *time.Time {
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			t.Fatal(err)
		}
		return &d
	}
	exec, adv := Executive, AdvisorCat
	cases := []struct {
		name string
		f    Facts
		want Status
	}{
		{"executive seat beats graduation", Facts{Affiliation: Student, GraduatedOn: day("2020-01-01"), CurrentSeat: &exec}, StatusCommittee},
		{"advisor seat", Facts{Affiliation: Faculty, CurrentSeat: &adv}, StatusAdvisor},
		{"former faculty advisor", Facts{Affiliation: Faculty, EverAdvisor: true}, StatusAdvisor},
		{"faculty speaker", Facts{Affiliation: Faculty}, StatusGuest},
		{"guest stays guest", Facts{Affiliation: Visitor, MemberSince: day("2020-01-01")}, StatusGuest},
		{"graduated today is alumni", Facts{Affiliation: Student, MemberSince: day("2020-01-01"), GraduatedOn: day("2026-11-02")}, StatusAlumni},
		{"graduation in the future", Facts{Affiliation: Student, MemberSince: day("2022-01-01"), GraduatedOn: day("2027-01-01")}, StatusMember},
		{"member", Facts{Affiliation: Student, MemberSince: day("2024-03-01")}, StatusMember},
		{"not yet a member", Facts{Affiliation: Student, MemberSince: day("2026-12-01")}, StatusGuest},
		{"student without membership", Facts{Affiliation: Student}, StatusGuest},
	}
	for _, c := range cases {
		if got := StatusOf(c.f, today); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}
