// Package service contains the application's business logic, sitting between
// the HTTP layer and the storage layer. It owns ID generation, timestamps,
// validation, and event emission.
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/events"
	"github.com/till/todo/internal/recurrence"
	"github.com/till/todo/internal/storage"
)

var (
	ErrValidation = errors.New("validation error")
	ErrNotFound   = errors.New("not found")
)

type Service struct {
	tasks    storage.TaskRepo
	projects storage.ProjectRepo
	audit    storage.AuditRepo
	comments storage.CommentRepo
	absences storage.AbsenceRepo
	epics    storage.EpicRepo
	bus      *events.Bus
	clock    func() time.Time
}

func New(tasks storage.TaskRepo, projects storage.ProjectRepo, audit storage.AuditRepo, comments storage.CommentRepo, absences storage.AbsenceRepo, epics storage.EpicRepo, bus *events.Bus) *Service {
	return &Service{
		tasks:    tasks,
		projects: projects,
		audit:    audit,
		comments: comments,
		absences: absences,
		epics:    epics,
		bus:      bus,
		clock:    func() time.Time { return time.Now().UTC() },
	}
}

// SetClock is for tests.
func (s *Service) SetClock(fn func() time.Time) { s.clock = fn }

func newID(prefix string) string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return prefix + "_" + hex.EncodeToString(b[:])
}

// ---------- Tasks ----------

type TaskInput struct {
	Title            string
	Description      string
	ProjectID        string
	EpicID           string
	Priority         domain.Priority
	VisibleFrom      domain.LocalDate
	Deadline         domain.LocalDate
	RecurrenceRule   string
	RecurrenceMode   domain.RecurrenceMode
	RecurrenceAnchor domain.LocalDate
}

// recurrenceFields bundles the three fields that only make sense together.
type recurrenceFields struct {
	Rule   string
	Mode   domain.RecurrenceMode
	Anchor domain.LocalDate
}

// normalizeRecurrence validates rule, mode and series anchor as one unit. An
// empty rule clears the other two — a one-off task has no series to speak of —
// and an empty mode means RecurFixed, which keeps pre-existing API clients
// working unchanged. fallbackAnchor seeds the series date when a task becomes
// recurring without one being given (normally the task's visible_from).
func normalizeRecurrence(in recurrenceFields, fallbackAnchor domain.LocalDate) (recurrenceFields, error) {
	if in.Rule == "" {
		return recurrenceFields{}, nil
	}
	parsed, err := recurrence.Parse(in.Rule)
	if err != nil {
		return recurrenceFields{}, fmt.Errorf("%w: recurrence_rule: %s", ErrValidation, err.Error())
	}
	if in.Mode == "" {
		in.Mode = domain.RecurFixed
	}
	if !in.Mode.Valid() {
		return recurrenceFields{}, fmt.Errorf("%w: recurrence_mode must be %q or %q",
			ErrValidation, domain.RecurFixed, domain.RecurAfterDone)
	}
	if in.Mode == domain.RecurAfterDone && parsed.HasCalendarAnchor() {
		return recurrenceFields{}, fmt.Errorf("%w: recurrence_mode %q allows only FREQ and INTERVAL, not BYDAY/BYMONTHDAY/BYMONTH",
			ErrValidation, domain.RecurAfterDone)
	}
	out := recurrenceFields{Rule: parsed.Canonical(), Mode: in.Mode, Anchor: in.Anchor}
	if in.Mode == domain.RecurAfterDone {
		// This mode anchors on the completion day, so a stored series date
		// would only be misleading.
		out.Anchor = domain.LocalDate{}
	} else if out.Anchor.IsZero() {
		out.Anchor = fallbackAnchor
	}
	return out, nil
}

