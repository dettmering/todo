// Package recurrence parses a pragmatic subset of RFC 5545 RRULE strings and
// computes the next occurrence after a given date. Supported parts:
//
//	FREQ=DAILY|WEEKLY|MONTHLY|YEARLY  (required)
//	INTERVAL=<positive int>           (default 1)
//	BYDAY=MO,TU,WE,TH,FR,SA,SU        (WEEKLY only)
//	BYMONTHDAY=1..31                  (MONTHLY/YEARLY)
//	BYMONTH=1..12                     (YEARLY)
//
// Anything beyond this is rejected. Dates are timezone-naive (domain.LocalDate).
package recurrence

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/till/todo/internal/domain"
)

type Freq string

const (
	Daily   Freq = "DAILY"
	Weekly  Freq = "WEEKLY"
	Monthly Freq = "MONTHLY"
	Yearly  Freq = "YEARLY"
)

type Rule struct {
	Freq       Freq
	Interval   int            // ≥ 1
	ByDay      []time.Weekday // WEEKLY only
	ByMonthDay int            // 0 = unset
	ByMonth    time.Month     // 0 = unset
}

var weekdayCodes = map[string]time.Weekday{
	"SU": time.Sunday,
	"MO": time.Monday,
	"TU": time.Tuesday,
	"WE": time.Wednesday,
	"TH": time.Thursday,
	"FR": time.Friday,
	"SA": time.Saturday,
}

var weekdayShort = map[time.Weekday]string{
	time.Sunday:    "SU",
	time.Monday:    "MO",
	time.Tuesday:   "TU",
	time.Wednesday: "WE",
	time.Thursday:  "TH",
	time.Friday:    "FR",
	time.Saturday:  "SA",
}

