package httpapi

import "net/http"

// contentSecurityPolicy is the policy served with every response.
//
// The frontend makes this policy unusually easy to tighten: it has no inline
// event handlers, no inline <script>, and loads nothing from a foreign origin —
// so scripts are restricted to 'self' without an 'unsafe-inline' escape hatch,
// which is the directive that actually matters against injected markup.
//
// Two deliberate loosenings:
//
//   - style-src 'unsafe-inline' — the absence timeline positions its spans with
//     style="left:…%" attributes built into HTML strings (renderAbsenceTimeline
//     in app.js), and the share page carries its stylesheet inline because it
//     must stand alone as a single document. Both would need a nonce or hash
//     otherwise; inline *style* is a far weaker vector than inline script.
//   - img-src data: — style.css draws the checkbox tick from a data:-URI SVG.
//
// Not set: require-trusted-types-for 'script'. It would be the natural
// enforcement for the escaping discipline in app.js, but it rejects every raw
// string assigned to innerHTML — which is currently how the whole UI renders.
// It only becomes an option once rendering goes through a single escaping
// helper.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; " +
	"connect-src 'self'; " +
	"object-src 'none'; " +
	"base-uri 'none'; " +
	"form-action 'self'; " +
	"frame-ancestors 'none'"

// SecurityHeaders wraps a handler so every response carries the same baseline
// security headers. It is applied to the *root* mux in cmd/server, not just to
// the API, so the SPA shell, the embedded assets and the share page are covered
// by construction — no route can be added and forgotten.
//
// Headers are set before the inner handler runs, so a handler that needs a
// stricter value can still override it (the share page does this for
// Referrer-Policy, where the header is load-bearing: its token sits in the URL).
//
// HSTS is deliberately absent — TLS terminates at the reverse proxy, which is
// the layer that knows whether HTTPS is actually in play.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
