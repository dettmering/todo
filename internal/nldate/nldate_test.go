package nldate

import (
	"testing"
	"time"

	"github.com/till/todo/internal/domain"
)

// 2026-05-21 = Thursday. Used as `today` in all examples so the test
// expectations are easy to reason about.
var today = domain.LocalDate{Year: 2026, Month: time.May, Day: 21}

func d(y int, m time.Month, day int) domain.LocalDate {
	return domain.LocalDate{Year: y, Month: m, Day: day}
}

func TestParseDate(t *testing.T) {
	cases := []struct {
		in   string
		want domain.LocalDate
	}{
		{"heute", d(2026, time.May, 21)},
		{"heu", d(2026, time.May, 21)},
		{"morgen", d(2026, time.May, 22)},
		{"mor", d(2026, time.May, 22)},
		{"übermorgen", d(2026, time.May, 23)},
		{"gestern", d(2026, time.May, 20)},
		{"freitag", d(2026, time.May, 22)},
		{"fr", d(2026, time.May, 22)},
		{"donnerstag", d(2026, time.May, 21)}, // today is Thursday
		{"montag", d(2026, time.May, 25)},
		{"nächsten Montag", d(2026, time.May, 25)},
		{"nächste woche", d(2026, time.May, 25)},
		{"nächsten monat", d(2026, time.June, 21)},
		{"nächstes jahr", d(2027, time.May, 21)},
		{"in 5 tagen", d(2026, time.May, 26)},
		{"+5", d(2026, time.May, 26)},
		{"+5 tage", d(2026, time.May, 26)},
		{"in 3 wochen", d(2026, time.June, 11)},
		{"monatsende", d(2026, time.May, 31)},
		{"ende des monats", d(2026, time.May, 31)},
		{"mitte januar", d(2027, time.January, 15)},
		{"anfang juni", d(2026, time.June, 1)},
		{"ende dezember", d(2026, time.December, 31)},
		{"27.1.", d(2027, time.January, 27)}, // past in 2026, roll forward
		{"27.5.", d(2026, time.May, 27)},
		{"27.5.2026", d(2026, time.May, 27)},
		{"27.05.2026", d(2026, time.May, 27)},
		{"2026-12-24", d(2026, time.December, 24)},
		{"KW 32", d(2026, time.August, 3)},
		{"kw32", d(2026, time.August, 3)},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseDate(tc.in, today)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestParseRejects(t *testing.T) {
	for _, in := range []string{"", "gibberish", "32.13.", "30.02.2026"} {
		if in == "" {
			// empty is allowed (returns zero result, no error)
			r, err := Parse(in, today)
			if err != nil || !r.Date.IsZero() {
				t.Errorf("empty: expected zero/nil, got %+v %v", r, err)
			}
			continue
		}
		if _, err := ParseDate(in, today); err == nil {
			t.Errorf("%q: expected error", in)
		}
	}
}

func TestMonatsendeClampForFebruary(t *testing.T) {
	feb := domain.LocalDate{Year: 2027, Month: time.February, Day: 5} // non-leap year
	got, err := ParseDate("monatsende", feb)
	if err != nil {
		t.Fatal(err)
	}
	if got != d(2027, time.February, 28) {
		t.Errorf("got %v want 2027-02-28", got)
	}
}

func TestNextMonthClamp(t *testing.T) {
	jan31 := domain.LocalDate{Year: 2026, Month: time.January, Day: 31}
	got, err := ParseDate("nächsten monat", jan31)
	if err != nil {
		t.Fatal(err)
	}
	if got != d(2026, time.February, 28) {
		t.Errorf("got %v want 2026-02-28", got)
	}
}
