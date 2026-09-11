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
	valid := []AbsenceType{AbsenceVacation, AbsenceHoliday, AbsenceBusinessTrip, AbsenceSick, AbsenceOther}
	for _, v := range valid {
		if !v.Valid() {
			t.Errorf("%q should be valid", v)
		}
	}
	for _, v := range []AbsenceType{"", "sabbatical", "Urlaub"} {
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

func TestLocalDateDaysUntil(t *testing.T) {
	cases := []struct {
		name     string
		from, to string
		want     int
	}{
		{"same day", "2026-07-07", "2026-07-07", 0},
		{"next day", "2026-07-07", "2026-07-08", 1},
		{"backwards", "2026-07-08", "2026-07-07", -1},
		{"across month", "2026-07-07", "2026-08-03", 27},
		{"across year", "2026-12-30", "2027-01-02", 3},
		// March 2026 crosses the European DST switch (29 March); anchoring at UTC
		// midnight keeps the count at whole days.
		{"across dst switch", "2026-03-28", "2026-03-30", 2},
		{"leap day", "2028-02-28", "2028-03-01", 2},
		{"non-leap february", "2026-02-28", "2026-03-01", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := mustDate(t, c.from).DaysUntil(mustDate(t, c.to))
			if got != c.want {
				t.Errorf("%s.DaysUntil(%s) = %d, want %d", c.from, c.to, got, c.want)
			}
		})
	}
}

