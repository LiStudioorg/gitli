// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
)

// requireAdmin writes 401/403 and returns nil when the caller is not an admin.
func requireAdmin(w http.ResponseWriter, r *http.Request) *domain.User {
	u := requireUser(w, r)
	if u == nil {
		return nil
	}
	if !u.IsAdmin {
		writeError(w, http.StatusForbidden, "需要管理员权限")
		return nil
	}
	return u
}

// AdminOnly is a defense-in-depth guard for the /admin route group. Handlers
// still call requireAdmin, but an unauthenticated or non-admin request is
// rejected here first so a future handler cannot accidentally omit the check.
func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFrom(r.Context())
		if u == nil {
			writeError(w, http.StatusUnauthorized, "需要登录")
			return
		}
		if !u.IsAdmin {
			writeError(w, http.StatusForbidden, "需要管理员权限")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// countAll runs a parameter-free COUNT(*) query and returns the total.
func (s *Server) countAll(r *http.Request, table string) (int64, error) {
	q := s.deps.DB.Rebind(`SELECT COUNT(*) FROM ` + table)
	var n int64
	err := s.deps.DB.QueryRowContext(r.Context(), q).Scan(&n)
	return n, err
}

// handleAdminStats returns aggregate instance counters for administrators.
func (s *Server) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	if requireAdmin(w, r) == nil {
		return
	}
	counts := map[string]int64{}
	for _, tbl := range []string{
		"users", "repos", "orgs", "issues", "pulls",
		"gists", "releases", "webhooks", "access_tokens",
	} {
		n, err := s.countAll(r, tbl)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "count "+tbl+": "+err.Error())
			return
		}
		counts[tbl] = n
	}
	out := map[string]any{
		"users":          counts["users"],
		"repos":          counts["repos"],
		"orgs":           counts["orgs"],
		"issues":         counts["issues"],
		"pulls":          counts["pulls"],
		"gists":          counts["gists"],
		"releases":       counts["releases"],
		"webhooks":       counts["webhooks"],
		"tokens":         counts["access_tokens"],
		"uptime_seconds": s.deps.Metrics.Uptime().Seconds(),
	}
	writeJSON(w, http.StatusOK, out)
}
