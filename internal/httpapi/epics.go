package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/service"
	"github.com/till/todo/internal/storage"
)

type epicCreateReq struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ProjectID   string `json:"project_id"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	SortOrder   int    `json:"sort_order"`
}

// epicPatchReq deliberately has no status field: the status moves only through
// POST /api/epics/{id}/complete and /reopen, so the "no open tasks left" rule
// lives in exactly one place. Same split as tasks.
type epicPatchReq struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	ProjectID   *string `json:"project_id"`
	StartDate   *string `json:"start_date"`
	EndDate     *string `json:"end_date"`
	SortOrder   *int    `json:"sort_order"`
	Archived    *bool   `json:"archived"`
}

// writeEpic renders one epic with its project name and task tally. The counts
// come from a single pass over all tasks — cheap enough at this scale, and it
// keeps the count logic in the service where the rest of it lives.
func (h *Handler) writeEpic(w http.ResponseWriter, r *http.Request, e *domain.Epic, status int) {
	counts, err := h.svc.EpicTaskCounts(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, status, toEpicDTO(e, h.projectName(r.Context(), e.ProjectID), counts[e.ID]))
}

func (h *Handler) handleListEpics(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := storage.EpicFilter{
		ProjectID:       q.Get("project_id"),
		IncludeArchived: q.Get("include_archived") == "true",
	}
	switch s := q.Get("status"); s {
	case "":
		// no constraint — both running and finished epics
	case string(domain.EpicActive), string(domain.EpicCompleted):
		f.Status = domain.EpicStatus(s)
	default:
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "status must be active or completed"})
		return
	}
	es, err := h.svc.ListEpics(r.Context(), f)
	if err != nil {
		writeError(w, err)
		return
	}
	names, err := h.projectNames(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	counts, err := h.svc.EpicTaskCounts(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]epicDTO, 0, len(es))
	for _, e := range es {
		out = append(out, toEpicDTO(e, names[e.ProjectID], counts[e.ID]))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) handleCreateEpic(w http.ResponseWriter, r *http.Request) {
	var req epicCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "invalid JSON: " + err.Error()})
		return
	}
	start, err := domain.ParseDate(req.StartDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "invalid start_date: " + err.Error()})
		return
	}
	end, err := domain.ParseDate(req.EndDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "invalid end_date: " + err.Error()})
		return
	}
	e, err := h.svc.CreateEpic(r.Context(), service.EpicInput{
		Title: req.Title, Description: req.Description, ProjectID: req.ProjectID,
		Start: start, End: end, SortOrder: req.SortOrder,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	h.writeEpic(w, r, e, http.StatusCreated)
}

func (h *Handler) handleGetEpic(w http.ResponseWriter, r *http.Request) {
	e, err := h.svc.GetEpic(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	h.writeEpic(w, r, e, http.StatusOK)
}

func (h *Handler) handlePatchEpic(w http.ResponseWriter, r *http.Request) {
	var req epicPatchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "invalid JSON: " + err.Error()})
		return
	}
	patch := service.EpicPatch{
		Title: req.Title, Description: req.Description, ProjectID: req.ProjectID,
		SortOrder: req.SortOrder, Archived: req.Archived,
	}
	if req.StartDate != nil {
		d, err := domain.ParseDate(*req.StartDate)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResp{Error: "invalid start_date: " + err.Error()})
			return
		}
		patch.Start = &d
	}
	if req.EndDate != nil {
		d, err := domain.ParseDate(*req.EndDate)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResp{Error: "invalid end_date: " + err.Error()})
			return
		}
		patch.End = &d
	}
	e, err := h.svc.UpdateEpic(r.Context(), r.PathValue("id"), patch)
	if err != nil {
		writeError(w, err)
		return
	}
	h.writeEpic(w, r, e, http.StatusOK)
}

func (h *Handler) handleCompleteEpic(w http.ResponseWriter, r *http.Request) {
	e, err := h.svc.CompleteEpic(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	h.writeEpic(w, r, e, http.StatusOK)
}

func (h *Handler) handleReopenEpic(w http.ResponseWriter, r *http.Request) {
	e, err := h.svc.ReopenEpic(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	h.writeEpic(w, r, e, http.StatusOK)
}

func (h *Handler) handleDeleteEpic(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteEpic(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