func TestReturnDate(t *testing.T) {
	// Vacation Mon–Fri 2026-06-15..19, so work resumes Monday 2026-06-22.
	vacation := &Absence{Type: AbsenceVacation, Start: mustDate(t, "2026-06-15"), End: mustDate(t, "2026-06-19")}
	trip := &Absence{Type: AbsenceBusinessTrip, Start: mustDate(t, "2026-06-15"), End: mustDate(t, "2026-06-19")}
	sick := &Absence{Type: AbsenceSick, Start: mustDate(t, "2026-06-17"), End: mustDate(t, "2026-06-17")}

	cases := []struct {
		name     string
		absences []*Absence
		today    string
		want     string // "" = zero date, i.e. nothing to announce
	}{
		{"available", []*Absence{vacation}, "2026-06-10", ""},
		{"no absences at all", nil, "2026-06-10", ""},
		{"business trip stays reachable", []*Absence{trip}, "2026-06-17", ""},
		{"mid-vacation", []*Absence{vacation}, "2026-06-17", "2026-06-22"},
		{"first day of vacation", []*Absence{vacation}, "2026-06-15", "2026-06-22"},
		{"last day of vacation", []*Absence{vacation}, "2026-06-19", "2026-06-22"},
		{"single sick day", []*Absence{sick}, "2026-06-17", "2026-06-18"},
		// A blocking absence wins over a concurrent trip, and the trip days it
		// covers still count as workdays afterwards.
		{"sick during a trip", []*Absence{trip, sick}, "2026-06-17", "2026-06-18"},
		// Absence ending on a Friday pushes the return past the weekend.
		{"ends friday", []*Absence{{Type: AbsenceVacation, Start: mustDate(t, "2026-06-08"), End: mustDate(t, "2026-06-12")}}, "2026-06-12", "2026-06-15"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ReturnDate(c.absences, mustDate(t, c.today))
			if c.want == "" {
				if !got.IsZero() {
					t.Errorf("ReturnDate = %s, want zero date", got.String())
				}
				return
			}
			if got.String() != c.want {
				t.Errorf("ReturnDate = %s, want %s", got.String(), c.want)
			}
		})
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
		{"weekday no absence", "2026-06-10", nil, "2026-06-11"},                    // Wed -> Thu
		{"friday skips weekend", "2026-06-12", nil, "2026-06-15"},                  // Fri -> Mon
		{"vacation whole week", "2026-06-12", []Absence{vacation}, "2026-06-22"},   // Fri -> Mon after vacation week
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

func TestAbsencesForView(t *testing.T) {
	today := mustDate(t, "2026-06-12")
	mk := func(start, end string) *Absence {
		return &Absence{Start: mustDate(t, start), End: mustDate(t, end)}
	}
	lastYear := mk("2025-12-20", "2025-12-31")      // ended last year -> dropped
	earlyThisYear := mk("2026-01-05", "2026-01-09") // past, this year -> kept
	pastThisYear := mk("2026-05-01", "2026-05-05")  // past, this year -> kept
	current := mk("2026-06-10", "2026-06-15")       // current -> kept
	upcoming := mk("2026-07-01", "2026-07-10")      // upcoming -> kept
	nextYear := mk("2027-01-02", "2027-01-09")      // upcoming next year -> kept
	spanning := mk("2025-12-28", "2026-01-02")      // ends this year -> kept

	in := []*Absence{lastYear, earlyThisYear, pastThisYear, current, upcoming, nextYear, spanning}
	got := AbsencesForView(in, today)
	want := []*Absence{earlyThisYear, pastThisYear, current, upcoming, nextYear, spanning}

	if len(got) != len(want) {
		t.Fatalf("got %d absences, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestCurrentAvailability(t *testing.T) {
	today := mustDate(t, "2026-06-12")
	mk := func(typ AbsenceType, start, end string) *Absence {
		return &Absence{Type: typ, Start: mustDate(t, start), End: mustDate(t, end)}
	}
	cases := []struct {
		name string
		in   []*Absence
		want AvailabilityStatus
	}{
		{"no absences", nil, StatusAvailable},
		{"absence not covering today", []*Absence{mk(AbsenceVacation, "2026-07-01", "2026-07-05")}, StatusAvailable},
		{"vacation covers today", []*Absence{mk(AbsenceVacation, "2026-06-10", "2026-06-15")}, StatusAway},
		{"sick covers today", []*Absence{mk(AbsenceSick, "2026-06-12", "2026-06-12")}, StatusAway},
		{"business trip covers today", []*Absence{mk(AbsenceBusinessTrip, "2026-06-11", "2026-06-13")}, StatusBusinessTrip},
		{"blocking wins over trip", []*Absence{
			mk(AbsenceBusinessTrip, "2026-06-11", "2026-06-13"),
			mk(AbsenceVacation, "2026-06-12", "2026-06-12"),
		}, StatusAway},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CurrentAvailability(c.in, today); got != c.want {
				t.Errorf("CurrentAvailability = %v, want %v", got, c.want)
			}
		})
	}
}

// TestAvailabilityOn pins the situation matrix behind GET /api/availability:
// Working and OnBusinessTrip are independent axes, a blocking absence wins the
// status, and the return date is only announced while work is blocked.
func TestAvailabilityOn(t *testing.T) {
	day := mustDate(t, "2026-06-12") // a Friday
	mk := func(typ AbsenceType, start, end string) *Absence {
		return &Absence{Type: typ, Start: mustDate(t, start), End: mustDate(t, end)}
	}
	cases := []struct {
		name           string
		in             []*Absence
		status         AvailabilityStatus
		working        bool
		onBusinessTrip bool
		returnDate     string // "" = zero
		covering       int
	}{
		{"normal day", nil, StatusAvailable, true, false, "", 0},
		{"absence elsewhere", []*Absence{mk(AbsenceVacation, "2026-07-01", "2026-07-05")}, StatusAvailable, true, false, "", 0},
		{"business trip", []*Absence{mk(AbsenceBusinessTrip, "2026-06-11", "2026-06-13")}, StatusBusinessTrip, true, true, "", 1},
		{"vacation", []*Absence{mk(AbsenceVacation, "2026-06-10", "2026-06-15")}, StatusAway, false, false, "2026-06-16", 1},
		{"sick single day", []*Absence{mk(AbsenceSick, "2026-06-12", "2026-06-12")}, StatusAway, false, false, "2026-06-15", 1},
		{"vacation and trip same day", []*Absence{
			mk(AbsenceBusinessTrip, "2026-06-11", "2026-06-13"),
			mk(AbsenceVacation, "2026-06-12", "2026-06-12"),
		}, StatusAway, false, true, "2026-06-15", 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := AvailabilityOn(c.in, day)
			if got.Date != day {
				t.Errorf("Date = %s, want %s", got.Date.String(), day.String())
			}
			if got.Status != c.status {
				t.Errorf("Status = %v, want %v", got.Status, c.status)
			}
			if got.Working != c.working {
				t.Errorf("Working = %v, want %v", got.Working, c.working)
			}
			if got.OnBusinessTrip != c.onBusinessTrip {
				t.Errorf("OnBusinessTrip = %v, want %v", got.OnBusinessTrip, c.onBusinessTrip)
			}
			if c.returnDate == "" {
				if !got.ReturnDate.IsZero() {
					t.Errorf("ReturnDate = %s, want zero", got.ReturnDate.String())
				}
			} else if got.ReturnDate.String() != c.returnDate {
				t.Errorf("ReturnDate = %s, want %s", got.ReturnDate.String(), c.returnDate)
			}
			if len(got.Absences) != c.covering {
				t.Errorf("len(Absences) = %d, want %d", len(got.Absences), c.covering)
			}
		})
	}
}

func TestAvailabilityStatusString(t *testing.T) {
	cases := map[AvailabilityStatus]string{
		StatusAvailable:    "available",
		StatusAway:         "away",
		StatusBusinessTrip: "business_trip",
	}
	for st, want := range cases {
		if got := st.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", st, got, want)
		}
	}
}

