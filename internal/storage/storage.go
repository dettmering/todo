// Package storage defines repository interfaces. Concrete implementations
// (SQLite today, possibly others later) live in subpackages.
package storage

import (
	"context"

	"github.com/till/todo/internal/domain"
)

// TaskFilter captures the query knobs the list view and API need.
// All fields are optional; the zero value means "no constraint".
type TaskFilter struct {
	ProjectID   string
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
	SortDefault     TaskSort = ""           // Lastenheft 5.6 recommended default
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
}

type ProjectRepo interface {
	Create(ctx context.Context, p *domain.Project) error
	Get(ctx context.Context, id string) (*domain.Project, error)
	Update(ctx context.Context, p *domain.Project) error
	List(ctx context.Context, includeArchived bool) ([]*domain.Project, error)
	Delete(ctx context.Context, id string) error
}
