// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
)

// resolveRepo loads a repository by owner/repo names (from query or body) and
// checks the caller's permission, writing an error response on failure.
func (s *Server) resolveRepo(w http.ResponseWriter, r *http.Request, owner, repoName string, need domain.Perm) (*domain.Repo, *domain.User, bool) {
	if err := validateName("owner", owner); err != nil {
		writeError(w, http.StatusBadRequest, "无效的所有者")
		return nil, nil, false
	}
	if err := validateName("repo", repoName); err != nil {
		writeError(w, http.StatusBadRequest, "无效的仓库")
		return nil, nil, false
	}
	ownerUser, err := s.deps.Users.ByUsername(r.Context(), owner)
	if err != nil {
		writeError(w, http.StatusNotFound, "仓库不存在")
		return nil, nil, false
	}
	rp, err := s.deps.Repos.ByOwnerName(r.Context(), ownerUser.ID, repoName)
	if err != nil {
		writeError(w, http.StatusNotFound, "仓库不存在")
		return nil, nil, false
	}
	if deleted, derr := s.deps.Repos.IsDeleted(r.Context(), rp.ID); derr == nil && deleted {
		writeError(w, http.StatusGone, "仓库已删除")
		return nil, nil, false
	}
	u := userFrom(r.Context())
	var userID int64
	var isAdmin bool
	if u != nil {
		userID = u.ID
		isAdmin = u.IsAdmin
	}
	perm, err := s.deps.Repos.PermissionFor(r.Context(), rp, userID, isAdmin, s.deps.Orgs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return nil, nil, false
	}
	if rp.IsPrivate && perm < domain.PermRead {
		writeError(w, http.StatusNotFound, "仓库不存在")
		return nil, nil, false
	}
	if need >= domain.PermWrite && s.deps.Repos.IsArchived(r.Context(), rp) {
		writeError(w, http.StatusForbidden, "仓库已归档（只读）")
		return nil, nil, false
	}
	if perm < need {
		writeError(w, http.StatusForbidden, "权限不足")
		return nil, nil, false
	}
	return rp, u, true
}
