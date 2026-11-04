package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/dettmering/todo/internal/domain"
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

// CountByTasks reads the per-task comment counts out of idx_comments_task. The
// list views show nothing but this number, so the bodies never have to leave
// the database for them.
func (r *CommentRepo) CountByTasks(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT task_id, count(*) FROM comments GROUP BY task_id`)
	if err != nil {
		return nil, fmt.Errorf("count comments by task: %w", err)
	}
	defer rows.Close()
	out := make(map[string]int)
	for rows.Next() {
		var taskID string
		var n int
		if err := rows.Scan(&taskID, &n); err != nil {
			return nil, err
		}
		out[taskID] = n
	}
	return out, rows.Err()
}
