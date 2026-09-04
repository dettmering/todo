// Package httpapi exposes the REST API.
package httpapi

import (
	"net/http"

	"github.com/till/todo/internal/service"
)

type Handler struct {
	svc *service.Service
	mux *http.ServeMux
	// icalToken guards GET /api/absences.ics. Empty disables the feed entirely
	// (the route then 404s), so it can never be served without a token.
	icalToken string
	// shareToken guards GET /api/absences.html (anonymized public overview).
	// Empty disables the page (404), so it can never be served without a token.
	shareToken string
	// shareName is the person the data belongs to. It personalizes the share
	// page's status line ("<name> is currently available") and titles the iCal
	// feed ("<name>s Abwesenheiten"). Empty falls back to generic wording in both.
	shareName string
}

// New returns an http.Handler that serves the REST API described in
// Lastenheft 5.9. The web UI is served separately by the caller. icalToken
// enables the public iCal absences feed when non-empty (see handleAbsencesICS);
// shareToken enables the anonymized public absences page (see handleAbsencesShare);
// shareName personalizes that page's status line and the iCal calendar name.
func New(svc *service.Service, icalToken, shareToken, shareName string) http.Handler {
	h := &Handler{svc: svc, mux: http.NewServeMux(), icalToken: icalToken, shareToken: shareToken, shareName: shareName}
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
	h.mux.HandleFunc("GET /api/tasks/{id}/audit", h.handleTaskAudit)
	h.mux.HandleFunc("GET /api/tasks/{id}/comments", h.handleListComments)
	h.mux.HandleFunc("POST /api/tasks/{id}/comments", h.handleCreateComment)

	h.mux.HandleFunc("GET /api/counts", h.handleCounts)

	h.mux.HandleFunc("GET /api/audit", h.handleListAudit)

	h.mux.HandleFunc("GET /api/projects", h.handleListProjects)
	h.mux.HandleFunc("POST /api/projects", h.handleCreateProject)
	h.mux.HandleFunc("GET /api/projects/{id}", h.handleGetProject)
	h.mux.HandleFunc("PATCH /api/projects/{id}", h.handlePatchProject)
	h.mux.HandleFunc("DELETE /api/projects/{id}", h.handleDeleteProject)

	h.mux.HandleFunc("GET /api/epics", h.handleListEpics)
	h.mux.HandleFunc("POST /api/epics", h.handleCreateEpic)
	h.mux.HandleFunc("GET /api/epics/{id}", h.handleGetEpic)
	h.mux.HandleFunc("PATCH /api/epics/{id}", h.handlePatchEpic)
	h.mux.HandleFunc("POST /api/epics/{id}/complete", h.handleCompleteEpic)
	h.mux.HandleFunc("POST /api/epics/{id}/reopen", h.handleReopenEpic)
	h.mux.HandleFunc("DELETE /api/epics/{id}", h.handleDeleteEpic)

	h.mux.HandleFunc("GET /api/absences", h.handleListAbsences)
	h.mux.HandleFunc("GET /api/absences.ics", h.handleAbsencesICS)
	h.mux.HandleFunc("GET /api/absences.html", h.handleAbsencesShare)
	h.mux.HandleFunc("GET /api/share-links", h.handleShareLinks)
	h.mux.HandleFunc("POST /api/absences", h.handleCreateAbsence)
	h.mux.HandleFunc("GET /api/absences/{id}", h.handleGetAbsence)
	h.mux.HandleFunc("PATCH /api/absences/{id}", h.handlePatchAbsence)
	h.mux.HandleFunc("DELETE /api/absences/{id}", h.handleDeleteAbsence)
	h.mux.HandleFunc("GET /api/next-workday", h.handleNextWorkday)

	h.mux.HandleFunc("POST /api/parse", h.handleParse)
}
