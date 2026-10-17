// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
)

// handleListDependencies implements GET /api/v1/repos/{owner}/{repo}/dependencies.
func (s *Server) handleListDependencies(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	list, err := s.deps.Deps.List(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Group by ecosystem + counts.
	groups := map[string][]any{}
	counts := map[string]int{}
	for _, d := range list {
		groups[d.Ecosystem] = append(groups[d.Ecosystem], d)
		counts[d.Ecosystem]++
	}
	writeJSON(w, http.StatusOK, map[string]any{"dependencies": list, "groups": groups, "counts": counts})
}

// handleRefreshDependencies forces a re-parse at the default branch.
func (s *Server) handleRefreshDependencies(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermWrite)
	if !ok {
		return
	}
	path, err := s.repoPath(r, rp)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	// repoPath helper is per-server; use the layout directly for the refresh.
	n, err := s.deps.RefreshDeps(r.Context(), rp.ID, path, rp.DefaultBranch)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "count": n})
}
