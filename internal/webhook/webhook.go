// Package webhook forwards every event from the in-process bus to a single
// external HTTP endpoint. This is the minimal mapping of Lastenheft 5.11:
// one URL, no auth, no per-event filtering, no retry.
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/dettmering/todo/internal/domain"
	"github.com/dettmering/todo/internal/events"
)

const defaultTimeout = 10 * time.Second

type Dispatcher struct {
	url    string
	client *http.Client
}

func New(url string) *Dispatcher {
	return &Dispatcher{
		url:    url,
		client: &http.Client{Timeout: defaultTimeout},
	}
}

type payload struct {
	Event     string `json:"event"`
	Timestamp string `json:"timestamp"`
	Data      any    `json:"data"`
}

// Handle is an events.Handler. It runs on the bus goroutine, so the HTTP
// round-trip never blocks the originating request. The incoming ctx is
// ignored on purpose: it's the request context, which is canceled as soon
// as the HTTP handler returns — long before our async delivery completes.
func (d *Dispatcher) Handle(_ context.Context, e events.Event) {
	body, err := json.Marshal(payload{
		Event:     string(e.Type),
		Timestamp: e.Timestamp.UTC().Format(time.RFC3339),
		Data:      convertData(e.Data),
	})
	if err != nil {
		slog.Error("webhook marshal", "type", e.Type, "err", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.url, bytes.NewReader(body))
	if err != nil {
		slog.Error("webhook request", "type", e.Type, "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		slog.Warn("webhook send failed", "type", e.Type, "err", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		slog.Warn("webhook non-2xx", "type", e.Type, "status", resp.StatusCode)
	}
}

// convertData reshapes domain types into the snake_case payload structure
// shown in Lastenheft 5.11. Anything we don't recognize is passed through.
func convertData(v any) any {
	switch d := v.(type) {
	case *domain.Task:
		return taskPayload(d)
	case *domain.Project:
		return projectPayload(d)
	default:
		return v
	}
}

type taskJSON struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	ProjectID      string  `json:"project_id"`
	Status         string  `json:"status"`
	Priority       int     `json:"priority"`
	VisibleFrom    string  `json:"visible_from,omitempty"`
	Deadline       string  `json:"deadline,omitempty"`
	RecurrenceRule string  `json:"recurrence,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	CompletedAt    *string `json:"completed_at,omitempty"`
}

func taskPayload(t *domain.Task) taskJSON {
	p := taskJSON{
		ID:             t.ID,
		Title:          t.Title,
		Description:    t.Description,
		ProjectID:      t.ProjectID,
		Status:         string(t.Status),
		Priority:       int(t.Priority),
		VisibleFrom:    t.VisibleFrom.String(),
		Deadline:       t.Deadline.String(),
		RecurrenceRule: t.RecurrenceRule,
		CreatedAt:      t.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      t.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if t.CompletedAt != nil {
		s := t.CompletedAt.UTC().Format(time.RFC3339)
		p.CompletedAt = &s
	}
	return p
}

type projectJSON struct {
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

func projectPayload(p *domain.Project) projectJSON {
	return projectJSON{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		SortOrder:   p.SortOrder,
		Archived:    p.Archived,
		Color:       p.Color,
		CreatedAt:   p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   p.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
