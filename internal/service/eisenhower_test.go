package service

import (
	"context"
	"errors"
	"testing"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/storage"
)

func TestCreateTaskDefaultsUrgency(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	task, err := svc.CreateTask(ctx, TaskInput{Title: "irgendwann"})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if task.Urgency != domain.U4 || task.Quadrant() != domain.QuadrantEliminate {
		t.Errorf("got U%d in %q, want U4 in eliminate", task.Urgency, task.Quadrant())
	}
	if _, err := svc.CreateTask(ctx, TaskInput{Title: "x", Urgency: 5}); !errors.Is(err, ErrValidation) {
		t.Errorf("urgency 5: got %v, want ErrValidation", err)
	}
}

func TestUpdateTaskQuadrant(t *testing.T) {
	cases := []struct {
		name    string
		patch   func() TaskPatch
		wantP   domain.Priority
		wantU   domain.Urgency
		wantErr bool
	}{
		{
			name:  "quadrant moves both axes minimally",
			patch: func() TaskPatch { q := domain.QuadrantDo; return TaskPatch{Quadrant: &q} },
			wantP: domain.P2, wantU: domain.U2,
		},
		{
			name:  "urgency alone",
			patch: func() TaskPatch { u := domain.U1; return TaskPatch{Urgency: &u} },
			wantP: domain.P4, wantU: domain.U1,
		},
		{
			name: "quadrant with priority is rejected",
			patch: func() TaskPatch {
				q, p := domain.QuadrantDo, domain.P1
				return TaskPatch{Quadrant: &q, Priority: &p}
			},
			wantErr: true,
		},
		{
			name:    "unknown quadrant is rejected",
			patch:   func() TaskPatch { q := domain.Quadrant("later"); return TaskPatch{Quadrant: &q} },
			wantErr: true,
		},
		{
			name:    "urgency out of range is rejected",
			patch:   func() TaskPatch { u := domain.Urgency(0); return TaskPatch{Urgency: &u} },
			wantErr: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			svc := newTestService(t)
			task, _ := svc.CreateTask(ctx, TaskInput{Title: "t"})
			got, err := svc.UpdateTask(ctx, task.ID, c.patch())
			if c.wantErr {
				if !errors.Is(err, ErrValidation) {
					t.Fatalf("got %v, want ErrValidation", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateTask: %v", err)
			}
			if got.Priority != c.wantP || got.Urgency != c.wantU {
				t.Errorf("got P%d/U%d, want P%d/U%d", got.Priority, got.Urgency, c.wantP, c.wantU)
			}
			stored, _ := svc.GetTask(ctx, task.ID)
			if stored.Urgency != c.wantU {
				t.Errorf("stored urgency U%d, want U%d", stored.Urgency, c.wantU)
			}
		})
	}
}

func TestUpdateTaskUrgencyIsAudited(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	task, _ := svc.CreateTask(ctx, TaskInput{Title: "t"})
	u := domain.U1
	if _, err := svc.UpdateTask(ctx, task.ID, TaskPatch{Urgency: &u}); err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	entries, err := svc.ListAudit(ctx, storage.AuditFilter{TaskID: task.ID})
	if err != nil {
		t.Fatalf("ListAudit: %v", err)
	}
	for _, e := range entries {
		if e.Action == domain.AuditUrgencyChanged {
			if e.OldValue != "D4" || e.NewValue != "D1" {
				t.Errorf("got %q → %q, want D4 → D1", e.OldValue, e.NewValue)
			}
			return
		}
	}
	t.Errorf("no urgency_changed entry in %d audit entries", len(entries))
}
