package service

import (
	"context"
	"testing"

	"github.com/dettmering/todo/internal/domain"
	"github.com/dettmering/todo/internal/storage"
)

// The per-epic tallies are computed in SQL now, while domain.CountEpicTasks
// still states the rule in Go. This pins the two together: if one side ever
// starts counting deleted tasks or tasks without an epic, the other disagrees.
func TestEpicTaskCountsMatchesDomainRule(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	p := newTestProject(t, svc, "Renovierung")
	e1, _ := svc.CreateEpic(ctx, EpicInput{Title: "Küche", ProjectID: p.ID})
	e2, _ := svc.CreateEpic(ctx, EpicInput{Title: "Bad", ProjectID: p.ID})

	svc.CreateTask(ctx, TaskInput{Title: "offen 1", EpicID: e1.ID})
	svc.CreateTask(ctx, TaskInput{Title: "offen 2", EpicID: e1.ID})
	done, _ := svc.CreateTask(ctx, TaskInput{Title: "fertig", EpicID: e1.ID})
	gone, _ := svc.CreateTask(ctx, TaskInput{Title: "geloescht", EpicID: e1.ID})
	svc.CreateTask(ctx, TaskInput{Title: "anderes Epic", EpicID: e2.ID})
	svc.CreateTask(ctx, TaskInput{Title: "ohne Epic", ProjectID: p.ID})
	if _, err := svc.CompleteTask(ctx, done.ID); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}
	if err := svc.DeleteTask(ctx, gone.ID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}

	got, err := svc.EpicTaskCounts(ctx)
	if err != nil {
		t.Fatalf("EpicTaskCounts: %v", err)
	}
	all, err := svc.ListTasks(ctx, storage.TaskFilter{})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	want := domain.CountEpicTasks(all)
	if len(got) != len(want) {
		t.Fatalf("got %d epics with counts, want %d (got %+v, want %+v)", len(got), len(want), got, want)
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("epic %s: got %+v, want %+v", id, got[id], w)
		}
	}
	if got[e1.ID] != (domain.EpicCounts{Open: 2, Done: 1}) {
		t.Errorf("epic 1: got %+v, want {Open:2 Done:1}", got[e1.ID])
	}
}

func TestOpenTaskCounts(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	p := newTestProject(t, svc, "Renovierung")
	e, _ := svc.CreateEpic(ctx, EpicInput{Title: "Küche", ProjectID: p.ID})
	today := date(t, "2026-09-04")

	// Visible: no date at all, an earlier date, and today itself.
	svc.CreateTask(ctx, TaskInput{Title: "ohne Datum", ProjectID: p.ID, EpicID: e.ID})
	svc.CreateTask(ctx, TaskInput{Title: "gestern", ProjectID: p.ID, VisibleFrom: date(t, "2026-09-03")})
	svc.CreateTask(ctx, TaskInput{Title: "heute", ProjectID: p.ID, VisibleFrom: today})
	// Not visible yet, but still an open task of the project.
	svc.CreateTask(ctx, TaskInput{Title: "morgen", ProjectID: p.ID, VisibleFrom: date(t, "2026-09-05")})
	// Inbox, and two tasks that must not count at all.
	svc.CreateTask(ctx, TaskInput{Title: "inbox"})
	done, _ := svc.CreateTask(ctx, TaskInput{Title: "fertig", ProjectID: p.ID, EpicID: e.ID})
	gone, _ := svc.CreateTask(ctx, TaskInput{Title: "geloescht", ProjectID: p.ID, EpicID: e.ID})
	if _, err := svc.CompleteTask(ctx, done.ID); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}
	if err := svc.DeleteTask(ctx, gone.ID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}

	counts, err := svc.OpenTaskCounts(ctx, today)
	if err != nil {
		t.Fatalf("OpenTaskCounts: %v", err)
	}
	if got, want := counts.ByProject[p.ID], 4; got != want {
		t.Errorf("project count = %d, want %d", got, want)
	}
	if got, want := counts.ByProject["inbox"], 1; got != want {
		t.Errorf("inbox count = %d, want %d", got, want)
	}
	if got, want := counts.ByEpic[e.ID], 1; got != want {
		t.Errorf("epic count = %d, want %d", got, want)
	}
	// Four of the five open tasks are visible today; "morgen" is not.
	if got, want := counts.VisibleToday, 4; got != want {
		t.Errorf("visible today = %d, want %d", got, want)
	}
}

// The tallies must agree with what the list endpoints return for the same day,
// since the sidebar badge and the list below it are read as one number.
func TestOpenTaskCountsAgreeWithListing(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	p := newTestProject(t, svc, "Renovierung")
	today := date(t, "2026-09-04")
	svc.CreateTask(ctx, TaskInput{Title: "a", ProjectID: p.ID})
	svc.CreateTask(ctx, TaskInput{Title: "b", ProjectID: p.ID, VisibleFrom: date(t, "2026-09-05")})
	svc.CreateTask(ctx, TaskInput{Title: "c", ProjectID: p.ID, VisibleFrom: date(t, "2026-01-01")})

	counts, err := svc.OpenTaskCounts(ctx, today)
	if err != nil {
		t.Fatalf("OpenTaskCounts: %v", err)
	}
	visible, err := svc.ListTasks(ctx, storage.TaskFilter{
		Status: domain.StatusOpen, OnlyVisible: true, Today: today,
	})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if counts.VisibleToday != len(visible) {
		t.Errorf("VisibleToday = %d, but the visible listing has %d tasks", counts.VisibleToday, len(visible))
	}
	open, err := svc.ListTasks(ctx, storage.TaskFilter{ProjectID: p.ID, Status: domain.StatusOpen})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if counts.ByProject[p.ID] != len(open) {
		t.Errorf("ByProject = %d, but the project listing has %d open tasks", counts.ByProject[p.ID], len(open))
	}
}

func TestOpenTaskCountsEmptyStore(t *testing.T) {
	svc := newTestService(t)
	counts, err := svc.OpenTaskCounts(context.Background(), date(t, "2026-09-04"))
	if err != nil {
		t.Fatalf("OpenTaskCounts: %v", err)
	}
	if counts.VisibleToday != 0 || len(counts.ByProject) != 0 || len(counts.ByEpic) != 0 {
		t.Errorf("empty store gave %+v", counts)
	}
}

func TestCommentCounts(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	a, _ := svc.CreateTask(ctx, TaskInput{Title: "mit Kommentaren"})
	b, _ := svc.CreateTask(ctx, TaskInput{Title: "ohne"})
	if _, err := svc.AddComment(ctx, a.ID, "erster"); err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	if _, err := svc.AddComment(ctx, a.ID, "zweiter"); err != nil {
		t.Fatalf("AddComment: %v", err)
	}

	counts, err := svc.CommentCounts(ctx)
	if err != nil {
		t.Fatalf("CommentCounts: %v", err)
	}
	if counts[a.ID] != 2 {
		t.Errorf("task a: got %d comments, want 2", counts[a.ID])
	}
	if _, ok := counts[b.ID]; ok {
		t.Errorf("task without comments should be absent from the map, got %d", counts[b.ID])
	}
}
