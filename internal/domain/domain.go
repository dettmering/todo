// Package domain defines the core business objects of the todo application.
// Types here have no dependency on storage, transport, or rendering.
package domain

import "time"

type TaskStatus string

const (
	StatusOpen      TaskStatus = "open"
	StatusCompleted TaskStatus = "completed"
	StatusDeleted   TaskStatus = "deleted"
)

// Priority follows the Lastenheft model: 1 = highest, 4 = lowest.
type Priority int

const (
	P1 Priority = 1
	P2 Priority = 2
	P3 Priority = 3
	P4 Priority = 4
)

func (p Priority) Valid() bool { return p >= P1 && p <= P4 }

// LocalDate represents a calendar date without time-of-day, stored as YYYY-MM-DD.
// Used for VisibleFrom and Deadline so timezone shifts can't move them.
type LocalDate struct {
	Year  int
	Month time.Month
	Day   int
}

func DateOf(t time.Time) LocalDate {
	y, m, d := t.Date()
	return LocalDate{Year: y, Month: m, Day: d}
}

func (d LocalDate) IsZero() bool { return d == LocalDate{} }

func (d LocalDate) String() string {
	if d.IsZero() {
		return ""
	}
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}

// Before reports whether d is strictly before other.
func (d LocalDate) Before(other LocalDate) bool {
	if d.Year != other.Year {
		return d.Year < other.Year
	}
	if d.Month != other.Month {
		return d.Month < other.Month
	}
	return d.Day < other.Day
}

// AfterOrEqual reports whether d is on or after other.
func (d LocalDate) AfterOrEqual(other LocalDate) bool { return !d.Before(other) }

// AddDays returns the date n calendar days after d (n may be negative).
func (d LocalDate) AddDays(n int) LocalDate {
	t := time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC).AddDate(0, 0, n)
	return DateOf(t)
}

// DaysUntil returns the number of calendar days from d to other — positive when
// other lies in the future, negative when it's in the past. Both operands are
// timezone-naive and anchored at UTC midnight here, so no zone shift or DST
// transition can bend the result.
func (d LocalDate) DaysUntil(other LocalDate) int {
	from := time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
	to := time.Date(other.Year, other.Month, other.Day, 0, 0, 0, 0, time.UTC)
	return int(to.Sub(from).Hours()) / 24
}

// Weekday reports which day of the week d falls on.
func (d LocalDate) Weekday() time.Weekday {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC).Weekday()
}

// ISOWeek returns the ISO 8601 calendar week d falls in (1–53). The year the
// week belongs to is deliberately dropped: callers render the bare number next
// to dates that already carry the year.
func (d LocalDate) ISOWeek() int {
	_, w := time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC).ISOWeek()
	return w
}

// IsWeekend reports whether d falls on a Saturday or Sunday.
func (d LocalDate) IsWeekend() bool {
	wd := d.Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

func ParseDate(s string) (LocalDate, error) {
	if s == "" {
		return LocalDate{}, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return LocalDate{}, err
	}
	return DateOf(t), nil
}

// Task is the central aggregate of the application.
type Task struct {
	ID             string
	Title          string
	Description    string
	ProjectID      string
	Status         TaskStatus
	Priority       Priority
	VisibleFrom    LocalDate // when the task starts appearing in the list
	Deadline       LocalDate // when it must be done
	RecurrenceRule string    // RRULE-compatible, empty if not recurring
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CompletedAt    *time.Time
}

// VisibleOn reports whether the task should appear in the standard list on the given date.
// Mirrors Lastenheft 5.5: Status=open AND (VisibleFrom empty OR VisibleFrom <= today).
func (t Task) VisibleOn(today LocalDate) bool {
	if t.Status != StatusOpen {
		return false
	}
	if t.VisibleFrom.IsZero() {
		return true
	}
	return !today.Before(t.VisibleFrom)
}

// Overdue reports whether the task's deadline has passed.
func (t Task) Overdue(today LocalDate) bool {
	if t.Deadline.IsZero() || t.Status != StatusOpen {
		return false
	}
	return t.Deadline.Before(today)
}

// Project groups tasks thematically.
type Project struct {
	ID          string
	Name        string
	Slug        string // URL-safe identifier, derived from Name; stable across renames
	Description string
	SortOrder   int
	Archived    bool
	Color       string // CSS hex color like "#246fe0", empty means default
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// InboxID is the reserved id of the default project.
const InboxID = "inbox"
