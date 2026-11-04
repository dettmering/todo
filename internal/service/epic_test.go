package service

import (
	"context"
	"errors"
	"testing"

	"github.com/dettmering/todo/internal/domain"
	"github.com/dettmering/todo/internal/storage"
)

// newTestProject creates a real project to hang epics off — the Inbox is
// deliberately not allowed to hold them.
func newTestProject(t *testing.T, svc *Service, name string) *domain.Project {
	t.Helper()
	p, err := svc.CreateProject(context.Background(), ProjectInput{Name: name})
	if err != nil {
		t.Fatalf("CreateProject(%q): %v", name, err)
	}
	return p
}

func TestEpicCRUD(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	p := newTestProject(t, svc, "Renovierung")

	e, err := svc.CreateEpic(ctx, EpicInput{
		Title: "Küche renovieren", ProjectID: p.ID,
		Start: date(t, "2026-09-01"), End: date(t, "2027-03-31"),
	})
	if err != nil {
		t.Fatalf("CreateEpic: %v", err)
	}
	if e.ID == "" || e.Status != domain.EpicActive {
		t.Fatalf("unexpected epic: %+v", e)
	}
	if e.Slug != "kueche-renovieren" {
		t.Errorf("Slug = %q, want %q", e.Slug, "kueche-renovieren")
	}
	if e.Archived || e.CompletedAt != nil {
		t.Errorf("fresh epic should be neither archived nor completed: %+v", e)
	}

	got, err := svc.GetEpic(ctx, e.ID)
	if err != nil || got.Title != "Küche renovieren" {
		t.Fatalf("GetEpic = %+v, %v", got, err)
	}

	// A patch touches only what it names; the slug survives a rename so shared
	// links keep working.
	upd, err := svc.UpdateEpic(ctx, e.ID, EpicPatch{Title: ptr("Küche neu")})
	if err != nil {
		t.Fatalf("UpdateEpic: %v", err)
	}
	if upd.Title != "Küche neu" {
		t.Errorf("Title = %q, want %q", upd.Title, "Küche neu")
	}
	if upd.Slug != "kueche-renovieren" {
		t.Errorf("Slug = %q, want it unchanged after a rename", upd.Slug)
	}
	if upd.End.String() != "2027-03-31" {
		t.Errorf("End = %q, want the untouched original", upd.End.String())
	}

	list, err := svc.ListEpics(ctx, storage.EpicFilter{})
	if err != nil || len(list) != 1 {
		t.Fatalf("ListEpics = %v, %v; want 1 entry", list, err)
	}
}

func TestCreateEpicValidation(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	p := newTestProject(t, svc, "Renovierung")

	cases := []struct {
		name string
		in   EpicInput
	}{
		{"no title", EpicInput{ProjectID: p.ID}},
		{"blank title", EpicInput{Title: "   ", ProjectID: p.ID}},
		{"no project", EpicInput{Title: "x"}},
		// The Inbox is the unsorted pile, not an undertaking.
		{"inbox project", EpicInput{Title: "x", ProjectID: domain.InboxID}},
		{"unknown project", EpicInput{Title: "x", ProjectID: "project_nope"}},
		{"end before start", EpicInput{
			Title: "x", ProjectID: p.ID,
			Start: date(t, "2027-01-01"), End: date(t, "2026-01-01"),
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := svc.CreateEpic(ctx, c.in); !errors.Is(err, ErrValidation) {
				t.Errorf("got %v, want ErrValidation", err)
			}
		})
	}

	// Dates are optional on both ends — an initiative may have no fixed frame.
	if _, err := svc.CreateEpic(ctx, EpicInput{Title: "Offen", ProjectID: p.ID}); err != nil {
		t.Errorf("CreateEpic without dates: %v", err)
	}
}

func TestEpicSlugUniqueness(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	p := newTestProject(t, svc, "Renovierung")

	first, err := svc.CreateEpic(ctx, EpicInput{Title: "Küche", ProjectID: p.ID})
	if err != nil {
		t.Fatalf("CreateEpic: %v", err)
	}
	second, err := svc.CreateEpic(ctx, EpicInput{Title: "Küche", ProjectID: p.ID})
	if err != nil {
		t.Fatalf("CreateEpic (duplicate title): %v", err)
	}
	if first.Slug != "kueche" || second.Slug != "kueche-2" {
		t.Errorf("slugs = %q / %q, want kueche / kueche-2", first.Slug, second.Slug)
	}
}