// Parse returns the rule, or an error if the input is malformed or uses an
// unsupported clause. Empty input returns (zero, nil) so callers can pass
// optional rules through.
func Parse(s string) (Rule, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Rule{}, nil
	}
	r := Rule{Interval: 1}
	for _, part := range strings.Split(s, ";") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return Rule{}, fmt.Errorf("invalid clause %q", part)
		}
		key := strings.ToUpper(strings.TrimSpace(kv[0]))
		val := strings.ToUpper(strings.TrimSpace(kv[1]))
		switch key {
		case "FREQ":
			switch Freq(val) {
			case Daily, Weekly, Monthly, Yearly:
				r.Freq = Freq(val)
			default:
				return Rule{}, fmt.Errorf("unsupported FREQ %q", val)
			}
		case "INTERVAL":
			n, err := strconv.Atoi(val)
			if err != nil || n < 1 {
				return Rule{}, fmt.Errorf("invalid INTERVAL %q", val)
			}
			r.Interval = n
		case "BYDAY":
			for _, code := range strings.Split(val, ",") {
				w, ok := weekdayCodes[strings.TrimSpace(code)]
				if !ok {
					return Rule{}, fmt.Errorf("invalid BYDAY %q", code)
				}
				r.ByDay = append(r.ByDay, w)
			}
		case "BYMONTHDAY":
			n, err := strconv.Atoi(val)
			if err != nil || n < 1 || n > 31 {
				return Rule{}, fmt.Errorf("invalid BYMONTHDAY %q", val)
			}
			r.ByMonthDay = n
		case "BYMONTH":
			n, err := strconv.Atoi(val)
			if err != nil || n < 1 || n > 12 {
				return Rule{}, fmt.Errorf("invalid BYMONTH %q", val)
			}
			r.ByMonth = time.Month(n)
		default:
			return Rule{}, fmt.Errorf("unsupported clause %q", key)
		}
	}
	if r.Freq == "" {
		return Rule{}, fmt.Errorf("FREQ is required")
	}
	if len(r.ByDay) > 0 && r.Freq != Weekly {
		return Rule{}, fmt.Errorf("BYDAY only valid for WEEKLY")
	}
	if r.ByMonth != 0 && r.Freq != Yearly {
		return Rule{}, fmt.Errorf("BYMONTH only valid for YEARLY")
	}
	// Dedupe BYDAY.
	if len(r.ByDay) > 0 {
		seen := map[time.Weekday]bool{}
		out := r.ByDay[:0]
		for _, w := range r.ByDay {
			if !seen[w] {
				seen[w] = true
				out = append(out, w)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
		r.ByDay = out
	}
	return r, nil
}

// Canonical re-serializes the rule in a stable form so equal rules compare equal.
func (r Rule) Canonical() string {
	if r.Freq == "" {
		return ""
	}
	parts := []string{"FREQ=" + string(r.Freq)}
	if r.Interval > 1 {
		parts = append(parts, "INTERVAL="+strconv.Itoa(r.Interval))
	}
	if len(r.ByDay) > 0 {
		codes := make([]string, len(r.ByDay))
		for i, w := range r.ByDay {
			codes[i] = weekdayShort[w]
		}
		parts = append(parts, "BYDAY="+strings.Join(codes, ","))
	}
	if r.ByMonth != 0 {
		parts = append(parts, "BYMONTH="+strconv.Itoa(int(r.ByMonth)))
	}
	if r.ByMonthDay != 0 {
		parts = append(parts, "BYMONTHDAY="+strconv.Itoa(r.ByMonthDay))
	}
	return strings.Join(parts, ";")
}

// HasCalendarAnchor reports whether the rule pins occurrences to specific
// calendar positions (weekday, day of month, month). Such rules are meaningless
// in RecurAfterDone mode — "every Monday, starting when you tick it off" has no
// coherent reading — so the service rejects the combination.
func (r Rule) HasCalendarAnchor() bool {
	return len(r.ByDay) > 0 || r.ByMonthDay != 0 || r.ByMonth != 0
}

// Next returns the smallest date strictly after `after` that matches the rule.
// It also rolls forward so the returned date is on or after `notBefore` —
// completing a recurring task long after its visible_from shouldn't spawn an
// instance with an already-past date.
func (r Rule) Next(after, notBefore domain.LocalDate) (domain.LocalDate, error) {
	if r.Freq == "" {
		return domain.LocalDate{}, fmt.Errorf("empty rule")
	}
	if after.IsZero() {
		after = notBefore
	}
	base := toTime(after)
	out, err := r.nextAfter(base)
	if err != nil {
		return domain.LocalDate{}, err
	}
	if !notBefore.IsZero() {
		nb := toTime(notBefore)
		for out.Before(nb) {
			out, err = r.nextAfter(out)
			if err != nil {
				return domain.LocalDate{}, err
			}
		}
	}
	return domain.DateOf(out), nil
}

func (r Rule) nextAfter(base time.Time) (time.Time, error) {
	switch r.Freq {
	case Daily:
		return base.AddDate(0, 0, r.Interval), nil
	case Weekly:
		if len(r.ByDay) == 0 {
			return base.AddDate(0, 0, 7*r.Interval), nil
		}
		// Anchor on the Monday of base's ISO week so INTERVAL counts whole weeks.
		anchor := mondayOf(base)
		// Scan up to interval*7 + 7 days ahead to find the next matching day.
		for i := 1; i <= r.Interval*7+7; i++ {
			c := base.AddDate(0, 0, i)
			weekOffset := int(c.Sub(anchor).Hours()/24) / 7
			if weekOffset%r.Interval != 0 {
				continue
			}
			if containsWeekday(r.ByDay, c.Weekday()) {
				return c, nil
			}
		}
		return time.Time{}, fmt.Errorf("weekly rule produced no match (rule likely impossible)")
	case Monthly:
		day := r.ByMonthDay
		if day == 0 {
			day = base.Day()
		}
		// Try base.month + interval, base.month + 2*interval, ... until day fits and date > base.
		for step := 1; step <= 24; step++ {
			y := base.Year()
			m := int(base.Month()) + step*r.Interval
			for m > 12 {
				m -= 12
				y++
			}
			t := clampToMonth(y, time.Month(m), day)
			if t.After(base) {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("monthly rule produced no match")
	case Yearly:
		m := r.ByMonth
		if m == 0 {
			m = base.Month()
		}
		day := r.ByMonthDay
		if day == 0 {
			day = base.Day()
		}
		for step := 0; step <= 8; step++ {
			y := base.Year() + step*r.Interval
			t := clampToMonth(y, m, day)
			if t.After(base) {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("yearly rule produced no match")
	}
	return time.Time{}, fmt.Errorf("unsupported FREQ %q", r.Freq)
}

func clampToMonth(year int, m time.Month, day int) time.Time {
	// Last day of month: go to the 1st of the next month and step back.
	first := time.Date(year, m, 1, 0, 0, 0, 0, time.UTC)
	last := first.AddDate(0, 1, -1).Day()
	if day > last {
		day = last
	}
	return time.Date(year, m, day, 0, 0, 0, 0, time.UTC)
}

func mondayOf(t time.Time) time.Time {
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7 // Sunday → 7 so Monday = 1
	}
	return time.Date(t.Year(), t.Month(), t.Day()-(wd-1), 0, 0, 0, 0, time.UTC)
}

func containsWeekday(ws []time.Weekday, w time.Weekday) bool {
	for _, x := range ws {
		if x == w {
			return true
		}
	}
	return false
}

func toTime(d domain.LocalDate) time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
}

// HumanLabel returns a short German label describing the rule, e.g. "Täglich",
// "Jeden Montag", "Alle 2 Wochen". Empty string for the zero rule. In
// RecurAfterDone mode the interval restarts on completion, which the label
// spells out ("Alle 4 Wochen ab Erledigung").
func HumanLabel(s string, mode domain.RecurrenceMode) string {
	base := baseLabel(s)
	if base == "" {
		return ""
	}
	if mode == domain.RecurAfterDone {
		return base + " ab Erledigung"
	}
	return base
}

func baseLabel(s string) string {
	r, err := Parse(s)
	if err != nil || r.Freq == "" {
		return ""
	}
	switch r.Freq {
	case Daily:
		if r.Interval == 1 {
			return "Täglich"
		}
		return fmt.Sprintf("Alle %d Tage", r.Interval)
	case Weekly:
		if len(r.ByDay) > 0 {
			if r.Interval == 1 && isWeekdaySet(r.ByDay) {
				return "Jeden Werktag"
			}
			names := make([]string, len(r.ByDay))
			for i, w := range r.ByDay {
				names[i] = weekdayDE[w]
			}
			joined := strings.Join(names, ", ")
			if r.Interval == 1 {
				return "Jeden " + joined
			}
			return fmt.Sprintf("Alle %d Wochen am %s", r.Interval, joined)
		}
		if r.Interval == 1 {
			return "Wöchentlich"
		}
		return fmt.Sprintf("Alle %d Wochen", r.Interval)
	case Monthly:
		if r.ByMonthDay > 0 {
			if r.Interval == 1 {
				return fmt.Sprintf("Am %d. jedes Monats", r.ByMonthDay)
			}
			return fmt.Sprintf("Am %d. alle %d Monate", r.ByMonthDay, r.Interval)
		}
		if r.Interval == 1 {
			return "Monatlich"
		}
		return fmt.Sprintf("Alle %d Monate", r.Interval)
	case Yearly:
		base := "Jährlich"
		if r.Interval > 1 {
			base = fmt.Sprintf("Alle %d Jahre", r.Interval)
		}
		if r.ByMonth != 0 && r.ByMonthDay != 0 {
			return fmt.Sprintf("%s am %d. %s", base, r.ByMonthDay, monthDE[r.ByMonth])
		}
		return base
	}
	return ""
}

var weekdayDE = map[time.Weekday]string{
	time.Monday:    "Montag",
	time.Tuesday:   "Dienstag",
	time.Wednesday: "Mittwoch",
	time.Thursday:  "Donnerstag",
	time.Friday:    "Freitag",
	time.Saturday:  "Samstag",
	time.Sunday:    "Sonntag",
}

var monthDE = map[time.Month]string{
	time.January: "Januar", time.February: "Februar", time.March: "März",
	time.April: "April", time.May: "Mai", time.June: "Juni",
	time.July: "Juli", time.August: "August", time.September: "September",
	time.October: "Oktober", time.November: "November", time.December: "Dezember",
}

func isWeekdaySet(ws []time.Weekday) bool {
	if len(ws) != 5 {
		return false
	}
	want := map[time.Weekday]bool{time.Monday: true, time.Tuesday: true, time.Wednesday: true, time.Thursday: true, time.Friday: true}
	for _, w := range ws {
		if !want[w] {
			return false
		}
	}
	return true
}
