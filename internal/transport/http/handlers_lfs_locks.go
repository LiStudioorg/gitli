// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"errors"
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/lfs"
)

type createLockReq struct {
	Path string `json:"path"`
}

func (s *Server) handleCreateLFSLock(w http.ResponseWriter, r *http.Request) {
	rp, u, ok := s.loadRepoAndPerm(w, r, domain.PermWrite)
	if !ok {
		return
	}
	var req createLockReq
	if !decodeJSON(w, r, &req) {
		return
	}
	lock, err := s.deps.LFS.Lock(r.Context(), rp.ID, u.ID, req.Path)
	if err != nil {
		switch {
		case errors.Is(err, lfs.ErrLockInvalid):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, lfs.ErrLocked):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusCreated, lock)
}

func (s *Server) handleUnlockLFSLock(w http.ResponseWriter, r *http.Request) {
	rp, u, ok := s.loadRepoAndPerm(w, r, domain.PermWrite)
	if !ok {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.deps.LFS.Unlock(r.Context(), rp.ID, u.ID, id); err != nil {
		if errors.Is(err, lfs.ErrNotFound) {
			writeError(w, http.StatusNotFound, "lock not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleListLFSLocks(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	locks, err := s.deps.LFS.ListLocks(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"locks": locks})
}