func TestCompleteEpicRequiresAllTasksDone(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	p := newTestProject(t, svc, "Renovierung")
	e, err := svc.CreateEpic(ctx, EpicInput{Title: "Küche", ProjectID: p.ID})
	if err != nil {
		t.Fatalf("CreateEpic: %v", err)
	}
	open, err := svc.CreateTask(ctx, TaskInput{Title: "Fliesen aussuchen", EpicID: e.ID})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	if _, err := svc.CompleteEpic(ctx, e.ID); !errors.Is(err, ErrValidation) {
		t.Fatalf("CompleteEpic with an open task = %v, want ErrValidation", err)
	}

	// A soft-deleted task is not work left to do and must not block either.
	dead, err := svc.CreateTask(ctx, TaskInput{Title: "Verworfen", EpicID: e.ID})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if err := svc.DeleteTask(ctx, dead.ID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	if _, err := svc.CompleteTask(ctx, open.ID); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	done, err := svc.CompleteEpic(ctx, e.ID)
	if err != nil {
		t.Fatalf("CompleteEpic: %v", err)
	}
	if done.Status != domain.EpicCompleted || done.CompletedAt == nil {
		t.Fatalf("epic not completed: %+v", done)
	}

	// Completing twice is a no-op, not an error.
	if again, err := svc.CompleteEpic(ctx, e.ID); err != nil || again.Status != domain.EpicCompleted {
		t.Errorf("second CompleteEpic = %+v, %v; want idempotent", again, err)
	}
}

func TestReopeningTaskReopensEpic(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	p := newTestProject(t, svc, "Renovierung")
	e, _ := svc.CreateEpic(ctx, EpicInput{Title: "Küche", ProjectID: p.ID})
	task, err := svc.CreateTask(ctx, TaskInput{Title: "Fliesen", EpicID: e.ID})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if _, err := svc.CompleteTask(ctx, task.ID); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}
	if _, err := svc.CompleteEpic(ctx, e.ID); err != nil {
		t.Fatalf("CompleteEpic: %v", err)
	}

	// Putting work back on the pile puts the epic back on the pile — a finished
	// epic with an open task under it would be a lie.
	if _, err := svc.ReopenTask(ctx, task.ID); err != nil {
		t.Fatalf("ReopenTask: %v", err)
	}
	got, err := svc.GetEpic(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetEpic: %v", err)
	}
	if got.Status != domain.EpicActive || got.CompletedAt != nil {
		t.Errorf("epic = %+v, want active again with no completed_at", got)
	}
}

func TestTaskFollowsItsEpicsProject(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	pA := newTestProject(t, svc, "Renovierung")
	pB := newTestProject(t, svc, "Garten")
	e, _ := svc.CreateEpic(ctx, EpicInput{Title: "Küche", ProjectID: pA.ID})

	// On create, the epic wins over a submitted project_id.
	task, err := svc.CreateTask(ctx, TaskInput{Title: "Fliesen", ProjectID: pB.ID, EpicID: e.ID})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if task.ProjectID != pA.ID {
		t.Errorf("ProjectID = %q, want the epic's project %q", task.ProjectID, pA.ID)
	}

	// Moving the task to another project takes it out of the epic: the project
	// the user just picked is the later instruction, and an epic in a different
	// project than its task is not a readable state.
	moved, err := svc.UpdateTask(ctx, task.ID, TaskPatch{ProjectID: &pB.ID})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if moved.EpicID != "" || moved.ProjectID != pB.ID {
		t.Errorf("task = %+v, want no epic and project %q", moved, pB.ID)
	}

	// Moving it to the epic's own project is no reason to drop anything — but
	// the link is already gone by now, so re-link first and check.
	if _, err := svc.UpdateTask(ctx, task.ID, TaskPatch{EpicID: &e.ID}); err != nil {
		t.Fatalf("re-link: %v", err)
	}
	same, err := svc.UpdateTask(ctx, task.ID, TaskPatch{ProjectID: &pA.ID})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if same.EpicID != e.ID {
		t.Errorf("EpicID = %q, want the link kept when the project does not change", same.EpicID)
	}
	if _, err := svc.UpdateTask(ctx, task.ID, TaskPatch{EpicID: ptr("")}); err != nil {
		t.Fatalf("clear epic: %v", err)
	}
	if _, err := svc.UpdateTask(ctx, task.ID, TaskPatch{ProjectID: &pB.ID}); err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}

	// Re-linking pulls it back into the epic's project.
	back, err := svc.UpdateTask(ctx, task.ID, TaskPatch{EpicID: &e.ID})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if back.ProjectID != pA.ID {
		t.Errorf("ProjectID = %q, want %q", back.ProjectID, pA.ID)
	}
}

