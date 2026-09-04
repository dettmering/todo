package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/storage"
)

type EpicRepo struct{ db *sql.DB }

const epicCols = `id, title, slug, description, project_id, status,
	start_date, end_date, sort_order, archived, created_at, updated_at, completed_at`

func (r *EpicRepo) Create(ctx context.Context, e *domain.Epic) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO epics (`+epicCols+`)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Title, e.Slug, e.Description, e.ProjectID, string(e.Status),
		e.Start.String(), e.End.String(), e.SortOrder, boolToInt(e.Archived),
		timeString(e.CreatedAt), timeString(e.UpdatedAt), nullableTime(e.CompletedAt),
	)
	return err
}

func (r *EpicRepo) Get(ctx context.Context, id string) (*domain.Epic, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+epicCols+` FROM epics WHERE id = ?`, id)
	e, err := scanEpic(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return e, err
}

func (r *EpicRepo) Update(ctx context.Context, e *domain.Epic) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE epics
		   SET title = ?, slug = ?, description = ?, project_id = ?, status = ?,
		       start_date = ?, end_date = ?, sort_order = ?, archived = ?,
		       updated_at = ?, completed_at = ?
		 WHERE id = ?`,
		e.Title, e.Slug, e.Description, e.ProjectID, string(e.Status),
		e.Start.String(), e.End.String(), e.SortOrder, boolToInt(e.Archived),
		timeString(e.UpdatedAt), nullableTime(e.CompletedAt), e.ID,
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

func (r *EpicRepo) List(ctx context.Context, f storage.EpicFilter) ([]*domain.Epic, error) {
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
	if !f.IncludeArchived {
		where = append(where, `archived = 0`)
	}
	q := `SELECT ` + epicCols + ` FROM epics`
	if len(where) > 0 {
		q += ` WHERE ` + strings.Join(where, " AND ")
	}
	q += ` ORDER BY sort_order ASC, title ASC`

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list epics: %w", err)
	}
	defer rows.Close()
	var out []*domain.Epic
	for rows.Next() {
		e, err := scanEpic(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func scanEpic(s rowScanner) (*domain.Epic, error) {
	var (
		e                    domain.Epic
		status               string
		start, end           string
		archived             int
		createdAt, updatedAt string
		completedAt          sql.NullString
	)
	if err := s.Scan(
		&e.ID, &e.Title, &e.Slug, &e.Description, &e.ProjectID, &status,
		&start, &end, &e.SortOrder, &archived, &createdAt, &updatedAt, &completedAt,
	); err != nil {
		return nil, err
	}
	e.Status = domain.EpicStatus(status)
	e.Archived = archived != 0
	var err error
	if e.Start, err = domain.ParseDate(start); err != nil {
		return nil, err
	}
	if e.End, err = domain.ParseDate(end); err != nil {
		return nil, err
	}
	if e.CreatedAt, err = parseTime(createdAt); err != nil {
		return nil, err
	}
	if e.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return nil, err
	}
	if completedAt.Valid {
		ct, err := parseTime(completedAt.String)
		if err != nil {
			return nil, err
		}
		e.CompletedAt = &ct
	}
	return &e, nil
}
