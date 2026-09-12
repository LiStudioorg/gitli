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
			if _, _, err := s.repos.Get(r.Context(), owner, name); err != nil {
				return "", "", false
			}
			return s.repos.DiskPath(owner, name), owner + "/" + name + ".git", true
		},
		Authorize: func(r *http.Request, svc git.Service) bool {
			owner, name, ok := parseGitPath(r.URL.Path)
			if !ok {
				return false
			}
			repo, _, err := s.repos.Get(r.Context(), owner, name)
			if err != nil {
				return false
			}
			write := svc == git.ReceivePack

			// 匿名读 public 仓库
			if !write && auth.CanAccess(r.Context(), s.q, UserFromContext(r.Context()), repo, false) {
				return true
			}
			// Basic Auth（PAT 或密码）
			user, ok2 := auth.BasicAuthUser(r.Context(), s.q, r)
			if !ok2 {
				return false
			}
			return auth.CanAccess(r.Context(), s.q, user, repo, write)
		},
	}
}

var _ = repos.VisibilityPublic
