// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/pathperm"
)

func (s *Server) handleListPathPermissions(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	rules, err := s.deps.PathPerms.List(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rules == nil {
		rules = []pathperm.Rule{}
	}
	writeJSON(w, http.StatusOK, rules)
}

type pathpermCreateReq struct {
	Pattern    string `json:"pattern"`
	Permission string `json:"permission"`
}

func (s *Server) handleCreatePathPermission(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermAdmin)
	if !ok {
		return
	}
	var req pathpermCreateReq
	if !decodeJSON(w, r, &req) {
		return
	}
	rule, err := s.deps.PathPerms.Create(r.Context(), rp.ID, req.Pattern, req.Permission)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) handleDeletePathPermission(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermAdmin)
	if !ok {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule id")
		return
	}
	if err := s.deps.PathPerms.Delete(r.Context(), rp.ID, id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleCommitSignature(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	sha := urlParam(r, "sha")
	sig, err := s.deps.Signatures.Get(r.Context(), rp.ID, sha)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if sig == nil {
		path, err := s.repoPath(r, rp)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		sig, err = s.deps.Signatures.Verify(r.Context(), path, sha)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		sig.RepoID = rp.ID
		if err := s.deps.Signatures.Save(r.Context(), sig); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, sig)
}
