// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/git"
)

// handleGitInfoRefs advertises refs for the smart HTTP protocol.
func (s *Server) handleGitInfoRefs(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	if !git.IsSmartService(service) {
		writeError(w, http.StatusBadRequest, "unsupported service")
		return
	}
	need := domain.PermRead
	if service == git.ServiceReceivePack {
		need = domain.PermWrite
	}
	rp, _, ok := s.gitRepoAccess(w, r, service, need)
	if !ok {
		return
	}
	path, err := s.repoPath(r, rp)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", git.InfoRefsContentType(service))
	w.Header().Set("Cache-Control", "no-cache")
	if err := git.WritePacketLinePrefix(w, service); err != nil {
		return
	}
	if err := s.deps.Git.ServeSmartHTTP(r.Context(), path, service, git.InfoRefsContentType(service), nil, w); err != nil {
		s.deps.Log.Warn("git info/refs failed", "error", err)
	}
}

// handleGitService handles the upload-pack / receive-pack POST phase.
func (s *Server) handleGitService(w http.ResponseWriter, r *http.Request, service string) {
	if !git.IsSmartService(service) {
		writeError(w, http.StatusBadRequest, "unsupported service")
		return
	}
	need := domain.PermRead
	if service == git.ServiceReceivePack {
		need = domain.PermWrite
	}
	rp, gu, ok := s.gitRepoAccess(w, r, service, need)
	if !ok {
		return
	}
	path, err := s.repoPath(r, rp)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	body := io.Reader(r.Body)
	if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid gzip body")
			return
		}
		defer func() { _ = gz.Close() }()
		body = gz
	}
	w.Header().Set("Content-Type", git.ResultContentType(service))
	w.Header().Set("Cache-Control", "no-cache")
	extraEnv := map[string]string{}
	if gu != nil {
		extraEnv["GITLI_HOOK_ACTOR"] = gu.Username
		if gu.IsAdmin {
			extraEnv["GITLI_HOOK_IS_ADMIN"] = "1"
		}
	}
	_ = rp
	if err := s.deps.Git.ServeSmartHTTPWithEnv(r.Context(), path, service, git.ResultContentType(service), body, w, extraEnv); err != nil {
		s.deps.Log.Warn("git service failed", "service", service, "error", err)
	}
}

// gitRepoAccess resolves owner/repo and checks permission. It authenticates via
// Basic auth (username + PAT) when a token is not already present on the context.
func (s *Server) gitRepoAccess(w http.ResponseWriter, r *http.Request, service string, need domain.Perm) (*domain.Repo, *domain.User, bool) {
	owner := urlParam(r, "owner")
	name := urlParam(r, "repo")
	name = strings.TrimSuffix(name, ".git")
	if err := validateName("owner", owner); err != nil {
		writeError(w, http.StatusBadRequest, "invalid owner")
		return nil, nil, false
	}
	if err := validateName("repo", name); err != nil {
		writeError(w, http.StatusBadRequest, "invalid repo")
		return nil, nil, false
	}

	u := userFrom(r.Context())
	if u == nil {
		if gu, ok := s.authenticateGitBasic(r); ok {
			u = gu
		}
	}

	ownerUser, err := s.deps.Users.ByUsername(r.Context(), owner)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return nil, nil, false
	}
	rp, err := s.deps.Repos.ByOwnerName(r.Context(), ownerUser.ID, name)
	if err != nil {
		// Follow a repository rename with a 301 to the canonical git path.
		if targetID, rerr := s.deps.Repos.ResolveRedirect(r.Context(), owner, name); rerr == nil {
			if target, rerr := s.deps.Repos.ByID(r.Context(), targetID); rerr == nil {
				sub := strings.TrimPrefix(r.URL.Path, "/"+owner+"/"+name)
				http.Redirect(w, r, "/"+owner+"/"+target.Name+sub, http.StatusMovedPermanently)
				return nil, nil, false
			}
		}
		writeError(w, http.StatusNotFound, "repository not found")
		return nil, nil, false
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
		s.gitAuthChallenge(w)
		return nil, nil, false
	}
	// Anonymous read access is allowed for public repositories.
	if need <= domain.PermRead && !rp.IsPrivate {
		return rp, u, true
	}
	if perm < need {
		if u == nil {
			s.gitAuthChallenge(w)
		} else {
			writeError(w, http.StatusForbidden, "insufficient permissions")
		}
		return nil, nil, false
	}
	return rp, u, true
}

// gitAuthChallenge asks the git client for Basic credentials.
func (s *Server) gitAuthChallenge(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="gitli"`)
	writeError(w, http.StatusUnauthorized, "authentication required")
}

// authenticateGitBasic validates HTTP Basic auth against PATs and user passwords.
func (s *Server) authenticateGitBasic(r *http.Request) (*domain.User, bool) {
	username, password, ok := r.BasicAuth()
	if !ok || username == "" {
		return nil, false
	}
	ctx := r.Context()
	// Password may be a PAT (not tied to a username) or the account password.
	if strings.HasPrefix(password, "gitli_") {
		tok, err := s.deps.Tokens.Resolve(ctx, password)
		if err != nil {
			return nil, false
		}
		target, err := s.deps.Users.ByID(ctx, tok.UserID)
		if err != nil {
			return nil, false
		}
		if username != target.Username && username != "git" && username != "oauth2" {
			return nil, false
		}
		return target, true
	}
	u, err := s.deps.Auth.VerifyPassword(ctx, username, password)
	if err != nil {
		return nil, false
	}
	return u, true
}