func TestTaskCannotJoinUnavailableEpic(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	p := newTestProject(t, svc, "Renovierung")

	finished, _ := svc.CreateEpic(ctx, EpicInput{Title: "Dach", ProjectID: p.ID})
	if _, err := svc.CompleteEpic(ctx, finished.ID); err != nil {
		t.Fatalf("CompleteEpic: %v", err)
	}
	archived, _ := svc.CreateEpic(ctx, EpicInput{Title: "Keller", ProjectID: p.ID})
	if err := svc.DeleteEpic(ctx, archived.ID); err != nil {
		t.Fatalf("DeleteEpic: %v", err)
	}

	for _, c := range []struct {
		name   string
		epicID string
	}{
		{"completed epic", finished.ID},
		{"archived epic", archived.ID},
		{"unknown epic", "epic_nope"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := svc.CreateTask(ctx, TaskInput{Title: "x", EpicID: c.epicID}); !errors.Is(err, ErrValidation) {
				t.Errorf("got %v, want ErrValidation", err)
			}
		})
	}
}

func TestMovingEpicMovesItsTasks(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	pA := newTestProject(t, svc, "Renovierung")
	pB := newTestProject(t, svc, "Garten")
	e, _ := svc.CreateEpic(ctx, EpicInput{Title: "Küche", ProjectID: pA.ID})
	t1, _ := svc.CreateTask(ctx, TaskInput{Title: "Fliesen", EpicID: e.ID})
	t2, _ := svc.CreateTask(ctx, TaskInput{Title: "Armatur", EpicID: e.ID})
	loose, _ := svc.CreateTask(ctx, TaskInput{Title: "Ohne Epic", ProjectID: pA.ID})

	if _, err := svc.UpdateEpic(ctx, e.ID, EpicPatch{ProjectID: &pB.ID}); err != nil {
		t.Fatalf("UpdateEpic: %v", err)
	}
	for _, id := range []string{t1.ID, t2.ID} {
		got, err := svc.GetTask(ctx, id)
		if err != nil {
			t.Fatalf("GetTask: %v", err)
		}
		if got.ProjectID != pB.ID {
			t.Errorf("task %q project = %q, want %q", got.Title, got.ProjectID, pB.ID)
		}
	}
	// A task that never belonged to the epic stays where it was.
	if got, _ := svc.GetTask(ctx, loose.ID); got.ProjectID != pA.ID {
		t.Errorf("unrelated task moved to %q, want %q", got.ProjectID, pA.ID)
	}
}

