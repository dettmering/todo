package httpapi

import (
	"net/http"
	"time"

	"github.com/dettmering/todo/internal/storage"
)

// handleListAudit serves GET /api/audit. Query parameters:
//
//	task_id        — restrict to one task
//	date           — single calendar day (YYYY-MM-DD)
//	from, to       — inclusive calendar-day range (YYYY-MM-DD)
//
// Date filters are interpreted as UTC days and translated into a half-open
// [from, next-day-after-to) interval on the at column.
func (h *Handler) handleListAudit(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := storage.AuditFilter{TaskID: q.Get("task_id")}
	gte, lte, err := parseDateRangeParams(q, "date", "from", "to", "week")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: err.Error()})
		return
	}
	if !gte.IsZero() {
		f.FromUTC = time.Date(gte.Year, gte.Month, gte.Day, 0, 0, 0, 0, time.UTC)
	}
	if !lte.IsZero() {
		f.ToUTC = time.Date(lte.Year, lte.Month, lte.Day+1, 0, 0, 0, 0, time.UTC)
	}
	writeAuditList(w, h, r, f)
}

// handleTaskAudit serves GET /api/tasks/{id}/audit — convenience shortcut
// equivalent to /api/audit?task_id={id}.
func (h *Handler) handleTaskAudit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	writeAuditList(w, h, r, storage.AuditFilter{TaskID: id})
}

func writeAuditList(w http.ResponseWriter, h *Handler, r *http.Request, f storage.AuditFilter) {
	entries, err := h.svc.ListAudit(r.Context(), f)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]auditDTO, 0, len(entries))
	for _, e := range entries {
		out = append(out, toAuditDTO(e))
	}
	writeJSON(w, http.StatusOK, out)
}

