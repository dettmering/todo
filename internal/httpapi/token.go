package httpapi

import (
	"crypto/subtle"
	"net/http"
)

// requireToken guards the routes that are exempted from the reverse-proxy
// BasicAuth (the iCal feed and the share page): for those the `token` query
// parameter is the *only* access control.
//
// It reports whether the request may proceed. When it returns false the
// response has already been written:
//
//   - configured == "" — the feature is disabled, so the route 404s and can
//     never be reached unprotected.
//   - token mismatch — 401. The comparison is constant-time.
func requireToken(w http.ResponseWriter, r *http.Request, configured string) bool {
	if configured == "" {
		http.NotFound(w, r)
		return false
	}
	token := r.URL.Query().Get("token")
	if subtle.ConstantTimeCompare([]byte(token), []byte(configured)) != 1 {
		http.Error(w, "invalid or missing token", http.StatusUnauthorized)
		return false
	}
	return true
}
