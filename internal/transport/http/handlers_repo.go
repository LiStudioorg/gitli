// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/infra/gitcmd"
	"github.com/LiStudioorg/gitli/internal/service/repo"
)

type createRepoReq struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	DefaultBranch string `json:"default_branch"`
	IsPrivate     bool   `json:"is_private"`
	Org           string `json:"org"`
}

func (s *Server) handleCreateRepo(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req createRepoReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := validateName("repo", req.Name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	params := repo.CreateParams{
		OwnerID:       u.ID,
		OwnerType:     domain.OwnerUser,
		Name:          req.Name,
		Description:   req.Description,
		DefaultBranch: req.DefaultBranch,
		IsPrivate:     req.IsPrivate,
	}
	if req.Org != "" {
		o, err := s.deps.Orgs.ByName(r.Context(), req.Org)
		if err != nil {
			writeError(w, http.StatusBadRequest, "org not found")
			return
		}
		params.OwnerType = domain.OwnerOrg
		params.OrgID = &o.ID
		params.OwnerID = o.OwnerID
	}
	rp, err := s.deps.Repos.Create(r.Context(), params)
	if err != nil {
		if errors.Is(err, repo.ErrExists) {
			writeError(w, http.StatusConflict, "repository already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "repo.create", "success")
	s.enqueueLanguageRefresh(rp.ID)
	writeJSON(w, http.StatusCreated, rp)
}

// loadRepoAndPerm resolves owner/repo URL params and checks permission.
func (s *Server) loadRepoAndPerm(w http.ResponseWriter, r *http.Request, need domain.Perm) (*domain.Repo, *domain.User, bool) {
	owner := urlParam(r, "owner")
	name := urlParam(r, "repo")
	if err := validateName("owner", owner); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return nil, nil, false
	}
	if err := validateName("repo", name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return nil, nil, false
	}
	ownerUser, err := s.deps.Users.ByUsername(r.Context(), owner)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return nil, nil, false
	}
	rp, err := s.deps.Repos.ByOwnerName(r.Context(), ownerUser.ID, name)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return nil, nil, false
	}
	u := userFrom(r.Context())
	// Package/other non-browser clients authenticate with HTTP Basic instead of
	// a session cookie; fall back to it when no session user is present.
	if u == nil {
		if gu, ok := s.authenticateGitBasic(r); ok {
			u = gu
		}
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
		return nil, nil, false
	}
	if rp.IsPrivate && perm < domain.PermRead {
		writeError(w, http.StatusNotFound, "repository not found")
		return nil, nil, false
	}
	if perm < need {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return nil, nil, false
	}
	return rp, u, true
}

func (s *Server) handleGetRepo(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	langs, _ := s.deps.Repos.Languages(r.Context(), rp.ID)
	if len(langs) == 0 {
		if path, err := s.repoPath(r, rp); err == nil {
			if detected, derr := s.deps.Git.DetectLanguages(r.Context(), path, rp.DefaultBranch); derr == nil && len(detected) > 0 {
				stats := make([]repo.LanguageStat, 0, len(detected))
				for _, d := range detected {
					stats = append(stats, repo.LanguageStat{Language: d.Language, Bytes: d.Bytes, Percentage: d.Percentage})
				}
				if s.deps.Repos.ReplaceLanguages(r.Context(), rp.ID, stats) == nil {
					langs = stats
				}
			}
		}
	}
	if langs == nil {
		langs = []repo.LanguageStat{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"repo":      rp,
		"languages": langs,
	})
}

// handleCloneInfo returns clone URLs for the repo based on configuration.
func (s *Server) handleCloneInfo(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	owner := urlParam(r, "owner")
	ext := s.deps.External
	writeJSON(w, http.StatusOK, map[string]any{
		"https":                ext.HTTPSCloneURL(owner, rp.Name),
		"ssh":                  ext.SSHCloneURL(owner, rp.Name),
		"clone_default_branch": rp.DefaultBranch,
	})
}

// handleArchive streams a zip/tar.gz archive or git bundle.
func (s *Server) handleArchive(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	path, err := s.repoPath(r, rp)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	format := urlParam(r, "format")
	ref := r.URL.Query().Get("ref")
	stream, wait, err := s.deps.Git.ArchiveStream(r.Context(), path, ref, format)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer func() { _ = stream.Close() }()

	base := fmt.Sprintf("%s-%s", rp.Name, refOrDefault(ref, rp.DefaultBranch))
	switch format {
	case "zip":
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename=\""+base+".zip\"")
	case "tar.gz", "targz", "tgz":
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", "attachment; filename=\""+base+".tar.gz\"")
	case "bundle":
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=\""+base+".bundle\"")
	}
	if _, err := io.Copy(w, stream); err != nil {
		s.deps.Log.Warn("archive stream failed", "error", err)
	}
	_ = wait()
}

