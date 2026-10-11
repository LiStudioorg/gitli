// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
	importer "github.com/LiStudioorg/gitli/internal/service/import"
	"github.com/LiStudioorg/gitli/internal/service/mirror"
)

type createMirrorReq struct {
	RemoteURL       string `json:"remote_url"`
	Direction       string `json:"direction"`
	IntervalMinutes int    `json:"interval_minutes"`
}

// handleCreateMirror configures a new repository mirror.
func (s *Server) handleCreateMirror(w http.ResponseWriter, r *http.Request) {
	rp, u, ok := s.loadRepoAndPerm(w, r, domain.PermWrite)
	if !ok {
		return
	}
	var req createMirrorReq
	if !decodeJSON(w, r, &req) {
		return
	}
	m, err := s.deps.Mirrors.Create(r.Context(), rp.ID, req.RemoteURL, req.Direction, req.IntervalMinutes)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.auditAction(r, u.ID, "mirror.create", "success")
	writeJSON(w, http.StatusCreated, m)
}

// handleListMirrors lists all mirrors configured for a repository.
func (s *Server) handleListMirrors(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	mirrors, err := s.deps.Mirrors.ListByRepo(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if mirrors == nil {
		mirrors = []*mirror.Mirror{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"mirrors": mirrors})
}

type createImportReq struct {
	Source    string `json:"source"`
	SourceURL string `json:"source_url"`
}

// handleCreateImport registers a repository import job.
func (s *Server) handleCreateImport(w http.ResponseWriter, r *http.Request) {
	rp, u, ok := s.loadRepoAndPerm(w, r, domain.PermWrite)
	if !ok {
		return
	}
	var req createImportReq
	if !decodeJSON(w, r, &req) {
		return
	}
	job, err := s.deps.Imports.Create(r.Context(), rp.ID, req.Source, req.SourceURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.auditAction(r, u.ID, "import.create", "success")
	writeJSON(w, http.StatusCreated, job)
}

// handleListImports lists all import jobs for a repository.
func (s *Server) handleListImports(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	jobs, err := s.deps.Imports.ListByRepo(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if jobs == nil {
		jobs = []*importer.Job{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"imports": jobs})
}
