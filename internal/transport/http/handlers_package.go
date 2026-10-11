// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"errors"
	"io"
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/pkgsvc"
)

// packageTypeValid reports whether typ is a supported package format.
func packageTypeValid(typ string) bool {
	switch typ {
	case "generic", "npm", "go", "maven":
		return true
	default:
		return false
	}
}

// handleListPackageVersions lists every published version of a named package.
func (s *Server) handleListPackageVersions(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	typ := urlParam(r, "type")
	if !packageTypeValid(typ) {
		writeError(w, http.StatusBadRequest, "unsupported package type")
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name query parameter is required")
		return
	}
	versions, err := s.deps.PkgSvc.ListVersions(r.Context(), rp.ID, typ, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"versions": versions})
}

// handlePublishPackageVersion creates a version and streams the uploaded file
// into the blob store without buffering it in memory.
func (s *Server) handlePublishPackageVersion(w http.ResponseWriter, r *http.Request) {
	rp, u, ok := s.loadRepoAndPerm(w, r, domain.PermWrite)
	if !ok {
		return
	}
	typ := urlParam(r, "type")
	if !packageTypeValid(typ) {
		writeError(w, http.StatusBadRequest, "unsupported package type")
		return
	}
	name := urlParam(r, "name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "package name is required")
		return
	}
	version := urlParam(r, "version")
	if err := pkgsvc.ValidateVersion(version); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		writeError(w, http.StatusBadRequest, "filename query parameter is required")
		return
	}
	id, err := s.deps.PkgSvc.PublishVersion(r.Context(), rp.ID, typ, name, version, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	digest, err := s.deps.PkgSvc.AddFile(r.Context(), id, filename, r.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "package.publish", "success")
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "digest": digest})
}

// handleDownloadPackageFile streams a stored package file to the client.
func (s *Server) handleDownloadPackageFile(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	typ := urlParam(r, "type")
	if !packageTypeValid(typ) {
		writeError(w, http.StatusBadRequest, "unsupported package type")
		return
	}
	name := urlParam(r, "name")
	version := urlParam(r, "version")
	filename := urlParam(r, "filename")
	v, err := s.deps.PkgSvc.GetVersion(r.Context(), rp.ID, typ, name, version)
	if err != nil {
		if errors.Is(err, pkgsvc.ErrNotFound) {
			writeError(w, http.StatusNotFound, "version not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	rc, _, err := s.deps.PkgSvc.OpenFile(r.Context(), v.ID, filename)
	if err != nil {
		if errors.Is(err, pkgsvc.ErrNotFound) {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = rc.Close() }()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, rc); err != nil {
		s.deps.Log.Warn("package file stream failed", "error", err)
	}
}