func (s *Service) CreateTask(ctx context.Context, in TaskInput) (*domain.Task, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrValidation)
	}
	// An epic decides the project: a task linked to an initiative always lives
	// in that initiative's project, whatever project_id the caller submitted.
	if in.EpicID != "" {
		e, err := s.resolveEpic(ctx, in.EpicID)
		if err != nil {
			return nil, err
		}
		in.ProjectID = e.ProjectID
	}
	if in.ProjectID == "" {
		in.ProjectID = domain.InboxID
	}
	if _, err := s.projects.Get(ctx, in.ProjectID); err != nil {
		return nil, fmt.Errorf("%w: project %q not found", ErrValidation, in.ProjectID)
	}
	if in.Priority == 0 {
		in.Priority = domain.P4
	}
	if !in.Priority.Valid() {
		return nil, fmt.Errorf("%w: priority must be 1..4", ErrValidation)
	}
	rec, err := normalizeRecurrence(recurrenceFields{
		Rule: in.RecurrenceRule, Mode: in.RecurrenceMode, Anchor: in.RecurrenceAnchor,
	}, in.VisibleFrom)
	if err != nil {
		return nil, err
	}

	now := s.clock()
	t := &domain.Task{
		ID:               newID("task"),
		Title:            in.Title,
		Description:      in.Description,
		ProjectID:        in.ProjectID,
		EpicID:           in.EpicID,
		Status:           domain.StatusOpen,
		Priority:         in.Priority,
		VisibleFrom:      in.VisibleFrom,
		Deadline:         in.Deadline,
		RecurrenceRule:   rec.Rule,
		RecurrenceMode:   rec.Mode,
		RecurrenceAnchor: rec.Anchor,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.tasks.Create(ctx, t); err != nil {
		return nil, err
	}
	s.writeAudit(ctx, t, domain.AuditCreated, "", "", now)
	s.bus.Publish(ctx, events.TaskCreated, t)
	return t, nil
}

// TaskPatch carries optional updates; nil pointer = leave field unchanged.
type TaskPatch struct {
	Title            *string
	Description      *string
	ProjectID        *string
	EpicID           *string
	Priority         *domain.Priority
	VisibleFrom      *domain.LocalDate
	Deadline         *domain.LocalDate
	RecurrenceRule   *string
	RecurrenceMode   *domain.RecurrenceMode
	RecurrenceAnchor *domain.LocalDate
}

