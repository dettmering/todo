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
		`INSERT INTO projects (id, name, description, sort_order, archived, color, created_at, updated_at)
		 VALUES (?, ?, '', 0, 0, ?, ?, ?)`,
		domain.InboxID, "Inbox", "#246fe0", now, now,
	)
	return err
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
)

func (s *Store) Tasks() *TaskRepo       { return &TaskRepo{db: s.db} }
func (s *Store) Projects() *ProjectRepo { return &ProjectRepo{db: s.db} }
