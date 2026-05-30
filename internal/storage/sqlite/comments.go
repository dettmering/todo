package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/till/todo/internal/domain"
)

type CommentRepo struct{ db *sql.DB }

const commentCols = `id, task_id, body, created_at`

func (r *CommentRepo) Create(ctx context.Context, c *domain.Comment) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO comments (`+commentCols+`) VALUES (?, ?, ?, ?)`,
		c.ID, c.TaskID, c.Body, timeString(c.CreatedAt),
	)
	return err
}

func (r *CommentRepo) List(ctx context.Context, taskID string) ([]*domain.Comment, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+commentCols+` FROM comments WHERE task_id = ? ORDER BY created_at DESC, id DESC`,
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	defer rows.Close()
	var out []*domain.Comment
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CommentRepo) ListByTasks(ctx context.Context, taskIDs ...string) (map[string][]*domain.Comment, error) {
	q := `SELECT ` + commentCols + ` FROM comments`
	var args []any
	if len(taskIDs) > 0 {
		ph := strings.Repeat("?,", len(taskIDs))
		ph = ph[:len(ph)-1]
		q += ` WHERE task_id IN (` + ph + `)`
		for _, id := range taskIDs {
			args = append(args, id)
		}
	}
	// Global DESC order means each task's slice ends up newest-first too.
	q += ` ORDER BY created_at DESC, id DESC`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list comments by tasks: %w", err)
	}
	defer rows.Close()
	out := make(map[string][]*domain.Comment)
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		out[c.TaskID] = append(out[c.TaskID], c)
	}
	return out, rows.Err()
}

func scanComment(s rowScanner) (*domain.Comment, error) {
	var (
		c         domain.Comment
		createdAt string
	)
	if err := s.Scan(&c.ID, &c.TaskID, &c.Body, &createdAt); err != nil {
		return nil, err
	}
	t, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, err
	}
	c.CreatedAt = t
	return &c, nil
}
