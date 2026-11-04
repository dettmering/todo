// Package nldate parses German natural-language date phrases into a
// domain.LocalDate. Supported phrases (case-insensitive, leading/trailing
// whitespace ignored):
//
//	heute, heu, morgen, mor, übermorgen, gestern
//	montag..sonntag, plus 2-letter codes (mo, di, mi, do, fr, sa, so)
//	nächste woche               -> Monday of next ISO week
//	nächsten monat              -> same day next month (clamped)
//	nächstes jahr               -> same day next year
//	in N tage[n]|woche[n]|monate[n]|jahre[n]
//	+N, +N tage, +Nd            -> today + N days
//	anfang|mitte|ende <month>   -> 1st / 15th / last day of that month
//	monatsende, ende des monats -> last day of current month
//	monatsanfang                -> first day of current month
//	27.1.   /  27.01.           -> 27 January (current year, or next if past)
//	27.1.2027                   -> exact
//	2026-05-21                  -> ISO (also handled by domain.ParseDate)
//	kw 32, kw32                 -> Monday of ISO week 32
//
// Recurrence phrases ("jeden Montag", "alle 2 Wochen") are not implemented yet;
// see parseRecurrence below for the planned set.
package nldate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dettmering/todo/internal/domain"
)

// Result is what Parse returns: at most a visible-from date and (later) a
// recurrence rule. Either field can be its zero value if not extracted.
type Result struct {
	Date           domain.LocalDate
	RecurrenceRule string
}

// Parse turns natural-language German into a Result, anchored on `today`.
// Returns an error if nothing in `text` resolves to a date (and no recurrence
// phrase is recognised either).
func Parse(text string, today domain.LocalDate) (Result, error) {
	s := normalize(text)
	if s == "" {
		return Result{}, nil
	}
	r := Result{}
	if rule := parseRecurrence(s); rule != "" {
		r.RecurrenceRule = rule
	}
	if d, ok := parseDate(s, today); ok {
		r.Date = d
	}
	if r.Date.IsZero() && r.RecurrenceRule == "" {
		return Result{}, fmt.Errorf("cannot parse %q as a date", text)
	}
	return r, nil
}

