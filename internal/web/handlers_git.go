package web

import (
	"net/http"
	"strings"

	"gitli/internal/auth"
	"gitli/internal/git"
	"gitli/internal/repos"
)

// parseGitPath 解析 /{owner}/{repo}.git/... 路径。
func parseGitPath(p string) (owner, repo string, ok bool) {
	p = strings.TrimPrefix(p, "/")
	parts := strings.Split(p, "/")
	if len(parts) < 2 {
		return "", "", false
	}
	owner, name := parts[0], parts[1]
	if !strings.HasSuffix(name, ".git") {
		return "", "", false
	}
	return owner, strings.TrimSuffix(name, ".git"), true
}

// gitHTTP 处理 /{owner}/{repo}.git/* 的 smart HTTP 请求。
func (s *Server) gitHTTP() http.Handler {
	reposDir := s.repos.RootDir()
	return &git.SmartHTTPHandler{
		RelRoot: reposDir,
		RepoPath: func(r *http.Request) (string, string, bool) {
			owner, name, ok := parseGitPath(r.URL.Path)
			if !ok {
				return "", "", false
			}
			if _, err := s.repos.GetWithKind(r.Context(), owner, name); err != nil {
				return "", "", false
			}
			return s.repos.DiskPath(owner, name), owner + "/" + name + ".git", true
		},
		Authorize: func(r *http.Request, svc git.Service) bool {
			owner, name, ok := parseGitPath(r.URL.Path)
			if !ok {
				return false
			}
			oi, err := s.repos.GetWithKind(r.Context(), owner, name)
			if err != nil {
				return false
			}
			repo := oi.Repo
			write := svc == git.ReceivePack

			// Basic Auth：用户名 + PAT（全局或仓库级）或密码
			if u, p, ok := r.BasicAuth(); ok && u != "" && p != "" {
				user, pat := auth.BasicAuthGit(r.Context(), s.q, r)
				if user == nil {
					return false
				}
				if pat != nil {
					// PAT 作用域判定（仓库级 PAT 最小权限，全局 PAT 走用户权限）
					return auth.PATCanAccessRepo(r.Context(), s.q, *pat, *user, repo, write)
				}
				return auth.CanAccess(r.Context(), s.q, user, repo, write)
			}
			// 匿名：仅 public 可读
			if !write && auth.CanAccess(r.Context(), s.q, UserFromContext(r.Context()), repo, false) {
				return true
			}
			return false
		},
	}
}

var _ = repos.VisibilityPublic
