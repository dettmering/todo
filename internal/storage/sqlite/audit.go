package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/dettmering/todo/internal/domain"
	"github.com/dettmering/todo/internal/storage"
)

type AuditRepo struct{ db *sql.DB }

const auditCols = `id, task_id, task_title, action, old_value, new_value, at`

func (r *AuditRepo) Append(ctx context.Context, e *domain.AuditEntry) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_log (`+auditCols+`) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.TaskID, e.TaskTitle, string(e.Action), e.OldValue, e.NewValue, timeString(e.At),
	)
	return err
}

func (r *AuditRepo) List(ctx context.Context, f storage.AuditFilter) ([]*domain.AuditEntry, error) {
	var (
		where []string
		args  []any
	)
	if f.TaskID != "" {
		where = append(where, `task_id = ?`)
		args = append(args, f.TaskID)
	}
	if !f.FromUTC.IsZero() {
		where = append(where, `at >= ?`)
		args = append(args, timeString(f.FromUTC))
	}
	if !f.ToUTC.IsZero() {
		where = append(where, `at < ?`)
		args = append(args, timeString(f.ToUTC))
	}
	q := `SELECT ` + auditCols + ` FROM audit_log`
	if len(where) > 0 {
		q += ` WHERE ` + strings.Join(where, " AND ")
	}
	q += ` ORDER BY at DESC, id DESC`
	if f.Limit > 0 {
		q += fmt.Sprintf(` LIMIT %d`, f.Limit)
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	defer rows.Close()
	var out []*domain.AuditEntry
	for rows.Next() {
		e, err := scanAudit(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func scanAudit(s rowScanner) (*domain.AuditEntry, error) {
	var (
		e      domain.AuditEntry
		action string
		at     string
	)
	if err := s.Scan(&e.ID, &e.TaskID, &e.TaskTitle, &action, &e.OldValue, &e.NewValue, &at); err != nil {
		return nil, err
	}
	e.Action = domain.AuditAction(action)
	t, err := time.Parse(time.RFC3339, at)
	if err != nil {
		return nil, err
	}
	e.At = t
	return &e, nil
}
