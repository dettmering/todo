package httpapi

import (
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dettmering/todo/internal/domain"
)

// icalProdID identifies the generating software (RFC 5545 §3.7.3).
const icalProdID = "-//till//todo//DE"

// icalCalendarDescription fills the RFC 7986 DESCRIPTION property — the blurb a
// client may show beneath the calendar name in its subscription list.
const icalCalendarDescription = "Urlaub, Dienstreisen und sonstige Abwesenheiten"

// icalRefreshInterval is how often subscribers should re-fetch, as an ISO 8601
// duration. Absences are usually planned weeks ahead, but sick days aren't —
// four hours keeps a same-day entry from sitting invisible for half a day. It
// feeds two properties that must not drift apart, hence the constant.
const icalRefreshInterval = "PT4H"

// absenceSummaryPrefix maps an absence type to the German word used as the
// VEVENT SUMMARY prefix in the calendar feed.
var absenceSummaryPrefix = map[domain.AbsenceType]string{
	domain.AbsenceVacation:     "Urlaub",
	domain.AbsenceHoliday:      "Feiertag",
	domain.AbsenceBusinessTrip: "Dienstreise",
	domain.AbsenceSick:         "Krank",
	domain.AbsenceOther:        "Abwesend",
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
	_, _ = io.WriteString(w, buildAbsencesICS(as, time.Now(), h.shareName))
}

// calendarName is the title subscribers see for the feed. SHARE_NAME — the same
// setting that personalizes the share page's status line — makes it "Tills
// Abwesenheiten" so the calendar is identifiable among a stack of
// subscriptions; without it the feed stays generic. German, like everything
// else in this feed (SUMMARY prefixes, DESCRIPTION).
func calendarName(name string) string {
	if name == "" {
		return "Abwesenheiten"
	}
	return name + "s Abwesenheiten"
}

// buildAbsencesICS renders absences as an RFC 5545 iCalendar document. Each
// absence becomes an all-day VEVENT. `now` supplies the DTSTAMP (a parameter so
// the output is deterministic in tests), `name` the person the calendar belongs
// to (empty = generic title, see calendarName).
//
// All-day events use DATE values whose DTEND is *exclusive*: an absence running
// through end date D must set DTEND to D+1, otherwise calendars drop the last
// day. See domain.Absence — Start/End are inclusive.
func buildAbsencesICS(absences []*domain.Absence, now time.Time, name string) string {
	var b strings.Builder
	writeICSLine(&b, "BEGIN:VCALENDAR")
	writeICSLine(&b, "VERSION:2.0")
	writeICSLine(&b, "PRODID:"+icalProdID)
	writeICSLine(&b, "CALSCALE:GREGORIAN")
	writeICSLine(&b, "METHOD:PUBLISH")
	// Calendar title and blurb, twice over: X-WR-CALNAME is the pre-standard
	// spelling everything reads, NAME/DESCRIPTION its RFC 7986 successors.
	// Clients that know both prefer the standard ones.
	title := calendarName(name)
	writeICSLine(&b, "X-WR-CALNAME:"+escapeICSText(title))
	writeICSLine(&b, "NAME:"+escapeICSText(title))
	writeICSLine(&b, "DESCRIPTION:"+escapeICSText(icalCalendarDescription))
	// Refresh hint, twice over as well: X-PUBLISHED-TTL is what Outlook reads,
	// REFRESH-INTERVAL the RFC 7986 property Apple Calendar and Thunderbird
	// honour. Google ignores both and polls on its own schedule.
	writeICSLine(&b, "X-PUBLISHED-TTL:"+icalRefreshInterval)
	writeICSLine(&b, "REFRESH-INTERVAL;VALUE=DURATION:"+icalRefreshInterval)
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
		// Absences are facts, never tentative invitations. Most clients assume
		// CONFIRMED anyway; a few want it spelled out.
		writeICSLine(&b, "STATUS:CONFIRMED")
		writeICSLine(&b, "SUMMARY:"+escapeICSText(absenceSummary(a)))
		// The free-text comment (booking numbers, hotel, flight times) becomes
		// the event body — that is where a calendar client shows the details
		// the title has no room for. Omitted when empty: an empty DESCRIPTION
		// makes some clients render a blank notes section.
		if c := strings.TrimSpace(a.Comment); c != "" {
			writeICSLine(&b, "DESCRIPTION:"+escapeICSText(c))
		}
		writeICSLine(&b, "CATEGORIES:"+escapeICSText(strings.ToUpper(string(a.Type))))
		// Every absence is free, whatever its type: the feed sits next to the
		// real work calendar as something to read, not to block with. Outlook
		// shades the whole day for any all-day event that isn't free, which
		// looks as if the absence were already blocked in the work calendar.
		// TRANSP is the portable field, BUSYSTATUS its Outlook counterpart.
		// Without the ALLDAYEVENT flag Outlook renders VALUE=DATE events
		// unreliably and drops some.
		writeICSLine(&b, "TRANSP:TRANSPARENT")
		writeICSLine(&b, "X-MICROSOFT-CDO-ALLDAYEVENT:TRUE")
		writeICSLine(&b, "X-MICROSOFT-CDO-BUSYSTATUS:FREE")
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
