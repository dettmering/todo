package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/storage"
)

type TaskRepo struct{ db *sql.DB }

const taskCols = `id, title, description, project_id, status, priority,
	visible_from, deadline, recurrence_rule, created_at, updated_at, completed_at`

func (r *TaskRepo) Create(ctx context.Context, t *domain.Task) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tasks (`+taskCols+`)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Title, t.Description, t.ProjectID, string(t.Status), int(t.Priority),
		t.VisibleFrom.String(), t.Deadline.String(), t.RecurrenceRule,
		timeString(t.CreatedAt), timeString(t.UpdatedAt), nullableTime(t.CompletedAt),
	)
	return err
}

func (r *TaskRepo) Get(ctx context.Context, id string) (*domain.Task, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+taskCols+` FROM tasks WHERE id = ?`, id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (r *TaskRepo) Update(ctx context.Context, t *domain.Task) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE tasks
		   SET title = ?, description = ?, project_id = ?, status = ?, priority = ?,
		       visible_from = ?, deadline = ?, recurrence_rule = ?, updated_at = ?, completed_at = ?
		 WHERE id = ?`,
		t.Title, t.Description, t.ProjectID, string(t.Status), int(t.Priority),
		t.VisibleFrom.String(), t.Deadline.String(), t.RecurrenceRule,
		timeString(t.UpdatedAt), nullableTime(t.CompletedAt), t.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *TaskRepo) List(ctx context.Context, f storage.TaskFilter) ([]*domain.Task, error) {
	var (
		where []string
		args  []any
	)
	if f.ProjectID != "" {
		where = append(where, `project_id = ?`)
		args = append(args, f.ProjectID)
	}
	if f.Status != "" {
		where = append(where, `status = ?`)
		args = append(args, string(f.Status))
	}
	if f.OnlyVisible {
		today := f.Today
		if today.IsZero() {
			today = domain.DateOf(time.Now())
		}
		where = append(where, `(visible_from = '' OR visible_from <= ?)`)
		args = append(args, today.String())
	}
	if !f.VisibleFromGTE.IsZero() {
		where = append(where, `visible_from != '' AND visible_from >= ?`)
		args = append(args, f.VisibleFromGTE.String())
	}
	if !f.VisibleFromLTE.IsZero() {
		where = append(where, `visible_from != '' AND visible_from <= ?`)
		args = append(args, f.VisibleFromLTE.String())
	}
	if len(f.Priorities) > 0 {
		ph := strings.Repeat("?,", len(f.Priorities))
		ph = ph[:len(ph)-1]
		where = append(where, `priority IN (`+ph+`)`)
		for _, p := range f.Priorities {
			args = append(args, int(p))
		}
	}

	q := `SELECT ` + taskCols + ` FROM tasks`
	if len(where) > 0 {
		q += ` WHERE ` + strings.Join(where, " AND ")
	}
	q += ` ORDER BY ` + orderClause(f.Sort)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()
	var out []*domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func orderClause(s storage.TaskSort) string {
	switch s {
	case storage.SortPriority:
		return `priority ASC, visible_from ASC`
	case storage.SortDeadline:
		return `(deadline = '') ASC, deadline ASC, priority ASC`
	case storage.SortVisibleFrom:
		return `(visible_from = '') ASC, visible_from ASC, priority ASC`
	case storage.SortCreatedAt:
		return `created_at ASC`
	default:
		// Lastenheft 5.6 default: overdue first, then priority, then deadline, then age.
		// Approximated here with a fixed today=now sub-expression at query time would
		// require a parameter; we keep it priority+deadline-first and let the UI sort
		// finer if needed.
		return `priority ASC, (deadline = '') ASC, deadline ASC, created_at ASC`
	}
}

// Delete is a soft delete: status -> deleted, completed_at left untouched.
func (r *TaskRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`,
		string(domain.StatusDeleted), timeString(time.Now().UTC()), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanTask(s rowScanner) (*domain.Task, error) {
	var (
		t                                                domain.Task
		status                                           string
		priority                                         int
		visibleFrom, deadline                            string
		createdAt, updatedAt                             string
		completedAt                                      sql.NullString
	)
	if err := s.Scan(
		&t.ID, &t.Title, &t.Description, &t.ProjectID, &status, &priority,
		&visibleFrom, &deadline, &t.RecurrenceRule,
		&createdAt, &updatedAt, &completedAt,
	); err != nil {
		return nil, err
	}
	t.Status = domain.TaskStatus(status)
	t.Priority = domain.Priority(priority)
	var err error
	if t.VisibleFrom, err = domain.ParseDate(visibleFrom); err != nil {
		return nil, err
	}
	if t.Deadline, err = domain.ParseDate(deadline); err != nil {
		return nil, err
	}
	if t.CreatedAt, err = parseTime(createdAt); err != nil {
		return nil, err
	}
	if t.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return nil, err
	}
	if completedAt.Valid {
		ct, err := parseTime(completedAt.String)
		if err != nil {
			return nil, err
		}
		t.CompletedAt = &ct
	}
	return &t, nil
}
