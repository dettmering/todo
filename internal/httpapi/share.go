package httpapi

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/till/todo/internal/domain"
)

// handleAbsencesShare serves an anonymized, human-readable HTML overview of the
// user's absences — a link to hand to colleagues so they have a current picture
// of availability. It is deliberately *anonymized*: no reason, no label, only
// the date range and a coarse "abwesend" / "unterwegs" status leave the server.
//
// Like the iCal feed this route is meant to be exempted from the reverse-proxy
// BasicAuth, so the `token` query parameter is the *only* guard — see
// requireToken. When SHARE_TOKEN is unset the page is disabled and returns 404,
// so it can never be reached unprotected.
func (h *Handler) handleAbsencesShare(w http.ResponseWriter, r *http.Request) {
	if !requireToken(w, r, h.shareToken) {
		return
	}
	all, err := h.svc.ListAbsences(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	today := domain.DateOf(time.Now())
	upcoming := domain.UpcomingAbsences(all, today)
	status := domain.CurrentAvailability(all, today)
	// "Last updated" reflects the actual last change to the absence data, not the
	// request time. Rendered in the server's local zone (TZ, e.g. Europe/Berlin);
	// stored timestamps are UTC.
	lastUpdated, hasUpdate := domain.LatestUpdate(all)
	if hasUpdate {
		lastUpdated = lastUpdated.Local()
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(w, buildAbsencesSharePage(upcoming, status, h.shareName, lastUpdated))
}

// availabilityPhrase builds the prominent current-status sentence, e.g.
// "Till is currently available" or, without a configured name, "Currently away".
func availabilityPhrase(status domain.AvailabilityStatus, name string) (text, cls string) {
	switch status {
	case domain.StatusAway:
		text, cls = "away", "away"
	case domain.StatusBusinessTrip:
		text, cls = "on a business trip", "trip"
	default:
		text, cls = "available", "available"
	}
	if name != "" {
		return name + " is currently " + text, cls
	}
	return "Currently " + text, cls
}

// buildAbsencesSharePage renders the anonymized absences as a self-contained
// HTML document (inline CSS, no external requests, light/dark aware). `status`
// drives the prominent current-availability banner, `name` personalizes it, and
// `lastUpdated` (already in the target zone) fills the "Last updated" footer —
// a zero value omits it (all parameters so tests stay deterministic).
//
// Anonymization happens here: only the date range and a two-state status are
// emitted. Business trips (which don't block work — see Absence.BlocksWork)
// render as "Business Trip"; everything else as "Away". Labels and exact types
// are never written to the output.
func buildAbsencesSharePage(absences []*domain.Absence, status domain.AvailabilityStatus, name string, lastUpdated time.Time) string {
	phrase, statusCls := availabilityPhrase(status, name)

	var rows strings.Builder
	if len(absences) == 0 {
		rows.WriteString(`<p class="empty">No scheduled absences.</p>`)
	} else {
		rows.WriteString(`<ul class="list">`)
		for _, a := range absences {
			status, cls := "Away", "away"
			if !a.BlocksWork() { // business trip
				status, cls = "Business Trip", "trip"
			}
			fmt.Fprintf(&rows,
				`<li><span class="range">%s</span><span class="badge %s">%s</span></li>`,
				html.EscapeString(formatRange(a.Start, a.End)),
				cls, status,
			)
		}
		rows.WriteString(`</ul>`)
	}

	return `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex, nofollow">
<title>Availability</title>
<style>
:root{color-scheme:light dark;--bg:#faf9f7;--card:#fff;--fg:#202020;--muted:#808080;--line:#eee;--away:#eb5757;--awaybg:#fdecec;--trip:#4073ff;--tripbg:#eaf0ff;--ok:#2fa36b;--okbg:#e7f6ee}
@media (prefers-color-scheme:dark){:root{--bg:#1a1a1a;--card:#242424;--fg:#e8e8e8;--muted:#8a8a8a;--line:#333;--away:#ff7a7a;--awaybg:#3a2626;--trip:#7aa0ff;--tripbg:#232d47;--ok:#5cd39b;--okbg:#1e3a2c}}
*{box-sizing:border-box}
body{margin:0;padding:2rem 1rem;background:var(--bg);color:var(--fg);font:16px/1.5 system-ui,-apple-system,Segoe UI,Roboto,sans-serif}
main{max-width:32rem;margin:0 auto;background:var(--card);border:1px solid var(--line);border-radius:12px;padding:1.5rem 1.75rem}
.head{padding-bottom:1rem;border-bottom:1px solid var(--line);margin-bottom:1.25rem}
h1{margin:0;font-size:1.4rem}
.sub{margin:0 0 1.25rem;color:var(--muted);font-size:.9rem}
.status{display:flex;align-items:center;gap:.6rem;font-size:1.15rem;font-weight:600;padding:.9rem 1.1rem;border-radius:10px;margin:0 0 1rem}
.status::before{content:"";width:.7rem;height:.7rem;border-radius:50%;background:currentColor;flex:none}
.status.available{color:var(--ok);background:var(--okbg)}
.status.away{color:var(--away);background:var(--awaybg)}
.status.trip{color:var(--trip);background:var(--tripbg)}
.list{list-style:none;margin:0;padding:0}
.list li{display:flex;align-items:center;justify-content:space-between;gap:1rem;padding:.7rem 0;border-bottom:1px solid var(--line)}
.list li:last-child{border-bottom:0}
.range{font-variant-numeric:tabular-nums}
.badge{flex:none;font-size:.8rem;font-weight:600;padding:.15rem .6rem;border-radius:999px}
.badge.away{color:var(--away);background:var(--awaybg)}
.badge.trip{color:var(--trip);background:var(--tripbg)}
.empty{color:var(--muted)}
footer{max-width:32rem;margin:1rem auto 0;color:var(--muted);font-size:.8rem;text-align:center}
</style>
</head>
<body>
<main>
<div class="head"><h1>Availability</h1></div>
<div class="status ` + statusCls + `">` + html.EscapeString(phrase) + `</div>
<p class="sub">Scheduled absences. “Business Trip” = travelling for work, usually reachable.</p>
` + rows.String() + `
</main>` + footerHTML(lastUpdated) + `
</body>
</html>
`
}

// footerHTML renders the "Last updated" footer, or nothing when no timestamp is
// available (zero time — e.g. no absences recorded yet).
func footerHTML(lastUpdated time.Time) string {
	if lastUpdated.IsZero() {
		return ""
	}
	return "\n<footer>Last updated: " + html.EscapeString(formatTimestampDate(lastUpdated)) + "</footer>"
}

var englishMonths = [...]string{
	"", "January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}

// formatTimestampDate renders a timestamp's calendar date in its own location as
// "2 March 2026" (no time-of-day).
func formatTimestampDate(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), englishMonths[t.Month()], t.Year())
}

// formatRange renders an inclusive date range for humans. A single day collapses
// to one date; a range within one month reads "10–18 July 2026", otherwise both
// endpoints are spelled out fully.
func formatRange(start, end domain.LocalDate) string {
	if start == end {
		return fmt.Sprintf("%d %s %d", start.Day, englishMonths[start.Month], start.Year)
	}
	if start.Year == end.Year && start.Month == end.Month {
		return fmt.Sprintf("%d–%d %s %d", start.Day, end.Day, englishMonths[start.Month], start.Year)
	}
	if start.Year == end.Year {
		return fmt.Sprintf("%d %s – %d %s %d",
			start.Day, englishMonths[start.Month], end.Day, englishMonths[end.Month], end.Year)
	}
	return fmt.Sprintf("%d %s %d – %d %s %d",
		start.Day, englishMonths[start.Month], start.Year,
		end.Day, englishMonths[end.Month], end.Year)
}
