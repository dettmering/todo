// Package httpapi exposes the REST API.
package httpapi

import (
	"net/http"

	"github.com/till/todo/internal/service"
)

type Handler struct {
	svc *service.Service
	mux *http.ServeMux
}

// New returns an http.Handler that serves the REST API described in
// Lastenheft 5.9. The web UI is served separately by the caller.
func New(svc *service.Service) http.Handler {
	h := &Handler{svc: svc, mux: http.NewServeMux()}
	h.routes()
	return h.mux
}

func (h *Handler) routes() {
	h.mux.HandleFunc("GET /api/tasks", h.handleListTasks)
	h.mux.HandleFunc("POST /api/tasks", h.handleCreateTask)
	h.mux.HandleFunc("GET /api/tasks/{id}", h.handleGetTask)
	h.mux.HandleFunc("PATCH /api/tasks/{id}", h.handlePatchTask)
	h.mux.HandleFunc("POST /api/tasks/{id}/complete", h.handleCompleteTask)
	h.mux.HandleFunc("POST /api/tasks/{id}/reopen", h.handleReopenTask)
	h.mux.HandleFunc("DELETE /api/tasks/{id}", h.handleDeleteTask)

	h.mux.HandleFunc("GET /api/projects", h.handleListProjects)
	h.mux.HandleFunc("POST /api/projects", h.handleCreateProject)
	h.mux.HandleFunc("GET /api/projects/{id}", h.handleGetProject)
	h.mux.HandleFunc("PATCH /api/projects/{id}", h.handlePatchProject)
	h.mux.HandleFunc("DELETE /api/projects/{id}", h.handleDeleteProject)
}
