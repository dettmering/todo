package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/events"
	"github.com/till/todo/internal/storage"
)

type EpicInput struct {
	Title       string
	Description string
	ProjectID   string
	Start       domain.LocalDate
	End         domain.LocalDate
	SortOrder   int
}

type EpicPatch struct {
	Title       *string
	Description *string
	ProjectID   *string
	Start       *domain.LocalDate
	End         *domain.LocalDate
	SortOrder   *int
	Archived    *bool
}

// ensureUniqueEpicSlug returns base if no other epic owns it, otherwise base-2,
// base-3, … The optional ownID is excluded so an epic keeps its own slug.
// Epic and project slugs live in separate URL namespaces (/epics/… vs
// /projects/…) and may therefore collide without harm.
func ensureUniqueEpicSlug(ctx context.Context, repo storage.EpicRepo, base, ownID string) (string, error) {
	epics, err := repo.List(ctx, storage.EpicFilter{IncludeArchived: true})
	if err != nil {
		return "", err
	}
	taken := make(map[string]bool, len(epics))
	for _, e := range epics {
		if e.ID == ownID || e.Slug == "" {
			continue
		}
		taken[e.Slug] = true
	}
	candidate := base
	for n := 2; taken[candidate]; n++ {
		candidate = fmt.Sprintf("%s-%d", base, n)
	}
	return candidate, nil
}

// validateEpicProject checks that an epic's target project exists and is a real
// project. The Inbox is deliberately excluded: it is the place tasks land
// before they are sorted, not an undertaking one can run for months.
func (s *Service) validateEpicProject(ctx context.Context, projectID string) error {
	if projectID == "" {
		return fmt.Errorf("%w: project_id is required", ErrValidation)
	}
	if projectID == domain.InboxID {
		return fmt.Errorf("%w: inbox cannot hold epics", ErrValidation)
	}
	if _, err := s.projects.Get(ctx, projectID); err != nil {
		return fmt.Errorf("%w: project %q not found", ErrValidation, projectID)
	}
	return nil
}

func (s *Service) CreateEpic(ctx context.Context, in EpicInput) (*domain.Epic, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrValidation)
	}
	if err := s.validateEpicProject(ctx, in.ProjectID); err != nil {
		return nil, err
	}
	if !domain.ValidEpicRange(in.Start, in.End) {
		return nil, fmt.Errorf("%w: end date cannot be before start date", ErrValidation)
	}
	slug, err := ensureUniqueEpicSlug(ctx, s.epics, domain.Slugify(in.Title), "")
	if err != nil {
		return nil, err
	}
	now := s.clock()
	e := &domain.Epic{
		ID:          newID("epic"),
		Title:       in.Title,
		Slug:        slug,
		Description: in.Description,
		ProjectID:   in.ProjectID,
		Status:      domain.EpicActive,
		Start:       in.Start,
		End:         in.End,
		SortOrder:   in.SortOrder,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.epics.Create(ctx, e); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, events.EpicCreated, e)
	return e, nil
}

