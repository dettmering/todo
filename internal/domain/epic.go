package domain

import (
	"sort"
	"time"
)

// EpicStatus is the workflow state of an epic. It is deliberately separate from
// Archived: a running epic that turned out to be a mistake is archived, an epic
// that was carried to the end is completed and stays in the record.
type EpicStatus string

const (
	// EpicActive is "laufend" — the epic is being worked on.
	EpicActive EpicStatus = "active"
	// EpicCompleted is "beendet". Only reachable when no task of the epic is
	// still open (see EpicCanComplete).
	EpicCompleted EpicStatus = "completed"
)

func (s EpicStatus) Valid() bool { return s == EpicActive || s == EpicCompleted }

// Epic is a long-running initiative inside one project — the layer between
// Project (a permanent bucket) and Task (a single step). Tasks point at it via
// Task.EpicID; an epic always belongs to exactly one project, and never to the
// Inbox, which is an inbox rather than an undertaking.
//
// Start and End are optional: an initiative may have no fixed beginning, no
// fixed end, or neither.
type Epic struct {
	ID          string
	Title       string
	Slug        string // URL-safe identifier, derived from Title; stable across renames
	Description string
	ProjectID   string
	Status      EpicStatus
	Start       LocalDate // optional; zero means "open-ended at the front"
	End         LocalDate // optional; zero means "open-ended at the back"
	SortOrder   int
	Archived    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
}

// ValidEpicRange reports whether a start/end pair is coherent. Either bound may
// be empty — only a pair where the end lies strictly before the start is wrong.
func ValidEpicRange(start, end LocalDate) bool {
	if start.IsZero() || end.IsZero() {
		return true
	}
	return !end.Before(start)
}

// Covers reports whether d lies inside the epic's window. An open-ended side
// never excludes a date, so an epic without any dates covers every day.
func (e Epic) Covers(d LocalDate) bool {
	if !e.Start.IsZero() && d.Before(e.Start) {
		return false
	}
	if !e.End.IsZero() && e.End.Before(d) {
		return false
	}
	return true
}

// Assignable reports whether tasks may be linked to this epic. An archived or
// already completed epic is not: the first is gone from every view, the second
// would have its "all tasks done" promise broken the moment an open task
// arrives.
func (e Epic) Assignable() bool { return !e.Archived && e.Status == EpicActive }

// EpicCounts is the task tally shown next to an epic: how much is left and how
// much is behind it.
type EpicCounts struct {
	Open int
	Done int
}

// Total is Open + Done. Deleted tasks are not counted anywhere.
func (c EpicCounts) Total() int { return c.Open + c.Done }

// CountEpicTasks tallies tasks per epic id. Tasks without an epic and
// soft-deleted tasks are ignored — a deleted task must not keep an epic from
// being completed, and it must not inflate the progress denominator either.
func CountEpicTasks(tasks []*Task) map[string]EpicCounts {
	out := make(map[string]EpicCounts)
	for _, t := range tasks {
		if t == nil || t.EpicID == "" || t.Status == StatusDeleted {
			continue
		}
		c := out[t.EpicID]
		if t.Status == StatusOpen {
			c.Open++
		} else {
			c.Done++
		}
		out[t.EpicID] = c
	}
	return out
}

// EpicCanComplete reports whether an epic whose tasks these are may be closed:
// no task may still be open. Completed and soft-deleted tasks do not block —
// a task that was thrown away is not work left to do.
func EpicCanComplete(tasks []*Task) bool { return OpenTaskCount(tasks) == 0 }

// OpenTaskCount counts the tasks still open, so callers can name the number in
// an error message rather than only saying "no".
func OpenTaskCount(tasks []*Task) int {
	n := 0
	for _, t := range tasks {
		if t != nil && t.Status == StatusOpen {
			n++
		}
	}
	return n
}

// ActiveEpics returns the epics that are neither archived nor completed,
// optionally narrowed to one project (empty projectID means all). Sorted by
// SortOrder, then title — the same order the sidebar and the project view show,
// so both read alike.
func ActiveEpics(epics []*Epic, projectID string) []*Epic {
	out := make([]*Epic, 0, len(epics))
	for _, e := range epics {
		if e == nil || e.Archived || e.Status != EpicActive {
			continue
		}
		if projectID != "" && e.ProjectID != projectID {
			continue
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].Title < out[j].Title
	})
	return out
}
