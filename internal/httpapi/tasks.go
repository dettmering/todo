package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/nldate"
	"github.com/till/todo/internal/service"
	"github.com/till/todo/internal/storage"
)

// parseISOWeek parses an ISO-8601 week string like "2026-W21" and returns the
// Monday and Sunday of that week as LocalDates.
func parseISOWeek(s string) (domain.LocalDate, domain.LocalDate, error) {
	if len(s) != 8 || s[4] != '-' || s[5] != 'W' {
		return domain.LocalDate{}, domain.LocalDate{}, fmt.Errorf("expected YYYY-Www")
	}
	var year, week int
	if _, err := fmt.Sscanf(s, "%4d-W%2d", &year, &week); err != nil {
		return domain.LocalDate{}, domain.LocalDate{}, err
	}
	if week < 1 || week > 53 {
		return domain.LocalDate{}, domain.LocalDate{}, fmt.Errorf("week out of range")
	}
	// ISO week 1 contains the year's first Thursday. Start from Jan 4 (always
	// in week 1), walk back to Monday, then advance week-1 weeks.
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, time.UTC)
	weekday := int(jan4.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday → 7
	}
	weekOneMonday := jan4.AddDate(0, 0, 1-weekday)
	monday := weekOneMonday.AddDate(0, 0, (week-1)*7)
	if y, w := monday.ISOWeek(); y != year || w != week {
		return domain.LocalDate{}, domain.LocalDate{}, fmt.Errorf("invalid ISO week %d for year %d", week, year)
	}
	sunday := monday.AddDate(0, 0, 6)
	return domain.DateOf(monday), domain.DateOf(sunday), nil
}

// parseDateInput accepts either an empty string, an ISO date (YYYY-MM-DD), or
// a German natural-language phrase ("morgen", "Freitag", "27.5.", ...).
// Returns the zero LocalDate for empty input.
func parseDateInput(s string) (domain.LocalDate, error) {
	if s == "" {
		return domain.LocalDate{}, nil
	}
	if d, err := domain.ParseDate(s); err == nil {
		return d, nil
	}
	today := domain.DateOf(time.Now().UTC())
	d, err := nldate.ParseDate(s, today)
	if err != nil {
		return domain.LocalDate{}, fmt.Errorf("could not parse %q", s)
	}
	return d, nil
}

type taskCreateReq struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	ProjectID      string `json:"project_id"`
	Priority       int    `json:"priority"`
	VisibleFrom    string `json:"visible_from"`
	Deadline       string `json:"deadline"`
	RecurrenceRule string `json:"recurrence_rule"`
}

func (h *Handler) handleListTasks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := storage.TaskFilter{
		ProjectID:   q.Get("project_id"),
		OnlyVisible: q.Get("scope") != "all",
	}
	switch q.Get("status") {
	case "":
		// default to open when scope != all
		if f.OnlyVisible {
			f.Status = domain.StatusOpen
		}
	case "open":
		f.Status = domain.StatusOpen
	case "completed":
		f.Status = domain.StatusCompleted
	case "deleted":
		f.Status = domain.StatusDeleted
	}
	week := q.Get("week")
	switch {
	case week != "":
		if q.Get("date") != "" || q.Get("from") != "" || q.Get("to") != "" {
			writeJSON(w, http.StatusBadRequest, errorResp{Error: "week cannot be combined with date/from/to"})
			return
		}
		mon, sun, err := parseISOWeek(week)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResp{Error: "week must be YYYY-Www (e.g. 2026-W21)"})
			return
		}
		f.VisibleFromGTE = mon
		f.VisibleFromLTE = sun
	case q.Get("date") != "":
		d, err := domain.ParseDate(q.Get("date"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResp{Error: "date must be YYYY-MM-DD"})
			return
		}
		f.VisibleFromGTE = d
		f.VisibleFromLTE = d
	default:
		if v := q.Get("from"); v != "" {
			d, err := domain.ParseDate(v)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, errorResp{Error: "from must be YYYY-MM-DD"})
				return
			}
			f.VisibleFromGTE = d
		}
		if v := q.Get("to"); v != "" {
			d, err := domain.ParseDate(v)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, errorResp{Error: "to must be YYYY-MM-DD"})
				return
			}
			f.VisibleFromLTE = d
		}
		if !f.VisibleFromGTE.IsZero() && !f.VisibleFromLTE.IsZero() && f.VisibleFromLTE.Before(f.VisibleFromGTE) {
			writeJSON(w, http.StatusBadRequest, errorResp{Error: "to must be on or after from"})
			return
		}
	}
	switch q.Get("sort") {
	case "priority":
		f.Sort = storage.SortPriority
	case "deadline":
		f.Sort = storage.SortDeadline
	case "visible_from":
		f.Sort = storage.SortVisibleFrom
	case "created_at":
		f.Sort = storage.SortCreatedAt
	}

	tasks, err := h.svc.ListTasks(r.Context(), f)
	if err != nil {
		writeError(w, err)
		return
	}
	names, err := h.projectNames(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]taskDTO, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, toTaskDTO(t, names[t.ProjectID]))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) projectNames(ctx context.Context) (map[string]string, error) {
	projects, err := h.svc.ListProjects(ctx, false)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(projects))
	for _, p := range projects {
		m[p.ID] = p.Name
	}
	return m, nil
}