// UpdateEpic applies a partial patch. The status is deliberately not patchable
// here — it moves only through CompleteEpic/ReopenEpic, so the "all tasks done"
// rule has exactly one place to live.
//
// Moving an epic to another project drags its tasks along. Leaving them behind
// would break the invariant the whole feature rests on: a task's epic always
// sits in the task's own project.
func (s *Service) UpdateEpic(ctx context.Context, id string, p EpicPatch) (*domain.Epic, error) {
	e, err := s.epics.Get(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	oldProjectID := e.ProjectID

	if p.Title != nil {
		title := strings.TrimSpace(*p.Title)
		if title == "" {
			return nil, fmt.Errorf("%w: title cannot be empty", ErrValidation)
		}
		e.Title = title
	}
	// The slug is derived once and then stays put, like a project's: renaming
	// an epic must not break links people have already shared.
	if e.Slug == "" {
		slug, err := ensureUniqueEpicSlug(ctx, s.epics, domain.Slugify(e.Title), e.ID)
		if err != nil {
			return nil, err
		}
		e.Slug = slug
	}
	if p.Description != nil {
		e.Description = *p.Description
	}
	if p.ProjectID != nil {
		if err := s.validateEpicProject(ctx, *p.ProjectID); err != nil {
			return nil, err
		}
		e.ProjectID = *p.ProjectID
	}
	if p.Start != nil {
		e.Start = *p.Start
	}
	if p.End != nil {
		e.End = *p.End
	}
	if p.SortOrder != nil {
		e.SortOrder = *p.SortOrder
	}
	if p.Archived != nil {
		e.Archived = *p.Archived
	}
	if !domain.ValidEpicRange(e.Start, e.End) {
		return nil, fmt.Errorf("%w: end date cannot be before start date", ErrValidation)
	}

	now := s.clock()
	e.UpdatedAt = now
	if err := s.epics.Update(ctx, e); err != nil {
		return nil, mapErr(err)
	}
	if e.ProjectID != oldProjectID {
		s.moveEpicTasks(ctx, e, oldProjectID, now)
	}
	s.bus.Publish(ctx, events.EpicUpdated, e)
	return e, nil
}

// moveEpicTasks re-homes every task of an epic after the epic changed project.
// Failures are logged rather than propagated: the epic has already moved, and
// refusing the whole request afterwards would leave the caller with no way to
// tell what happened.
func (s *Service) moveEpicTasks(ctx context.Context, e *domain.Epic, oldProjectID string, now time.Time) {
	tasks, err := s.tasks.List(ctx, storage.TaskFilter{EpicID: e.ID})
	if err != nil {
		slog.Error("move epic tasks: list", "epic_id", e.ID, "err", err)
		return
	}
	oldName := s.projectNameOrID(ctx, oldProjectID)
	newName := s.projectNameOrID(ctx, e.ProjectID)
	for _, t := range tasks {
		if t.ProjectID == e.ProjectID {
			continue
		}
		t.ProjectID = e.ProjectID
		t.UpdatedAt = now
		if err := s.tasks.Update(ctx, t); err != nil {
			slog.Error("move epic tasks: update", "task_id", t.ID, "err", err)
			continue
		}
		s.writeAudit(ctx, t, domain.AuditProjectChanged, oldName, newName, now)
		s.bus.Publish(ctx, events.TaskUpdated, t)
	}
}

// CompleteEpic closes an epic. It refuses while any task of the epic is still
// open — that is the whole point of the status. Completed and soft-deleted
// tasks do not block.
func (s *Service) CompleteEpic(ctx context.Context, id string) (*domain.Epic, error) {
	e, err := s.epics.Get(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	if e.Status == domain.EpicCompleted {
		return e, nil
	}
	open, err := s.tasks.List(ctx, storage.TaskFilter{EpicID: id, Status: domain.StatusOpen})
	if err != nil {
		return nil, err
	}
	if n := len(open); n > 0 {
		return nil, fmt.Errorf("%w: epic still has %d open task(s)", ErrValidation, n)
	}
	now := s.clock()
	e.Status = domain.EpicCompleted
	e.CompletedAt = &now
	e.UpdatedAt = now
	if err := s.epics.Update(ctx, e); err != nil {
		return nil, mapErr(err)
	}
	s.bus.Publish(ctx, events.EpicCompleted, e)
	return e, nil
}

// ReopenEpic reverts a completed epic to running. It is a no-op if the epic is
// already active. Also called implicitly when a task of a completed epic is
// reopened (see ReopenTask).
func (s *Service) ReopenEpic(ctx context.Context, id string) (*domain.Epic, error) {
	e, err := s.epics.Get(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	if e.Status == domain.EpicActive {
		return e, nil
	}
	if err := s.reopenEpicEntity(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// reopenEpicEntity flips an already-loaded epic back to active and publishes
// the update. Shared by ReopenEpic and the implicit reopen in ReopenTask.
func (s *Service) reopenEpicEntity(ctx context.Context, e *domain.Epic) error {
	now := s.clock()
	e.Status = domain.EpicActive
	e.CompletedAt = nil
	e.UpdatedAt = now
	if err := s.epics.Update(ctx, e); err != nil {
		return mapErr(err)
	}
	s.bus.Publish(ctx, events.EpicUpdated, e)
	return nil
}

// DeleteEpic hides the epic by archiving it, mirroring DeleteProject. Tasks
// keep their epic_id so historical labels still resolve, and no task ever loses
// data because an epic was tidied away.
func (s *Service) DeleteEpic(ctx context.Context, id string) error {
	e, err := s.epics.Get(ctx, id)
	if err != nil {
		return mapErr(err)
	}
	if e.Archived {
		return nil
	}
	e.Archived = true
	e.UpdatedAt = s.clock()
	if err := s.epics.Update(ctx, e); err != nil {
		return mapErr(err)
	}
	s.bus.Publish(ctx, events.EpicDeleted, e)
	return nil
}

func (s *Service) ListEpics(ctx context.Context, f storage.EpicFilter) ([]*domain.Epic, error) {
	return s.epics.List(ctx, f)
}

func (s *Service) GetEpic(ctx context.Context, id string) (*domain.Epic, error) {
	e, err := s.epics.Get(ctx, id)
	return e, mapErr(err)
}

// EpicTaskCounts returns an epic_id -> counts map covering every epic in one
// query. It used to load the whole task table and count in Go, which made
// GET /api/epics the slowest request of a view switch even though it answers
// with a handful of numbers; the store counts now.
func (s *Service) EpicTaskCounts(ctx context.Context) (map[string]domain.EpicCounts, error) {
	counts, err := s.tasks.CountByEpic(ctx)
	return counts, mapErr(err)
}

// OpenTaskCounts returns the tallies the sidebar shows — open tasks per project
// and per epic, plus how many are visible on `today`. The frontend used to
// download every open task and count them in the browser; the numbers are the
// only part it ever showed.
func (s *Service) OpenTaskCounts(ctx context.Context, today domain.LocalDate) (domain.OpenTaskCounts, error) {
	counts, err := s.tasks.CountOpen(ctx, today)
	return counts, mapErr(err)
}

// resolveEpic loads the epic a task wants to be linked to and rejects the ones
// that must not take new work: an archived epic is gone from every view, and a
// completed one would have its "everything is done" promise broken on the spot.
func (s *Service) resolveEpic(ctx context.Context, epicID string) (*domain.Epic, error) {
	e, err := s.epics.Get(ctx, epicID)
	if err != nil {
		return nil, fmt.Errorf("%w: epic %q not found", ErrValidation, epicID)
	}
	if !e.Assignable() {
		return nil, fmt.Errorf("%w: epic %q is %s and takes no tasks", ErrValidation, e.Title, epicStateWord(e))
	}
	return e, nil
}

func epicStateWord(e *domain.Epic) string {
	if e.Archived {
		return "archived"
	}
	return "completed"
}

// epicTitleOrID resolves an epic title for audit display, falling back to the
// raw id — better than dropping the entry. Empty id means "no epic".
func (s *Service) epicTitleOrID(ctx context.Context, id string) string {
	if id == "" {
		return ""
	}
	e, err := s.epics.Get(ctx, id)
	if err != nil || e == nil {
		return id
	}
	return e.Title
}
