package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/dettmering/todo/internal/domain"
	"github.com/dettmering/todo/internal/events"
	"github.com/dettmering/todo/internal/storage/sqlite"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"), nil)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return New(store.Tasks(), store.Projects(), store.Audit(), store.Comments(), store.Absences(), store.Epics(), events.NewBus())
}

func date(t *testing.T, s string) domain.LocalDate {
	t.Helper()
	d, err := domain.ParseDate(s)
	if err != nil {
		t.Fatalf("ParseDate(%q): %v", s, err)
	}
	return d
}

func TestAbsenceCRUD(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	a, err := svc.CreateAbsence(ctx, AbsenceInput{
		Type: "vacation", Label: "Kreta", Comment: "Buchung AB-12345",
		Start: date(t, "2026-06-15"), End: date(t, "2026-06-19"),
	})
	if err != nil {
		t.Fatalf("CreateAbsence: %v", err)
	}
	if a.ID == "" || a.Type != domain.AbsenceVacation {
		t.Fatalf("unexpected absence: %+v", a)
	}
	if a.Comment != "Buchung AB-12345" {
		t.Fatalf("Comment = %q, want %q", a.Comment, "Buchung AB-12345")
	}

	// The comment survives a round trip through storage.
	got, err := svc.GetAbsence(ctx, a.ID)
	if err != nil || got.Comment != "Buchung AB-12345" {
		t.Fatalf("GetAbsence = %+v, %v; want comment preserved", got, err)
	}

	list, err := svc.ListAbsences(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListAbsences = %v, %v; want 1 entry", list, err)
	}

	label := "Kreta 2026"
	upd, err := svc.UpdateAbsence(ctx, a.ID, AbsencePatch{Label: &label})
	if err != nil || upd.Label != label {
		t.Fatalf("UpdateAbsence = %+v, %v", upd, err)
	}
	// A patch that leaves the comment out must not clear it.
	if upd.Comment != "Buchung AB-12345" {
		t.Fatalf("Comment after label patch = %q, want unchanged", upd.Comment)
	}

	comment := "Flug LH123, Hotel Elounda"
	upd, err = svc.UpdateAbsence(ctx, a.ID, AbsencePatch{Comment: &comment})
	if err != nil || upd.Comment != comment {
		t.Fatalf("UpdateAbsence(comment) = %+v, %v", upd, err)
	}

	empty := ""
	upd, err = svc.UpdateAbsence(ctx, a.ID, AbsencePatch{Comment: &empty})
	if err != nil || upd.Comment != "" {
		t.Fatalf("UpdateAbsence(clear comment) = %+v, %v", upd, err)
	}

	if err := svc.DeleteAbsence(ctx, a.ID); err != nil {
		t.Fatalf("DeleteAbsence: %v", err)
	}
	if _, err := svc.GetAbsence(ctx, a.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetAbsence after delete = %v, want ErrNotFound", err)
	}
}

func TestCreateAbsenceValidation(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	// end before start
	if _, err := svc.CreateAbsence(ctx, AbsenceInput{
		Type: "vacation", Start: date(t, "2026-06-19"), End: date(t, "2026-06-15"),
	}); !errors.Is(err, ErrValidation) {
		t.Errorf("end<start: got %v, want ErrValidation", err)
	}

	// unknown type
	if _, err := svc.CreateAbsence(ctx, AbsenceInput{
		Type: "sabbatical", Start: date(t, "2026-06-15"), End: date(t, "2026-06-19"),
	}); !errors.Is(err, ErrValidation) {
		t.Errorf("bad type: got %v, want ErrValidation", err)
	}

	// empty type defaults to "other"
	a, err := svc.CreateAbsence(ctx, AbsenceInput{
		Type: "", Start: date(t, "2026-06-15"), End: date(t, "2026-06-19"),
	})
	if err != nil {
		t.Fatalf("empty type should default, got %v", err)
	}
	if a.Type != domain.AbsenceOther {
		t.Errorf("empty type = %q, want %q", a.Type, domain.AbsenceOther)
	}
}

func TestServiceNextWorkday(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	// Pin "today" to Friday 2026-06-12.
	svc.SetClock(func() time.Time { return time.Date(2026, time.June, 12, 9, 0, 0, 0, time.UTC) })

	// Without absences: Friday -> Monday.
	d, err := svc.NextWorkday(ctx)
	if err != nil || d.String() != "2026-06-15" {
		t.Fatalf("NextWorkday (no absence) = %s, %v; want 2026-06-15", d.String(), err)
	}

	// Vacation covering the next work week pushes to the following Monday.
	if _, err := svc.CreateAbsence(ctx, AbsenceInput{
		Type: "vacation", Start: date(t, "2026-06-15"), End: date(t, "2026-06-19"),
	}); err != nil {
		t.Fatalf("CreateAbsence: %v", err)
	}
	d, err = svc.NextWorkday(ctx)
	if err != nil || d.String() != "2026-06-22" {
		t.Fatalf("NextWorkday (vacation) = %s, %v; want 2026-06-22", d.String(), err)
	}
}

func TestServiceAvailability(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	// Pin "today" to Friday 2026-06-12.
	svc.SetClock(func() time.Time { return time.Date(2026, time.June, 12, 9, 0, 0, 0, time.UTC) })

	// Zero day => service clock; nothing on file => working, no trip.
	av, err := svc.Availability(ctx, domain.LocalDate{})
	if err != nil {
		t.Fatalf("Availability: %v", err)
	}
	if av.Date.String() != "2026-06-12" || !av.Working || av.OnBusinessTrip || len(av.Absences) != 0 {
		t.Fatalf("empty availability = %+v; want working on 2026-06-12", av)
	}

	if _, err := svc.CreateAbsence(ctx, AbsenceInput{
		Type: "business_trip", Start: date(t, "2026-06-11"), End: date(t, "2026-06-13"),
	}); err != nil {
		t.Fatalf("CreateAbsence trip: %v", err)
	}
	if _, err := svc.CreateAbsence(ctx, AbsenceInput{
		Type: "vacation", Start: date(t, "2026-06-15"), End: date(t, "2026-06-19"),
	}); err != nil {
		t.Fatalf("CreateAbsence vacation: %v", err)
	}

	// Today: only the trip covers => working, on trip, no return date.
	av, err = svc.Availability(ctx, domain.LocalDate{})
	if err != nil {
		t.Fatalf("Availability: %v", err)
	}
	if av.Status != domain.StatusBusinessTrip || !av.Working || !av.OnBusinessTrip || !av.ReturnDate.IsZero() || len(av.Absences) != 1 {
		t.Fatalf("trip availability = %+v; want business_trip/working/on trip", av)
	}

	// Explicit day inside the vacation => away, back the Monday after.
	av, err = svc.Availability(ctx, date(t, "2026-06-17"))
	if err != nil {
		t.Fatalf("Availability: %v", err)
	}
	if av.Status != domain.StatusAway || av.Working || av.OnBusinessTrip || av.ReturnDate.String() != "2026-06-22" {
		t.Fatalf("vacation availability = %+v; want away, back 2026-06-22", av)
	}
}