func TestDeleteEpicArchives(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	p := newTestProject(t, svc, "Renovierung")
	e, _ := svc.CreateEpic(ctx, EpicInput{Title: "Küche", ProjectID: p.ID})
	task, _ := svc.CreateTask(ctx, TaskInput{Title: "Fliesen", EpicID: e.ID})

	if err := svc.DeleteEpic(ctx, e.ID); err != nil {
		t.Fatalf("DeleteEpic: %v", err)
	}
	// Archiving twice is a no-op.
	if err := svc.DeleteEpic(ctx, e.ID); err != nil {
		t.Fatalf("second DeleteEpic: %v", err)
	}

	got, err := svc.GetEpic(ctx, e.ID)
	if err != nil || !got.Archived {
		t.Fatalf("GetEpic = %+v, %v; want an archived epic that still exists", got, err)
	}
	// The task keeps its link so its label still resolves.
	if kept, _ := svc.GetTask(ctx, task.ID); kept.EpicID != e.ID {
		t.Errorf("task lost its epic link: %+v", kept)
	}

	visible, err := svc.ListEpics(ctx, storage.EpicFilter{})
	if err != nil || len(visible) != 0 {
		t.Errorf("ListEpics = %v, %v; want the archived epic hidden", visible, err)
	}
	all, err := svc.ListEpics(ctx, storage.EpicFilter{IncludeArchived: true})
	if err != nil || len(all) != 1 {
		t.Errorf("ListEpics(include_archived) = %v, %v; want 1", all, err)
	}

	// Un-archiving brings it back.
	if _, err := svc.UpdateEpic(ctx, e.ID, EpicPatch{Archived: ptrBool(false)}); err != nil {
		t.Fatalf("un-archive: %v", err)
	}
	if back, _ := svc.ListEpics(ctx, storage.EpicFilter{}); len(back) != 1 {
		t.Errorf("epic did not come back: %v", back)
	}
}

func TestListEpicsFilters(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	pA := newTestProject(t, svc, "Renovierung")
	pB := newTestProject(t, svc, "Garten")
	svc.CreateEpic(ctx, EpicInput{Title: "Küche", ProjectID: pA.ID})
	dach, _ := svc.CreateEpic(ctx, EpicInput{Title: "Dach", ProjectID: pA.ID})
	svc.CreateEpic(ctx, EpicInput{Title: "Teich", ProjectID: pB.ID})
	if _, err := svc.CompleteEpic(ctx, dach.ID); err != nil {
		t.Fatalf("CompleteEpic: %v", err)
	}

	cases := []struct {
		name string
		f    storage.EpicFilter
		want int
	}{
		{"all", storage.EpicFilter{}, 3},
		{"one project", storage.EpicFilter{ProjectID: pA.ID}, 2},
		{"running only", storage.EpicFilter{Status: domain.EpicActive}, 2},
		{"finished only", storage.EpicFilter{Status: domain.EpicCompleted}, 1},
		{"running in one project", storage.EpicFilter{ProjectID: pA.ID, Status: domain.EpicActive}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := svc.ListEpics(ctx, c.f)
			if err != nil {
				t.Fatalf("ListEpics: %v", err)
			}
			if len(got) != c.want {
				t.Errorf("got %d epics, want %d", len(got), c.want)
			}
		})
	}
}

func TestEpicTaskCounts(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	p := newTestProject(t, svc, "Renovierung")
	e, _ := svc.CreateEpic(ctx, EpicInput{Title: "Küche", ProjectID: p.ID})
	svc.CreateTask(ctx, TaskInput{Title: "offen", EpicID: e.ID})
	done, _ := svc.CreateTask(ctx, TaskInput{Title: "fertig", EpicID: e.ID})
	svc.CreateTask(ctx, TaskInput{Title: "ohne Epic"})
	if _, err := svc.CompleteTask(ctx, done.ID); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	counts, err := svc.EpicTaskCounts(ctx)
	if err != nil {
		t.Fatalf("EpicTaskCounts: %v", err)
	}
	if counts[e.ID] != (domain.EpicCounts{Open: 1, Done: 1}) {
		t.Errorf("counts = %+v, want {Open:1 Done:1}", counts[e.ID])
	}
}

func TestRecurringTaskKeepsItsEpic(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	p := newTestProject(t, svc, "Renovierung")
	e, _ := svc.CreateEpic(ctx, EpicInput{Title: "Küche", ProjectID: p.ID})
	task, err := svc.CreateTask(ctx, TaskInput{
		Title: "Baustelle fegen", EpicID: e.ID,
		VisibleFrom: date(t, "2026-09-07"), RecurrenceRule: "FREQ=WEEKLY",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if _, err := svc.CompleteTask(ctx, task.ID); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}
	tasks, err := svc.ListTasks(ctx, storage.TaskFilter{EpicID: e.ID, Status: domain.StatusOpen})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("got %d open tasks, want the spawned successor to carry the epic", len(tasks))
	}
}

func ptr(s string) *string { return &s }
func ptrBool(b bool) *bool { return &b }
