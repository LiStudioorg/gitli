// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
)

// ANSI colour codes used by the text handler. They are dropped automatically
// when output is not a terminal (e.g. redirected to a file or a log collector).
const (
	ansiReset  = "\x1b[0m"
	ansiDim    = "\x1b[90m"
	ansiGreen  = "\x1b[32m"
	ansiCyan   = "\x1b[36m"
	ansiYellow = "\x1b[33m"
	ansiRed    = "\x1b[31m"
)

// textHandler renders compact, human-readable lines:
//
//	2026/10/17 23:51:42 [I] router: completed GET /api/v1/user/me for 1.2.3.4, 200 OK in 14.2ms @ handlers_user.go:42(GetUserInfo)
//
// Attributes are appended as key=value pairs so structured context is not lost.
type textHandler struct {
	mu      *sync.Mutex
	out     io.Writer
	level   slog.Level
	color   bool
	attrs   []slog.Attr
	handler string // optional module prefix, e.g. "router"
}

// newTextHandler builds a text handler. color must be decided by the caller
// (terminal detection + "not writing to a file").
func newTextHandler(out io.Writer, level slog.Level, color bool) *textHandler {
	return &textHandler{mu: &sync.Mutex{}, out: out, level: level, color: color}
}

func (h *textHandler) Enabled(_ context.Context, l slog.Level) bool { return l >= h.level }

func (h *textHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &clone
}

func (h *textHandler) WithGroup(string) slog.Handler { return h }

func (h *textHandler) paint(code, s string) string {
	if !h.color {
		return s
	}
	return code + s + ansiReset
}

func levelTag(l slog.Level) (tag, code string) {
	switch {
	case l >= slog.LevelError:
		return "[E]", ansiRed
	case l >= slog.LevelWarn:
		return "[W]", ansiYellow
	case l >= slog.LevelInfo:
		return "[I]", ansiGreen
	default:
		return "[D]", ansiCyan
	}
}

func (h *textHandler) Handle(_ context.Context, r slog.Record) error {
	tag, code := levelTag(r.Level)
	var b strings.Builder
	b.WriteString(h.paint(ansiDim, r.Time.Format("2006/01/02 15:04:05")))
	b.WriteByte(' ')
	b.WriteString(h.paint(code, tag))
	b.WriteByte(' ')
	if h.handler != "" {
		b.WriteString(h.handler)
		b.WriteString(": ")
	}
	b.WriteString(r.Message)

	writeAttr := func(a slog.Attr) bool {
		if a.Key == "" || a.Key == slog.TimeKey || a.Key == slog.LevelKey || a.Key == slog.MessageKey {
			return true
		}
		b.WriteByte(' ')
		b.WriteString(a.Key)
		b.WriteByte('=')
		b.WriteString(a.Value.String())
		return true
	}
	for _, a := range h.attrs {
		writeAttr(a)
	}
	r.Attrs(writeAttr)
	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.out, b.String())
	return err
}
