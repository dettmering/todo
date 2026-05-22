package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/recurrence"
	"github.com/till/todo/internal/service"
)

type errorResp struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		writeJSON(w, http.StatusBadRequest, errorResp{Error: err.Error()})
	case errors.Is(err, service.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResp{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errorResp{Error: err.Error()})
	}
}

// DTOs intentionally use snake_case keys to match the Lastenheft examples.

type taskDTO struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	ProjectID      string  `json:"project_id"`
	ProjectName    string  `json:"project_name"`
	Status         string  `json:"status"`
	Priority       int     `json:"priority"`
	VisibleFrom    string  `json:"visible_from,omitempty"`
	Deadline       string  `json:"deadline,omitempty"`
	RecurrenceRule  string `json:"recurrence_rule,omitempty"`
	RecurrenceLabel string `json:"recurrence_label,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	CompletedAt    *string `json:"completed_at,omitempty"`
}

func toTaskDTO(t *domain.Task, projectName string) taskDTO {
	d := taskDTO{
		ID:             t.ID,
		Title:          t.Title,
		Description:    t.Description,
		ProjectID:      t.ProjectID,
		ProjectName:    projectName,
		Status:         string(t.Status),
		Priority:       int(t.Priority),
		VisibleFrom:    t.VisibleFrom.String(),
		Deadline:       t.Deadline.String(),
		RecurrenceRule:  t.RecurrenceRule,
		RecurrenceLabel: recurrence.HumanLabel(t.RecurrenceRule),
		CreatedAt:       t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if t.CompletedAt != nil {
		s := t.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
		d.CompletedAt = &s
	}
	return d
}

type projectDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
	Archived    bool   `json:"archived"`
	Color       string `json:"color"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func toProjectDTO(p *domain.Project) projectDTO {
	return projectDTO{
		ID: p.ID, Name: p.Name, Slug: p.Slug, Description: p.Description,
		SortOrder: p.SortOrder, Archived: p.Archived, Color: p.Color,
		CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
