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
	"strings"
	"time"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/events"
	"github.com/till/todo/internal/storage"
)

var (
	ErrValidation = errors.New("validation error")
	ErrNotFound   = errors.New("not found")
)

type Service struct {
	tasks    storage.TaskRepo
	projects storage.ProjectRepo
	bus      *events.Bus
	clock    func() time.Time
}

func New(tasks storage.TaskRepo, projects storage.ProjectRepo, bus *events.Bus) *Service {
	return &Service{tasks: tasks, projects: projects, bus: bus, clock: func() time.Time { return time.Now().UTC() }}
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
	Title          string
	Description    string
	ProjectID      string
	Priority       domain.Priority
	VisibleFrom    domain.LocalDate
	Deadline       domain.LocalDate
	RecurrenceRule string
}

func (s *Service) CreateTask(ctx context.Context, in TaskInput) (*domain.Task, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrValidation)
	}
	if in.ProjectID == "" {
		in.ProjectID = domain.InboxID
	}
	if _, err := s.projects.Get(ctx, in.ProjectID); err != nil {
		return nil, fmt.Errorf("%w: project %q not found", ErrValidation, in.ProjectID)
	}
	if in.Priority == 0 {
		in.Priority = domain.P3
	}
	if !in.Priority.Valid() {
		return nil, fmt.Errorf("%w: priority must be 1..4", ErrValidation)
	}

	now := s.clock()
	t := &domain.Task{
		ID:             newID("task"),
		Title:          in.Title,
		Description:    in.Description,
		ProjectID:      in.ProjectID,
		Status:         domain.StatusOpen,
		Priority:       in.Priority,
		VisibleFrom:    in.VisibleFrom,
		Deadline:       in.Deadline,
		RecurrenceRule: in.RecurrenceRule,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.tasks.Create(ctx, t); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, events.TaskCreated, t)
	return t, nil
}

// TaskPatch carries optional updates; nil pointer = leave field unchanged.
type TaskPatch struct {
	Title          *string
	Description    *string
	ProjectID      *string
	Priority       *domain.Priority
	VisibleFrom    *domain.LocalDate
	Deadline       *domain.LocalDate
	RecurrenceRule *string
}

func (s *Service) UpdateTask(ctx context.Context, id string, p TaskPatch) (*domain.Task, error) {
	t, err := s.tasks.Get(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	deadlineChanged := false
	rescheduled := false

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
	if p.Priority != nil {
		if !p.Priority.Valid() {
			return nil, fmt.Errorf("%w: priority must be 1..4", ErrValidation)
		}
		t.Priority = *p.Priority
	}
	if p.VisibleFrom != nil {
		if *p.VisibleFrom != t.VisibleFrom {
			rescheduled = true
		}
		t.VisibleFrom = *p.VisibleFrom
	}
	if p.Deadline != nil {
		if *p.Deadline != t.Deadline {
			deadlineChanged = true
		}
		t.Deadline = *p.Deadline
	}
	if p.RecurrenceRule != nil {
		t.RecurrenceRule = *p.RecurrenceRule
	}
	t.UpdatedAt = s.clock()
	if err := s.tasks.Update(ctx, t); err != nil {
		return nil, mapErr(err)
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
	s.bus.Publish(ctx, events.TaskCompleted, t)
	// Recurrence handling (Modell B from Lastenheft 4.6) is intentionally not
	// implemented yet — when an RRULE engine is added, emit task.recurrence_generated
	// here after creating the next instance.
	return t, nil
}

func (s *Service) DeleteTask(ctx context.Context, id string) error {
	t, err := s.tasks.Get(ctx, id)
	if err != nil {
		return mapErr(err)
	}
	if err := s.tasks.Delete(ctx, id); err != nil {
		return mapErr(err)
	}
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
	now := s.clock()
	p := &domain.Project{
		ID:          newID("project"),
		Name:        in.Name,
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
