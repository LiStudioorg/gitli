// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"fmt"
	"net/http"
	"time"
)

// handleSSE streams server-sent events for the authenticated user.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	fmt.Fprintf(w, "event: ready\ndata: {\"user_id\":%d}\n\n", u.ID)
	flusher.Flush()

	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = fmt.Fprintf(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}
