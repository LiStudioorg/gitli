// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"encoding/json"
	"errors"
	"fmt"
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

// apiError is the canonical JSON error envelope. Code is a stable,
// machine-readable identifier that clients must use instead of matching the
// human-readable (and localised) Error text.
type apiError struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, apiError{Error: msg})
}

// writeErrorCode writes an error envelope with a stable machine-readable code.
func writeErrorCode(w http.ResponseWriter, status int, msg, code string) {
	writeJSON(w, status, apiError{Error: msg, Code: code})
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
			writeErrorCode(w, http.StatusNotFound, notFoundMsg, "not_found")
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

// trustedProxies holds the parsed CIDRs whose X-Forwarded-For is believed.
// Empty by default: XFF is then ignored entirely, which prevents attackers
// from spoofing their IP to bypass rate limits and IP allowlists (H-03).
var trustedProxies []*net.IPNet

// SetTrustedProxies configures which upstream proxies may set XFF.
func SetTrustedProxies(cidrs []string) error {
	parsed := make([]*net.IPNet, 0, len(cidrs))
	for _, raw := range cidrs {
		c := strings.TrimSpace(raw)
		if c == "" {
			continue
		}
		if !strings.Contains(c, "/") {
			ip := net.ParseIP(c)
			if ip == nil {
				return fmt.Errorf("trusted_proxies: 无效的 IP %q", c)
			}
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			c = fmt.Sprintf("%s/%d", ip.String(), bits)
		}
		_, network, err := net.ParseCIDR(c)
		if err != nil {
			return fmt.Errorf("trusted_proxies: 无效的 CIDR %q", raw)
		}
		parsed = append(parsed, network)
	}
	trustedProxies = parsed
	return nil
}

func isTrustedProxy(addr string) bool {
	ip := net.ParseIP(strings.TrimSpace(addr))
	if ip == nil {
		return false
	}
	for _, n := range trustedProxies {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// clientIP returns the effective client address. X-Forwarded-For is only
// consulted when the immediate peer is a configured trusted proxy, and then the
// right-most non-trusted hop is used (the value appended by our own proxy).
func clientIP(r *http.Request) string {
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteHost = r.RemoteAddr
	}
	if len(trustedProxies) == 0 || !isTrustedProxy(remoteHost) {
		return remoteHost
	}
	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return remoteHost
	}
	parts := strings.Split(xff, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		hop := strings.TrimSpace(parts[i])
		if hop == "" {
			continue
		}
		if isTrustedProxy(hop) {
			continue
		}
		return hop
	}
	return remoteHost
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
