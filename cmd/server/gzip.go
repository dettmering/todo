package main

import (
	"compress/gzip"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Responses below this many bytes are sent as they are: the gzip header and
// trailer alone are 18 bytes, and a short JSON answer gains nothing. Only
// checked when the handler announced a Content-Length; without one we compress.
const gzipMinSize = 512

var gzipPool = sync.Pool{
	New: func() any { return gzip.NewWriter(nil) },
}

// gzipped compresses text responses for clients that ask for it. Everything
// this server sends is text — JSON, JS, CSS, HTML, iCal — and the API answers
// compress about twentyfold, which is the difference between a fast and a slow
// view switch over a real connection.
//
// Deliberately not wrapped around the MCP route: that one streams, and a
// compressor sitting in front of a stream holds bytes back.
func gzipped(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The header goes on every answer, compressed or not: a cache that keeps
		// one variant must not hand it to a client that asked for the other.
		w.Header().Add("Vary", "Accept-Encoding")
		// A range request wants exactly those bytes of the original file;
		// compressing them would answer a different question.
		if !acceptsGzip(r.Header.Get("Accept-Encoding")) || r.Header.Get("Range") != "" {
			next.ServeHTTP(w, r)
			return
		}
		gw := &gzipWriter{ResponseWriter: w}
		defer gw.close()
		next.ServeHTTP(gw, r)
	})
}

// acceptsGzip reports whether the client asked for gzip. "gzip;q=0" is an
// explicit refusal, not an offer.
func acceptsGzip(header string) bool {
	for _, part := range strings.Split(header, ",") {
		name, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if name != "gzip" && name != "*" {
			continue
		}
		if q, ok := strings.CutPrefix(strings.TrimSpace(params), "q="); ok {
			if f, err := strconv.ParseFloat(q, 64); err == nil && f == 0 {
				continue
			}
		}
		return true
	}
	return false
}

type gzipState int

const (
	gzipUndecided gzipState = iota
	gzipCompressing
	gzipPassthrough
)

// gzipWriter decides on the first write whether this particular response is
// worth compressing — that is the first moment the handler's Content-Type and
// Content-Length are known.
type gzipWriter struct {
	http.ResponseWriter
	state gzipState
	gz    *gzip.Writer
}

func (w *gzipWriter) WriteHeader(code int) {
	w.decide(code)
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipWriter) Write(b []byte) (int, error) {
	w.decide(http.StatusOK)
	if w.state == gzipCompressing {
		return w.gz.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (w *gzipWriter) decide(code int) {
	if w.state != gzipUndecided {
		return
	}
	h := w.Header()
	w.state = gzipPassthrough
	switch {
	case code == http.StatusNoContent || code == http.StatusNotModified:
	case code == http.StatusPartialContent:
	case h.Get("Content-Encoding") != "":
	case !compressibleType(h.Get("Content-Type")):
	case knownShort(h.Get("Content-Length")):
	default:
		// The compressed body has a different length, and the handler's count is
		// now wrong — drop it and let the response be chunked.
		h.Del("Content-Length")
		h.Set("Content-Encoding", "gzip")
		w.gz = gzipPool.Get().(*gzip.Writer)
		w.gz.Reset(w.ResponseWriter)
		w.state = gzipCompressing
	}
}

func (w *gzipWriter) close() {
	if w.state == gzipCompressing {
		_ = w.gz.Close()
		gzipPool.Put(w.gz)
		w.gz = nil
	}
}

// Flush keeps a flushing handler working: the compressor has to let its own
// buffer go before the underlying writer flushes, or nothing reaches the client.
func (w *gzipWriter) Flush() {
	if w.state == gzipCompressing {
		_ = w.gz.Flush()
	}
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap lets http.ResponseController reach the real writer (deadlines etc.).
func (w *gzipWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func knownShort(contentLength string) bool {
	n, err := strconv.Atoi(contentLength)
	return err == nil && n < gzipMinSize
}

// compressibleType reports whether a body of this type is worth the CPU.
// An empty type means the handler left it to Go's sniffer, which only runs on
// the first write — too late to decide here, so it counts as "don't".
func compressibleType(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	switch {
	case strings.HasPrefix(ct, "text/"),
		ct == "application/json",
		ct == "application/javascript",
		ct == "application/xml",
		ct == "image/svg+xml":
		return true
	}
	return false
}
