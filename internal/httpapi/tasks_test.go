package httpapi

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/till/todo/internal/domain"
)

func TestParseISOWeek(t *testing.T) {
	cases := []struct {
		in      string
		wantMon domain.LocalDate
		wantSun domain.LocalDate
	}{
		{"2026-W21", date(2026, 5, 18), date(2026, 5, 24)},
		// W1 of 2026 starts in the previous calendar year.
		{"2026-W01", date(2025, 12, 29), date(2026, 1, 4)},
		// 2026 is a 53-week ISO year (starts on Thursday).
		{"2026-W53", date(2026, 12, 28), date(2027, 1, 3)},
		// 2020 is a 53-week ISO year (leap year starting Wednesday).
		{"2020-W53", date(2020, 12, 28), date(2021, 1, 3)},
		// W1 of 2021 starts after the 2020-W53 overflow.
		{"2021-W01", date(2021, 1, 4), date(2021, 1, 10)},
		// 2024 starts on Monday, so W1 begins on Jan 1.
		{"2024-W01", date(2024, 1, 1), date(2024, 1, 7)},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			mon, sun, err := parseISOWeek(c.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if mon != c.wantMon {
				t.Errorf("monday = %v, want %v", mon, c.wantMon)
			}
			if sun != c.wantSun {
				t.Errorf("sunday = %v, want %v", sun, c.wantSun)
			}
		})
	}
}

func TestParseISOWeekInvalid(t *testing.T) {
	cases := []string{
		"",
		"2026",
		"2026-W",
		"2026-W00",       // week 0 doesn't exist
		"2026-W54",       // out of range
		"2025-W53",       // 2025 has only 52 ISO weeks
		"2026-21",        // missing "W"
		"2026-W21-extra", // trailing garbage
		"abcd-Wef",
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			if _, _, err := parseISOWeek(c); err == nil {
				t.Errorf("expected error for %q, got nil", c)
			}
		})
	}
}

func date(y, m, d int) domain.LocalDate {
	return domain.LocalDate{Year: y, Month: time.Month(m), Day: d}
}

// The sidebar reads counts[project] with no guard for a missing container, so
// the endpoint must answer with empty objects rather than JSON null.
func TestToCountsDTONeverNull(t *testing.T) {
	d := toCountsDTO(domain.OpenTaskCounts{VisibleToday: 3})
	if d.Projects == nil || d.Epics == nil {
		t.Fatalf("nil maps leak into the response: %+v", d)
	}
	body, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got, want := string(body), `{"today":3,"projects":{},"epics":{}}`; got != want {
		t.Errorf("body = %s, want %s", got, want)
	}
}
