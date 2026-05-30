package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

// parseDateRangeParams reads a date-range from the given query parameter
// names. dateP is a single-day shortcut (sets both bounds). weekP accepts
// an ISO week (YYYY-Www) and sets bounds to Mon..Sun. dateP and weekP
// cannot be combined with each other or with fromP/toP.
func parseDateRangeParams(q url.Values, dateP, fromP, toP, weekP string) (domain.LocalDate, domain.LocalDate, error) {
	week := q.Get(weekP)
	date := q.Get(dateP)
	from := q.Get(fromP)
	to := q.Get(toP)
	switch {
	case week != "":
		if date != "" || from != "" || to != "" {
			return domain.LocalDate{}, domain.LocalDate{}, fmt.Errorf("%s cannot be combined with %s/%s/%s", weekP, dateP, fromP, toP)
		}
		mon, sun, err := parseISOWeek(week)
		if err != nil {
			return domain.LocalDate{}, domain.LocalDate{}, fmt.Errorf("%s must be YYYY-Www (e.g. 2026-W21)", weekP)
		}
		return mon, sun, nil
	case date != "":
		d, err := domain.ParseDate(date)
		if err != nil {
			return domain.LocalDate{}, domain.LocalDate{}, fmt.Errorf("%s must be YYYY-MM-DD", dateP)
		}
		return d, d, nil
	default:
		var gte, lte domain.LocalDate
		if from != "" {
			d, err := domain.ParseDate(from)
			if err != nil {
				return domain.LocalDate{}, domain.LocalDate{}, fmt.Errorf("%s must be YYYY-MM-DD", fromP)
			}
			gte = d
		}
		if to != "" {
			d, err := domain.ParseDate(to)
			if err != nil {
				return domain.LocalDate{}, domain.LocalDate{}, fmt.Errorf("%s must be YYYY-MM-DD", toP)
			}
			lte = d
		}
		if !gte.IsZero() && !lte.IsZero() && lte.Before(gte) {
			return domain.LocalDate{}, domain.LocalDate{}, fmt.Errorf("%s must be on or after %s", toP, fromP)
		}
		return gte, lte, nil
	}
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
	visGTE, visLTE, err := parseDateRangeParams(q, "date", "from", "to", "week")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: err.Error()})
		return
	}
	f.VisibleFromGTE = visGTE
	f.VisibleFromLTE = visLTE
	compGTE, compLTE, err := parseDateRangeParams(q, "completed_date", "completed_from", "completed_to", "completed_week")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: err.Error()})
		return
	}
	f.CompletedAtGTE = compGTE
	f.CompletedAtLTE = compLTE
	if today := q.Get("today"); today != "" {
		d, err := domain.ParseDate(today)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResp{Error: "today must be YYYY-MM-DD"})
			return
		}
		f.Today = d
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
	commentsByTask, err := h.svc.CommentsByTasks(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]taskDTO, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, toTaskDTO(t, names[t.ProjectID], commentsByTask[t.ID]))
	}
	writeJSON(w, http.StatusOK, out)
}

// commentsFor returns one task's comments (newest first), swallowing errors
// (a comment-fetch failure should never block returning the task itself).
func (h *Handler) commentsFor(ctx context.Context, taskID string) []*domain.Comment {
	comments, err := h.svc.ListComments(ctx, taskID)
	if err != nil {
		return nil
	}
	return comments
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
	writeJSON(w, http.StatusCreated, toTaskDTO(t, h.projectName(r.Context(), t.ProjectID), h.commentsFor(r.Context(), t.ID)))
}

func (h *Handler) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.svc.GetTask(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTaskDTO(t, h.projectName(r.Context(), t.ProjectID), h.commentsFor(r.Context(), t.ID)))
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
	writeJSON(w, http.StatusOK, toTaskDTO(t, h.projectName(r.Context(), t.ProjectID), h.commentsFor(r.Context(), t.ID)))
}

func (h *Handler) handleCompleteTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.svc.CompleteTask(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTaskDTO(t, h.projectName(r.Context(), t.ProjectID), h.commentsFor(r.Context(), t.ID)))
}

func (h *Handler) handleReopenTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.svc.ReopenTask(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTaskDTO(t, h.projectName(r.Context(), t.ProjectID), h.commentsFor(r.Context(), t.ID)))
}

func (h *Handler) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.DeleteTask(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
