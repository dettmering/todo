package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/service"
	"github.com/till/todo/internal/storage"
)

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
	out := make([]taskDTO, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, toTaskDTO(t))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req taskCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "invalid JSON: " + err.Error()})
		return
	}
	vf, err := domain.ParseDate(req.VisibleFrom)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "visible_from: " + err.Error()})
		return
	}
	dl, err := domain.ParseDate(req.Deadline)
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
	writeJSON(w, http.StatusCreated, toTaskDTO(t))
}

func (h *Handler) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.svc.GetTask(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTaskDTO(t))
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
		d, err := domain.ParseDate(*req.VisibleFrom)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResp{Error: "visible_from: " + err.Error()})
			return
		}
		patch.VisibleFrom = &d
	}
	if req.Deadline != nil {
		d, err := domain.ParseDate(*req.Deadline)
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
	writeJSON(w, http.StatusOK, toTaskDTO(t))
}

func (h *Handler) handleCompleteTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.svc.CompleteTask(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTaskDTO(t))
}

func (h *Handler) handleReopenTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.svc.ReopenTask(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTaskDTO(t))
}

func (h *Handler) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.DeleteTask(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
