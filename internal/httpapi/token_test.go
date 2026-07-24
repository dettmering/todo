package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireToken(t *testing.T) {
	cases := []struct {
		name       string
		configured string
		url        string
		wantOK     bool
		wantStatus int
	}{
		{"disabled", "", "/x?token=whatever", false, http.StatusNotFound},
		{"disabled without token", "", "/x", false, http.StatusNotFound},
		{"missing token", "secret", "/x", false, http.StatusUnauthorized},
		{"empty token", "secret", "/x?token=", false, http.StatusUnauthorized},
		{"wrong token", "secret", "/x?token=nope", false, http.StatusUnauthorized},
		{"prefix of token", "secret", "/x?token=sec", false, http.StatusUnauthorized},
		{"match", "secret", "/x?token=secret", true, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)

			got := requireToken(rec, req, tc.configured)

			if got != tc.wantOK {
				t.Errorf("requireToken = %v, want %v", got, tc.wantOK)
			}
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}