func refOrDefault(ref, def string) string {
	if ref != "" {
		return ref
	}
	if def != "" {
		return def
	}
	return "HEAD"
}

func (s *Server) handleDeleteRepo(w http.ResponseWriter, r *http.Request) {
	rp, u, ok := s.loadRepoAndPerm(w, r, domain.PermAdmin)
	if !ok {
		return
	}
	if err := s.deps.Repos.Delete(r.Context(), rp.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "repo.delete", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type transferReq struct {
	NewOwner string `json:"new_owner"`
}

func (s *Server) handleTransferRepo(w http.ResponseWriter, r *http.Request) {
	rp, u, ok := s.loadRepoAndPerm(w, r, domain.PermAdmin)
	if !ok {
		return
	}
	var req transferReq
	if !decodeJSON(w, r, &req) {
		return
	}
	target, err := s.deps.Users.ByUsername(r.Context(), req.NewOwner)
	if err != nil {
		writeError(w, http.StatusBadRequest, "target user not found")
		return
	}
	if err := s.deps.Repos.Transfer(r.Context(), rp.ID, target.ID, domain.OwnerUser, nil); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "repo.transfer", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// enqueueLanguageRefresh schedules a background language recomputation.
func (s *Server) enqueueLanguageRefresh(repoID int64) {
	if s.deps.Queue == nil {
		return
	}
	s.deps.Queue.Enqueue("refresh-languages", strconv.FormatInt(repoID, 10))
}

func (s *Server) repoPath(r *http.Request, rp *domain.Repo) (string, error) {
	owner := urlParam(r, "owner")
	return gitcmd.ValidateRepoPath(s.deps.FS.Root(), s.deps.FS.Path(owner, rp.Name))
}

func (s *Server) handleBranches(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	path, err := s.repoPath(r, rp)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	branches, err := s.deps.Git.Branches(r.Context(), path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, branches)
}

func (s *Server) handleListProtections(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	branches, err := s.deps.Repos.ListBranchProtections(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, branches)
}

type protectionReq struct {
	Pattern       string `json:"pattern"`
	RequireReview bool   `json:"require_reviews"`
	RequireSigned bool   `json:"require_signed"`
}

func (s *Server) handleAddBranchProtection(w http.ResponseWriter, r *http.Request) {
	rp, u, ok := s.loadRepoAndPerm(w, r, domain.PermAdmin)
	if !ok {
		return
	}
	var req protectionReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.Repos.AddBranchProtection(r.Context(), rp.ID, req.Pattern, req.RequireReview, req.RequireSigned); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "repo.branch_protection_add", "success")
	writeJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
}

func (s *Server) handleAddTagProtection(w http.ResponseWriter, r *http.Request) {
	rp, u, ok := s.loadRepoAndPerm(w, r, domain.PermAdmin)
	if !ok {
		return
	}
	var req protectionReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.Repos.AddTagProtection(r.Context(), rp.ID, req.Pattern); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "repo.tag_protection_add", "success")
	writeJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
}

type collaboratorReq struct {
	Username   string `json:"username"`
	Permission string `json:"permission"`
}

func (s *Server) handleAddCollaborator(w http.ResponseWriter, r *http.Request) {
	rp, u, ok := s.loadRepoAndPerm(w, r, domain.PermAdmin)
	if !ok {
		return
	}
	var req collaboratorReq
	if !decodeJSON(w, r, &req) {
		return
	}
	target, err := s.deps.Users.ByUsername(r.Context(), req.Username)
	if err != nil {
		writeError(w, http.StatusBadRequest, "user not found")
		return
	}
	perm := domain.ParsePerm(req.Permission)
	if perm == domain.PermNone {
		perm = domain.PermRead
	}
	if err := s.deps.Repos.AddCollaborator(r.Context(), rp.ID, target.ID, perm); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "repo.collaborator_add", "success")
	writeJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
}
