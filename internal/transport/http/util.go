// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/LiStudioorg/gitli/internal/service/auth"
	"github.com/LiStudioorg/gitli/internal/service/comment"
	"github.com/LiStudioorg/gitli/internal/service/deploykey"
	"github.com/LiStudioorg/gitli/internal/service/discussion"
	"github.com/LiStudioorg/gitli/internal/service/ipallow"
	"github.com/LiStudioorg/gitli/internal/service/issue"
	"github.com/LiStudioorg/gitli/internal/service/notification"
	"github.com/LiStudioorg/gitli/internal/service/oauth"
	"github.com/LiStudioorg/gitli/internal/service/pat"
	"github.com/LiStudioorg/gitli/internal/service/rbac"
	"github.com/LiStudioorg/gitli/internal/service/savedreply"
	"github.com/LiStudioorg/gitli/internal/service/user"
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

// notFoundErrs 汇总各 service 的归属类 NotFound 哨兵。
//
// 这些错误来自 `WHERE id = ? AND <owner> = ?` 命中 0 行的写操作：要么资源不存在，
// 要么存在但不属于调用方。两者都必须回 404 而不是 500 —— 500 既误导用户
// （"服务器错误"会诱使他重试），也让越权探测凭空拿到一个区分信号。
var notFoundErrs = []error{
	auth.ErrSessionNotFound,
	comment.ErrNotFound,
	deploykey.ErrNotFound,
	discussion.ErrNotFound,
	ipallow.ErrNotFound,
	issue.ErrNotFound,
	notification.ErrNotFound,
	oauth.ErrNotFound,
	pat.ErrNotFound,
	rbac.ErrNotFound,
	savedreply.ErrNotFound,
	user.ErrNotFound,
}

// writeServiceError maps a service-layer error to the right HTTP status.
// notFoundMsg is the user-facing text used when the error means "no such row
// owned by you".
func writeServiceError(w http.ResponseWriter, err error, notFoundMsg string) {
	for _, sentinel := range notFoundErrs {
		if errors.Is(err, sentinel) {
			writeError(w, http.StatusNotFound, notFoundMsg)
			return
		}
	}
	writeError(w, http.StatusInternalServerError, err.Error())
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
