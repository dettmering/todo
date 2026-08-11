package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders_SetsBaseline(t *testing.T) {
	h := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/anything", nil))

	for header, want := range map[string]string{
		"Content-Security-Policy": contentSecurityPolicy,
		"X-Content-Type-Options":  "nosniff",
		"Referrer-Policy":         "no-referrer",
	} {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

// The wrapped handler must keep the last word: the share page relies on being
// able to set its own Referrer-Policy, because its token travels in the URL.
func TestSecurityHeaders_HandlerCanOverride(t *testing.T) {
	h := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "same-origin")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := rec.Header().Get("Referrer-Policy"); got != "same-origin" {
		t.Errorf("handler override lost: Referrer-Policy = %q, want %q", got, "same-origin")
	}
}

// The point of the policy is script-src without 'unsafe-inline'; everything else
// is hygiene. Loosening it should take a deliberate edit to this test too.
func TestContentSecurityPolicy_ScriptSrcIsStrict(t *testing.T) {
	directives := map[string]string{}
	for _, d := range strings.Split(contentSecurityPolicy, ";") {
		name, value, _ := strings.Cut(strings.TrimSpace(d), " ")
		directives[name] = value
	}

	for _, tc := range []struct {
		directive string
		want      string
	}{
		{"default-src", "'self'"},
		{"script-src", "'self'"},
		{"object-src", "'none'"},
		{"base-uri", "'none'"},
		{"frame-ancestors", "'none'"},
	} {
		if got, ok := directives[tc.directive]; !ok {
			t.Errorf("directive %q missing from policy", tc.directive)
		} else if got != tc.want {
			t.Errorf("%s = %q, want %q", tc.directive, got, tc.want)
		}
	}
}
