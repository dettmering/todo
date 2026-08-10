package domain

import (
	"testing"
	"time"
)

// The interesting cases are all at the year boundary, where the ISO week and
// the calendar year disagree: a week belongs to whichever year holds its
// Thursday.
func TestLocalDateISOWeek(t *testing.T) {
	tests := []struct {
		name string
		date LocalDate
		want int
	}{
		{"mid-year", LocalDate{2026, time.August, 10}, 33},
		{"first thursday is week 1", LocalDate{2026, time.January, 1}, 1},
		{"monday before it belongs to week 1 too", LocalDate{2025, time.December, 29}, 1},
		{"sunday before that is still week 52", LocalDate{2025, time.December, 28}, 52},
		{"long year has a week 53", LocalDate{2026, time.December, 31}, 53},
		{"and it runs into January", LocalDate{2027, time.January, 3}, 53},
		{"new week 1 starts on the Monday after", LocalDate{2027, time.January, 4}, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.date.ISOWeek(); got != tc.want {
				t.Errorf("ISOWeek(%s) = %d, want %d", tc.date, got, tc.want)
			}
		})
	}
}
