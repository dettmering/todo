package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dettmering/todo/internal/domain"
)

type AbsenceRepo struct{ db *sql.DB }

const absenceCols = `id, type, label, comment, start_date, end_date, created_at, updated_at`

func (r *AbsenceRepo) Create(ctx context.Context, a *domain.Absence) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO absences (`+absenceCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, string(a.Type), a.Label, a.Comment, a.Start.String(), a.End.String(),
		timeString(a.CreatedAt), timeString(a.UpdatedAt),
	)
	return err
}

func (r *AbsenceRepo) Get(ctx context.Context, id string) (*domain.Absence, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+absenceCols+` FROM absences WHERE id = ?`, id)
	a, err := scanAbsence(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (r *AbsenceRepo) Update(ctx context.Context, a *domain.Absence) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE absences SET type = ?, label = ?, comment = ?, start_date = ?, end_date = ?, updated_at = ? WHERE id = ?`,
		string(a.Type), a.Label, a.Comment, a.Start.String(), a.End.String(), timeString(a.UpdatedAt), a.ID,
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

func (r *AbsenceRepo) List(ctx context.Context) ([]*domain.Absence, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+absenceCols+` FROM absences ORDER BY start_date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Absence
	for rows.Next() {
		a, err := scanAbsence(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AbsenceRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM absences WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanAbsence(s rowScanner) (*domain.Absence, error) {
	var a domain.Absence
	var typ, start, end, createdAt, updatedAt string
	if err := s.Scan(&a.ID, &typ, &a.Label, &a.Comment, &start, &end, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	a.Type = domain.AbsenceType(typ)
	var err error
	if a.Start, err = domain.ParseDate(start); err != nil {
		return nil, err
	}
	if a.End, err = domain.ParseDate(end); err != nil {
		return nil, err
	}
	if a.CreatedAt, err = parseTime(createdAt); err != nil {
		return nil, err
	}
	if a.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return nil, err
	}
	return &a, nil
}
