// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
)

// handleUserProfile implements GET /api/v1/users/{username}.
func (s *Server) handleUserProfile(w http.ResponseWriter, r *http.Request) {
	name := urlParam(r, "username")
	if err := validateName("username", name); err != nil {
		writeError(w, http.StatusBadRequest, "无效的用户名")
		return
	}
	user, err := s.deps.Users.ByUsername(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusNotFound, "用户不存在")
		return
	}
	repos, _ := s.deps.Repos.ListByOwner(r.Context(), user.ID, 30, 0)
	if repos == nil {
		// Always send an array: the client indexes into it and a JSON null
		// would crash rendering (blank profile page).
		repos = []*domain.Repo{}
	}
	followers, following, _ := s.deps.Follows.Counts(r.Context(), user.ID)
	var isFollowing bool
	if u := userFrom(r.Context()); u != nil {
		isFollowing, _ = s.deps.Follows.IsFollowing(r.Context(), u.ID, user.ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":         publicUser(user),
		"repos":        repos,
		"followers":    followers,
		"following":    following,
		"is_following": isFollowing,
	})
}

func (s *Server) handleFollowUser(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	target, err := s.deps.Users.ByUsername(r.Context(), urlParam(r, "username"))
	if err != nil {
		writeError(w, http.StatusNotFound, "用户不存在")
		return
	}
	if err := s.deps.Follows.Follow(r.Context(), u.ID, target.ID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "following": true})
}

func (s *Server) handleUnfollowUser(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	target, err := s.deps.Users.ByUsername(r.Context(), urlParam(r, "username"))
	if err != nil {
		writeError(w, http.StatusNotFound, "用户不存在")
		return
	}
	if err := s.deps.Follows.Unfollow(r.Context(), u.ID, target.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "following": false})
}

func (s *Server) handleListFollowers(w http.ResponseWriter, r *http.Request) {
	s.listFollowUsers(w, r, true)
}

func (s *Server) handleListFollowing(w http.ResponseWriter, r *http.Request) {
	s.listFollowUsers(w, r, false)
}

func (s *Server) listFollowUsers(w http.ResponseWriter, r *http.Request, followers bool) {
	target, err := s.deps.Users.ByUsername(r.Context(), urlParam(r, "username"))
	if err != nil {
		writeError(w, http.StatusNotFound, "用户不存在")
		return
	}
	var ids []int64
	if followers {
		ids, _ = s.deps.Follows.ListFollowers(r.Context(), target.ID, 100)
	} else {
		ids, _ = s.deps.Follows.ListFollowing(r.Context(), target.ID, 100)
	}
	users := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		if u, err := s.deps.Users.ByID(r.Context(), id); err == nil {
			users = append(users, publicUser(u))
		}
	}
	writeJSON(w, http.StatusOK, users)
}
