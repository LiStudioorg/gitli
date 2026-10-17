// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"
	"strconv"
)

// handleGlobalSearch implements GET /api/v1/search?q=&limit=.
func (s *Server) handleGlobalSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "q parameter required")
		return
	}
	limit := 30
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	res, err := s.deps.GlobalSearch.Search(r.Context(), q, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}
