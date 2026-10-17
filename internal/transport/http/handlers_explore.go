// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"
	"strconv"
)

// handleTrending implements GET /api/v1/explore/trending?language=&limit=.
func (s *Server) handleTrending(w http.ResponseWriter, r *http.Request) {
	if s.deps.Explore == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	limit := 25
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	repos, err := s.deps.Explore.Trending(r.Context(), r.URL.Query().Get("language"), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, repos)
}

// handleTopic implements GET /api/v1/explore/topics/{topic}?language=.
func (s *Server) handleTopic(w http.ResponseWriter, r *http.Request) {
	if s.deps.Explore == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	repos, err := s.deps.Explore.ByTopic(r.Context(), urlParam(r, "topic"), r.URL.Query().Get("language"), 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, repos)
}

// handleTopTopics implements GET /api/v1/explore/topics.
func (s *Server) handleTopTopics(w http.ResponseWriter, r *http.Request) {
	if s.deps.Explore == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	topics, err := s.deps.Explore.TopTopics(r.Context(), 30)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, topics)
}
