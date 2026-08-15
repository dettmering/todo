package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/storage"
)

// at pins the service clock to 09:00 on the given day.
func at(svc *Service, y int, m time.Month, d int) {
	svc.SetClock(func() time.Time { return time.Date(y, m, d, 9, 0, 0, 0, time.UTC) })
}

// completeAndFindSuccessor ticks off the task and returns the follow-up
// instance spawned by the recurrence, identified by its shared title.
func completeAndFindSuccessor(t *testing.T, svc *Service, ctx context.Context, task *domain.Task) *domain.Task {
	t.Helper()
	if _, err := svc.CompleteTask(ctx, task.ID); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}
	list, err := svc.ListTasks(ctx, storage.TaskFilter{Status: domain.StatusOpen})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	for _, x := range list {
		if x.Title == task.Title && x.ID != task.ID {
			return x
		}
	}
	t.Fatalf("no successor spawned for %q", task.Title)
	return nil
}

func TestCompleteRecurringAnchors(t *testing.T) {
	cases := []struct {
		name        string
		rule        string
		mode        domain.RecurrenceMode
		visibleFrom string
		completeOn  [3]int // year, month, day
		want        string
	}{
		{
			// The birthday case: two days late, still lands on its own day.
			name: "fixed yearly ignores a late completion",
			rule: "FREQ=YEARLY", mode: domain.RecurFixed,
			visibleFrom: "2026-08-15", completeOn: [3]int{2026, 8, 17}, want: "2027-08-15",
		},
		{
			// The coffee-machine case: the interval restarts on completion.
			name: "after_completion restarts the interval",
			rule: "FREQ=WEEKLY;INTERVAL=4", mode: domain.RecurAfterDone,
			visibleFrom: "2026-07-20", completeOn: [3]int{2026, 8, 17}, want: "2026-09-14",
		},
		{
			// Ticked off early, the floating interval still counts from today.
			name: "after_completion counts from an early completion",
			rule: "FREQ=WEEKLY;INTERVAL=4", mode: domain.RecurAfterDone,
			visibleFrom: "2026-08-20", completeOn: [3]int{2026, 8, 15}, want: "2026-09-12",
		},
		{
			name: "after_completion monthly",
			rule: "FREQ=MONTHLY", mode: domain.RecurAfterDone,
			visibleFrom: "2026-06-01", completeOn: [3]int{2026, 8, 17}, want: "2026-09-17",
		},
		{
			// An unset mode is the pre-migration encoding of "fixed".
			name: "empty mode behaves like fixed",
			rule: "FREQ=MONTHLY;BYMONTHDAY=10", mode: "",
			visibleFrom: "2026-03-10", completeOn: [3]int{2026, 8, 17}, want: "2026-09-10",
		},
		{
			// Fixed mode skips missed occurrences rather than piling them up.
			name: "fixed skips occurrences missed long ago",
			rule: "FREQ=MONTHLY;BYMONTHDAY=10", mode: domain.RecurFixed,
			visibleFrom: "2026-01-10", completeOn: [3]int{2026, 8, 17}, want: "2026-09-10",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			svc := newTestService(t)
			at(svc, 2026, time.January, 1)

			task, err := svc.CreateTask(ctx, TaskInput{
				Title:          tc.name,
				VisibleFrom:    date(t, tc.visibleFrom),
				RecurrenceRule: tc.rule,
				RecurrenceMode: tc.mode,
			})
			if err != nil {
				t.Fatalf("CreateTask: %v", err)
			}

			at(svc, tc.completeOn[0], time.Month(tc.completeOn[1]), tc.completeOn[2])
			next := completeAndFindSuccessor(t, svc, ctx, task)

			if got := next.VisibleFrom.String(); got != tc.want {
				t.Errorf("visible_from = %s, want %s", got, tc.want)
			}
			// The successor has to carry the mode, or the series degrades to
			// "fixed" after the first completion.
			if next.RecurrenceMode != task.RecurrenceMode {
				t.Errorf("mode = %q, want %q", next.RecurrenceMode, task.RecurrenceMode)
			}
		})
	}
}

