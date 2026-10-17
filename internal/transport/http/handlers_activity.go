// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"
	"strconv"

	"github.com/LiStudioorg/gitli/internal/domain"
)

func (s *Server) handleMyRepos(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	repos, err := s.deps.Repos.ListAccessible(r.Context(), u.ID, 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, repos)
}

func (s *Server) handleActivities(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	perPage := 20
	if v := r.URL.Query().Get("per_page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			perPage = n
		}
	}
	items, err := s.deps.Activities.ListByActor(r.Context(), u.ID, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []*domain.Activity{}
	}
	writeJSON(w, http.StatusOK, items)
}

// handleExploreActivity serves the public cross-repo activity timeline.
func (s *Server) handleExploreActivity(w http.ResponseWriter, r *http.Request) {
	limit := 30
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	items, err := s.deps.Activities.ListPublic(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []*domain.Activity{}
	}
	writeJSON(w, http.StatusOK, items)
}

// handleRepoActivities serves a single repository's activity timeline.
func (s *Server) handleRepoActivities(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.resolveRepo(w, r, urlParam(r, "owner"), urlParam(r, "repo"), domain.PermRead)
	if !ok {
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	items, err := s.deps.Activities.ListByRepo(r.Context(), rp.ID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []*domain.Activity{}
	}
	writeJSON(w, http.StatusOK, items)
}
