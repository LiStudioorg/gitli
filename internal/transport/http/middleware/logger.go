// SPDX-License-Identifier: AGPL-3.0
package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// Logger returns a request logging middleware. The completion line is written
// by app.RequestLogger so it can include the handler's file:line(function) and
// escalate slow requests to WARN/ERROR.
func Logger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			sw := &siteWriter{ResponseWriter: ww}
			next.ServeHTTP(sw, r)
			site := sw.site
			if site == "" {
				site = "unknown"
			}
			RequestLoggerAt(log, r.Method, r.URL.Path, clientIP(r), ww.Status(), time.Since(start), site)
		})
	}
}

// clientIP resolves the caller address, honouring configured trusted proxies.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

// siteWriter captures the handler location at the moment the response headers
// are written. At that point the handler's frame is still on the stack, which
// is not true once ServeHTTP has returned.
type siteWriter struct {
	http.ResponseWriter
	site string
}

func (w *siteWriter) WriteHeader(status int) {
	if w.site == "" {
		w.site = callerSite()
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *siteWriter) Write(b []byte) (int, error) {
	if w.site == "" {
		w.site = callerSite()
	}
	return w.ResponseWriter.Write(b)
}

func (w *siteWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
