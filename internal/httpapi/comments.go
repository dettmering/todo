package httpapi

import (
	"encoding/json"
	"net/http"
)

// handleListComments serves GET /api/tasks/{id}/comments — the task's comments,
// newest first.
func (h *Handler) handleListComments(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	comments, err := h.svc.ListComments(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]commentDTO, 0, len(comments))
	for _, c := range comments {
		out = append(out, toCommentDTO(c))
	}
	writeJSON(w, http.StatusOK, out)
}

type commentCreateReq struct {
	Body string `json:"body"`
}

// handleCreateComment serves POST /api/tasks/{id}/comments. Body is Markdown
// source; it is stored verbatim and rendered client-side. This is also the
// endpoint to use for setting comments programmatically via the API.
func (h *Handler) handleCreateComment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req commentCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "invalid JSON: " + err.Error()})
		return
	}
	c, err := h.svc.AddComment(r.Context(), id, req.Body)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toCommentDTO(c))
}