func TestLatestUpdate(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if _, ok := LatestUpdate(nil); ok {
			t.Errorf("expected ok=false for no absences")
		}
	})
	t.Run("picks max UpdatedAt", func(t *testing.T) {
		older := &Absence{UpdatedAt: time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)}
		newer := &Absence{UpdatedAt: time.Date(2026, 6, 3, 14, 30, 0, 0, time.UTC)}
		mid := &Absence{UpdatedAt: time.Date(2026, 6, 2, 8, 0, 0, 0, time.UTC)}
		got, ok := LatestUpdate([]*Absence{older, newer, mid})
		if !ok {
			t.Fatalf("expected ok=true")
		}
		if !got.Equal(newer.UpdatedAt) {
			t.Errorf("LatestUpdate = %v, want %v", got, newer.UpdatedAt)
		}
	})
}

func TestUpcomingAbsences(t *testing.T) {
	today := mustDate(t, "2026-06-12")
	mk := func(start, end string) *Absence {
		return &Absence{Start: mustDate(t, start), End: mustDate(t, end)}
	}
	past := mk("2026-05-01", "2026-05-05")           // ended before today -> dropped
	endedYesterday := mk("2026-06-01", "2026-06-11") // ended before today -> dropped
	endsToday := mk("2026-06-05", "2026-06-12")      // End == today -> kept (boundary)
	current := mk("2026-06-10", "2026-06-20")        // running -> kept
	soon := mk("2026-06-15", "2026-06-18")           // upcoming -> kept
	later := mk("2026-08-01", "2026-08-10")          // upcoming -> kept

	// Deliberately unsorted input to exercise the Start-ascending sort.
	in := []*Absence{later, current, past, soon, endsToday, endedYesterday}
	got := UpcomingAbsences(in, today)
	want := []*Absence{endsToday, current, soon, later}

	if len(got) != len(want) {
		t.Fatalf("got %d absences, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestAbsencesInRange(t *testing.T) {
	mk := func(start, end string) *Absence {
		return &Absence{Start: mustDate(t, start), End: mustDate(t, end)}
	}
	// Window = calendar week Mon 2026-06-15 .. Sun 2026-06-21.
	from, to := mustDate(t, "2026-06-15"), mustDate(t, "2026-06-21")

	before := mk("2026-06-08", "2026-06-12")       // ends before window -> out
	endsInside := mk("2026-06-12", "2026-06-16")   // running into window -> in
	inside := mk("2026-06-17", "2026-06-18")       // fully inside -> in
	running := mk("2026-06-10", "2026-06-30")      // spans the whole window -> in
	startsInside := mk("2026-06-21", "2026-06-25") // touches last day -> in
	after := mk("2026-06-22", "2026-06-24")        // starts after window -> out

	in := []*Absence{before, endsInside, inside, running, startsInside, after}
	got := AbsencesInRange(in, from, to)
	want := []*Absence{endsInside, inside, running, startsInside}
	if len(got) != len(want) {
		t.Fatalf("got %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %+v, want %+v", i, got[i], want[i])
		}
	}

	// Open lower bound: everything ending on/after `to`'s start side; here no from.
	gotOpenFrom := AbsencesInRange(in, LocalDate{}, to)
	if len(gotOpenFrom) != 5 { // all except `after`
		t.Errorf("open-from: got %d, want 5", len(gotOpenFrom))
	}
	// Open upper bound: everything not ending before `from`.
	gotOpenTo := AbsencesInRange(in, from, LocalDate{})
	if len(gotOpenTo) != 5 { // all except `before`
		t.Errorf("open-to: got %d, want 5", len(gotOpenTo))
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

func TestShareWindow(t *testing.T) {
	tests := []struct {
		name              string
		today, start, end string
	}{
		// Every day of a week must yield the same window — that is the point of
		// anchoring on Monday instead of on today.
		{"monday", "2026-08-03", "2026-08-03", "2026-10-25"},
		{"saturday", "2026-08-08", "2026-08-03", "2026-10-25"},
		{"sunday", "2026-08-09", "2026-08-03", "2026-10-25"},
		{"next monday moves it one week on", "2026-08-10", "2026-08-10", "2026-11-01"},
		{"across the year boundary", "2026-12-31", "2026-12-28", "2027-03-21"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start, end := ShareWindow(mustDate(t, tc.today))
			if start.String() != tc.start || end.String() != tc.end {
				t.Errorf("ShareWindow(%s) = %s..%s, want %s..%s", tc.today, start, end, tc.start, tc.end)
			}
			// The window is inclusive on both ends and exactly 12 weeks long.
			if days := start.DaysUntil(end) + 1; days != ShareWindowWeeks*7 {
				t.Errorf("window spans %d days, want %d", days, ShareWindowWeeks*7)
			}
		})
	}
}
