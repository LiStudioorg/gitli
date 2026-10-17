// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"errors"
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/cisecret"
)

type setTopicsReq struct {
	Topics []string `json:"topics"`
}

func (s *Server) handleListTopics(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	topics, err := s.deps.Topics.List(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if topics == nil {
		topics = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"topics": topics})
}

func (s *Server) handleSetTopics(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermWrite)
	if !ok {
		return
	}
	var req setTopicsReq
	if !decodeJSON(w, r, &req) {
		return
	}
	existing, err := s.deps.Topics.List(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, name := range existing {
		if err := s.deps.Topics.Remove(r.Context(), rp.ID, name); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	for _, name := range req.Topics {
		if err := s.deps.Topics.Add(r.Context(), rp.ID, name); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	topics, err := s.deps.Topics.List(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if topics == nil {
		topics = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"topics": topics})
}

type setCISecretReq struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (s *Server) handleListCISecrets(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermAdmin)
	if !ok {
		return
	}
	names, err := s.deps.CISecrets.List(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if names == nil {
		names = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"secrets": names})
}

func (s *Server) handleSetCISecret(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermAdmin)
	if !ok {
		return
	}
	var req setCISecretReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.CISecrets.Set(r.Context(), rp.ID, req.Name, req.Value); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleDeleteCISecret(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermAdmin)
	if !ok {
		return
	}
	name := urlParam(r, "name")
	if err := s.deps.CISecrets.Delete(r.Context(), rp.ID, name); err != nil {
		if errors.Is(err, cisecret.ErrNotFound) {
			writeError(w, http.StatusNotFound, "密钥不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
