package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/till/todo/internal/domain"
)

type ProjectRepo struct{ db *sql.DB }

const projectCols = `id, name, description, sort_order, archived, color, created_at, updated_at`

func (r *ProjectRepo) Create(ctx context.Context, p *domain.Project) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO projects (`+projectCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Description, p.SortOrder, boolToInt(p.Archived), p.Color,
		timeString(p.CreatedAt), timeString(p.UpdatedAt),
	)
	return err
}

func (r *ProjectRepo) Get(ctx context.Context, id string) (*domain.Project, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+projectCols+` FROM projects WHERE id = ?`, id)
	p, err := scanProject(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (r *ProjectRepo) Update(ctx context.Context, p *domain.Project) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE projects SET name = ?, description = ?, sort_order = ?, archived = ?, color = ?, updated_at = ? WHERE id = ?`,
		p.Name, p.Description, p.SortOrder, boolToInt(p.Archived), p.Color, timeString(p.UpdatedAt), p.ID,
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

func (r *ProjectRepo) List(ctx context.Context, includeArchived bool) ([]*domain.Project, error) {
	q := `SELECT ` + projectCols + ` FROM projects`
	if !includeArchived {
		q += ` WHERE archived = 0`
	}
	q += ` ORDER BY sort_order ASC, name ASC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ProjectRepo) Delete(ctx context.Context, id string) error {
	if id == domain.InboxID {
		return errors.New("inbox cannot be deleted")
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProject(s rowScanner) (*domain.Project, error) {
	var p domain.Project
	var archived int
	var createdAt, updatedAt string
	if err := s.Scan(&p.ID, &p.Name, &p.Description, &p.SortOrder, &archived, &p.Color, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	p.Archived = archived != 0
	var err error
	if p.CreatedAt, err = parseTime(createdAt); err != nil {
		return nil, err
	}
	if p.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
