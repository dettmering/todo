package domain

import (
	"sort"
	"time"
)

// AbsenceType classifies an absence. Business trips are special: they don't
// block work, so the "next workday" scheduling treats them as normal days.
type AbsenceType string

const (
	AbsenceVacation     AbsenceType = "vacation"
	AbsenceHoliday      AbsenceType = "holiday"
	AbsenceBusinessTrip AbsenceType = "business_trip"
	AbsenceSick         AbsenceType = "sick"
	AbsenceOther        AbsenceType = "other"
)

func (t AbsenceType) Valid() bool {
	switch t {
	case AbsenceVacation, AbsenceHoliday, AbsenceBusinessTrip, AbsenceSick, AbsenceOther:
		return true
	}
	return false
}

// Absence is a period the user is away (vacation, business trip, sick, …).
// Start and End are inclusive calendar dates.
type Absence struct {
	ID        string
	Type      AbsenceType
	Label     string // optional free-text name, e.g. "Kreta"
	Start     LocalDate
	End       LocalDate
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Covers reports whether d lies within [Start, End] (inclusive).
func (a Absence) Covers(d LocalDate) bool {
	return !d.Before(a.Start) && !a.End.Before(d)
}

// BlocksWork reports whether no work happens during this absence. Business
// trips count as regular workdays and therefore do not block.
func (a Absence) BlocksWork() bool { return a.Type != AbsenceBusinessTrip }

// AbsencesForView returns the absences to display in the absences view: all
// upcoming and current ones, plus past ones that ended in the current calendar
// year. Past absences that already ended in an earlier year are dropped. Input
// order is preserved.
func AbsencesForView(absences []*Absence, today LocalDate) []*Absence {
	out := make([]*Absence, 0, len(absences))
	for _, a := range absences {
		if a.End.Year >= today.Year {
			out = append(out, a)
		}
	}
	return out
}

// AvailabilityStatus is the coarse present-day state shown on the public share
// page ("currently available / away / on a business trip").
type AvailabilityStatus int

const (
	StatusAvailable AvailabilityStatus = iota
	StatusAway
	StatusBusinessTrip
)

// CurrentAvailability reports the coarse availability on `today`: Away if any
// work-blocking absence covers today, BusinessTrip if only a (non-blocking)
// business trip does, otherwise Available. A blocking absence always wins over
// a concurrent business trip.
func CurrentAvailability(absences []*Absence, today LocalDate) AvailabilityStatus {
	onTrip := false
	for _, a := range absences {
		if !a.Covers(today) {
			continue
		}
		if a.BlocksWork() {
			return StatusAway
		}
		onTrip = true
	}
	if onTrip {
		return StatusBusinessTrip
	}
	return StatusAvailable
}

// ReturnDate reports when work resumes: the next workday after `today` (see
// NextWorkday). It only answers while a work-blocking absence actually covers
// today — someone who is available, or merely on a business trip (work goes on),
// has no return date to announce, and the zero LocalDate is returned instead.
// Used for the "Back on …" line of the public share page.
func ReturnDate(absences []*Absence, today LocalDate) LocalDate {
	if CurrentAvailability(absences, today) != StatusAway {
		return LocalDate{}
	}
	vals := make([]Absence, 0, len(absences))
	for _, a := range absences {
		vals = append(vals, *a)
	}
	return NextWorkday(today, vals)
}

// LatestUpdate returns the most recent UpdatedAt across the given absences, i.e.
// when the absence data was last touched. The bool is false when there are no
// absences (or none carry a timestamp), so callers can omit a "last updated"
// note rather than show a zero date.
func LatestUpdate(absences []*Absence) (time.Time, bool) {
	var latest time.Time
	for _, a := range absences {
		if a.UpdatedAt.After(latest) {
			latest = a.UpdatedAt
		}
	}
	return latest, !latest.IsZero()
}

// UpcomingAbsences returns the absences that are still current or in the future
// (End >= today), sorted by Start ascending. Past absences are dropped — they
// carry no signal for an availability view. Ties on Start keep input order.
// Used by the anonymized public share page.
func UpcomingAbsences(absences []*Absence, today LocalDate) []*Absence {
	out := make([]*Absence, 0, len(absences))
	for _, a := range absences {
		if !a.End.Before(today) {
			out = append(out, a)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Start.Before(out[j].Start)
	})
	return out
}

// ShareWindowWeeks is the horizon the public share page shows — both the
// timeline rail and the list below it cover the same span, so a reader can't
// find an entry in one and miss it in the other.
//
// The frontend's own rail repeats this number as TIMELINE_WEEKS in
// internal/web/static/app.js; it can't import from here. Change one, change
// the other, or the two absence views cover different horizons.
const ShareWindowWeeks = 12

// ShareWindow returns the inclusive window [start, end] the share page covers:
// the Monday of today's week plus ShareWindowWeeks weeks. Anchoring on Monday
// (rather than on today) keeps the window from crawling forward mid-week.
func ShareWindow(today LocalDate) (start, end LocalDate) {
	start = today.AddDays(-((int(today.Weekday()) + 6) % 7))
	return start, start.AddDays(ShareWindowWeeks*7 - 1)
}

// AbsencesInRange returns the absences that overlap the inclusive window
// [from, to] — i.e. any absence touching the window, including ones that are
// currently running (started before `from` and not yet ended). A zero `from`
// means no lower bound, a zero `to` means no upper bound. Input order is kept.
func AbsencesInRange(absences []*Absence, from, to LocalDate) []*Absence {
	out := make([]*Absence, 0, len(absences))
	for _, a := range absences {
		if !from.IsZero() && a.End.Before(from) {
			continue // ended before the window starts
		}
		if !to.IsZero() && to.Before(a.Start) {
			continue // starts after the window ends
		}
		out = append(out, a)
	}
	return out
}

// NextWorkday returns the first day strictly after `from` that is neither a
// weekend nor covered by a blocking absence (business trips are not blocking).
// The 366-iteration cap guards against an unexpected infinite loop.
func NextWorkday(from LocalDate, absences []Absence) LocalDate {
	d := from
	for i := 0; i < 366; i++ {
		d = d.AddDays(1)
		if d.IsWeekend() {
			continue
		}
		blocked := false
		for _, a := range absences {
			if a.BlocksWork() && a.Covers(d) {
				blocked = true
				break
			}
		}
		if !blocked {
			return d
		}
	}
	return d
}