// ParseDate is a convenience wrapper for callers that only want a date.
func ParseDate(text string, today domain.LocalDate) (domain.LocalDate, error) {
	r, err := Parse(text, today)
	if err != nil {
		return domain.LocalDate{}, err
	}
	if r.Date.IsZero() {
		return domain.LocalDate{}, fmt.Errorf("no date in %q", text)
	}
	return r.Date, nil
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	// Collapse internal whitespace.
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func parseDate(s string, today domain.LocalDate) (domain.LocalDate, bool) {
	if d, err := domain.ParseDate(s); err == nil {
		return d, true
	}
	// Rules tried top-down. First match wins.
	for _, fn := range dateRules {
		if d, ok := fn(s, today); ok {
			return d, true
		}
	}
	return domain.LocalDate{}, false
}

type dateRule func(s string, today domain.LocalDate) (domain.LocalDate, bool)

var dateRules = []dateRule{
	ruleToday,
	ruleTomorrow,
	ruleDayAfterTomorrow,
	ruleYesterday,
	ruleNextWeek,
	ruleNextMonth,
	ruleNextYear,
	ruleMonatsende,
	ruleMonatsanfang,
	ruleMonthAnchor, // anfang|mitte|ende <month>
	ruleInN,
	rulePlusN,
	ruleKW,
	ruleDDMMYYYY,
	ruleDDMM,
	ruleWeekday, // last so "nächsten montag" patterns above can grab first
}

// ---------- individual rules ----------

func ruleToday(s string, today domain.LocalDate) (domain.LocalDate, bool) {
	if s == "heute" || s == "heu" {
		return today, true
	}
	return domain.LocalDate{}, false
}

func ruleTomorrow(s string, today domain.LocalDate) (domain.LocalDate, bool) {
	if s == "morgen" || s == "mor" {
		return addDays(today, 1), true
	}
	return domain.LocalDate{}, false
}

func ruleDayAfterTomorrow(s string, today domain.LocalDate) (domain.LocalDate, bool) {
	if s == "übermorgen" || s == "uebermorgen" {
		return addDays(today, 2), true
	}
	return domain.LocalDate{}, false
}

func ruleYesterday(s string, today domain.LocalDate) (domain.LocalDate, bool) {
	if s == "gestern" {
		return addDays(today, -1), true
	}
	return domain.LocalDate{}, false
}

func ruleNextWeek(s string, today domain.LocalDate) (domain.LocalDate, bool) {
	if s == "nächste woche" || s == "naechste woche" || s == "nächsten woche" || s == "naechsten woche" {
		return mondayOfNextISOWeek(today), true
	}
	return domain.LocalDate{}, false
}

func ruleNextMonth(s string, today domain.LocalDate) (domain.LocalDate, bool) {
	if s == "nächsten monat" || s == "naechsten monat" || s == "nächster monat" || s == "naechster monat" {
		// Same day next month, clamped to the new month's length so Jan 31 + 1
		// becomes Feb 28/29 rather than Mar 3 (which time.AddDate would do).
		y, m := today.Year, today.Month+1
		if m > 12 {
			m -= 12
			y++
		}
		return clampToMonth(y, m, today.Day), true
	}
	return domain.LocalDate{}, false
}

func ruleNextYear(s string, today domain.LocalDate) (domain.LocalDate, bool) {
	if s == "nächstes jahr" || s == "naechstes jahr" {
		return clampToMonth(today.Year+1, today.Month, today.Day), true
	}
	return domain.LocalDate{}, false
}

func ruleMonatsende(s string, today domain.LocalDate) (domain.LocalDate, bool) {
	switch s {
	case "monatsende", "ende des monats", "ende monat":
		return lastDayOf(today.Year, today.Month), true
	}
	return domain.LocalDate{}, false
}

func ruleMonatsanfang(s string, today domain.LocalDate) (domain.LocalDate, bool) {
	switch s {
	case "monatsanfang", "anfang des monats", "anfang monat":
		return domain.LocalDate{Year: today.Year, Month: today.Month, Day: 1}, true
	}
	return domain.LocalDate{}, false
}

var reMonthAnchor = regexp.MustCompile(`^(anfang|mitte|ende)\s+(.+)$`)

func ruleMonthAnchor(s string, today domain.LocalDate) (domain.LocalDate, bool) {
	m := reMonthAnchor.FindStringSubmatch(s)
	if m == nil {
		return domain.LocalDate{}, false
	}
	mon, ok := monthByName(m[2])
	if !ok {
		return domain.LocalDate{}, false
	}
	year := today.Year
	var day int
	switch m[1] {
	case "anfang":
		day = 1
	case "mitte":
		day = 15
	case "ende":
		day = lastDayOf(year, mon).Day
	}
	candidate := domain.LocalDate{Year: year, Month: mon, Day: day}
	if candidate.Before(today) {
		year++
		if m[1] == "ende" {
			day = lastDayOf(year, mon).Day
		}
		candidate = domain.LocalDate{Year: year, Month: mon, Day: day}
	}
	return candidate, true
}

var reInN = regexp.MustCompile(`^in\s+(\d+)\s+(tag|tage|tagen|woche|wochen|monat|monaten|jahr|jahren)$`)

func ruleInN(s string, today domain.LocalDate) (domain.LocalDate, bool) {
	m := reInN.FindStringSubmatch(s)
	if m == nil {
		return domain.LocalDate{}, false
	}
	n, _ := strconv.Atoi(m[1])
	switch m[2] {
	case "tag", "tage", "tagen":
		return addDays(today, n), true
	case "woche", "wochen":
		return addDays(today, n*7), true
	case "monat", "monaten":
		y, mo := today.Year, int(today.Month)+n
		for mo > 12 {
			mo -= 12
			y++
		}
		return clampToMonth(y, time.Month(mo), today.Day), true
	case "jahr", "jahren":
		return clampToMonth(today.Year+n, today.Month, today.Day), true
	}
	return domain.LocalDate{}, false
}

var rePlusN = regexp.MustCompile(`^\+(\d+)(?:\s*(d|tage|tagen))?$`)

func rulePlusN(s string, today domain.LocalDate) (domain.LocalDate, bool) {
	m := rePlusN.FindStringSubmatch(s)
	if m == nil {
		return domain.LocalDate{}, false
	}
	n, _ := strconv.Atoi(m[1])
	return addDays(today, n), true
}

var reKW = regexp.MustCompile(`^kw\s*(\d{1,2})(?:\s*[/ -]\s*(\d{4}))?$`)

func ruleKW(s string, today domain.LocalDate) (domain.LocalDate, bool) {
	m := reKW.FindStringSubmatch(s)
	if m == nil {
		return domain.LocalDate{}, false
	}
	week, _ := strconv.Atoi(m[1])
	if week < 1 || week > 53 {
		return domain.LocalDate{}, false
	}
	year := today.Year
	if m[2] != "" {
		year, _ = strconv.Atoi(m[2])
	}
	mon := mondayOfISOWeek(year, week)
	// If no year given and the week's Monday is already past, roll to next year.
	if m[2] == "" && mon.Before(today) {
		mon = mondayOfISOWeek(year+1, week)
	}
	return mon, true
}

var reDDMMYYYY = regexp.MustCompile(`^(\d{1,2})\.(\d{1,2})\.(\d{4})$`)

func ruleDDMMYYYY(s string, today domain.LocalDate) (domain.LocalDate, bool) {
	m := reDDMMYYYY.FindStringSubmatch(s)
	if m == nil {
		return domain.LocalDate{}, false
	}
	d, _ := strconv.Atoi(m[1])
	mo, _ := strconv.Atoi(m[2])
	y, _ := strconv.Atoi(m[3])
	if !validYMD(y, mo, d) {
		return domain.LocalDate{}, false
	}
	return domain.LocalDate{Year: y, Month: time.Month(mo), Day: d}, true
}

var reDDMM = regexp.MustCompile(`^(\d{1,2})\.(\d{1,2})\.?$`)

func ruleDDMM(s string, today domain.LocalDate) (domain.LocalDate, bool) {
	m := reDDMM.FindStringSubmatch(s)
	if m == nil {
		return domain.LocalDate{}, false
	}
	d, _ := strconv.Atoi(m[1])
	mo, _ := strconv.Atoi(m[2])
	if !validYMD(today.Year, mo, d) {
		return domain.LocalDate{}, false
	}
	candidate := domain.LocalDate{Year: today.Year, Month: time.Month(mo), Day: d}
	if candidate.Before(today) {
		candidate.Year++
	}
	return candidate, true
}

func ruleWeekday(s string, today domain.LocalDate) (domain.LocalDate, bool) {
	// Strip an optional leading "nächsten"/"nächste"/"nächster"/"nächstes" so
	// e.g. "nächsten freitag" still resolves; we currently treat it the same as
	// the bare weekday (next upcoming occurrence; today if today matches).
	for _, prefix := range []string{"nächsten ", "nächste ", "nächster ", "nächstes ", "naechsten ", "naechste ", "naechster ", "naechstes "} {
		s = strings.TrimPrefix(s, prefix)
	}
	w, ok := weekdayByName(s)
	if !ok {
		return domain.LocalDate{}, false
	}
	cur := toTime(today).Weekday()
	delta := (int(w) - int(cur) + 7) % 7
	return addDays(today, delta), true
}

// ---------- helpers ----------

func addDays(d domain.LocalDate, n int) domain.LocalDate {
	t := toTime(d).AddDate(0, 0, n)
	return domain.DateOf(t)
}

func toTime(d domain.LocalDate) time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
}

