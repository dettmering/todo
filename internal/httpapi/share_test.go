package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/till/todo/internal/domain"
)

// shareUpdated is a fixed clock for deterministic "Stand" footers.
// shareUpdated is a fixed "last updated" timestamp for deterministic footers.
var shareUpdated = time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)

func TestBuildAbsencesSharePage_Anonymized(t *testing.T) {
	// A vacation with a label must render only the range + "abwesend": neither
	// the label ("Kreta") nor the type ("vacation"/"Urlaub") may leak.
	a := absence("absence_1", "vacation", "Kreta", date(2026, 7, 10), date(2026, 7, 24))
	out := buildAbsencesSharePage([]*domain.Absence{a}, domain.StatusAvailable, "Till", domain.LocalDate{}, shareUpdated)

	if !strings.Contains(out, "Away") {
		t.Errorf("expected status 'Away' in:\n%s", out)
	}
	for _, leak := range []string{"Kreta", "vacation", "Urlaub"} {
		if strings.Contains(out, leak) {
			t.Errorf("anonymization leak: %q present in output:\n%s", leak, out)
		}
	}
}

func TestBuildAbsencesSharePage_BusinessTripIsUnterwegs(t *testing.T) {
	trip := absence("absence_2", "business_trip", "Berlin", date(2026, 8, 3), date(2026, 8, 5))
	vac := absence("absence_3", "sick", "", date(2026, 8, 10), date(2026, 8, 12))
	out := buildAbsencesSharePage([]*domain.Absence{trip, vac}, domain.StatusAvailable, "Till", domain.LocalDate{}, shareUpdated)

	if !strings.Contains(out, "Business Trip") {
		t.Errorf("business trip should render as 'Business Trip' in:\n%s", out)
	}
	if !strings.Contains(out, "Away") {
		t.Errorf("sick absence should render as 'Away' in:\n%s", out)
	}
	if strings.Contains(out, "Berlin") {
		t.Errorf("business trip label 'Berlin' leaked in:\n%s", out)
	}
}

func TestBuildAbsencesSharePage_Legend(t *testing.T) {
	// The legend sits below the list and explains both badges — but only when
	// there is a list to explain.
	a := absence("absence_4", "vacation", "", date(2026, 7, 10), date(2026, 7, 24))
	out := buildAbsencesSharePage([]*domain.Absence{a}, domain.StatusAvailable, "Till", domain.LocalDate{}, shareUpdated)

	if !strings.Contains(out, `<dl class="legend">`) {
		t.Errorf("expected legend in:\n%s", out)
	}
	if i, j := strings.Index(out, `<ul class="list">`), strings.Index(out, `<dl class="legend">`); i > j {
		t.Errorf("legend must come after the list (list at %d, legend at %d)", i, j)
	}
	for _, want := range []string{"not working, not reachable", "travelling for work, usually reachable"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing legend text %q in:\n%s", want, out)
		}
	}

	// The .legend CSS rule always ships in the <style> block, so look for the
	// rendered element rather than the bare word.
	if empty := buildAbsencesSharePage(nil, domain.StatusAvailable, "Till", domain.LocalDate{}, shareUpdated); strings.Contains(empty, `<dl class="legend">`) {
		t.Errorf("empty state should carry no legend:\n%s", empty)
	}
}

func TestBuildAbsencesSharePage_Empty(t *testing.T) {
	out := buildAbsencesSharePage(nil, domain.StatusAvailable, "Till", domain.LocalDate{}, shareUpdated)
	if !strings.Contains(out, "No scheduled absences") {
		t.Errorf("expected empty-state message in:\n%s", out)
	}
	if !strings.Contains(out, "Last updated: 7 July 2026") {
		t.Errorf("expected footer date in:\n%s", out)
	}
	if strings.Contains(out, "12:00") {
		t.Errorf("footer should not include a time-of-day in:\n%s", out)
	}
}

func TestBuildAbsencesSharePage_NoTimestampOmitsFooter(t *testing.T) {
	// Zero lastUpdated (e.g. no absences recorded yet) => no "Last updated" footer.
	out := buildAbsencesSharePage(nil, domain.StatusAvailable, "Till", domain.LocalDate{}, time.Time{})
	if strings.Contains(out, "Last updated") {
		t.Errorf("expected no footer for zero timestamp in:\n%s", out)
	}
}

