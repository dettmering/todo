package httpapi

import (
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/till/todo/internal/domain"
)

// icalProdID identifies the generating software (RFC 5545 §3.7.3).
const icalProdID = "-//till//todo//DE"

// absenceSummaryPrefix maps an absence type to the German word used as the
// VEVENT SUMMARY prefix in the calendar feed.
var absenceSummaryPrefix = map[domain.AbsenceType]string{
	domain.AbsenceVacation:     "Urlaub",
	domain.AbsenceHoliday:      "Feiertag",
	domain.AbsenceBusinessTrip: "Dienstreise",
	domain.AbsenceSick:         "Krank",
	domain.AbsenceOther:        "Abwesend",
}

// absenceBusyStatus maps an absence type to the Outlook-specific
// X-MICROSOFT-CDO-BUSYSTATUS value. Everything reads as "out of office"
// (purple hatching in Outlook, Teams flips to "not in the office"); only a
// business trip is WORKINGELSEWHERE, mirroring domain.Absence.BlocksWork —
// it's the one type where work still happens. Types not listed default to OOF.
var absenceBusyStatus = map[domain.AbsenceType]string{
	domain.AbsenceBusinessTrip: "WORKINGELSEWHERE",
}

// absenceBusyFields returns the free/busy pair for an absence: the RFC 5545
// TRANSP value every client understands, plus its Outlook counterpart.
func absenceBusyFields(a *domain.Absence) (transp, busyStatus string) {
	busyStatus = absenceBusyStatus[a.Type]
	if busyStatus == "" {
		busyStatus = "OOF"
	}
	// A business trip shouldn't block meeting scheduling — the user is
	// reachable, just elsewhere. Real absences do block.
	if a.Type == domain.AbsenceBusinessTrip {
		return "TRANSPARENT", busyStatus
	}
	return "OPAQUE", busyStatus
}

// handleAbsencesICS serves the absences as a subscribable iCalendar feed.
//
// This route is meant to be exempted from the reverse-proxy BasicAuth (calendar
// clients can't do interactive login), so the `token` query parameter is the
// *only* thing guarding it — see requireToken. When ICAL_TOKEN is unset the feed
// is disabled and returns 404 so it can never be reached unprotected.
func (h *Handler) handleAbsencesICS(w http.ResponseWriter, r *http.Request) {
	if !requireToken(w, r, h.icalToken) {
		return
	}
	as, err := h.svc.ListAbsences(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", `inline; filename="abwesenheiten.ics"`)
	_, _ = io.WriteString(w, buildAbsencesICS(as, time.Now()))
}

// buildAbsencesICS renders absences as an RFC 5545 iCalendar document. Each
// absence becomes an all-day VEVENT. `now` supplies the DTSTAMP (a parameter so
// the output is deterministic in tests).
//
// All-day events use DATE values whose DTEND is *exclusive*: an absence running
// through end date D must set DTEND to D+1, otherwise calendars drop the last
// day. See domain.Absence — Start/End are inclusive.
func buildAbsencesICS(absences []*domain.Absence, now time.Time) string {
	var b strings.Builder
	writeICSLine(&b, "BEGIN:VCALENDAR")
	writeICSLine(&b, "VERSION:2.0")
	writeICSLine(&b, "PRODID:"+icalProdID)
	writeICSLine(&b, "CALSCALE:GREGORIAN")
	writeICSLine(&b, "METHOD:PUBLISH")
	writeICSLine(&b, "X-WR-CALNAME:Abwesenheiten")
	// Refresh hint honoured by Outlook for subscribed calendars.
	writeICSLine(&b, "X-PUBLISHED-TTL:PT12H")
	stamp := icalDateTime(now)
	for _, a := range absences {
		writeICSLine(&b, "BEGIN:VEVENT")
		// Stable UID so re-fetching updates the event instead of duplicating it.
		writeICSLine(&b, "UID:"+a.ID+"@todo")
		writeICSLine(&b, "DTSTAMP:"+stamp)
		writeICSLine(&b, "SEQUENCE:0")
		if !a.CreatedAt.IsZero() {
			writeICSLine(&b, "CREATED:"+icalDateTime(a.CreatedAt))
		}
		if !a.UpdatedAt.IsZero() {
			writeICSLine(&b, "LAST-MODIFIED:"+icalDateTime(a.UpdatedAt))
		}
		writeICSLine(&b, "DTSTART;VALUE=DATE:"+icalDate(a.Start))
		writeICSLine(&b, "DTEND;VALUE=DATE:"+icalDate(a.End.AddDays(1)))
		writeICSLine(&b, "SUMMARY:"+escapeICSText(absenceSummary(a)))
		writeICSLine(&b, "CATEGORIES:"+escapeICSText(strings.ToUpper(string(a.Type))))
		// Free/busy signalling: TRANSP is the portable field, the
		// X-MICROSOFT-CDO-* pair the Outlook-specific equivalent. Without the
		// ALLDAYEVENT flag Outlook renders VALUE=DATE events unreliably and
		// drops some.
		transp, busyStatus := absenceBusyFields(a)
		writeICSLine(&b, "TRANSP:"+transp)
		writeICSLine(&b, "X-MICROSOFT-CDO-ALLDAYEVENT:TRUE")
		writeICSLine(&b, "X-MICROSOFT-CDO-BUSYSTATUS:"+busyStatus)
		writeICSLine(&b, "END:VEVENT")
	}
	writeICSLine(&b, "END:VCALENDAR")
	return b.String()
}

// absenceSummary builds the human-readable event title, e.g. "Urlaub: Kreta"
// or just "Urlaub" when the absence has no label.
func absenceSummary(a *domain.Absence) string {
	prefix := absenceSummaryPrefix[a.Type]
	if prefix == "" {
		prefix = "Abwesend"
	}
	if a.Label != "" {
		return prefix + ": " + a.Label
	}
	return prefix
}

// icalDate formats a LocalDate as an RFC 5545 DATE value (YYYYMMDD).
func icalDate(d domain.LocalDate) string {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC).Format("20060102")
}

// icalDateTime formats a timestamp as an RFC 5545 UTC DATE-TIME (…Z).
func icalDateTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// escapeICSText escapes a value for an iCalendar TEXT field (RFC 5545 §3.3.11):
// backslash, semicolon and comma are backslash-escaped, newlines become "\n",
// and stray carriage returns are dropped.
func escapeICSText(s string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`;`, `\;`,
		`,`, `\,`,
		"\n", `\n`,
		"\r", "",
	).Replace(s)
}

// writeICSLine writes one logical content line terminated by CRLF, folding
// lines longer than 75 octets with a CRLF + leading space continuation
// (RFC 5545 §3.1) without splitting a multi-byte UTF-8 rune.
func writeICSLine(b *strings.Builder, line string) {
	const limit = 75
	max := limit
	for len(line) > max {
		cut := max
		for cut > 0 && !utf8.RuneStart(line[cut]) {
			cut-- // back off so we don't split a rune
		}
		b.WriteString(line[:cut])
		b.WriteString("\r\n ")
		line = line[cut:]
		max = limit - 1 // continuation lines carry a leading space
	}
	b.WriteString(line)
	b.WriteString("\r\n")
}
