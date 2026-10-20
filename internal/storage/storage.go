// Package storage defines repository interfaces. Concrete implementations
// (SQLite today, possibly others later) live in subpackages.
package storage

import (
	"context"
	"time"

	"github.com/till/todo/internal/domain"
)

// TaskFilter captures the query knobs the list view and API need.
// All fields are optional; the zero value means "no constraint".
type TaskFilter struct {
	ProjectID string
	// EpicID narrows to the tasks of one epic. There is deliberately no
	// "tasks without an epic" mode: the views that need that group a project's
	// tasks client-side, which they have to load anyway.
	EpicID      string
	Status      domain.TaskStatus
	OnlyVisible bool // applies the visible_from <= today rule
	Today       domain.LocalDate
	Priorities  []domain.Priority
	Sort        TaskSort

	// VisibleFromGTE / VisibleFromLTE filter on the visible_from column.
	// Tasks with an empty visible_from are excluded when either bound is set,
	// since they have no scheduled date to compare against.
	VisibleFromGTE domain.LocalDate
	VisibleFromLTE domain.LocalDate

	// CompletedAtGTE / CompletedAtLTE filter on the completed_at column,
	// interpreted as full UTC days (inclusive on both ends). Tasks that were
	// never completed are excluded when either bound is set.
	CompletedAtGTE domain.LocalDate
	CompletedAtLTE domain.LocalDate
}

type TaskSort string

const (
	SortDefault     TaskSort = "" // Lastenheft 5.6 recommended default
	SortPriority    TaskSort = "priority"
	SortDeadline    TaskSort = "deadline"
	SortVisibleFrom TaskSort = "visible_from"
	SortCreatedAt   TaskSort = "created_at"
)

type TaskRepo interface {
	Create(ctx context.Context, t *domain.Task) error
	Get(ctx context.Context, id string) (*domain.Task, error)
	Update(ctx context.Context, t *domain.Task) error
	List(ctx context.Context, f TaskFilter) ([]*domain.Task, error)
	Delete(ctx context.Context, id string) error // soft delete

	// CountByEpic tallies open and done tasks per epic id. Same rules as
	// domain.CountEpicTasks — deleted tasks and tasks without an epic stay out
	// — but the counting happens in the store, so a tally does not cost one
	// materialised task per row in the table.
	CountByEpic(ctx context.Context) (map[string]domain.EpicCounts, error)
	// CountOpen tallies open tasks per project and per epic, plus how many of
	// them are visible on the given date. Counting, not listing: the caller
	// wants numbers, and the numbers must not get more expensive as the
	// archive grows.
	CountOpen(ctx context.Context, today domain.LocalDate) (domain.OpenTaskCounts, error)
	// CountOpenByProject tallies open tasks per project id — the number every
	// project response carries. Projects without open tasks are missing.
	CountOpenByProject(ctx context.Context) (map[string]int, error)
}

type ProjectRepo interface {
	Create(ctx context.Context, p *domain.Project) error
	Get(ctx context.Context, id string) (*domain.Project, error)
	Update(ctx context.Context, p *domain.Project) error
	List(ctx context.Context, includeArchived bool) ([]*domain.Project, error)
	Delete(ctx context.Context, id string) error
}

// EpicFilter narrows the epic listing. Zero values mean "no constraint";
// archived epics are hidden unless IncludeArchived is set, mirroring
// ProjectRepo.List.
type EpicFilter struct {
	ProjectID       string
	Status          domain.EpicStatus
	IncludeArchived bool
}

// EpicRepo stores long-running initiatives. There is no Delete: an epic is
// archived through Update (see service.DeleteEpic), the same way a project is,
// so tasks keep resolving their epic label.
type EpicRepo interface {
	Create(ctx context.Context, e *domain.Epic) error
	Get(ctx context.Context, id string) (*domain.Epic, error)
	Update(ctx context.Context, e *domain.Epic) error
	List(ctx context.Context, f EpicFilter) ([]*domain.Epic, error)
}

type AbsenceRepo interface {
	Create(ctx context.Context, a *domain.Absence) error
	Get(ctx context.Context, id string) (*domain.Absence, error)
	Update(ctx context.Context, a *domain.Absence) error
	List(ctx context.Context) ([]*domain.Absence, error)
	Delete(ctx context.Context, id string) error
}

// AuditFilter narrows the audit log query. Zero values mean "no constraint".
// FromUTC is inclusive, ToUTC is exclusive — the caller is expected to expand
// a calendar-day range into the right UTC half-open interval.
type AuditFilter struct {
	TaskID  string
	FromUTC time.Time
	ToUTC   time.Time
	Limit   int
}

type AuditRepo interface {
	Append(ctx context.Context, e *domain.AuditEntry) error
	List(ctx context.Context, f AuditFilter) ([]*domain.AuditEntry, error)
}

// CommentRepo stores task comments. Comments are append-only; there is no
// update or delete path today.
type CommentRepo interface {
	Create(ctx context.Context, c *domain.Comment) error
	// List returns the comments for one task, newest first.
	List(ctx context.Context, taskID string) ([]*domain.Comment, error)
	// CountByTasks returns a task_id -> number of comments map over every task.
	// Tasks without comments are simply absent from the map. The list views show
	// nothing but this number; the bodies of one task are fetched through List
	// when its modal opens.
	CountByTasks(ctx context.Context) (map[string]int, error)
}
