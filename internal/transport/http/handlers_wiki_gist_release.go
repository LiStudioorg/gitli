// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/infra/secutil"
)

type wikiPageReq struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (s *Server) handleListWikiPages(w http.ResponseWriter, r *http.Request) {
	owner := urlParam(r, "owner")
	repoName := urlParam(r, "repo")
	rp, _, ok := s.resolveRepo(w, r, owner, repoName, domain.PermRead)
	if !ok {
		return
	}
	pages, err := s.deps.Wiki.List(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pages)
}

func (s *Server) handleGetWikiPage(w http.ResponseWriter, r *http.Request) {
	owner := urlParam(r, "owner")
	repoName := urlParam(r, "repo")
	rp, _, ok := s.resolveRepo(w, r, owner, repoName, domain.PermRead)
	if !ok {
		return
	}
	page, err := s.deps.Wiki.BySlug(r.Context(), rp.ID, urlParam(r, "slug"))
	if err != nil {
		writeError(w, http.StatusNotFound, "wiki page not found")
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) handleCreateWikiPage(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	owner := urlParam(r, "owner")
	repoName := urlParam(r, "repo")
	rp, _, ok := s.resolveRepo(w, r, owner, repoName, domain.PermWrite)
	if !ok {
		return
	}
	var req wikiPageReq
	if !decodeJSON(w, r, &req) {
		return
	}
	page, err := s.deps.Wiki.Create(r.Context(), rp.ID, req.Slug, req.Title, req.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "wiki.create", "success")
	writeJSON(w, http.StatusCreated, page)
}

func (s *Server) handleUpdateWikiPage(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	owner := urlParam(r, "owner")
	repoName := urlParam(r, "repo")
	rp, _, ok := s.resolveRepo(w, r, owner, repoName, domain.PermWrite)
	if !ok {
		return
	}
	page, err := s.deps.Wiki.BySlug(r.Context(), rp.ID, urlParam(r, "slug"))
	if err != nil {
		writeError(w, http.StatusNotFound, "wiki page not found")
		return
	}
	var req wikiPageReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Title != "" {
		page.Title = req.Title
	}
	page.Body = req.Body
	if err := s.deps.Wiki.Update(r.Context(), page); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "wiki.update", "success")
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) handleDeleteWikiPage(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	owner := urlParam(r, "owner")
	repoName := urlParam(r, "repo")
	rp, _, ok := s.resolveRepo(w, r, owner, repoName, domain.PermWrite)
	if !ok {
		return
	}
	if err := s.deps.Wiki.Delete(r.Context(), rp.ID, urlParam(r, "slug")); err != nil {
		writeError(w, http.StatusNotFound, "wiki page not found")
		return
	}
	s.auditAction(r, u.ID, "wiki.delete", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type createGistReq struct {
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
}

func (s *Server) handleListGists(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	gists, err := s.deps.Gists.ListByUser(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, gists)
}

func (s *Server) handleCreateGist(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req createGistReq
	if !decodeJSON(w, r, &req) {
		return
	}
	g, err := s.deps.Gists.Create(r.Context(), u.ID, req.Description, req.IsPublic)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "gist.create", "success")
	writeJSON(w, http.StatusCreated, g)
}

func (s *Server) handleGetGist(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	g, err := s.deps.Gists.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "gist not found")
		return
	}
	if !g.IsPublic && g.UserID != u.ID && !u.IsAdmin {
		writeError(w, http.StatusNotFound, "gist not found")
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (s *Server) handleDeleteGist(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.deps.Gists.Delete(r.Context(), u.ID, id); err != nil {
		writeError(w, http.StatusNotFound, "gist not found")
		return
	}
	s.auditAction(r, u.ID, "gist.delete", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type createReleaseReq struct {
	Tag  string `json:"tag"`
	Name string `json:"name"`
	Body string `json:"body"`
}

func (s *Server) handleListReleases(w http.ResponseWriter, r *http.Request) {
	owner := urlParam(r, "owner")
	repoName := urlParam(r, "repo")
	rp, _, ok := s.resolveRepo(w, r, owner, repoName, domain.PermRead)
	if !ok {
		return
	}
	releases, err := s.deps.Releases.List(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, releases)
}

func (s *Server) handleGetRelease(w http.ResponseWriter, r *http.Request) {
	owner := urlParam(r, "owner")
	repoName := urlParam(r, "repo")
	rp, _, ok := s.resolveRepo(w, r, owner, repoName, domain.PermRead)
	if !ok {
		return
	}
	rel, err := s.deps.Releases.ByTag(r.Context(), rp.ID, urlParam(r, "tag"))
	if err != nil {
		writeError(w, http.StatusNotFound, "release not found")
		return
	}
	writeJSON(w, http.StatusOK, rel)
}

func (s *Server) handleCreateRelease(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	owner := urlParam(r, "owner")
	repoName := urlParam(r, "repo")
	rp, _, ok := s.resolveRepo(w, r, owner, repoName, domain.PermWrite)
	if !ok {
		return
	}
	var req createReleaseReq
	if !decodeJSON(w, r, &req) {
		return
	}
	rel, err := s.deps.Releases.Create(r.Context(), rp.ID, u.ID, req.Tag, req.Name, req.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "release.create", "success")
	writeJSON(w, http.StatusCreated, rel)
}

func (s *Server) handlePublishRelease(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	owner := urlParam(r, "owner")
	repoName := urlParam(r, "repo")
	rp, _, ok := s.resolveRepo(w, r, owner, repoName, domain.PermWrite)
	if !ok {
		return
	}
	rel, err := s.deps.Releases.ByTag(r.Context(), rp.ID, urlParam(r, "tag"))
	if err != nil {
		writeError(w, http.StatusNotFound, "release not found")
		return
	}
	if err := s.deps.Releases.Publish(r.Context(), rel.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "release.publish", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleListReleaseAssets(w http.ResponseWriter, r *http.Request) {
	owner := urlParam(r, "owner")
	repoName := urlParam(r, "repo")
	rp, _, ok := s.resolveRepo(w, r, owner, repoName, domain.PermRead)
	if !ok {
		return
	}
	rel, err := s.deps.Releases.ByTag(r.Context(), rp.ID, urlParam(r, "tag"))
	if err != nil {
		writeError(w, http.StatusNotFound, "release not found")
		return
	}
	assets, err := s.deps.Releases.ListAssets(r.Context(), rel.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, assets)
}

// handleUploadReleaseAsset streams an uploaded file to disk and records it.
func (s *Server) handleUploadReleaseAsset(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	owner := urlParam(r, "owner")
	repoName := urlParam(r, "repo")
	rp, _, ok := s.resolveRepo(w, r, owner, repoName, domain.PermWrite)
	if !ok {
		return
	}
	rel, err := s.deps.Releases.ByTag(r.Context(), rp.ID, urlParam(r, "tag"))
	if err != nil {
		writeError(w, http.StatusNotFound, "release not found")
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name query parameter required")
		return
	}
	if err := secutil.ValidateName("asset", name); err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset name")
		return
	}
	dir := filepath.Join(s.deps.Cfg.Storage.Attachments, "releases", strconv.FormatInt(rel.ID, 10))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	dst := filepath.Join(dir, name)
	f, err := os.Create(dst)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() { _ = f.Close() }()
	size, err := io.Copy(f, r.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	asset, err := s.deps.Releases.AddAsset(r.Context(), rel.ID, name, size, dst)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "release.asset_upload", "success")
	writeJSON(w, http.StatusCreated, asset)
}

// handleDownloadReleaseAsset streams a stored release asset.
func (s *Server) handleDownloadReleaseAsset(w http.ResponseWriter, r *http.Request) {
	owner := urlParam(r, "owner")
	repoName := urlParam(r, "repo")
	rp, _, ok := s.resolveRepo(w, r, owner, repoName, domain.PermRead)
	if !ok {
		return
	}
	rel, err := s.deps.Releases.ByTag(r.Context(), rp.ID, urlParam(r, "tag"))
	if err != nil {
		writeError(w, http.StatusNotFound, "release not found")
		return
	}
	assets, err := s.deps.Releases.ListAssets(r.Context(), rel.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	name := urlParam(r, "name")
	for _, a := range assets {
		if a.Name == name {
			f, err := os.Open(a.StoredPath)
			if err != nil {
				writeError(w, http.StatusNotFound, "asset file missing")
				return
			}
			defer func() { _ = f.Close() }()
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", "attachment; filename=\""+a.Name+"\"")
			_, _ = io.Copy(w, f)
			return
		}
	}
	writeError(w, http.StatusNotFound, "asset not found")
}
