// SPDX-License-Identifier: AGPL-3.0
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// RequestLogger writes one router completion line and escalates slow requests
// to WARN (>=500ms) or ERROR (>=1s) so latency regressions are visible.
func RequestLogger(log *slog.Logger, method, path, ip string, status int, dur time.Duration) {
	RequestLoggerAt(log, method, path, ip, status, dur, callerSite())
}

// RequestLoggerAt logs with an explicitly captured handler location.
func RequestLoggerAt(log *slog.Logger, method, path, ip string, status int, dur time.Duration, site string) {
	level := slog.LevelInfo
	switch {
	case dur >= time.Second:
		level = slog.LevelError
	case dur >= 500*time.Millisecond:
		level = slog.LevelWarn
	}
	log.Log(context.Background(), level, "router: completed "+method+" "+path+" for "+ip+
		", "+strconv.Itoa(status)+" "+http.StatusText(status)+" in "+formatMS(dur)+" @"+site)
}

// callerSite returns the first frame inside the http transport handlers.
func callerSite() string {
	pcs := make([]uintptr, 24)
	n := runtime.Callers(2, pcs)
	if n == 0 {
		return "unknown"
	}
	frames := runtime.CallersFrames(pcs[:n])
	for {
		f, more := frames.Next()
		if strings.Contains(f.Function, "internal/transport/http.") &&
			!strings.Contains(f.Function, "/middleware.") {
			name := f.Function[strings.LastIndex(f.Function, ".")+1:]
			// Helper writers (writeJSON/writeError) sit between the handler and
			// the socket; walk further up to report the actual handler.
			if !helperFuncs[name] {
				file := f.File
				if i := strings.LastIndexByte(file, '/'); i >= 0 {
					file = file[i+1:]
				}
				return file + ":" + strconv.Itoa(f.Line) + "(" + name + ")"
			}
		}
		if !more {
			break
		}
	}
	return "unknown"
}

// helperFuncs are request-response utilities, not handlers.
var helperFuncs = map[string]bool{
	"writeJSON": true, "writeError": true, "writeErrorCode": true,
	"Write": true, "WriteHeader": true, "WriteHeaderNow": true,
}

func formatMS(d time.Duration) string {
	return strconv.FormatFloat(float64(d.Microseconds())/1000.0, 'f', 1, 64) + "ms"
}
