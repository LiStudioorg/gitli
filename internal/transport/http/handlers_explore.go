// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"
	"strconv"

	"github.com/LiStudioorg/gitli/internal/domain"
)

// handleExploreRepos serves a public, unauthenticated listing of repositories.
func (s *Server) handleExploreRepos(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	var (
		repos []*domain.Repo
		err   error
	)
	if q := r.URL.Query().Get("q"); q != "" {
		repos, err = s.deps.Repos.Search(r.Context(), q, limit, offset)
	} else {
		repos, err = s.deps.Repos.ListPublic(r.Context(), limit, offset)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if repos == nil {
		repos = []*domain.Repo{}
	}

	ownerCache := make(map[int64]string)
	out := make([]map[string]any, 0, len(repos))
	for _, rp := range repos {
		owner, ok := ownerCache[rp.OwnerID]
		if !ok {
			if u, uerr := s.deps.Users.ByID(r.Context(), rp.OwnerID); uerr == nil && u != nil {
				owner = u.Username
			}
			ownerCache[rp.OwnerID] = owner
		}
		out = append(out, map[string]any{
			"id":             rp.ID,
			"name":           rp.Name,
			"owner":          owner,
			"owner_id":       rp.OwnerID,
			"description":    rp.Description,
			"default_branch": rp.DefaultBranch,
			"updated_at":     rp.UpdatedAt,
			"stars_count":    rp.StarsCount,
		})
	}
	writeJSON(w, http.StatusOK, out)
}
