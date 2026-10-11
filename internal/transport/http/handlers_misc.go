// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/badge"
	"github.com/LiStudioorg/gitli/internal/service/lfs"
)

// handleSearch performs a repo-scoped code search at a ref.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	q := r.URL.Query().Get("q")
	if strings.TrimSpace(q) == "" {
		writeError(w, http.StatusBadRequest, "q query param required")
		return
	}
	path, err := s.repoPath(r, rp)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ref := r.URL.Query().Get("ref")
	results, err := s.deps.Search.Grep(r.Context(), path, q, ref, 100)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"query": q, "ref": ref, "results": results})
}

// lfsBatchObject is one entry of an LFS batch request/response.
type lfsBatchObject struct {
	Oid    string `json:"oid"`
	Size   int64  `json:"size"`
	Action string `json:"action,omitempty"`
	Href   string `json:"href,omitempty"`
}

type lfsBatchRequest struct {
	Operation string           `json:"operation"`
	Objects   []lfsBatchObject `json:"objects"`
}

// handleLFSBatch reports download/upload actions for a set of objects.
func (s *Server) handleLFSBatch(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.loadRepoAndPerm(w, r, domain.PermWrite)
	if !ok {
		return
	}
	var req lfsBatchRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	base := "/api/v1/repos/" + urlParam(r, "owner") + "/" + urlParam(r, "repo") + "/lfs/objects/"
	out := make([]lfsBatchObject, 0, len(req.Objects))
	for _, obj := range req.Objects {
		if !s.deps.LFS.VerifyOID(obj.Oid) {
			writeError(w, http.StatusBadRequest, "invalid oid")
			return
		}
		action := "upload"
		if s.deps.LFS.Exists(obj.Oid) {
			action = "download"
		}
		out = append(out, lfsBatchObject{
			Oid:    obj.Oid,
			Size:   obj.Size,
			Action: action,
			Href:   base + obj.Oid,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"objects": out})
}

// handleLFSUpload streams an LFS object body to disk.
func (s *Server) handleLFSUpload(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermWrite)
	if !ok {
		return
	}
	oid := urlParam(r, "oid")
	if !s.deps.LFS.VerifyOID(oid) {
		writeError(w, http.StatusBadRequest, "invalid oid")
		return
	}
	size := r.ContentLength
	if size < 0 {
		size = 0
	}
	if err := s.deps.LFS.Put(r.Context(), rp.ID, oid, size, r.Body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// handleLFSDownload streams an LFS object to the client.
func (s *Server) handleLFSDownload(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	oid := urlParam(r, "oid")
	rc, _, err := s.deps.LFS.Get(r.Context(), rp.ID, oid)
	if err != nil {
		if errors.Is(err, lfs.ErrNotFound) {
			writeError(w, http.StatusNotFound, "object not found")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer func() { _ = rc.Close() }()
	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := io.Copy(w, rc); err != nil {
		s.deps.Log.Warn("lfs download stream failed", "error", err)
	}
}

// handleBadge renders a public SVG status badge for a repository.
func (s *Server) handleBadge(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "unknown"
	}
	label := r.URL.Query().Get("label")
	if label == "" {
		label = "build"
	}
	svg := badge.SVG(label, status, badge.ColorForStatus(status))
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte(svg))
}

// handleRSS renders a public RSS 2.0 feed for a repository.
func (s *Server) handleRSS(w http.ResponseWriter, r *http.Request) {
	s.writeFeed(w, r, false)
}

// handleAtom renders a public Atom 1.0 feed for a repository.
func (s *Server) handleAtom(w http.ResponseWriter, r *http.Request) {
	s.writeFeed(w, r, true)
}

func (s *Server) writeFeed(w http.ResponseWriter, r *http.Request, atom bool) {
	owner := urlParam(r, "owner")
	repoName := urlParam(r, "repo")
	if err := validateName("owner", owner); err != nil {
		writeError(w, http.StatusBadRequest, "invalid owner")
		return
	}
	if err := validateName("repo", repoName); err != nil {
		writeError(w, http.StatusBadRequest, "invalid repo")
		return
	}
	title := owner + "/" + repoName
	link := s.deps.External.HTTPBase + "/" + owner + "/" + repoName

	var (
		body []byte
		err  error
	)
	if atom {
		body, err = badge.Atom(title, link, nil)
	} else {
		body, err = badge.RSS(title, link, "Recent activity for "+title, nil)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if atom {
		w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	}
	_, _ = w.Write(body)
}