func TestBuildAbsencesSharePage_StatusLine(t *testing.T) {
	cases := []struct {
		name   string
		status domain.AvailabilityStatus
		person string
		want   string
	}{
		{"available named", domain.StatusAvailable, "Till", "Till is currently available"},
		{"away named", domain.StatusAway, "Till", "Till is currently away"},
		{"trip named", domain.StatusBusinessTrip, "Till", "Till is currently on a business trip"},
		{"available anon", domain.StatusAvailable, "", "Currently available"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := buildAbsencesSharePage(nil, c.status, c.person, domain.LocalDate{}, shareUpdated)
			if !strings.Contains(out, c.want) {
				t.Errorf("missing status line %q in:\n%s", c.want, out)
			}
		})
	}
}

func TestBuildAbsencesSharePage_BackOn(t *testing.T) {
	// The return date sits between the status banner and the absence list.
	out := buildAbsencesSharePage(nil, domain.StatusAway, "Till", date(2026, 8, 3), shareUpdated)

	if !strings.Contains(out, "Back on <strong>Monday, 3 August 2026</strong>") {
		t.Errorf("missing 'Back on' line in:\n%s", out)
	}
	i, j := strings.Index(out, `class="status`), strings.Index(out, `<h2>`)
	if k := strings.Index(out, `class="back"`); k < i || k > j {
		t.Errorf("'Back on' line must sit between status (%d) and heading (%d), got %d", i, j, k)
	}
}

func TestBuildAbsencesSharePage_BackOnOmittedWhenZero(t *testing.T) {
	// Available or on a business trip => domain.ReturnDate yields a zero date and
	// the line disappears rather than rendering an empty or bogus date.
	for _, status := range []domain.AvailabilityStatus{domain.StatusAvailable, domain.StatusBusinessTrip} {
		out := buildAbsencesSharePage(nil, status, "Till", domain.LocalDate{}, shareUpdated)
		if strings.Contains(out, "Back on") {
			t.Errorf("status %v should carry no 'Back on' line:\n%s", status, out)
		}
	}
}

func TestFormatWeekdayDate(t *testing.T) {
	cases := []struct {
		name string
		in   domain.LocalDate
		want string
	}{
		{"monday", date(2026, 8, 3), "Monday, 3 August 2026"},
		{"first of month", date(2026, 3, 1), "Sunday, 1 March 2026"},
		{"year end", date(2026, 12, 31), "Thursday, 31 December 2026"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatWeekdayDate(c.in); got != c.want {
				t.Errorf("formatWeekdayDate = %q, want %q", got, c.want)
			}
		})
	}
}

func TestFormatRange(t *testing.T) {
	cases := []struct {
		name       string
		start, end domain.LocalDate
		want       string
	}{
		{"single day", date(2026, 3, 2), date(2026, 3, 2), "2 March 2026"},
		{"same month", date(2026, 7, 10), date(2026, 7, 18), "10–18 July 2026"},
		{"cross month", date(2026, 7, 30), date(2026, 8, 3), "30 July – 3 August 2026"},
		{"cross year", date(2026, 12, 30), date(2027, 1, 2), "30 December 2026 – 2 January 2027"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatRange(c.start, c.end); got != c.want {
				t.Errorf("formatRange = %q, want %q", got, c.want)
			}
		})
	}
}

// --- Handler token gating (does not touch the service layer) ---

func TestHandleAbsencesShare_Disabled(t *testing.T) {
	h := &Handler{shareToken: ""}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/absences.html?token=whatever", nil)

	h.handleAbsencesShare(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 when page disabled", rec.Code)
	}
}

func TestHandleAbsencesShare_BadToken(t *testing.T) {
	h := &Handler{shareToken: "secret"}
	cases := map[string]string{
		"missing": "/api/absences.html",
		"wrong":   "/api/absences.html?token=nope",
		"empty":   "/api/absences.html?token=",
	}
	for name, url := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, url, nil)

			h.handleAbsencesShare(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", rec.Code)
			}
		})
	}
}