func clampToMonth(year int, m time.Month, day int) domain.LocalDate {
	last := lastDayOf(year, m).Day
	if day > last {
		day = last
	}
	return domain.LocalDate{Year: year, Month: m, Day: day}
}

func lastDayOf(year int, m time.Month) domain.LocalDate {
	first := time.Date(year, m, 1, 0, 0, 0, 0, time.UTC)
	end := first.AddDate(0, 1, -1)
	return domain.DateOf(end)
}

func validYMD(y, m, d int) bool {
	if m < 1 || m > 12 || d < 1 {
		return false
	}
	last := lastDayOf(y, time.Month(m)).Day
	return d <= last
}

func mondayOfISOWeek(year, week int) domain.LocalDate {
	// Jan 4 is always in ISO week 1. Find the Monday on or before Jan 4, then
	// advance (week-1) full weeks.
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
	weekday := int(jan4.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := jan4.AddDate(0, 0, -(weekday - 1))
	monday = monday.AddDate(0, 0, (week-1)*7)
	return domain.DateOf(monday)
}

func mondayOfNextISOWeek(today domain.LocalDate) domain.LocalDate {
	// Find this week's Monday, then add 7 days.
	t := toTime(today)
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	monday := t.AddDate(0, 0, -(wd - 1) + 7)
	return domain.DateOf(monday)
}

var monthNames = map[string]time.Month{
	"januar": time.January, "jan": time.January,
	"februar": time.February, "feb": time.February,
	"märz": time.March, "maerz": time.March, "mär": time.March, "mar": time.March,
	"april": time.April, "apr": time.April,
	"mai":  time.May,
	"juni": time.June, "jun": time.June,
	"juli": time.July, "jul": time.July,
	"august": time.August, "aug": time.August,
	"september": time.September, "sep": time.September, "sept": time.September,
	"oktober": time.October, "okt": time.October, "oct": time.October,
	"november": time.November, "nov": time.November,
	"dezember": time.December, "dez": time.December, "dec": time.December,
}

func monthByName(s string) (time.Month, bool) {
	m, ok := monthNames[s]
	return m, ok
}

var weekdayNames = map[string]time.Weekday{
	"montag": time.Monday, "mo": time.Monday, "mon": time.Monday,
	"dienstag": time.Tuesday, "di": time.Tuesday, "die": time.Tuesday,
	"mittwoch": time.Wednesday, "mi": time.Wednesday, "mit": time.Wednesday,
	"donnerstag": time.Thursday, "do": time.Thursday, "don": time.Thursday,
	"freitag": time.Friday, "fr": time.Friday, "fre": time.Friday,
	"samstag": time.Saturday, "sa": time.Saturday, "sam": time.Saturday, "sonnabend": time.Saturday,
	"sonntag": time.Sunday, "so": time.Sunday, "son": time.Sunday,
}

func weekdayByName(s string) (time.Weekday, bool) {
	w, ok := weekdayNames[s]
	return w, ok
}

// ---------- recurrence (planned, not yet implemented) ----------
//
// parseRecurrence returns an RRULE string (matching internal/recurrence's
// supported subset) or "" if no recurrence phrase is recognised. The
// implementation below is intentionally a stub — the patterns to support are
// documented here so a future implementation has a concrete checklist:
//
//	"täglich" / "jeden tag"                 -> FREQ=DAILY
//	"jeden N. tag" / "alle N tage"          -> FREQ=DAILY;INTERVAL=N
//	"jeden montag" .. "jeden sonntag"       -> FREQ=WEEKLY;BYDAY=<2-letter>
//	"jeden mo, mi, fr"                      -> FREQ=WEEKLY;BYDAY=MO,WE,FR
//	"jeden werktag"                         -> FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR
//	"wöchentlich"                           -> FREQ=WEEKLY
//	"alle N wochen" / "alle N wochen am MO" -> FREQ=WEEKLY;INTERVAL=N[;BYDAY=MO]
//	"monatlich"                             -> FREQ=MONTHLY
//	"jeden N. des monats"                   -> FREQ=MONTHLY;BYMONTHDAY=N
//	"alle N monate"                         -> FREQ=MONTHLY;INTERVAL=N
//	"jährlich"                              -> FREQ=YEARLY
//	"jedes jahr am DD.MM."                  -> FREQ=YEARLY;BYMONTH=MM;BYMONTHDAY=DD
//
// Edge cases worth considering when implementing:
//   - "jeden ersten Montag im Monat" needs BYSETPOS which the current
//     recurrence package does not support yet.
//   - Mixed phrases like "jeden Freitag ab 01.06." combine a recurrence with
//     a start date — Parse should be able to return both Date and RecurrenceRule.
func parseRecurrence(s string) string {
	// TODO: implement the patterns documented above. Returning "" leaves the
	// task without a recurrence rule, which keeps current behavior unchanged.
	_ = s
	return ""
}
