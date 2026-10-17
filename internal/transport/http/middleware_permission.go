// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/audit"
	"github.com/LiStudioorg/gitli/internal/service/rbac"
)

// RequirePermission returns middleware enforcing a resource:action permission
// on the repository identified by the {owner}/{repo} URL parameters.
func (s *Server) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := requireUser(w, r)
			if u == nil {
				return
			}
			rp, ok := s.repoFromURL(r.Context(), urlParam(r, "owner"), trimGitSuffix(urlParam(r, "repo")))
			if !ok {
				writeError(w, http.StatusNotFound, "仓库不存在")
				return
			}
			if !s.deps.RBAC.Check(r.Context(), u.ID, u.IsAdmin, rbac.Requirement{
				Resource: resource, Action: action, RepoID: rp.ID,
			}) {
				s.auditPermissionDenied(r, u.ID, resource, action, rp.ID)
				writeError(w, http.StatusForbidden, "权限不足")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireOrgPermission enforces a resource:action permission within an org
// identified by the {org} URL parameter.
func (s *Server) RequireOrgPermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := requireUser(w, r)
			if u == nil {
				return
			}
			orgName := urlParam(r, "org")
			if orgName == "" {
				orgName = r.URL.Query().Get("org")
			}
			o, err := s.deps.Orgs.ByName(r.Context(), orgName)
			if err != nil {
				writeError(w, http.StatusNotFound, "组织不存在")
				return
			}
			if s.deps.IPAllow != nil && !u.IsAdmin && !strings.Contains(r.URL.Path, "/ip-allowlist") {
				ok, ierr := s.deps.IPAllow.Check(r.Context(), o.ID, clientIP(r))
				if ierr == nil && !ok {
					s.auditAction(r, u.ID, "org.ip_allowlist_denied", "denied")
					writeError(w, http.StatusForbidden, "你的 IP 不在该组织白名单内")
					return
				}
			}
			if !s.deps.RBAC.Check(r.Context(), u.ID, u.IsAdmin, rbac.Requirement{
				Resource: resource, Action: action, OrgID: o.ID,
			}) {
				s.auditPermissionDenied(r, u.ID, resource, action, 0)
				writeError(w, http.StatusForbidden, "权限不足")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// repoFromURL loads a repository by owner login and repo name.
func (s *Server) repoFromURL(ctx context.Context, owner, name string) (*domain.Repo, bool) {
	if owner == "" || name == "" {
		return nil, false
	}
	if validateName("owner", owner) != nil || validateName("repo", name) != nil {
		return nil, false
	}
	ownerUser, err := s.deps.Users.ByUsername(ctx, owner)
	if err != nil {
		return nil, false
	}
	rp, err := s.deps.Repos.ByOwnerName(ctx, ownerUser.ID, name)
	if err != nil {
		return nil, false
	}
	return rp, true
}

func (s *Server) auditPermissionDenied(r *http.Request, userID int64, resource, action string, repoID int64) {
	detail := resource + ":" + action
	_ = s.deps.Audit.Record(r.Context(), audit.Entry{
		ActorID: &userID, Action: "permission.denied", TargetType: "repo",
		IP: clientIP(r), Detail: detail,
	})
}

func trimGitSuffix(s string) string {
	if len(s) > 4 && s[len(s)-4:] == ".git" {
		return s[:len(s)-4]
	}
	return s
}