func (h *Handler) projectName(ctx context.Context, id string) string {
	if id == "" {
		return ""
	}
	p, err := h.svc.GetProject(ctx, id)
	if err != nil {
		return ""
	}
	return p.Name
}

func (h *Handler) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req taskCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "invalid JSON: " + err.Error()})
		return
	}
	vf, err := parseDateInput(req.VisibleFrom)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "visible_from: " + err.Error()})
		return
	}
	dl, err := parseDateInput(req.Deadline)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "deadline: " + err.Error()})
		return
	}
	in := service.TaskInput{
		Title:          req.Title,
		Description:    req.Description,
		ProjectID:      req.ProjectID,
		Priority:       domain.Priority(req.Priority),
		VisibleFrom:    vf,
		Deadline:       dl,
		RecurrenceRule: req.RecurrenceRule,
	}
	t, err := h.svc.CreateTask(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toTaskDTO(t, h.projectName(r.Context(), t.ProjectID)))
}

func (h *Handler) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.svc.GetTask(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTaskDTO(t, h.projectName(r.Context(), t.ProjectID)))
}

type taskPatchReq struct {
	Title          *string `json:"title"`
	Description    *string `json:"description"`
	ProjectID      *string `json:"project_id"`
	Priority       *int    `json:"priority"`
	VisibleFrom    *string `json:"visible_from"`
	Deadline       *string `json:"deadline"`
	RecurrenceRule *string `json:"recurrence_rule"`
}

func (h *Handler) handlePatchTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req taskPatchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "invalid JSON: " + err.Error()})
		return
	}
	patch := service.TaskPatch{
		Title:          req.Title,
		Description:    req.Description,
		ProjectID:      req.ProjectID,
		RecurrenceRule: req.RecurrenceRule,
	}
	if req.Priority != nil {
		p := domain.Priority(*req.Priority)
		patch.Priority = &p
	}
	if req.VisibleFrom != nil {
		d, err := parseDateInput(*req.VisibleFrom)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResp{Error: "visible_from: " + err.Error()})
			return
		}
		patch.VisibleFrom = &d
	}
	if req.Deadline != nil {
		d, err := parseDateInput(*req.Deadline)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResp{Error: "deadline: " + err.Error()})
			return
		}
		patch.Deadline = &d
	}
	t, err := h.svc.UpdateTask(r.Context(), id, patch)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTaskDTO(t, h.projectName(r.Context(), t.ProjectID)))
}

func (h *Handler) handleCompleteTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.svc.CompleteTask(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTaskDTO(t, h.projectName(r.Context(), t.ProjectID)))
}

func (h *Handler) handleReopenTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.svc.ReopenTask(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTaskDTO(t, h.projectName(r.Context(), t.ProjectID)))
}

func (h *Handler) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.DeleteTask(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
