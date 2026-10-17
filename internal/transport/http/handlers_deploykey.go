// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
)

func (s *Server) handleListDeployKeys(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermAdmin)
	if !ok {
		return
	}
	keys, err := s.deps.DeployKeys.List(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, keys)
}

type deployKeyReq struct {
	Title     string `json:"title"`
	PublicKey string `json:"public_key"`
	ReadOnly  *bool  `json:"read_only"`
}

func (s *Server) handleAddDeployKey(w http.ResponseWriter, r *http.Request) {
	rp, u, ok := s.loadRepoAndPerm(w, r, domain.PermAdmin)
	if !ok {
		return
	}
	var req deployKeyReq
	if !decodeJSON(w, r, &req) {
		return
	}
	readOnly := true
	if req.ReadOnly != nil {
		readOnly = *req.ReadOnly
	}
	key, err := s.deps.DeployKeys.Add(r.Context(), rp.ID, req.Title, req.PublicKey, readOnly)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.auditAction(r, u.ID, "deploy_key.create", "success")
	writeJSON(w, http.StatusCreated, key)
}

type updateDeployKeyReq struct {
	ReadOnly bool `json:"read_only"`
}

func (s *Server) handleUpdateDeployKey(w http.ResponseWriter, r *http.Request) {
	rp, u, ok := s.loadRepoAndPerm(w, r, domain.PermAdmin)
	if !ok {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	var req updateDeployKeyReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.DeployKeys.SetReadOnly(r.Context(), rp.ID, id, req.ReadOnly); err != nil {
		writeError(w, http.StatusNotFound, "部署密钥不存在")
		return
	}
	s.auditAction(r, u.ID, "deploy_key.update", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleDeleteDeployKey(w http.ResponseWriter, r *http.Request) {
	rp, u, ok := s.loadRepoAndPerm(w, r, domain.PermAdmin)
	if !ok {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	if err := s.deps.DeployKeys.Delete(r.Context(), rp.ID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "deploy_key.delete", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
