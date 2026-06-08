package domain

import (
	"testing"
	"time"
)

func mustDate(t *testing.T, s string) LocalDate {
	t.Helper()
	d, err := ParseDate(s)
	if err != nil {
		t.Fatalf("ParseDate(%q): %v", s, err)
	}
	return d
}

func TestAbsenceTypeValid(t *testing.T) {
	valid := []AbsenceType{AbsenceVacation, AbsenceBusinessTrip, AbsenceSick, AbsenceOther}
	for _, v := range valid {
		if !v.Valid() {
			t.Errorf("%q should be valid", v)
		}
	}
	for _, v := range []AbsenceType{"", "holiday", "Urlaub"} {
		if v.Valid() {
			t.Errorf("%q should be invalid", v)
		}
	}
}

func TestAbsenceCovers(t *testing.T) {
	a := Absence{Start: mustDate(t, "2026-06-15"), End: mustDate(t, "2026-06-19")}
	cases := []struct {
		date string
		want bool
	}{
		{"2026-06-14", false}, // day before
		{"2026-06-15", true},  // start (inclusive)
		{"2026-06-17", true},  // middle
		{"2026-06-19", true},  // end (inclusive)
		{"2026-06-20", false}, // day after
	}
	for _, c := range cases {
		if got := a.Covers(mustDate(t, c.date)); got != c.want {
			t.Errorf("Covers(%s) = %v, want %v", c.date, got, c.want)
		}
	}
}

func TestAbsenceBlocksWork(t *testing.T) {
	if (Absence{Type: AbsenceBusinessTrip}).BlocksWork() {
		t.Error("business trip should not block work")
	}
	for _, typ := range []AbsenceType{AbsenceVacation, AbsenceSick, AbsenceOther} {
		if !(Absence{Type: typ}).BlocksWork() {
			t.Errorf("%q should block work", typ)
		}
	}
}

func TestLocalDateAddDaysAndWeekend(t *testing.T) {
	// 2026-06-12 is a Friday.
	fri := mustDate(t, "2026-06-12")
	if fri.IsWeekend() {
		t.Error("Friday should not be weekend")
	}
	sat := fri.AddDays(1)
	if sat.String() != "2026-06-13" || !sat.IsWeekend() {
		t.Errorf("AddDays(1) = %s, weekend=%v; want 2026-06-13 weekend", sat.String(), sat.IsWeekend())
	}
	// month boundary
	if got := mustDate(t, "2026-06-30").AddDays(1).String(); got != "2026-07-01" {
		t.Errorf("2026-06-30 + 1 = %s, want 2026-07-01", got)
	}
}

func TestNextWorkday(t *testing.T) {
	vacation := Absence{Type: AbsenceVacation, Start: mustDate(t, "2026-06-15"), End: mustDate(t, "2026-06-19")}
	trip := Absence{Type: AbsenceBusinessTrip, Start: mustDate(t, "2026-06-15"), End: mustDate(t, "2026-06-19")}

	cases := []struct {
		name     string
		from     string
		absences []Absence
		want     string
	}{
		{"weekday no absence", "2026-06-10", nil, "2026-06-11"},     // Wed -> Thu
		{"friday skips weekend", "2026-06-12", nil, "2026-06-15"},   // Fri -> Mon
		{"vacation whole week", "2026-06-12", []Absence{vacation}, "2026-06-22"}, // Fri -> Mon after vacation week
		{"business trip not skipped", "2026-06-12", []Absence{trip}, "2026-06-15"}, // Fri -> Mon (trip day allowed)
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := NextWorkday(mustDate(t, c.from), c.absences)
			if got.String() != c.want {
				t.Errorf("NextWorkday(%s) = %s, want %s", c.from, got.String(), c.want)
			}
		})
	}
}

// guard against the helper drifting from real calendar weekdays
func TestKnownWeekdays(t *testing.T) {
	if wd := time.Date(2026, time.June, 12, 0, 0, 0, 0, time.UTC).Weekday(); wd != time.Friday {
		t.Fatalf("2026-06-12 is %v, expected Friday", wd)
	}
	if wd := time.Date(2026, time.June, 15, 0, 0, 0, 0, time.UTC).Weekday(); wd != time.Monday {
		t.Fatalf("2026-06-15 is %v, expected Monday", wd)
	}
}
