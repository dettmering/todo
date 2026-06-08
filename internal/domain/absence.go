package domain

import "time"

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
