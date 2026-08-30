package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/service"
)

type absenceCreateReq struct {
	Type      string `json:"type"`
	Label     string `json:"label"`
	Comment   string `json:"comment"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type absencePatchReq struct {
	Type      *string `json:"type"`
	Label     *string `json:"label"`
	Comment   *string `json:"comment"`
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
}

func (h *Handler) handleListAbsences(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	fromRaw, toRaw := q.Get("from"), q.Get("to")

	var (
		as  []*domain.Absence
		err error
	)
	if fromRaw != "" || toRaw != "" {
		// Range mode: return absences overlapping [from, to], including running ones.
		from, ferr := domain.ParseDate(fromRaw)
		if ferr != nil {
			writeJSON(w, http.StatusBadRequest, errorResp{Error: "invalid from: " + ferr.Error()})
			return
		}
		to, terr := domain.ParseDate(toRaw)
		if terr != nil {
			writeJSON(w, http.StatusBadRequest, errorResp{Error: "invalid to: " + terr.Error()})
			return
		}
		as, err = h.svc.ListAbsencesInRange(r.Context(), from, to)
	} else {
		// View mode: upcoming/current plus past from the current year.
		today, _ := domain.ParseDate(q.Get("today"))
		as, err = h.svc.ListAbsencesForView(r.Context(), today)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]absenceDTO, 0, len(as))
	for _, a := range as {
		out = append(out, toAbsenceDTO(a))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) handleCreateAbsence(w http.ResponseWriter, r *http.Request) {
	var req absenceCreateReq
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
	a, err := h.svc.CreateAbsence(r.Context(), service.AbsenceInput{
		Type: req.Type, Label: req.Label, Comment: req.Comment, Start: start, End: end,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toAbsenceDTO(a))
}

func (h *Handler) handleGetAbsence(w http.ResponseWriter, r *http.Request) {
	a, err := h.svc.GetAbsence(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAbsenceDTO(a))
}

func (h *Handler) handlePatchAbsence(w http.ResponseWriter, r *http.Request) {
	var req absencePatchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "invalid JSON: " + err.Error()})
		return
	}
	patch := service.AbsencePatch{Type: req.Type, Label: req.Label, Comment: req.Comment}
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
	a, err := h.svc.UpdateAbsence(r.Context(), r.PathValue("id"), patch)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAbsenceDTO(a))
}

func (h *Handler) handleDeleteAbsence(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteAbsence(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleNextWorkday(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.NextWorkday(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"date": d.String()})
}
