package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/till/todo/internal/domain"
)

func absence(id, typ, label string, start, end domain.LocalDate) *domain.Absence {
	return &domain.Absence{
		ID:    id,
		Type:  domain.AbsenceType(typ),
		Label: label,
		Start: start,
		End:   end,
	}
}

// stamp is the fixed DTSTAMP used across tests for deterministic output.
var stamp = time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)

func TestBuildAbsencesICS_Envelope(t *testing.T) {
	out := buildAbsencesICS(nil, stamp)

	// Every line must be CRLF-terminated (RFC 5545 §3.1).
	if strings.Contains(strings.ReplaceAll(out, "\r\n", ""), "\n") {
		t.Errorf("output contains a bare LF, want CRLF only:\n%q", out)
	}
	for _, want := range []string{
		"BEGIN:VCALENDAR\r\n",
		"VERSION:2.0\r\n",
		"PRODID:" + icalProdID + "\r\n",
		"END:VCALENDAR\r\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// No absences => no events.
	if strings.Contains(out, "BEGIN:VEVENT") {
		t.Errorf("expected no VEVENT for empty input:\n%s", out)
	}
}

func TestBuildAbsencesICS_DTENDIsExclusive(t *testing.T) {
	// Kreta 10.–24. July (inclusive) => DTEND must be the 25th (end + 1 day).
	a := absence("absence_1", "vacation", "Kreta", date(2026, 7, 10), date(2026, 7, 24))
	out := buildAbsencesICS([]*domain.Absence{a}, stamp)

	for _, want := range []string{
		"DTSTART;VALUE=DATE:20260710\r\n",
		"DTEND;VALUE=DATE:20260725\r\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestBuildAbsencesICS_SingleDayDTEND(t *testing.T) {
	// A one-day absence still needs DTEND = start + 1.
	a := absence("absence_x", "sick", "", date(2026, 3, 2), date(2026, 3, 2))
	out := buildAbsencesICS([]*domain.Absence{a}, stamp)

	if !strings.Contains(out, "DTSTART;VALUE=DATE:20260302\r\n") {
		t.Errorf("missing DTSTART in:\n%s", out)
	}
	if !strings.Contains(out, "DTEND;VALUE=DATE:20260303\r\n") {
		t.Errorf("missing DTEND (start+1) in:\n%s", out)
	}
}

func TestBuildAbsencesICS_Summary(t *testing.T) {
	cases := []struct {
		typ, label string
		want       string
	}{
		{"vacation", "Kreta", "SUMMARY:Urlaub: Kreta\r\n"},
		{"vacation", "", "SUMMARY:Urlaub\r\n"},
		{"business_trip", "Berlin", "SUMMARY:Dienstreise: Berlin\r\n"},
		{"sick", "", "SUMMARY:Krank\r\n"},
		{"holiday", "", "SUMMARY:Feiertag\r\n"},
		{"other", "Sonstiges", "SUMMARY:Abwesend: Sonstiges\r\n"},
		// Unknown type falls back to "Abwesend".
		{"weird", "", "SUMMARY:Abwesend\r\n"},
	}
	for _, c := range cases {
		t.Run(c.typ+"/"+c.label, func(t *testing.T) {
			a := absence("absence_1", c.typ, c.label, date(2026, 1, 1), date(2026, 1, 2))
			out := buildAbsencesICS([]*domain.Absence{a}, stamp)
			if !strings.Contains(out, c.want) {
				t.Errorf("missing %q in:\n%s", c.want, out)
			}
		})
	}
}

func TestBuildAbsencesICS_EscapesSpecialChars(t *testing.T) {
	a := absence("absence_1", "other", `A; B, C\D`, date(2026, 1, 1), date(2026, 1, 2))
	out := buildAbsencesICS([]*domain.Absence{a}, stamp)

	want := `SUMMARY:Abwesend: A\; B\, C\\D` + "\r\n"
	if !strings.Contains(out, want) {
		t.Errorf("special chars not escaped, want %q in:\n%s", want, out)
	}
}

func TestBuildAbsencesICS_StableUIDAndStamp(t *testing.T) {
	a := absence("absence_abc", "vacation", "Kreta", date(2026, 7, 10), date(2026, 7, 24))
	out := buildAbsencesICS([]*domain.Absence{a}, stamp)

	if !strings.Contains(out, "UID:absence_abc@todo\r\n") {
		t.Errorf("missing stable UID in:\n%s", out)
	}
	if !strings.Contains(out, "DTSTAMP:20260707T120000Z\r\n") {
		t.Errorf("missing/incorrect DTSTAMP in:\n%s", out)
	}
}

func TestBuildAbsencesICS_FoldsLongLines(t *testing.T) {
	long := strings.Repeat("x", 200)
	a := absence("absence_1", "other", long, date(2026, 1, 1), date(2026, 1, 2))
	out := buildAbsencesICS([]*domain.Absence{a}, stamp)

	// No physical line (between CRLFs) may exceed 75 octets.
	for _, line := range strings.Split(out, "\r\n") {
		if len(line) > 75 {
			t.Errorf("unfolded line of %d octets: %q", len(line), line)
		}
	}
	// Continuation lines start with a single space.
	if !strings.Contains(out, "\r\n x") {
		t.Errorf("expected space-prefixed continuation line in:\n%s", out)
	}
}

// --- Handler token gating (does not touch the service layer) ---

func TestHandleAbsencesICS_Disabled(t *testing.T) {
	h := &Handler{icalToken: ""}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/absences.ics?token=whatever", nil)

	h.handleAbsencesICS(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 when feed disabled", rec.Code)
	}
}

func TestHandleAbsencesICS_BadToken(t *testing.T) {
	h := &Handler{icalToken: "secret"}
	cases := map[string]string{
		"missing": "/api/absences.ics",
		"wrong":   "/api/absences.ics?token=nope",
		"empty":   "/api/absences.ics?token=",
	}
	for name, url := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, url, nil)

			h.handleAbsencesICS(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", rec.Code)
			}
		})
	}
}