// Postponing a single occurrence must not drag the series along: a birthday on
// 1 August, pushed to Wednesday the 19th and ticked off there, is still due on
// 1 August next year.
func TestPostponeDoesNotMoveSeries(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	at(svc, 2026, time.July, 1)

	task, err := svc.CreateTask(ctx, TaskInput{
		Title:          "Hans gratulieren",
		VisibleFrom:    date(t, "2026-08-01"),
		RecurrenceRule: "FREQ=YEARLY",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if got := task.RecurrenceAnchor.String(); got != "2026-08-01" {
		t.Fatalf("anchor after create = %s, want 2026-08-01", got)
	}

	// Postpone this one occurrence to Wednesday the 19th.
	moved := date(t, "2026-08-19")
	task, err = svc.UpdateTask(ctx, task.ID, TaskPatch{VisibleFrom: &moved})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if got := task.RecurrenceAnchor.String(); got != "2026-08-01" {
		t.Errorf("postponing moved the series to %s, want it to stay on 2026-08-01", got)
	}

	at(svc, 2026, time.August, 19)
	next := completeAndFindSuccessor(t, svc, ctx, task)
	if got := next.VisibleFrom.String(); got != "2027-08-01" {
		t.Errorf("visible_from = %s, want 2027-08-01", got)
	}
	// The fresh occurrence is un-postponed, so both dates coincide again.
	if got := next.RecurrenceAnchor.String(); got != "2027-08-01" {
		t.Errorf("anchor = %s, want 2027-08-01", got)
	}
}

// Moving the series itself is an explicit act, separate from postponing.
func TestExplicitAnchorMovesSeries(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	at(svc, 2026, time.July, 1)

	task, err := svc.CreateTask(ctx, TaskInput{
		Title:          "Hans gratulieren",
		VisibleFrom:    date(t, "2026-08-01"),
		RecurrenceRule: "FREQ=YEARLY",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// The birthday was wrong all along — move the series to the 19th.
	anchor := date(t, "2026-08-19")
	if _, err := svc.UpdateTask(ctx, task.ID, TaskPatch{RecurrenceAnchor: &anchor}); err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}

	at(svc, 2026, time.August, 19)
	task, err = svc.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	next := completeAndFindSuccessor(t, svc, ctx, task)
	if got := next.VisibleFrom.String(); got != "2027-08-19" {
		t.Errorf("visible_from = %s, want 2027-08-19", got)
	}
}

// after_completion re-anchors on every completion, so it stores no series date.
func TestAfterCompletionKeepsNoAnchor(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	at(svc, 2026, time.August, 1)

	task, err := svc.CreateTask(ctx, TaskInput{
		Title:            "Kaffeemaschine",
		VisibleFrom:      date(t, "2026-08-01"),
		RecurrenceRule:   "FREQ=WEEKLY;INTERVAL=4",
		RecurrenceMode:   domain.RecurAfterDone,
		RecurrenceAnchor: date(t, "2026-08-01"),
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if !task.RecurrenceAnchor.IsZero() {
		t.Errorf("anchor = %s, want empty for after_completion", task.RecurrenceAnchor)
	}

	at(svc, 2026, time.August, 17)
	next := completeAndFindSuccessor(t, svc, ctx, task)
	if !next.RecurrenceAnchor.IsZero() {
		t.Errorf("successor anchor = %s, want empty", next.RecurrenceAnchor)
	}
	if got := next.VisibleFrom.String(); got != "2026-09-14" {
		t.Errorf("visible_from = %s, want 2026-09-14", got)
	}
}

// A postponed occurrence keeps its deadline window relative to the series date,
// not to the date it was dragged to.
func TestPostponeKeepsDeadlineWindow(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	at(svc, 2026, time.July, 1)

	task, err := svc.CreateTask(ctx, TaskInput{
		Title:          "Bericht",
		VisibleFrom:    date(t, "2026-08-01"),
		Deadline:       date(t, "2026-08-08"), // 7-day window
		RecurrenceRule: "FREQ=YEARLY",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	moved := date(t, "2026-08-19")
	task, err = svc.UpdateTask(ctx, task.ID, TaskPatch{VisibleFrom: &moved})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}

	at(svc, 2026, time.August, 19)
	next := completeAndFindSuccessor(t, svc, ctx, task)
	if got := next.VisibleFrom.String(); got != "2027-08-01" {
		t.Errorf("visible_from = %s, want 2027-08-01", got)
	}
	if got := next.Deadline.String(); got != "2027-08-08" {
		t.Errorf("deadline = %s, want 2027-08-08", got)
	}
}

// The deadline window must survive both anchors: a task that appears Monday and
// is due Friday stays a 4-day job on its next occurrence.
func TestCompleteRecurringKeepsDeadlineWindow(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	at(svc, 2026, time.August, 1)

	task, err := svc.CreateTask(ctx, TaskInput{
		Title:          "Fensterlänge",
		VisibleFrom:    date(t, "2026-08-10"), // Monday
		Deadline:       date(t, "2026-08-14"), // Friday
		RecurrenceRule: "FREQ=WEEKLY;INTERVAL=2",
		RecurrenceMode: domain.RecurAfterDone,
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	at(svc, 2026, time.August, 17)
	next := completeAndFindSuccessor(t, svc, ctx, task)

	if got := next.VisibleFrom.String(); got != "2026-08-31" {
		t.Errorf("visible_from = %s, want 2026-08-31", got)
	}
	if got := next.Deadline.String(); got != "2026-09-04" {
		t.Errorf("deadline = %s, want 2026-09-04", got)
	}
}

func TestRecurrenceModeValidation(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	// "Every Monday, starting when you tick it off" has no coherent reading.
	for _, rule := range []string{
		"FREQ=WEEKLY;BYDAY=MO",
		"FREQ=MONTHLY;BYMONTHDAY=1",
		"FREQ=YEARLY;BYMONTH=3;BYMONTHDAY=1",
	} {
		_, err := svc.CreateTask(ctx, TaskInput{
			Title: "x", RecurrenceRule: rule, RecurrenceMode: domain.RecurAfterDone,
		})
		if !errors.Is(err, ErrValidation) {
			t.Errorf("%s with after_completion: got %v, want ErrValidation", rule, err)
		}
	}

	// Unknown mode values are rejected rather than silently treated as fixed.
	if _, err := svc.CreateTask(ctx, TaskInput{
		Title: "x", RecurrenceRule: "FREQ=DAILY", RecurrenceMode: "whenever",
	}); !errors.Is(err, ErrValidation) {
		t.Errorf("bogus mode: got %v, want ErrValidation", err)
	}

	// A recurring task without an explicit mode is stored as "fixed"...
	task, err := svc.CreateTask(ctx, TaskInput{Title: "x", RecurrenceRule: "FREQ=DAILY"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if task.RecurrenceMode != domain.RecurFixed {
		t.Errorf("default mode = %q, want %q", task.RecurrenceMode, domain.RecurFixed)
	}

	// ...and a one-off task carries no mode at all.
	plain, err := svc.CreateTask(ctx, TaskInput{Title: "y", RecurrenceMode: domain.RecurAfterDone})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if plain.RecurrenceMode != "" {
		t.Errorf("mode without rule = %q, want empty", plain.RecurrenceMode)
	}
}

// Rule and mode are validated as a pair, even when a patch touches only one.
func TestUpdateTaskRecurrenceModePairing(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	task, err := svc.CreateTask(ctx, TaskInput{
		Title: "Kaffeemaschine", RecurrenceRule: "FREQ=WEEKLY;BYDAY=MO",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Switching only the mode must trip over the existing BYDAY anchor.
	mode := domain.RecurAfterDone
	if _, err := svc.UpdateTask(ctx, task.ID, TaskPatch{RecurrenceMode: &mode}); !errors.Is(err, ErrValidation) {
		t.Errorf("mode-only patch onto BYDAY rule: got %v, want ErrValidation", err)
	}

	// Changing both at once is the supported path.
	rule := "FREQ=WEEKLY;INTERVAL=4"
	updated, err := svc.UpdateTask(ctx, task.ID, TaskPatch{RecurrenceRule: &rule, RecurrenceMode: &mode})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if updated.RecurrenceMode != domain.RecurAfterDone {
		t.Errorf("mode = %q, want %q", updated.RecurrenceMode, domain.RecurAfterDone)
	}

	// Clearing the rule clears the mode with it.
	empty := ""
	cleared, err := svc.UpdateTask(ctx, task.ID, TaskPatch{RecurrenceRule: &empty})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if cleared.RecurrenceMode != "" {
		t.Errorf("mode after clearing rule = %q, want empty", cleared.RecurrenceMode)
	}
}
