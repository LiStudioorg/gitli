// SPDX-License-Identifier: AGPL-3.0
package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
	"sync"
)

var gzipPool = sync.Pool{
	New: func() any {
		w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
		return w
	},
}

// compressibleTypes lists response types that benefit from gzip. Binary git
// packfiles are already compressed, so they are deliberately excluded.
var compressibleTypes = []string{
	"application/json",
	"application/javascript",
	"application/xml",
	"application/xhtml+xml",
	"application/manifest+json",
	"image/svg+xml",
}

// Compress gzips text responses when the client advertises support. Git smart
// HTTP streams are skipped because git already negotiates its own encoding.
func Compress() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") || skipCompressPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			gw := &gzipResponseWriter{ResponseWriter: w}
			if compressiblePath(r.URL.Path) {
				gw.compress = true
				gw.decided = true
			}
			defer gw.close()
			next.ServeHTTP(gw, r)
		})
	}
}

func skipCompressPath(p string) bool {
	for _, suffix := range []string{
		"/info/refs", "/git-upload-pack", "/git-receive-pack",
		"/archive/", "/blobs/", "/files/", "/attachments/", "/avatars/",
	} {
		if strings.HasSuffix(p, suffix) || strings.Contains(p, suffix) {
			return true
		}
	}
	return false
}

// compressiblePath recognises static assets by extension, because
// http.FileServer sets Content-Type only after our WriteHeader runs.
func compressiblePath(p string) bool {
	for _, ext := range []string{".js", ".css", ".svg", ".json", ".webmanifest", ".map", ".html", ".txt"} {
		if strings.HasSuffix(p, ext) {
			return true
		}
	}
	return false
}

func shouldCompress(contentType string) bool {
	if contentType == "" {
		return false
	}
	for _, t := range compressibleTypes {
		if strings.HasPrefix(contentType, t) {
			return true
		}
	}
	return false
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer      *gzip.Writer
	wroteHeader bool
	decided     bool
	compress    bool
	status      int
}

func (g *gzipResponseWriter) WriteHeader(status int) {
	if g.wroteHeader {
		return
	}
	g.wroteHeader = true
	g.status = status
	// Content-Type is set by the handler before WriteHeader, so the decision
	// must be made here rather than in Write.
	if !g.decided {
		g.compress = shouldCompress(g.Header().Get("Content-Type"))
		g.decided = true
	}
	if status == http.StatusNoContent || status == http.StatusNotModified {
		g.ResponseWriter.WriteHeader(status)
		return
	}
	if g.compress {
		h := g.Header()
		h.Del("Content-Length")
		h.Set("Content-Encoding", "gzip")
		h.Add("Vary", "Accept-Encoding")
	}
	g.ResponseWriter.WriteHeader(status)
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	if !g.wroteHeader {
		g.WriteHeader(http.StatusOK)
	}
	if !g.compress {
		return g.ResponseWriter.Write(b)
	}
	if g.writer == nil {
		gw := gzipPool.Get().(*gzip.Writer)
		gw.Reset(g.ResponseWriter)
		g.writer = gw
	}
	return g.writer.Write(b)
}

func (g *gzipResponseWriter) close() {
	if g.writer != nil {
		_ = g.writer.Close()
		gzipPool.Put(g.writer)
		g.writer = nil
	}
}
