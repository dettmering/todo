package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestAcceptsGzip(t *testing.T) {
	cases := map[string]bool{
		"":                          false,
		"gzip":                      true,
		"deflate, gzip":             true,
		"gzip;q=0.8":                true,
		"gzip;q=0":                  false,
		"identity;q=1, gzip;q=0":    false,
		"*":                         true,
		"br":                        false,
		"deflate, gzip;q=0, br;q=1": false,
	}
	for in, want := range cases {
		if got := acceptsGzip(in); got != want {
			t.Errorf("acceptsGzip(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestCompressibleType(t *testing.T) {
	cases := map[string]bool{
		"application/json; charset=utf-8": true,
		"text/html; charset=utf-8":        true,
		"text/calendar":                   true,
		"image/svg+xml":                   true,
		"image/png":                       false,
		"font/woff2":                      false,
		"application/octet-stream":        false,
		"":                                false, // sniffed too late to decide here
	}
	for in, want := range cases {
		if got := compressibleType(in); got != want {
			t.Errorf("compressibleType(%q) = %v, want %v", in, got, want)
		}
	}
}

// jsonHandler answers with `size` bytes of JSON-ish text.
func jsonHandler(size int, setLength bool) http.Handler {
	body := strings.Repeat("a", size)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if setLength {
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		}
		_, _ = w.Write([]byte(body))
	})
}

func serve(t *testing.T, h http.Handler, acceptEncoding string, headers map[string]string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	if acceptEncoding != "" {
		req.Header.Set("Accept-Encoding", acceptEncoding)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	gzipped(h).ServeHTTP(rec, req)
	return rec.Result()
}

func TestGzippedCompressesLongTextResponse(t *testing.T) {
	res := serve(t, jsonHandler(4096, false), "gzip", nil)
	if got := res.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}
	if got := res.Header.Get("Content-Length"); got != "" {
		t.Errorf("stale Content-Length survived: %q", got)
	}
	raw, _ := io.ReadAll(res.Body)
	if len(raw) >= 4096 {
		t.Errorf("compressed body is %d bytes, no smaller than the original 4096", len(raw))
	}
	zr, err := gzip.NewReader(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("body is not valid gzip: %v", err)
	}
	body, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("read gzip body: %v", err)
	}
	if len(body) != 4096 || strings.Trim(string(body), "a") != "" {
		t.Errorf("decompressed body does not match what the handler wrote")
	}
}

func TestGzippedPassesThrough(t *testing.T) {
	cases := []struct {
		name    string
		handler http.Handler
		accept  string
		headers map[string]string
	}{
		{"client did not ask", jsonHandler(4096, false), "", nil},
		{"client refused", jsonHandler(4096, false), "gzip;q=0", nil},
		{"range request", jsonHandler(4096, false), "gzip", map[string]string{"Range": "bytes=0-99"}},
		{"short answer", jsonHandler(64, true), "gzip", nil},
		{"binary type", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte(strings.Repeat("a", 4096)))
		}), "gzip", nil},
		{"already encoded", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Encoding", "br")
			_, _ = w.Write([]byte(strings.Repeat("a", 4096)))
		}), "gzip", nil},
		{"no content", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNoContent)
		}), "gzip", nil},
		{"not modified", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/css")
			w.WriteHeader(http.StatusNotModified)
		}), "gzip", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := serve(t, c.handler, c.accept, c.headers)
			if got := res.Header.Get("Content-Encoding"); got == "gzip" {
				t.Errorf("response was compressed anyway (Content-Encoding: %q)", got)
			}
			// The header has to be there either way, or a cache hands the wrong
			// variant to the next client.
			if got := res.Header.Get("Vary"); !strings.Contains(got, "Accept-Encoding") {
				t.Errorf("Vary = %q, want it to mention Accept-Encoding", got)
			}
		})
	}
}

func TestGzippedKeepsStatusCode(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(strings.Repeat("a", 4096)))
	})
	res := serve(t, h, "gzip", nil)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", res.StatusCode)
	}
	if got := res.Header.Get("Content-Encoding"); got != "gzip" {
		t.Errorf("error bodies should compress too, got %q", got)
	}
}
