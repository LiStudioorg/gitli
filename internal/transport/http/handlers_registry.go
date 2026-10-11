// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/registry"
)

// handleV2Check implements the OCI API version check.
func (s *Server) handleV2Check(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
	w.WriteHeader(http.StatusOK)
}

// ociRepoAccess resolves owner/repo and enforces a permission for the OCI path.
func (s *Server) ociRepoAccess(w http.ResponseWriter, r *http.Request, need domain.Perm) (*domain.Repo, bool) {
	owner := urlParam(r, "owner")
	name := urlParam(r, "repo")
	u := userFrom(r.Context())
	if u == nil {
		if gu, ok := s.authenticateGitBasic(r); ok {
			u = gu
		}
	}
	ownerUser, err := s.deps.Users.ByUsername(r.Context(), owner)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return nil, false
	}
	rp, err := s.deps.Repos.ByOwnerName(r.Context(), ownerUser.ID, name)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return nil, false
	}
	var userID int64
	var isAdmin bool
	if u != nil {
		userID = u.ID
		isAdmin = u.IsAdmin
	}
	perm, err := s.deps.Repos.PermissionFor(r.Context(), rp, userID, isAdmin, s.deps.Orgs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return nil, false
	}
	if perm < need {
		if u == nil {
			s.gitAuthChallenge(w)
		} else {
			writeError(w, http.StatusForbidden, "insufficient permissions")
		}
		return nil, false
	}
	return rp, true
}

// handleBlobUploadStart begins a resumable blob upload session.
func (s *Server) handleBlobUploadStart(w http.ResponseWriter, r *http.Request) {
	rp, ok := s.ociRepoAccess(w, r, domain.PermWrite)
	if !ok {
		return
	}
	// Stage into <packages>/uploads/<uuid>; the registry service treats the
	// provided path as the staged file location.
	stagingDir := filepath.Join(s.deps.PackagesRoot, "uploads")
	if err := os.MkdirAll(stagingDir, 0o750); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	f, err := os.CreateTemp(stagingDir, "upload-*")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	stagedPath := f.Name()
	_ = f.Close()
	uuid, err := s.deps.Registry.StartUpload(r.Context(), rp.ID, stagedPath)
	if err != nil {
		_ = os.Remove(stagedPath)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Location", "/v2/"+urlParam(r, "owner")+"/"+urlParam(r, "repo")+"/blobs/uploads/"+uuid)
	w.Header().Set("Docker-Upload-Uuid", uuid)
	w.WriteHeader(http.StatusAccepted)
}

// handleBlobUploadFinish streams the upload body to the staged file and commits it.
func (s *Server) handleBlobUploadFinish(w http.ResponseWriter, r *http.Request) {
	rp, ok := s.ociRepoAccess(w, r, domain.PermWrite)
	if !ok {
		return
	}
	uuid := urlParam(r, "uuid")
	digest := r.URL.Query().Get("digest")
	if !s.deps.Registry.ValidateDigest(digest) {
		writeError(w, http.StatusBadRequest, "invalid digest")
		return
	}
	stagedPath, err := s.deps.Registry.StagedPath(r.Context(), rp.ID, uuid)
	if err != nil {
		writeError(w, http.StatusNotFound, "upload not found")
		return
	}
	f, err := os.OpenFile(stagedPath, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		writeError(w, http.StatusNotFound, "upload not found")
		return
	}
	if _, err := io.Copy(f, r.Body); err != nil {
		_ = f.Close()
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = f.Close()
	if err := s.deps.Registry.FinishUpload(r.Context(), rp.ID, stagedPath, uuid, digest); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Docker-Content-Digest", digest)
	w.WriteHeader(http.StatusCreated)
}

// handlePutManifest stores an image manifest or manifest list.
func (s *Server) handlePutManifest(w http.ResponseWriter, r *http.Request) {
	rp, ok := s.ociRepoAccess(w, r, domain.PermWrite)
	if !ok {
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 256<<10))
	if err != nil {
		writeError(w, http.StatusBadRequest, "read body failed")
		return
	}
	mediaType := r.Header.Get("Content-Type")
	digest := registry.DigestOf(body)
	reference := urlParam(r, "reference")
	if isDigestReference(reference) {
		_, err = s.deps.Registry.PutManifestList(r.Context(), rp.ID, urlParam(r, "repo"), reference, mediaType, body)
	} else {
		_, err = s.deps.Registry.PutManifest(r.Context(), rp.ID, urlParam(r, "repo"), reference, mediaType, body)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Location", "/v2/"+urlParam(r, "owner")+"/"+urlParam(r, "repo")+"/manifests/"+digest)
	w.WriteHeader(http.StatusCreated)
}

// handleGetManifest returns a stored manifest by tag or digest.
func (s *Server) handleGetManifest(w http.ResponseWriter, r *http.Request) {
	rp, ok := s.ociRepoAccess(w, r, domain.PermRead)
	if !ok {
		return
	}
	content, mediaType, digest, err := s.deps.Registry.GetManifest(r.Context(), rp.ID, urlParam(r, "repo"), urlParam(r, "reference"))
	if err != nil {
		writeError(w, http.StatusNotFound, "manifest unknown")
		return
	}
	if mediaType == "" {
		mediaType = "application/vnd.docker.distribution.manifest.v2+json"
	}
	w.Header().Set("Content-Type", mediaType)
	w.Header().Set("Docker-Content-Digest", digest)
	_, _ = w.Write(content)
}

// handleListTags lists repository tags.
func (s *Server) handleListTags(w http.ResponseWriter, r *http.Request) {
	rp, ok := s.ociRepoAccess(w, r, domain.PermRead)
	if !ok {
		return
	}
	tags, err := s.deps.Registry.ListTags(r.Context(), rp.ID, urlParam(r, "repo"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tags == nil {
		tags = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": urlParam(r, "repo"), "tags": tags})
}

// handleGetBlob streams a stored blob by digest.
func (s *Server) handleGetBlob(w http.ResponseWriter, r *http.Request) {
	_, ok := s.ociRepoAccess(w, r, domain.PermRead)
	if !ok {
		return
	}
	digest := urlParam(r, "digest")
	rc, err := s.deps.Registry.GetBlob(digest)
	if err != nil {
		writeError(w, http.StatusNotFound, "blob unknown")
		return
	}
	defer func() { _ = rc.Close() }()
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = io.Copy(w, rc)
}

func isDigestReference(ref string) bool {
	return strings.Contains(ref, ":")
}
