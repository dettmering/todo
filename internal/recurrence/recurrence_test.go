package recurrence

import (
	"testing"
	"time"

	"github.com/till/todo/internal/domain"
)

func d(y int, m time.Month, day int) domain.LocalDate {
	return domain.LocalDate{Year: y, Month: m, Day: day}
}

func TestNext(t *testing.T) {
	cases := []struct {
		name   string
		rule   string
		after  domain.LocalDate
		notBef domain.LocalDate
		want   domain.LocalDate
	}{
		{"daily", "FREQ=DAILY", d(2026, time.May, 20), d(2026, time.May, 20), d(2026, time.May, 21)},
		{"every 3 days", "FREQ=DAILY;INTERVAL=3", d(2026, time.May, 20), d(2026, time.May, 20), d(2026, time.May, 23)},
		{"weekly mon from sun", "FREQ=WEEKLY;BYDAY=MO", d(2026, time.May, 17), d(2026, time.May, 17), d(2026, time.May, 18)},
		{"weekly mon from mon", "FREQ=WEEKLY;BYDAY=MO", d(2026, time.May, 18), d(2026, time.May, 18), d(2026, time.May, 25)},
		{"weekday mo from sat", "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR", d(2026, time.May, 23), d(2026, time.May, 23), d(2026, time.May, 25)},
		{"every 2 weeks sat", "FREQ=WEEKLY;INTERVAL=2;BYDAY=SA", d(2026, time.May, 23), d(2026, time.May, 23), d(2026, time.June, 6)},
		{"monthly 1st", "FREQ=MONTHLY;BYMONTHDAY=1", d(2026, time.May, 15), d(2026, time.May, 15), d(2026, time.June, 1)},
		{"monthly 31st clamp", "FREQ=MONTHLY;BYMONTHDAY=31", d(2026, time.January, 31), d(2026, time.January, 31), d(2026, time.February, 28)},
		{"yearly", "FREQ=YEARLY;BYMONTH=1;BYMONTHDAY=1", d(2026, time.January, 1), d(2026, time.January, 1), d(2027, time.January, 1)},
		{"roll forward when late", "FREQ=WEEKLY;BYDAY=MO", d(2026, time.January, 5), d(2026, time.May, 20), d(2026, time.May, 25)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := Parse(tc.rule)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			got, err := r.Next(tc.after, tc.notBef)
			if err != nil {
				t.Fatalf("next: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestParseRejects(t *testing.T) {
	bad := []string{
		"FREQ=HOURLY",
		"INTERVAL=2",
		"FREQ=DAILY;INTERVAL=0",
		"FREQ=WEEKLY;BYDAY=XX",
		"FREQ=DAILY;BYDAY=MO",
		"FREQ=MONTHLY;BYMONTHDAY=32",
		"foo",
	}
	for _, s := range bad {
		if _, err := Parse(s); err == nil {
			t.Errorf("%q: expected error", s)
		}
	}
}

func TestHumanLabel(t *testing.T) {
	cases := map[string]string{
		"FREQ=DAILY":                         "Täglich",
		"FREQ=DAILY;INTERVAL=3":              "Alle 3 Tage",
		"FREQ=WEEKLY;BYDAY=MO":               "Jeden Montag",
		"FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR":   "Jeden Werktag",
		"FREQ=WEEKLY;INTERVAL=2":             "Alle 2 Wochen",
		"FREQ=MONTHLY;BYMONTHDAY=1":          "Am 1. jedes Monats",
		"FREQ=YEARLY;BYMONTH=1;BYMONTHDAY=1": "Jährlich am 1. Januar",
	}
	for rule, want := range cases {
		if got := HumanLabel(rule, domain.RecurFixed); got != want {
			t.Errorf("%s: got %q want %q", rule, got, want)
		}
	}
	// An empty mode is the legacy encoding of "fixed" and must read the same.
	if got := HumanLabel("FREQ=DAILY", ""); got != "Täglich" {
		t.Errorf("empty mode: got %q want %q", got, "Täglich")
	}
	// RecurAfterDone spells out that the interval restarts on completion.
	afterDone := map[string]string{
		"FREQ=WEEKLY;INTERVAL=4": "Alle 4 Wochen ab Erledigung",
		"FREQ=MONTHLY":           "Monatlich ab Erledigung",
	}
	for rule, want := range afterDone {
		if got := HumanLabel(rule, domain.RecurAfterDone); got != want {
			t.Errorf("%s: got %q want %q", rule, got, want)
		}
	}
	// No rule means no label, whatever the mode says.
	if got := HumanLabel("", domain.RecurAfterDone); got != "" {
		t.Errorf("empty rule: got %q want %q", got, "")
	}
}

func TestHasCalendarAnchor(t *testing.T) {
	cases := map[string]bool{
		"FREQ=DAILY":                false,
		"FREQ=WEEKLY;INTERVAL=4":    false,
		"FREQ=MONTHLY":              false,
		"FREQ=YEARLY":               false,
		"FREQ=WEEKLY;BYDAY=MO":      true,
		"FREQ=MONTHLY;BYMONTHDAY=1": true,
		"FREQ=YEARLY;BYMONTH=1":     true,
	}
	for rule, want := range cases {
		r, err := Parse(rule)
		if err != nil {
			t.Fatalf("Parse(%q): %v", rule, err)
		}
		if got := r.HasCalendarAnchor(); got != want {
			t.Errorf("%s: got %v want %v", rule, got, want)
		}
	}
}