func (s *Service) UpdateTask(ctx context.Context, id string, p TaskPatch) (*domain.Task, error) {
	t, err := s.tasks.Get(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	oldVisibleFrom := t.VisibleFrom
	oldDeadline := t.Deadline
	oldProjectID := t.ProjectID
	oldEpicID := t.EpicID
	oldPriority := t.Priority

	if p.Title != nil {
		title := strings.TrimSpace(*p.Title)
		if title == "" {
			return nil, fmt.Errorf("%w: title cannot be empty", ErrValidation)
		}
		t.Title = title
	}
	if p.Description != nil {
		t.Description = *p.Description
	}
	if p.ProjectID != nil {
		if _, err := s.projects.Get(ctx, *p.ProjectID); err != nil {
			return nil, fmt.Errorf("%w: project %q not found", ErrValidation, *p.ProjectID)
		}
		t.ProjectID = *p.ProjectID
	}
	// Epic and project are applied as a pair, epic last, and the later
	// instruction wins. Naming an epic drags the task into that epic's project,
	// so a project_id in the same patch loses. Naming only a project moves the
	// task and drops the epic link, because an epic in a different project than
	// its task is not a state the rest of the code can read.
	if p.EpicID != nil {
		if *p.EpicID == "" {
			t.EpicID = ""
		} else {
			e, err := s.resolveEpic(ctx, *p.EpicID)
			if err != nil {
				return nil, err
			}
			t.EpicID = e.ID
			t.ProjectID = e.ProjectID
		}
	} else if p.ProjectID != nil && t.EpicID != "" {
		if e, err := s.epics.Get(ctx, t.EpicID); err == nil && e.ProjectID != t.ProjectID {
			t.EpicID = ""
		}
	}
	if p.Priority != nil {
		if !p.Priority.Valid() {
			return nil, fmt.Errorf("%w: priority must be 1..4", ErrValidation)
		}
		t.Priority = *p.Priority
	}
	if p.VisibleFrom != nil {
		t.VisibleFrom = *p.VisibleFrom
	}
	if p.Deadline != nil {
		t.Deadline = *p.Deadline
	}
	// Rule, mode and series anchor are validated as one unit even when only one
	// of them is patched — switching an existing "jeden Montag" task to "ab
	// Erledigung" has to fail just as loudly as submitting both at once.
	//
	// Note what is deliberately absent: patching VisibleFrom does *not* touch
	// RecurrenceAnchor. Postponing one occurrence ("Geburtstag auf Mittwoch")
	// must leave the series where it is; moving the series is an explicit act
	// via RecurrenceAnchor.
	if p.RecurrenceRule != nil || p.RecurrenceMode != nil || p.RecurrenceAnchor != nil {
		cur := recurrenceFields{Rule: t.RecurrenceRule, Mode: t.RecurrenceMode, Anchor: t.RecurrenceAnchor}
		if p.RecurrenceRule != nil {
			cur.Rule = *p.RecurrenceRule
		}
		if p.RecurrenceMode != nil {
			cur.Mode = *p.RecurrenceMode
		}
		if p.RecurrenceAnchor != nil {
			cur.Anchor = *p.RecurrenceAnchor
		}
		rec, err := normalizeRecurrence(cur, t.VisibleFrom)
		if err != nil {
			return nil, err
		}
		t.RecurrenceRule, t.RecurrenceMode, t.RecurrenceAnchor = rec.Rule, rec.Mode, rec.Anchor
	}
	now := s.clock()
	t.UpdatedAt = now
	if err := s.tasks.Update(ctx, t); err != nil {
		return nil, mapErr(err)
	}
	rescheduled := t.VisibleFrom != oldVisibleFrom
	deadlineChanged := t.Deadline != oldDeadline
	projectChanged := t.ProjectID != oldProjectID
	epicChanged := t.EpicID != oldEpicID
	priorityChanged := t.Priority != oldPriority

	if rescheduled {
		s.auditDateChange(ctx, t, domain.AuditVisibleFromSet, domain.AuditVisibleFromCleared, domain.AuditVisibleFromChanged, oldVisibleFrom, t.VisibleFrom, now)
	}
	if deadlineChanged {
		s.auditDateChange(ctx, t, domain.AuditDeadlineSet, domain.AuditDeadlineCleared, domain.AuditDeadlineChanged, oldDeadline, t.Deadline, now)
	}
	if projectChanged {
		s.writeAudit(ctx, t, domain.AuditProjectChanged, s.projectNameOrID(ctx, oldProjectID), s.projectNameOrID(ctx, t.ProjectID), now)
	}
	if epicChanged {
		s.writeAudit(ctx, t, domain.AuditEpicChanged, s.epicTitleOrID(ctx, oldEpicID), s.epicTitleOrID(ctx, t.EpicID), now)
	}
	if priorityChanged {
		s.writeAudit(ctx, t, domain.AuditPriorityChanged, priorityLabel(oldPriority), priorityLabel(t.Priority), now)
	}

	s.bus.Publish(ctx, events.TaskUpdated, t)
	if rescheduled {
		s.bus.Publish(ctx, events.TaskRescheduled, t)
	}
	if deadlineChanged {
		s.bus.Publish(ctx, events.TaskDeadlineChanged, t)
	}
	return t, nil
}

// ReopenTask reverts a completed task to open and clears the completion timestamp.
// It is a no-op if the task is already open.
func (s *Service) ReopenTask(ctx context.Context, id string) (*domain.Task, error) {
	t, err := s.tasks.Get(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	if t.Status == domain.StatusOpen {
		return t, nil
	}
	t.Status = domain.StatusOpen
	t.CompletedAt = nil
	now := s.clock()
	t.UpdatedAt = now
	if err := s.tasks.Update(ctx, t); err != nil {
		return nil, mapErr(err)
	}
	s.writeAudit(ctx, t, domain.AuditReopened, "", "", now)
	s.bus.Publish(ctx, events.TaskUpdated, t)
	// An epic is only "beendet" while nothing under it is open. Putting work
	// back on the pile therefore puts the epic back on the pile too — the
	// alternative, refusing the reopen, would make a finished epic a trap.
	if t.EpicID != "" {
		if e, err := s.epics.Get(ctx, t.EpicID); err == nil && e.Status == domain.EpicCompleted {
			if err := s.reopenEpicEntity(ctx, e); err != nil {
				slog.Error("reopen epic after task reopen", "epic_id", e.ID, "err", err)
			}
		}
	}
	return t, nil
}

func (s *Service) CompleteTask(ctx context.Context, id string) (*domain.Task, error) {
	t, err := s.tasks.Get(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	if t.Status == domain.StatusCompleted {
		return t, nil
	}
	now := s.clock()
	t.Status = domain.StatusCompleted
	t.CompletedAt = &now
	t.UpdatedAt = now
	if err := s.tasks.Update(ctx, t); err != nil {
		return nil, mapErr(err)
	}
	s.writeAudit(ctx, t, domain.AuditCompleted, "", "", now)
	s.bus.Publish(ctx, events.TaskCompleted, t)

	// Modell B (Lastenheft 4.6): when a recurring task is completed, spawn a
	// fresh open instance for the next occurrence. The new task inherits all
	// scheduling-relevant fields; the next visible_from is rolled forward to
	// today if the original was already in the past.
	if t.RecurrenceRule != "" {
		if next, err := s.spawnNextInstance(ctx, t, now); err == nil && next != nil {
			s.writeAudit(ctx, next, domain.AuditCreated, "", "", now)
			s.bus.Publish(ctx, events.TaskCreated, next)
			s.bus.Publish(ctx, events.TaskRecurrenceGen, next)
		}
	}
	return t, nil
}

// spawnNextInstance creates the follow-up task for a recurring completion.
// Errors are returned but not propagated up: the completion has already been
// persisted, and we don't want a bad RRULE (validated at create/update) to
// retroactively fail the user's "done" click.
func (s *Service) spawnNextInstance(ctx context.Context, prev *domain.Task, now time.Time) (*domain.Task, error) {
	rule, err := recurrence.Parse(prev.RecurrenceRule)
	if err != nil {
		return nil, err
	}
	today := domain.DateOf(now)
	var nextVisible, window domain.LocalDate
	if prev.RecurrenceMode == domain.RecurAfterDone {
		// The interval restarts now: anchor on the completion day and take a
		// single step. No roll-forward — today is already "now", so the result
		// can never be in the past.
		window = prev.VisibleFrom
		nextVisible, err = rule.Next(today, domain.LocalDate{})
	} else {
		// Fixed mode steps from the series anchor, never from visible_from:
		// the latter may have been dragged around to postpone this one
		// occurrence, which must not move the series. Missed occurrences are
		// skipped so the result is not already in the past.
		window = prev.RecurrenceAnchor
		if window.IsZero() {
			window = prev.VisibleFrom // pre-anchor rows and API clients that omit it
		}
		nextVisible, err = rule.Next(window, today)
	}
	if err != nil {
		return nil, err
	}
	// Deadline rolls forward by the same delta as the date it was planned
	// against, so a task "appear Mon, due Fri" recurring weekly stays a 4-day
	// window even if this occurrence was postponed.
	var nextDeadline domain.LocalDate
	if !prev.Deadline.IsZero() {
		if window.IsZero() {
			window = today
		}
		nextDeadline = shiftDeadline(prev.Deadline, window, nextVisible)
	}
	t := &domain.Task{
		ID:             newID("task"),
		Title:          prev.Title,
		Description:    prev.Description,
		ProjectID:      prev.ProjectID,
		EpicID:         prev.EpicID,
		Status:         domain.StatusOpen,
		Priority:       prev.Priority,
		VisibleFrom:    nextVisible,
		Deadline:       nextDeadline,
		RecurrenceRule: prev.RecurrenceRule,
		RecurrenceMode: prev.RecurrenceMode,
		// The new occurrence starts life un-postponed, so its series date and
		// its visible_from coincide again.
		RecurrenceAnchor: anchorFor(prev.RecurrenceMode, nextVisible),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.tasks.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// anchorFor returns the series date to store for a freshly spawned occurrence.
// RecurAfterDone tasks re-anchor on every completion and keep none.
func anchorFor(mode domain.RecurrenceMode, next domain.LocalDate) domain.LocalDate {
	if mode == domain.RecurAfterDone {
		return domain.LocalDate{}
	}
	return next
}

func shiftDeadline(deadline, fromBase, toBase domain.LocalDate) domain.LocalDate {
	from := time.Date(fromBase.Year, fromBase.Month, fromBase.Day, 0, 0, 0, 0, time.UTC)
	to := time.Date(toBase.Year, toBase.Month, toBase.Day, 0, 0, 0, 0, time.UTC)
	dl := time.Date(deadline.Year, deadline.Month, deadline.Day, 0, 0, 0, 0, time.UTC)
	delta := to.Sub(from)
	return domain.DateOf(dl.Add(delta))
}

func (s *Service) DeleteTask(ctx context.Context, id string) error {
	t, err := s.tasks.Get(ctx, id)
	if err != nil {
		return mapErr(err)
	}
	if err := s.tasks.Delete(ctx, id); err != nil {
		return mapErr(err)
	}
	s.writeAudit(ctx, t, domain.AuditDeleted, "", "", s.clock())
	s.bus.Publish(ctx, events.TaskDeleted, t)
	return nil
}

func (s *Service) GetTask(ctx context.Context, id string) (*domain.Task, error) {
	t, err := s.tasks.Get(ctx, id)
	return t, mapErr(err)
}

func (s *Service) ListTasks(ctx context.Context, f storage.TaskFilter) ([]*domain.Task, error) {
	if f.Today.IsZero() {
		f.Today = domain.DateOf(s.clock())
	}
	return s.tasks.List(ctx, f)
}

// ---------- Projects ----------

type ProjectInput struct {
	Name        string
	Description string
	SortOrder   int
	Color       string
}

func (s *Service) CreateProject(ctx context.Context, in ProjectInput) (*domain.Project, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	if in.Color != "" && !isValidHexColor(in.Color) {
		return nil, fmt.Errorf("%w: color must be a hex like #1a2b3c", ErrValidation)
	}
	slug, err := ensureUniqueSlug(ctx, s.projects, domain.Slugify(in.Name), "")
	if err != nil {
		return nil, err
	}
	now := s.clock()
	p := &domain.Project{
		ID:          newID("project"),
		Name:        in.Name,
		Slug:        slug,
		Description: in.Description,
		SortOrder:   in.SortOrder,
		Color:       in.Color,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.projects.Create(ctx, p); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, events.ProjectCreated, p)
	return p, nil
}

type ProjectPatch struct {
	Name        *string
	Description *string
	SortOrder   *int
	Archived    *bool
	Color       *string
}

func isValidHexColor(s string) bool {
	if len(s) != 4 && len(s) != 7 {
		return false
	}
	if s[0] != '#' {
		return false
	}
	for _, c := range s[1:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func (s *Service) UpdateProject(ctx context.Context, id string, p ProjectPatch) (*domain.Project, error) {
	pr, err := s.projects.Get(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	if p.Name != nil {
		name := strings.TrimSpace(*p.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: name cannot be empty", ErrValidation)
		}
		pr.Name = name
	}
	if pr.Slug == "" {
		slug, err := ensureUniqueSlug(ctx, s.projects, domain.Slugify(pr.Name), pr.ID)
		if err != nil {
			return nil, err
		}
		pr.Slug = slug
	}
	if p.Description != nil {
		pr.Description = *p.Description
	}
	if p.SortOrder != nil {
		pr.SortOrder = *p.SortOrder
	}
	if p.Archived != nil {
		pr.Archived = *p.Archived
	}
	if p.Color != nil {
		if *p.Color != "" && !isValidHexColor(*p.Color) {
			return nil, fmt.Errorf("%w: color must be a hex like #1a2b3c", ErrValidation)
		}
		pr.Color = *p.Color
	}
	pr.UpdatedAt = s.clock()
	if err := s.projects.Update(ctx, pr); err != nil {
		return nil, mapErr(err)
	}
	s.bus.Publish(ctx, events.ProjectUpdated, pr)
	return pr, nil
}

// DeleteProject hides the project by archiving it. Tasks remain in the database
// untouched so historical data and event/webhook consistency are preserved.
func (s *Service) DeleteProject(ctx context.Context, id string) error {
	if id == domain.InboxID {
		return fmt.Errorf("%w: inbox cannot be deleted", ErrValidation)
	}
	pr, err := s.projects.Get(ctx, id)
	if err != nil {
		return mapErr(err)
	}
	if pr.Archived {
		return nil
	}
	pr.Archived = true
	pr.UpdatedAt = s.clock()
	if err := s.projects.Update(ctx, pr); err != nil {
		return mapErr(err)
	}
	s.bus.Publish(ctx, events.ProjectDeleted, pr)
	return nil
}

func (s *Service) ListProjects(ctx context.Context, includeArchived bool) ([]*domain.Project, error) {
	return s.projects.List(ctx, includeArchived)
}

func (s *Service) GetProject(ctx context.Context, id string) (*domain.Project, error) {
	pr, err := s.projects.Get(ctx, id)
	return pr, mapErr(err)
}

// ---------- Audit log ----------

func (s *Service) writeAudit(ctx context.Context, t *domain.Task, action domain.AuditAction, oldVal, newVal string, at time.Time) {
	if s.audit == nil {
		return
	}
	e := &domain.AuditEntry{
		ID:        newID("audit"),
		TaskID:    t.ID,
		TaskTitle: t.Title,
		Action:    action,
		OldValue:  oldVal,
		NewValue:  newVal,
		At:        at,
	}
	if err := s.audit.Append(ctx, e); err != nil {
		slog.Error("audit append", "task_id", t.ID, "action", action, "err", err)
	}
}

// auditDateChange picks the right action (set/cleared/changed) based on whether
// the date was added, removed, or moved.
func (s *Service) auditDateChange(ctx context.Context, t *domain.Task, setA, clearedA, changedA domain.AuditAction, oldD, newD domain.LocalDate, at time.Time) {
	switch {
	case oldD.IsZero() && !newD.IsZero():
		s.writeAudit(ctx, t, setA, "", newD.String(), at)
	case !oldD.IsZero() && newD.IsZero():
		s.writeAudit(ctx, t, clearedA, oldD.String(), "", at)
	default:
		s.writeAudit(ctx, t, changedA, oldD.String(), newD.String(), at)
	}
}

// projectNameOrID resolves a project name for audit display. Falls back to the
// raw ID if the project can't be looked up — better than dropping the entry.
func (s *Service) projectNameOrID(ctx context.Context, id string) string {
	if id == "" {
		return ""
	}
	p, err := s.projects.Get(ctx, id)
	if err != nil || p == nil {
		return id
	}
	return p.Name
}

// priorityLabel renders a priority as "P1".."P4" for audit display.
func priorityLabel(p domain.Priority) string {
	return fmt.Sprintf("P%d", int(p))
}

// ListAudit returns audit entries matching the given filter.
func (s *Service) ListAudit(ctx context.Context, f storage.AuditFilter) ([]*domain.AuditEntry, error) {
	if s.audit == nil {
		return nil, nil
	}
	return s.audit.List(ctx, f)
}

// ---------- Comments ----------

// AddComment attaches a Markdown note to a task. The body is trimmed and must
// be non-empty; the task must exist (and not be hard-gone). Works for any task
// status so notes can still be added to completed tasks.
func (s *Service) AddComment(ctx context.Context, taskID, body string) (*domain.Comment, error) {
	if s.comments == nil {
		return nil, fmt.Errorf("%w: comments not available", ErrValidation)
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, fmt.Errorf("%w: comment body is required", ErrValidation)
	}
	if _, err := s.tasks.Get(ctx, taskID); err != nil {
		return nil, mapErr(err)
	}
	c := &domain.Comment{
		ID:        newID("comment"),
		TaskID:    taskID,
		Body:      body,
		CreatedAt: s.clock(),
	}
	if err := s.comments.Create(ctx, c); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, events.CommentCreated, c)
	return c, nil
}

// ListComments returns a task's comments, newest first.
func (s *Service) ListComments(ctx context.Context, taskID string) ([]*domain.Comment, error) {
	if s.comments == nil {
		return nil, nil
	}
	return s.comments.List(ctx, taskID)
}

// CommentCounts returns a task_id -> number of comments map. Counting in the
// store instead of calling len() on every loaded comment: a task list shows the
// number, never the bodies.
func (s *Service) CommentCounts(ctx context.Context) (map[string]int, error) {
	if s.comments == nil {
		return map[string]int{}, nil
	}
	counts, err := s.comments.CountByTasks(ctx)
	return counts, mapErr(err)
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "not found") {
		return ErrNotFound
	}
	return err
}
