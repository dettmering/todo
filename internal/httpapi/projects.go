package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/service"
)

type projectCreateReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
	Color       string `json:"color"`
}

type projectPatchReq struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	SortOrder   *int    `json:"sort_order"`
	Archived    *bool   `json:"archived"`
	Color       *string `json:"color"`
}

// writeProject answers with one project and its open task count, like
// writeEpic does for epics: every project response carries the same shape.
func (h *Handler) writeProject(w http.ResponseWriter, r *http.Request, p *domain.Project, status int) {
	counts, err := h.svc.ProjectOpenCounts(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, status, toProjectDTO(p, counts[p.ID]))
}

func (h *Handler) handleListProjects(w http.ResponseWriter, r *http.Request) {
	includeArchived := r.URL.Query().Get("include_archived") == "true"
	ps, err := h.svc.ListProjects(r.Context(), includeArchived)
	if err != nil {
		writeError(w, err)
		return
	}
	counts, err := h.svc.ProjectOpenCounts(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]projectDTO, 0, len(ps))
	for _, p := range ps {
		out = append(out, toProjectDTO(p, counts[p.ID]))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req projectCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "invalid JSON: " + err.Error()})
		return
	}
	p, err := h.svc.CreateProject(r.Context(), service.ProjectInput{
		Name: req.Name, Description: req.Description, SortOrder: req.SortOrder, Color: req.Color,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	h.writeProject(w, r, p, http.StatusCreated)
}

func (h *Handler) handleGetProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := h.svc.GetProject(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	h.writeProject(w, r, p, http.StatusOK)
}

func (h *Handler) handlePatchProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req projectPatchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "invalid JSON: " + err.Error()})
		return
	}
	p, err := h.svc.UpdateProject(r.Context(), id, service.ProjectPatch{
		Name: req.Name, Description: req.Description, SortOrder: req.SortOrder, Archived: req.Archived, Color: req.Color,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	h.writeProject(w, r, p, http.StatusOK)
}

func (h *Handler) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.DeleteProject(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
