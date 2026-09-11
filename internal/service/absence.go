package service

import (
	"context"
	"fmt"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/events"
)

type AbsenceInput struct {
	Type    string
	Label   string
	Comment string
	Start   domain.LocalDate
	End     domain.LocalDate
}

type AbsencePatch struct {
	Type    *string
	Label   *string
	Comment *string
	Start   *domain.LocalDate
	End     *domain.LocalDate
}

func (s *Service) CreateAbsence(ctx context.Context, in AbsenceInput) (*domain.Absence, error) {
	typ := domain.AbsenceType(in.Type)
	if typ == "" {
		typ = domain.AbsenceOther
	}
	if !typ.Valid() {
		return nil, fmt.Errorf("%w: unknown absence type %q", ErrValidation, in.Type)
	}
	if in.Start.IsZero() || in.End.IsZero() {
		return nil, fmt.Errorf("%w: start and end date are required", ErrValidation)
	}
	if in.End.Before(in.Start) {
		return nil, fmt.Errorf("%w: end date cannot be before start date", ErrValidation)
	}
	now := s.clock()
	a := &domain.Absence{
		ID:        newID("absence"),
		Type:      typ,
		Label:     in.Label,
		Comment:   in.Comment,
		Start:     in.Start,
		End:       in.End,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.absences.Create(ctx, a); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, events.AbsenceCreated, a)
	return a, nil
}

func (s *Service) UpdateAbsence(ctx context.Context, id string, p AbsencePatch) (*domain.Absence, error) {
	a, err := s.absences.Get(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	if p.Type != nil {
		typ := domain.AbsenceType(*p.Type)
		if !typ.Valid() {
			return nil, fmt.Errorf("%w: unknown absence type %q", ErrValidation, *p.Type)
		}
		a.Type = typ
	}
	if p.Label != nil {
		a.Label = *p.Label
	}
	if p.Comment != nil {
		a.Comment = *p.Comment
	}
	if p.Start != nil {
		a.Start = *p.Start
	}
	if p.End != nil {
		a.End = *p.End
	}
	if a.Start.IsZero() || a.End.IsZero() {
		return nil, fmt.Errorf("%w: start and end date are required", ErrValidation)
	}
	if a.End.Before(a.Start) {
		return nil, fmt.Errorf("%w: end date cannot be before start date", ErrValidation)
	}
	a.UpdatedAt = s.clock()
	if err := s.absences.Update(ctx, a); err != nil {
		return nil, mapErr(err)
	}
	s.bus.Publish(ctx, events.AbsenceUpdated, a)
	return a, nil
}

func (s *Service) DeleteAbsence(ctx context.Context, id string) error {
	a, err := s.absences.Get(ctx, id)
	if err != nil {
		return mapErr(err)
	}
	if err := s.absences.Delete(ctx, id); err != nil {
		return mapErr(err)
	}
	s.bus.Publish(ctx, events.AbsenceDeleted, a)
	return nil
}

func (s *Service) ListAbsences(ctx context.Context) ([]*domain.Absence, error) {
	return s.absences.List(ctx)
}

// ListAbsencesForView returns the absences for the absences view: upcoming and
// current ones plus past ones from the current year (see domain.AbsencesForView).
// A zero `today` falls back to the service clock.
func (s *Service) ListAbsencesForView(ctx context.Context, today domain.LocalDate) ([]*domain.Absence, error) {
	if today.IsZero() {
		today = domain.DateOf(s.clock())
	}
	all, err := s.absences.List(ctx)
	if err != nil {
		return nil, err
	}
	return domain.AbsencesForView(all, today), nil
}

// ListAbsencesInRange returns the absences overlapping the inclusive window
// [from, to] (see domain.AbsencesInRange). Zero bounds are open-ended.
func (s *Service) ListAbsencesInRange(ctx context.Context, from, to domain.LocalDate) ([]*domain.Absence, error) {
	all, err := s.absences.List(ctx)
	if err != nil {
		return nil, err
	}
	return domain.AbsencesInRange(all, from, to), nil
}

func (s *Service) GetAbsence(ctx context.Context, id string) (*domain.Absence, error) {
	a, err := s.absences.Get(ctx, id)
	return a, mapErr(err)
}

// NextWorkday computes the next workday after today, skipping weekends and
// blocking absences (business trips count as workdays). See domain.NextWorkday.
func (s *Service) NextWorkday(ctx context.Context) (domain.LocalDate, error) {
	list, err := s.absences.List(ctx)
	if err != nil {
		return domain.LocalDate{}, err
	}
	absences := make([]domain.Absence, 0, len(list))
	for _, a := range list {
		absences = append(absences, *a)
	}
	today := domain.DateOf(s.clock())
	return domain.NextWorkday(today, absences), nil
}

// Availability answers "what is my situation on `day`?" — status, the
// working / on-business-trip booleans, the return date and the covering
// absences (see domain.AvailabilityOn). A zero `day` falls back to the service
// clock, so automation clients can ask without building a date themselves.
func (s *Service) Availability(ctx context.Context, day domain.LocalDate) (domain.Availability, error) {
	if day.IsZero() {
		day = domain.DateOf(s.clock())
	}
	all, err := s.absences.List(ctx)
	if err != nil {
		return domain.Availability{}, err
	}
	return domain.AvailabilityOn(all, day), nil
}
