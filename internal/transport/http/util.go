// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// apiError is the canonical JSON error envelope.
type apiError struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, apiError{Error: msg})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "无效的 JSON 请求体："+err.Error())
		return false
	}
	return true
}

func isHTTPS(externalURL string) bool {
	return strings.HasPrefix(strings.ToLower(externalURL), "https://")
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func urlParam(r *http.Request, name string) string {
	return chi.URLParam(r, name)
}

func pathInt(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, name), 10, 64)
}

// dispatch emits a repository event to webhooks if a dispatcher is wired.
func (s *Server) dispatch(r *http.Request, action string, repoID, actorID int64, payload map[string]any) {
	if s.deps.Dispatch == nil {
		return
	}
	s.deps.Dispatch(r.Context(), action, repoID, actorID, payload)
}

// jsonMarshalStrings encodes a []string to JSON, returning an error string.
func jsonMarshalStrings(v []string) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// writeSSE writes a single server-sent event with the given event name and data.
func writeSSE(w io.Writer, event, data string) {
	// Data may contain newlines; emit each line as a separate "data:" field.
	_, _ = io.WriteString(w, "event: "+event+"\n")
	for _, line := range strings.Split(data, "\n") {
		_, _ = io.WriteString(w, "data: "+line+"\n")
	}
	_, _ = io.WriteString(w, "\n")
}
