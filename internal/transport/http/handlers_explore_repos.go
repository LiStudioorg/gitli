// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"
	"strconv"

	"github.com/LiStudioorg/gitli/internal/domain"
)

// handleExploreRepos implements GET /api/v1/explore/repos (public): a list of
// public repositories, optionally filtered by a search query.
func (s *Server) handleExploreRepos(w http.ResponseWriter, r *http.Request) {
	limit := 20
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	query := r.URL.Query().Get("q")
	var (
		repos []*domain.Repo
		err   error
	)
	if query != "" {
		repos, err = s.deps.Repos.Search(r.Context(), query, limit, offset)
	} else {
		repos, err = s.deps.Repos.ListPublic(r.Context(), limit, offset)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	type row struct {
		ID          int64  `json:"id"`
		Owner       string `json:"owner"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Default     string `json:"default_branch"`
		Stars       int    `json:"stars_count"`
		UpdatedAt   any    `json:"updated_at"`
	}
	out := make([]row, 0, len(repos))
	for _, rp := range repos {
		owner := ""
		if u, uerr := s.deps.Users.ByID(r.Context(), rp.OwnerID); uerr == nil {
			owner = u.Username
		}
		out = append(out, row{ID: rp.ID, Owner: owner, Name: rp.Name, Description: rp.Description,
			Default: rp.DefaultBranch, Stars: rp.StarsCount, UpdatedAt: rp.UpdatedAt})
	}
	writeJSON(w, http.StatusOK, out)
}
