// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"
	"runtime"
	"strconv"
	"strings"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/platform"
)

// handleAdminListUsers implements GET /api/v1/admin/users?q=.
func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	users, err := s.deps.Users.ListAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	type row struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		IsAdmin   bool   `json:"is_admin"`
		IsActive  bool   `json:"is_active"`
		TOTP      bool   `json:"totp_enabled"`
		CreatedAt any    `json:"created_at"`
	}
	out := make([]row, 0, len(users))
	for _, u := range users {
		if q != "" && !strings.Contains(strings.ToLower(u.Username), q) && !strings.Contains(strings.ToLower(u.Email), q) {
			continue
		}
		out = append(out, row{u.ID, u.Username, u.Email, u.IsAdmin, u.IsActive, u.TOTPEnabled, u.CreatedAt})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAdminSetActive implements PATCH /api/v1/admin/users/{id} {active}.
func (s *Server) handleAdminSetActive(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	var req struct {
		Active *bool `json:"active"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Active == nil {
		writeError(w, http.StatusBadRequest, "必须指定 active 参数")
		return
	}
	u, err := s.deps.Users.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "用户不存在")
		return
	}
	if !*req.Active && u.IsAdmin {
		admins, _ := s.deps.Users.CountAdmins(r.Context())
		if admins <= 1 {
			writeError(w, http.StatusBadRequest, "不能停用最后一个活跃管理员")
			return
		}
	}
	if err := s.deps.Users.SetActive(r.Context(), id, *req.Active); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "admin.user.disable", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "active": *req.Active})
}

// handleAdminResetPassword sets a new random password and returns it once.
func (s *Server) handleAdminResetPassword(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	u, err := s.deps.Users.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "用户不存在")
		return
	}
	tmp, err := s.deps.Auth.ResetPassword(r.Context(), u)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "admin.user.reset_password", "success")
	writeJSON(w, http.StatusOK, map[string]any{"password": tmp})
}

// handleAdminReset2FA clears a user's 2FA.
func (s *Server) handleAdminReset2FA(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	u, err := s.deps.Users.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "用户不存在")
		return
	}
	if err := s.deps.Users.ResetTOTP(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "admin.user.reset_2fa", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// handleAdminListRepos implements GET /api/v1/admin/repos?q=.
func (s *Server) handleAdminListRepos(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	repos, err := s.deps.Repos.ListAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	type row struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		OwnerID   int64  `json:"owner_id"`
		IsPrivate bool   `json:"is_private"`
		Archived  bool   `json:"archived"`
		Deleted   bool   `json:"deleted"`
		SizeKB    int64  `json:"size_kb"`
		Default   string `json:"default_branch"`
	}
	out := make([]row, 0, len(repos))
	for _, rp := range repos {
		if q != "" && !strings.Contains(strings.ToLower(rp.Name), q) {
			continue
		}
		out = append(out, row{rp.ID, rp.Name, rp.OwnerID, rp.IsPrivate, rp.Archived, rp.DeletedAt != nil, rp.SizeKB, rp.DefaultBranch})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAdminArchiveRepo implements POST /api/v1/admin/repos/{id}/archive.
func (s *Server) handleAdminArchiveRepo(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	var req struct {
		Archived bool `json:"archived"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.Repos.Archive(r.Context(), id, req.Archived); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	action := "admin.repo.archive"
	if !req.Archived {
		action = "admin.repo.unarchive"
	}
	s.auditAction(r, id, action, "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "archived": req.Archived})
}

// handleAdminListOrgs implements GET /api/v1/admin/orgs.
func (s *Server) handleAdminListOrgs(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	orgs, err := s.deps.Orgs.ListAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, orgs)
}

// handleAdminSystem implements GET /api/v1/admin/system with runtime/system info.
func (s *Server) handleAdminSystem(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	dbStats := s.deps.DB.Stats()
	info := map[string]any{
		"go_version":    runtime.Version(),
		"goroutines":    runtime.NumGoroutine(),
		"heap_alloc_mb": m.HeapAlloc / (1 << 20),
		"sys_mb":        m.Sys / (1 << 20),
		"num_cpu":       runtime.NumCPU(),
		"db_open":       dbStats.OpenConnections,
		"db_in_use":     dbStats.InUse,
		"db_idle":       dbStats.Idle,
		"db_wait_count": dbStats.WaitCount,
	}
	if total, free, derr := platform.DiskUsage(s.deps.Cfg.Storage.Repos); derr == nil {
		info["disk_total_mb"] = total / (1 << 20)
		info["disk_free_mb"] = free / (1 << 20)
	}
	info["git_binary"] = s.deps.GitCmd.Path()
	writeJSON(w, http.StatusOK, info)
}

var _ = domain.PermAdmin
var _ = strconv.Itoa
