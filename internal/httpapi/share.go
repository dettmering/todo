package httpapi

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dettmering/todo/internal/domain"
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
	// Current and future absences, capped at the twelve-week window the timeline
	// draws: what starts beyond it is left off the page entirely, list included.
	_, windowEnd := domain.ShareWindow(today)
	upcoming := domain.AbsencesInRange(domain.UpcomingAbsences(all, today), domain.LocalDate{}, windowEnd)
	// Same status and return date as GET /api/availability — the share page
	// and the API never disagree. ReturnDate is zero while reachable; the page
	// then omits the "Back on" line entirely.
	av := domain.AvailabilityOn(all, today)
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
	_, _ = io.WriteString(w, buildAbsencesSharePage(upcoming, av.Status, h.shareName, today, av.ReturnDate, lastUpdated))
}

// shareLinksResponse carries the ready-to-use URLs of the two token-guarded
// absence routes. A field is omitted when its token is unset — the route then
// 404s anyway, so there is no link to offer.
type shareLinksResponse struct {
	ShareURL string `json:"share_url,omitempty"`
	ICalURL  string `json:"ical_url,omitempty"`
}

// handleShareLinks hands the tokenized share URLs to the internal UI, so its
// absences view can link to the share page and the iCal feed without the tokens
// being duplicated into the frontend (they only ever live in the environment).
//
// Unlike the two routes it describes, this one carries the tokens in its
// *response* and must therefore stay behind the reverse-proxy auth — it is a
// plain /api/ route for exactly that reason.
func (h *Handler) handleShareLinks(w http.ResponseWriter, r *http.Request) {
	var resp shareLinksResponse
	if h.shareToken != "" {
		resp.ShareURL = "/api/absences.html?token=" + url.QueryEscape(h.shareToken)
	}
	if h.icalToken != "" {
		resp.ICalURL = "/api/absences.ics?token=" + url.QueryEscape(h.icalToken)
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, resp)
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
// `today` marks the running absence and anchors the twelve-week timeline
// strip, `backOn` adds the "Back on …" line below
// the banner (zero = omit, see domain.AvailabilityOn), and `lastUpdated`
// (already in the target zone) fills the "Last updated" footer — a zero value
// omits it too (all parameters so tests stay deterministic).
//
// Anonymization happens here: only the date range and a two-state status are
// emitted. Business trips (which don't block work — see Absence.BlocksWork)
// render as "Business Trip"; everything else as "Away". Labels and exact types
// are never written to the output — that also holds for the timeline strip,
// which is derived from dates alone.
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
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&amp;family=JetBrains+Mono&amp;display=swap">
<style>
:root{
--sans:"Inter",system-ui,-apple-system,"Segoe UI",sans-serif;
--mono:"JetBrains Mono",ui-monospace,SFMono-Regular,monospace;
--bg:#fbfaf7;--paper:#fff;--edge:rgba(21,23,28,.16);--shade:rgba(21,23,28,.13);
--fg:#15171c;--muted:#666c76;--faint:#8a9099;--line:rgba(21,23,28,.12);
--track:#f3f1ec;--tick:rgba(21,23,28,.22);--offpaper:#8a9099;
--away:#b5342a;--awaybg:#f7e4e2;--trip:#3a4fa0;--tripbg:#e5e8f4;--ok:#2c7351;--now:#0b6e88}
@media (prefers-color-scheme:dark){:root{
--bg:#12151a;--paper:#191d23;--edge:rgba(232,233,236,.14);--shade:rgba(0,0,0,.5);
--fg:#e8e9ec;--muted:#9aa1ab;--faint:#7b838e;--line:rgba(232,233,236,.14);
--track:#232830;--tick:rgba(232,233,236,.26);--offpaper:#7b838e;
--away:#e86a5e;--awaybg:#3a201d;--trip:#8b9ce8;--tripbg:#1b2340;--ok:#6ecb8c;--now:#56c5e2}}
*{box-sizing:border-box}
body{margin:0;padding:clamp(1.5rem,5vw,3.5rem) 1rem 3rem;background:var(--bg);color:var(--fg);font-family:var(--sans);font-size:16px;line-height:1.55;-webkit-font-smoothing:antialiased}
main{max-width:32rem;margin:0 auto;background:var(--paper);border-radius:4px;padding:clamp(1.35rem,4vw,2rem) clamp(1.15rem,4vw,2rem);
outline:1px solid var(--edge);outline-offset:-1px;box-shadow:0 1px 2px var(--shade),0 12px 30px -14px var(--shade)}
.head{margin-bottom:.35rem}
/* Die Überschrift ist die Rubrik, nicht die Nachricht — die Nachricht ist der
   Status darunter. */
h1{margin:0;font-family:var(--mono);font-size:.688rem;font-weight:400;letter-spacing:.14em;text-transform:uppercase;color:var(--faint)}
/* Die Linie ist die Oberkante dieser Ueberschrift. Luft darueber (margin-top)
   und darunter (padding-top) sind derselbe Wert, sonst haengt der Strich
   schief zwischen den beiden Bloecken. */
h2{margin:1.1rem 0 .7rem;padding-top:1.1rem;border-top:1px solid var(--line);font-family:var(--mono);font-size:.688rem;font-weight:400;letter-spacing:.14em;text-transform:uppercase;color:var(--faint)}
/* Kein Statuspunkt davor: die Zeile ist selbst schon in der Statusfarbe
   gesetzt, der Punkt sagte dasselbe ein zweites Mal. */
.status{font-size:clamp(1.35rem,4.5vw,1.7rem);font-weight:600;letter-spacing:-.022em;line-height:1.15;margin:0 0 .5rem}
.status.available{color:var(--ok)}
.status.away{color:var(--away)}
.status.trip{color:var(--trip)}
.back{margin:0;color:var(--muted);font-family:var(--mono);font-size:.9rem;letter-spacing:.02em;font-variant-numeric:tabular-nums}
.back strong{color:var(--fg);font-weight:400}
/* Die Schiene ist das einzige Bild auf der Seite und darf das auch sein. */
.timeline{margin:1.5rem 0 0}
.tl-weeks{position:relative;height:.95rem;margin-bottom:.35rem;font-family:var(--mono);font-size:.625rem;letter-spacing:.04em;line-height:1;color:var(--faint);font-variant-numeric:tabular-nums}
.tl-weeks span{position:absolute;top:0;transform:translateX(-50%)}
.tl-weeks span.cur{color:var(--now);font-weight:400}
.tl-track{position:relative;height:.5rem;border-radius:999px;background-color:var(--track);background-image:repeating-linear-gradient(to right,transparent 0,transparent calc(100%/12 - 1px),var(--tick) calc(100%/12 - 1px),var(--tick) calc(100%/12))}
.tl-span{position:absolute;top:0;bottom:0;border-radius:999px;min-width:.5rem}
.tl-span.away{background:var(--away)}
.tl-span.trip{background:var(--trip)}
/* Raute statt Punkt, wie die Meilensteinmarke im Gantt-Diagramm -- Zwilling
   zu .tl-today in style.css. */
.tl-today{position:absolute;top:50%;width:.56rem;height:.56rem;background:var(--now);box-shadow:0 0 0 2px var(--paper);transform:translate(-50%,-50%) rotate(45deg)}
.tl-months{position:relative;height:1rem;margin-top:.55rem;font-family:var(--mono);font-size:.625rem;letter-spacing:.11em;text-transform:uppercase;color:var(--faint)}
.tl-months span{position:absolute;top:0}
.list{list-style:none;margin:1.25rem 0 0;padding:0}
.list li{display:flex;align-items:baseline;gap:.6rem;padding:.6rem 0}
.list li:last-child{border-bottom:0}
.list li .badge{margin-left:auto}
.range{font-family:var(--mono);font-size:.95rem;letter-spacing:.02em;font-variant-numeric:tabular-nums}
.now{flex:none;align-self:center;white-space:nowrap;font-family:var(--mono);font-size:.6875rem;font-weight:400;letter-spacing:.11em;text-transform:uppercase;color:var(--muted);border:1px solid var(--tick);padding:.1rem .55rem;border-radius:999px}
.badge{flex:none;align-self:center;white-space:nowrap;font-size:.8rem;font-weight:500;padding:.18rem .7rem;border-radius:999px}
.badge.away{color:var(--away);background:var(--awaybg)}
.badge.trip{color:var(--trip);background:var(--tripbg)}
.empty{color:var(--muted)}
.legend{display:grid;grid-template-columns:auto 1fr;gap:.5rem .75rem;align-items:center;margin:1.35rem 0 0;padding-top:1rem;border-top:1px solid var(--line);color:var(--muted);font-size:.85rem}
.legend dt,.legend dd{margin:0}
footer{max-width:32rem;margin:1.1rem auto 0;color:var(--offpaper);font-family:var(--mono);font-size:.625rem;letter-spacing:.11em;text-transform:uppercase;text-align:center}
@media (prefers-reduced-motion:reduce){*{transition:none!important}}
</style>
</head>
<body>
<main>
<div class="head"><h1>Availability</h1></div>
<div class="status ` + statusCls + `">` + html.EscapeString(phrase) + `</div>` + backOnHTML(today, backOn) + `
<h2>` + absencesHeading() + `</h2>
` + absenceTimelineHTML(absences, today) + rows.String() + `
</main>` + footerHTML(today, lastUpdated) + `
</body>
</html>
`
}

// absencesHeading names the absences section and states the horizon it covers,
// so readers see that the list is a window (domain.ShareWindowWeeks) and not
// everything on the calendar. The timeline below it carries no caption of its
// own — this heading labels both.
func absencesHeading() string {
	if domain.ShareWindowWeeks == 1 {
		return "Scheduled absences in the next week"
	}
	return fmt.Sprintf("Scheduled absences in the next %d weeks", domain.ShareWindowWeeks)
}

// absenceTimelineHTML renders the coming twelve weeks as a slim horizontal
// rail: colored pill-shaped spans for the absences, a "today" dot, thin ticks
// dividing the rail into calendar weeks, and month labels underneath. It
// answers "when are the gaps?" at a glance, which a
// list of ranges can't — the list below stays the readable source of truth,
// so the rail is aria-hidden decoration. Rail and list share the window from
// domain.ShareWindow, so both show the same absences; spans of running ones are
// clipped at the left edge. No absences = no rail.
//
// !! This rail has a twin: renderAbsenceTimeline in
// internal/web/static/app.js draws the same geometry client-side for the
// internal /absences view — same class names, same pos(), same half-cell
// offset on the today dot, same 9%/94% month-label rules. Nothing is shared
// across the language boundary (this page must stand alone, without app.js),
// so geometry changes have to land in both or the two views drift apart.
// Deliberately different here: the anonymized two-state coloring and the
// absence of tooltips — that difference *is* the anonymization, don't
// "unify" it away.
func absenceTimelineHTML(absences []*domain.Absence, today domain.LocalDate) string {
	if len(absences) == 0 {
		return ""
	}
	const totalDays = domain.ShareWindowWeeks * 7
	start, end := domain.ShareWindow(today)
	pos := func(d domain.LocalDate) float64 {
		return float64(start.DaysUntil(d)) * 100 / totalDays
	}
	var b strings.Builder
	b.WriteString(`<section class="timeline" aria-hidden="true">`)
	// ISO week numbers above the rail, centered over their cell — the bare
	// number, since "week" is what a two-digit label over a calendar strip can
	// only mean. The current week (always the first cell, the window starts on
	// its Monday) is emphasized to anchor the reading.
	b.WriteString(`<div class="tl-weeks">`)
	for i := 0; i < domain.ShareWindowWeeks; i++ {
		cls := ""
		if i == 0 {
			cls = ` class="cur"`
		}
		fmt.Fprintf(&b, `<span%s style="left:%.3f%%">%d</span>`,
			cls, (float64(i)+0.5)*100/domain.ShareWindowWeeks, start.AddDays(i*7).ISOWeek())
	}
	b.WriteString(`</div><div class="tl-track">`)
	for _, a := range absences {
		s, e := a.Start, a.End
		if end.Before(s) || e.Before(start) {
			continue
		}
		if s.Before(start) {
			s = start
		}
		if end.Before(e) {
			e = end
		}
		cls := "away"
		if !a.BlocksWork() {
			cls = "trip"
		}
		// Deliberately no title/tooltip: the list below spells the range out, and
		// a date string here would shadow the list's copy as the document's first
		// occurrence (the now-marker test anchors on that order).
		fmt.Fprintf(&b, `<div class="tl-span %s" style="left:%.3f%%;width:%.3f%%"></div>`,
			cls, pos(s), float64(s.DaysUntil(e)+1)*100/totalDays)
	}
	// The marker sits mid-cell on today's column.
	fmt.Fprintf(&b, `<div class="tl-today" style="left:%.3f%%"></div>`, pos(today)+100.0/(2*totalDays))
	b.WriteString(`</div><div class="tl-months">`)
	// Month labels under the track: the window's own month, then each first of a
	// month. Labels closer than ~9% to the previous one (or hugging the right
	// edge) are dropped rather than overlapped.
	last := -100.0
	for d := start; !end.Before(d); d = d.AddDays(1) {
		if d != start && d.Day != 1 {
			continue
		}
		p := pos(d)
		if p-last < 9 || p > 94 {
			continue
		}
		fmt.Fprintf(&b, `<span style="left:%.3f%%">%s</span>`, p, englishMonths[d.Month][:3])
		last = p
	}
	b.WriteString(`</div></section>`)
	return b.String()
}

// backOnHTML renders the "Back in N days on <weekday, date>" line shown under
// the status banner while the user is away. The distance leads because that's
// the part the reader is after; the exact date follows so a forwarded screenshot
// still says something true. A zero date (they're reachable — see
// domain.AvailabilityOn) renders nothing.
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
