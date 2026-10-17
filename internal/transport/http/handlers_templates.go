// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
)

func (s *Server) handleListIssueTemplates(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	if s.deps.ListIssueTemplates == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	path, err := s.repoPath(r, rp)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	tpls := s.deps.ListIssueTemplates(r.Context(), path, refOr(r, rp.DefaultBranch))
	writeJSON(w, http.StatusOK, tpls)
}

func (s *Server) handlePullRequestTemplate(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	if s.deps.PullRequestTemplate == nil {
		writeJSON(w, http.StatusOK, map[string]any{"body": ""})
		return
	}
	path, err := s.repoPath(r, rp)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	body := s.deps.PullRequestTemplate(r.Context(), path, refOr(r, rp.DefaultBranch))
	writeJSON(w, http.StatusOK, map[string]any{"body": body})
}

func refOr(r *http.Request, def string) string {
	if v := r.URL.Query().Get("ref"); v != "" {
		return v
	}
	return def
}
