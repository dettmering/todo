// Package sqlite is the SQLite-backed implementation of the storage interfaces.
package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/storage"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	if err := s.seedInbox(); err != nil {
		return nil, err
	}
	if err := s.backfillSlugs(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

// migrations are versioned via PRAGMA user_version. To extend the schema,
// append a new step — never edit or reorder existing ones.
type migrationStep struct {
	version int
	sql     string
}

var migrations = []migrationStep{
	{1, `CREATE TABLE IF NOT EXISTS projects (
		id           TEXT PRIMARY KEY,
		name         TEXT NOT NULL,
		description  TEXT NOT NULL DEFAULT '',
		sort_order   INTEGER NOT NULL DEFAULT 0,
		archived     INTEGER NOT NULL DEFAULT 0,
		created_at   TEXT NOT NULL,
		updated_at   TEXT NOT NULL
	);`},
	{2, `CREATE TABLE IF NOT EXISTS tasks (
		id              TEXT PRIMARY KEY,
		title           TEXT NOT NULL,
		description     TEXT NOT NULL DEFAULT '',
		project_id      TEXT NOT NULL REFERENCES projects(id) ON DELETE RESTRICT,
		status          TEXT NOT NULL DEFAULT 'open',
		priority        INTEGER NOT NULL DEFAULT 3,
		visible_from    TEXT NOT NULL DEFAULT '',
		deadline        TEXT NOT NULL DEFAULT '',
		recurrence_rule TEXT NOT NULL DEFAULT '',
		created_at      TEXT NOT NULL,
		updated_at      TEXT NOT NULL,
		completed_at    TEXT
	);`},
	{3, `CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id);`},
	{4, `CREATE INDEX IF NOT EXISTS idx_tasks_status  ON tasks(status);`},
	{5, `ALTER TABLE projects ADD COLUMN color TEXT NOT NULL DEFAULT '';`},
	{6, `ALTER TABLE projects ADD COLUMN slug TEXT NOT NULL DEFAULT '';`},
	{7, `CREATE UNIQUE INDEX IF NOT EXISTS idx_projects_slug ON projects(slug) WHERE slug != '';`},
	{8, `CREATE TABLE IF NOT EXISTS audit_log (
		id         TEXT PRIMARY KEY,
		task_id    TEXT NOT NULL,
		task_title TEXT NOT NULL DEFAULT '',
		action     TEXT NOT NULL,
		old_value  TEXT NOT NULL DEFAULT '',
		new_value  TEXT NOT NULL DEFAULT '',
		at         TEXT NOT NULL
	);`},
	{9, `CREATE INDEX IF NOT EXISTS idx_audit_task ON audit_log(task_id);`},
	{10, `CREATE INDEX IF NOT EXISTS idx_audit_at ON audit_log(at);`},
	{11, `CREATE TABLE IF NOT EXISTS comments (
		id         TEXT PRIMARY KEY,
		task_id    TEXT NOT NULL,
		body       TEXT NOT NULL,
		created_at TEXT NOT NULL
	);`},
	{12, `CREATE INDEX IF NOT EXISTS idx_comments_task ON comments(task_id);`},
	{13, `CREATE TABLE IF NOT EXISTS absences (
		id          TEXT PRIMARY KEY,
		type        TEXT NOT NULL,
		label       TEXT NOT NULL DEFAULT '',
		start_date  TEXT NOT NULL,
		end_date    TEXT NOT NULL,
		created_at  TEXT NOT NULL,
		updated_at  TEXT NOT NULL
	);`},
	// Existing rows default to 'fixed', which reproduces the pre-mode behaviour.
	{14, `ALTER TABLE tasks ADD COLUMN recurrence_mode TEXT NOT NULL DEFAULT 'fixed';`},
	{15, `ALTER TABLE tasks ADD COLUMN recurrence_anchor TEXT NOT NULL DEFAULT '';`},
	// Seed the series date of existing recurring tasks from visible_from, which
	// is what they anchored on before the column existed.
	{16, `UPDATE tasks SET recurrence_anchor = visible_from
	      WHERE recurrence_rule != '' AND recurrence_anchor = '';`},
	{17, `ALTER TABLE absences ADD COLUMN comment TEXT NOT NULL DEFAULT '';`},
}

func (s *Store) migrate() error {
	var current int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&current); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}
	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		if _, err := s.db.Exec(m.sql); err != nil {
			return fmt.Errorf("migration %d failed: %w", m.version, err)
		}
		// PRAGMA user_version doesn't accept bound parameters.
		if _, err := s.db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, m.version)); err != nil {
			return fmt.Errorf("bump user_version to %d: %w", m.version, err)
		}
	}
	return nil
}

func (s *Store) seedInbox() error {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM projects WHERE id = ?`, domain.InboxID).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO projects (id, name, slug, description, sort_order, archived, color, created_at, updated_at)
		 VALUES (?, ?, ?, '', 0, 0, ?, ?, ?)`,
		domain.InboxID, "Inbox", "inbox", "#246fe0", now, now,
	)
	return err
}

// backfillSlugs populates any project rows that still have an empty slug
// (e.g., projects that existed before migration 6 was introduced). Conflicts
// are resolved deterministically by appending "-2", "-3", … in the order the
// rows are processed (sort_order ASC, name ASC), so the result is stable.
func (s *Store) backfillSlugs() error {
	rows, err := s.db.Query(`SELECT id, name FROM projects WHERE slug = '' ORDER BY sort_order ASC, name ASC`)
	if err != nil {
		return fmt.Errorf("backfillSlugs select: %w", err)
	}
	type row struct{ id, name string }
	var pending []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.name); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, r)
	}
	rows.Close()
	if len(pending) == 0 {
		return nil
	}
	for _, r := range pending {
		base := domain.Slugify(r.name)
		slug := base
		for n := 2; ; n++ {
			var taken int
			if err := s.db.QueryRow(`SELECT COUNT(*) FROM projects WHERE slug = ?`, slug).Scan(&taken); err != nil {
				return err
			}
			if taken == 0 {
				break
			}
			slug = fmt.Sprintf("%s-%d", base, n)
		}
		if _, err := s.db.Exec(`UPDATE projects SET slug = ? WHERE id = ?`, slug, r.id); err != nil {
			return fmt.Errorf("backfillSlugs update %s: %w", r.id, err)
		}
	}
	return nil
}

// helpers shared by repos
func timeString(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, s)
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return timeString(*t)
}

// ErrNotFound is returned when a record cannot be located.
var ErrNotFound = errors.New("storage: not found")

// compile-time interface checks
var (
	_ storage.TaskRepo    = (*TaskRepo)(nil)
	_ storage.ProjectRepo = (*ProjectRepo)(nil)
	_ storage.AuditRepo   = (*AuditRepo)(nil)
	_ storage.CommentRepo = (*CommentRepo)(nil)
	_ storage.AbsenceRepo = (*AbsenceRepo)(nil)
)

func (s *Store) Tasks() *TaskRepo       { return &TaskRepo{db: s.db} }
func (s *Store) Projects() *ProjectRepo { return &ProjectRepo{db: s.db} }
func (s *Store) Audit() *AuditRepo      { return &AuditRepo{db: s.db} }
func (s *Store) Comments() *CommentRepo { return &CommentRepo{db: s.db} }
func (s *Store) Absences() *AbsenceRepo { return &AbsenceRepo{db: s.db} }
