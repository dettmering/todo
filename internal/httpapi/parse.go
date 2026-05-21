package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/till/todo/internal/domain"
	"github.com/till/todo/internal/nldate"
)

type parseReq struct {
	Text  string `json:"text"`
	Today string `json:"today,omitempty"` // optional override (for tests / clients in odd timezones)
}

type parseResp struct {
	Input          string `json:"input"`
	VisibleFrom    string `json:"visible_from,omitempty"`
	RecurrenceRule string `json:"recurrence_rule,omitempty"`
}

// handleParse exposes the natural-language parser (Lastenheft 5.9: POST /api/parse).
// Returns 422 if the input matches no known phrase so callers can distinguish
// "user typed garbage" from "server error".
func (h *Handler) handleParse(w http.ResponseWriter, r *http.Request) {
	var req parseReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResp{Error: "invalid JSON: " + err.Error()})
		return
	}
	today := domain.DateOf(time.Now().UTC())
	if req.Today != "" {
		if d, err := domain.ParseDate(req.Today); err == nil {
			today = d
		}
	}
	res, err := nldate.Parse(req.Text, today)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, errorResp{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, parseResp{
		Input:          req.Text,
		VisibleFrom:    res.Date.String(),
		RecurrenceRule: res.RecurrenceRule,
	})
}
