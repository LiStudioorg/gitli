// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"
	"strconv"

	"github.com/LiStudioorg/gitli/internal/domain"
)

// requireAdmin writes 401/403 and returns nil when the caller is not an admin.
func requireAdmin(w http.ResponseWriter, r *http.Request) *domain.User {
	u := requireUser(w, r)
	if u == nil {
		return nil
	}
	if !u.IsAdmin {
		writeError(w, http.StatusForbidden, "administrator privileges required")
		return nil
	}
	return u
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
		"users", "repos", "orgs", "issues", "pull_requests",
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
		"pulls":          counts["pull_requests"],
		"gists":          counts["gists"],
		"releases":       counts["releases"],
		"webhooks":       counts["webhooks"],
		"tokens":         counts["access_tokens"],
		"uptime_seconds": s.deps.Metrics.Uptime().Seconds(),
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAdminAudit lists recent audit log entries for administrators.
func (s *Server) handleAdminAudit(w http.ResponseWriter, r *http.Request) {
	if requireAdmin(w, r) == nil {
		return
	}
	limit := 50
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
		}
	}
	logs, err := s.deps.Audit.List(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": logs, "limit": limit, "offset": offset})
}
