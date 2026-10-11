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
		writeError(w, http.StatusBadRequest, "invalid owner")
		return nil, nil, false
	}
	if err := validateName("repo", repoName); err != nil {
		writeError(w, http.StatusBadRequest, "invalid repo")
		return nil, nil, false
	}
	ownerUser, err := s.deps.Users.ByUsername(r.Context(), owner)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return nil, nil, false
	}
	rp, err := s.deps.Repos.ByOwnerName(r.Context(), ownerUser.ID, repoName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
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
		writeError(w, http.StatusNotFound, "repository not found")
		return nil, nil, false
	}
	if perm < need {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return nil, nil, false
	}
	return rp, u, true
}
