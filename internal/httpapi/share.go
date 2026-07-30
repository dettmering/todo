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
	// Zero while reachable — the page then omits the "Back on" line entirely.
	backOn := domain.ReturnDate(all, today)
	// "Last updated" reflects the actual last change to the absence data, not the
	// request time. Rendered in the server's local zone (TZ, e.g. Europe/Berlin);
	// stored timestamps are UTC.
	lastUpdated, hasUpdate := domain.LatestUpdate(all)
	if hasUpdate {
		lastUpdated = lastUpdated.Local()
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	// The share token sits in the URL, so any outbound navigation would carry it
	// along in the Referer header — and with it access to the page. The document
	// links nowhere today, but that's one edit away from being untrue.
	w.Header().Set("Referrer-Policy", "no-referrer")
	_, _ = io.WriteString(w, buildAbsencesSharePage(upcoming, status, h.shareName, today, backOn, lastUpdated))
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
// drives the prominent current-availability banner, `name` personalizes it,
// `today` marks the running absence and dates the "in N days" hints, `backOn`
// adds the "Back on …" line below the banner (zero = omit, see
// domain.ReturnDate), and `lastUpdated` (already in the target zone) fills the
// "Last updated" footer — a zero value omits it too (all parameters so tests
// stay deterministic).
//
// Anonymization happens here: only the date range and a two-state status are
// emitted. Business trips (which don't block work — see Absence.BlocksWork)
// render as "Business Trip"; everything else as "Away". Labels and exact types
// are never written to the output.
func buildAbsencesSharePage(absences []*domain.Absence, status domain.AvailabilityStatus, name string, today, backOn domain.LocalDate, lastUpdated time.Time) string {
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
			// Which entry is running right now is otherwise left to the reader to
			// work out against today's date.
			now := ""
			if a.Covers(today) {
				now = `<span class="now">now</span>`
			}
			fmt.Fprintf(&rows,
				`<li><span class="range">%s</span>%s<span class="badge %s">%s</span></li>`,
				html.EscapeString(formatRange(a.Start, a.End)),
				now, cls, status,
			)
		}
		rows.WriteString(`</ul>`)
		// The legend explains the two badges; it only makes sense next to an
		// actual list, so the empty state stays legend-free.
		rows.WriteString(`<dl class="legend">` +
			`<dt><span class="badge away">Away</span></dt><dd>not working, not reachable</dd>` +
			`<dt><span class="badge trip">Business Trip</span></dt><dd>travelling for work, usually reachable</dd>` +
			`</dl>`)
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
h2{margin:1.5rem 0 .6rem;font-size:1.15rem;font-weight:600}
.status{display:flex;align-items:center;gap:.6rem;font-size:1.15rem;font-weight:600;padding:.9rem 1.1rem;border-radius:10px;margin:0 0 .75rem}
.back{margin:0 0 1rem;padding:0 .15rem;color:var(--muted);font-size:.95rem}
.back strong{color:var(--fg);font-weight:600;font-variant-numeric:tabular-nums}
.status::before{content:"";width:.7rem;height:.7rem;border-radius:50%;background:currentColor;flex:none}
.status.available{color:var(--ok);background:var(--okbg)}
.status.away{color:var(--away);background:var(--awaybg)}
.status.trip{color:var(--trip);background:var(--tripbg)}
.list{list-style:none;margin:0;padding:0}
.list li{display:flex;align-items:center;gap:.55rem;padding:.45rem 0}
.list li .badge{margin-left:auto}
.range{font-variant-numeric:tabular-nums}
.now{flex:none;font-size:.68rem;font-weight:600;letter-spacing:.05em;text-transform:uppercase;color:var(--muted);border:1px solid var(--line);padding:.05rem .4rem;border-radius:999px}
.badge{flex:none;font-size:.8rem;font-weight:600;padding:.15rem .6rem;border-radius:999px}
.badge.away{color:var(--away);background:var(--awaybg)}
.badge.trip{color:var(--trip);background:var(--tripbg)}
.empty{color:var(--muted)}
.legend{display:grid;grid-template-columns:auto 1fr;gap:.45rem .7rem;align-items:center;margin:1.1rem 0 0;padding-top:.9rem;border-top:1px solid var(--line);color:var(--muted);font-size:.8rem}
.legend dt,.legend dd{margin:0}
footer{max-width:32rem;margin:1rem auto 0;color:var(--muted);font-size:.8rem;text-align:center}
</style>
</head>
<body>
<main>
<div class="head"><h1>Availability</h1></div>
<div class="status ` + statusCls + `">` + html.EscapeString(phrase) + `</div>` + backOnHTML(today, backOn) + `
<h2>Scheduled absences</h2>
` + rows.String() + `
</main>` + footerHTML(today, lastUpdated) + `
</body>
</html>
`
}

// backOnHTML renders the "Back in N days on <weekday, date>" line shown under
// the status banner while the user is away. The distance leads because that's
// the part the reader is after; the exact date follows so a forwarded screenshot
// still says something true. A zero date (they're reachable — see
// domain.ReturnDate) renders nothing.
func backOnHTML(today, backOn domain.LocalDate) string {
	if backOn.IsZero() {
		return ""
	}
	days := today.DaysUntil(backOn)
	if days == 1 {
		// "tomorrow" is already a complete time adverbial, so English wants an
		// apposition after it rather than a second preposition — "back tomorrow on
		// Tuesday" reads like a translation slip. The weekday goes with it: next to
		// "tomorrow" it carries nothing.
		return backLine("Back tomorrow, ", formatRange(backOn, backOn))
	}
	if rel := relativeDays(days); rel != "" {
		return backLine("Back "+rel+" on ", formatWeekdayDate(backOn))
	}
	return backLine("Back on ", formatWeekdayDate(backOn))
}

// backLine wraps a finished return sentence, emphasising its date half.
func backLine(lead, date string) string {
	return "\n<p class=\"back\">" + html.EscapeString(lead) +
		"<strong>" + html.EscapeString(date) + "</strong></p>"
}

// relativeDays turns a distance in days into the phrase that saves the reader
// from counting on a calendar — the absolute date always stays next to it, so
// this only ever has to be roughly right, never load-bearing. Every result slots
// into "Back <phrase> on <date>", so they all read as "in …".
//
// Deliberately absent: anything of the "next Monday" family. Whether that means
// the coming Monday or the one after is genuinely contested between speakers,
// and the weekday is already spelled out beside it. Non-positive distances
// return "" — the caller only has future dates, and "in 0 days" helps nobody.
//
// "tomorrow" needs a different sentence around it than the "in …" phrases; see
// backOnHTML, which builds that one itself.
func relativeDays(days int) string {
	switch {
	case days <= 0:
		return ""
	case days == 1:
		return "tomorrow"
	case days < 7:
		return fmt.Sprintf("in %d days", days)
	case days < 14:
		return "in a week"
	default:
		return fmt.Sprintf("in %d weeks", days/7)
	}
}

// footerHTML renders the "Last updated" footer, or nothing when no timestamp is
// available (zero time — e.g. no absences recorded yet).
//
// This is the one date on the page that reads better as pure age: the question
// behind it isn't "when exactly" but "can I still trust this". "2 months ago" is
// an instant verdict, "30 May 2026" makes the reader subtract. The exact date
// stays available on hover.
func footerHTML(today domain.LocalDate, lastUpdated time.Time) string {
	if lastUpdated.IsZero() {
		return ""
	}
	exact := formatTimestampDate(lastUpdated)
	age := relativeDaysAgo(domain.DateOf(lastUpdated).DaysUntil(today))
	if age == "" { // clock skew: a timestamp from the future
		return "\n<footer>Last updated: " + html.EscapeString(exact) + "</footer>"
	}
	return "\n<footer title=\"" + html.EscapeString(exact) + "\">Last updated " +
		html.EscapeString(age) + "</footer>"
}

// relativeDaysAgo is relativeDays looking backwards, for the "Last updated"
// footer. It coarsens faster than its forward twin: past a fortnight nobody
// counts days, they just want to know whether the page is stale, so weeks and
// then months carry the signal. A negative distance (a timestamp from the
// future — clock skew) returns "", leaving the caller to print the plain date.
func relativeDaysAgo(days int) string {
	switch {
	case days < 0:
		return ""
	case days == 0:
		return "today"
	case days == 1:
		return "yesterday"
	case days < 7:
		return fmt.Sprintf("%d days ago", days)
	case days < 14:
		return "a week ago"
	case days < 60:
		return fmt.Sprintf("%d weeks ago", days/7)
	default:
		return fmt.Sprintf("%d months ago", days/30)
	}
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

// formatWeekdayDate renders a single date with its weekday — "Monday, 3 August
// 2026". The weekday is what makes a return date readable at a glance ("ah,
// Monday") without counting days on a calendar.
func formatWeekdayDate(d domain.LocalDate) string {
	return fmt.Sprintf("%s, %d %s %d",
		d.Weekday().String(), d.Day, englishMonths[d.Month], d.Year)
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
