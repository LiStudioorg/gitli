// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/ide"
)

func (s *Server) handleIDETree(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	path, err := s.repoPath(r, rp)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ref := r.URL.Query().Get("ref")
	dir := r.URL.Query().Get("dir")
	entries, err := s.deps.IDE.ListDir(r.Context(), path, ref, dir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if entries == nil {
		entries = []ide.FileEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) handleIDEFile(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		writeError(w, http.StatusBadRequest, "必须提供路径")
		return
	}
	path, err := s.repoPath(r, rp)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ref := r.URL.Query().Get("ref")
	content, err := s.deps.IDE.ReadFile(r.Context(), path, ref, filePath, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (s *Server) handleIDESymbols(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	path, err := s.repoPath(r, rp)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ref := r.URL.Query().Get("ref")
	q := r.URL.Query().Get("q")
	symbols, err := s.deps.IDE.Symbols(r.Context(), path, ref, q, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if symbols == nil {
		symbols = []ide.Symbol{}
	}
	writeJSON(w, http.StatusOK, symbols)
}
