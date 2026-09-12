package web

import (
	"net/http"
	"strconv"

	"gitli/internal/db"
	"gitli/internal/git"
	"gitli/internal/repos"
)

type adminStats struct {
	Users int64
	Repos int64
}

type adminUserData struct {
	Users []db.User
	Flash Flash
}

type adminRepoData struct {
	Repos []db.ListAllReposWithOwnerRow
	Flash Flash
}

// requireAdmin 管理后台保护：未登录或非 admin 一律 404（不暴露存在性）。
func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	user := UserFromContext(r.Context())
	if user == nil || user.IsAdmin != 1 {
		s.renderError(w, http.StatusNotFound, "页面不存在")
		return false
	}
	return true
}

// GET /admin
func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	users, _ := s.q.CountAllUsers(r.Context())
	repos, _ := s.q.CountAllRepos(r.Context())
	s.render(w, http.StatusOK, "admin", pageData{
		Title: "管理后台",
		User:  UserFromContext(r.Context()),
		CSRF:  CSRFFromContext(r.Context()),
		Data:  adminStats{Users: users, Repos: repos},
	})
}

// GET /admin/users
func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	list, err := s.q.ListAllUsers(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	s.render(w, http.StatusOK, "admin_users", pageData{
		Title: "用户管理",
		User:  UserFromContext(r.Context()),
		CSRF:  CSRFFromContext(r.Context()),
		Data:  adminUserData{Users: list},
	})
}

// POST /admin/users/{id}/delete
func (s *Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	user := UserFromContext(r.Context())
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "无效 ID")
		return
	}
	if id == user.ID {
		s.render(w, http.StatusBadRequest, "admin_users", pageData{
			Title: "用户管理",
			User:  user,
			CSRF:  CSRFFromContext(r.Context()),
			Data:  adminUserData{Flash: Flash{Kind: "error", Message: "不能删除自己"}},
		})
		return
	}
	if err := s.q.DeleteUserByID(r.Context(), id); err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// GET /admin/repos
func (s *Server) handleAdminRepos(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	list, err := s.q.ListAllReposWithOwner(r.Context())
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	s.render(w, http.StatusOK, "admin_repos", pageData{
		Title: "仓库管理",
		User:  UserFromContext(r.Context()),
		CSRF:  CSRFFromContext(r.Context()),
		Data:  adminRepoData{Repos: list},
	})
}

// POST /admin/repos/{id}/delete：删 DB 记录 + 尽力删磁盘目录。
func (s *Server) handleAdminDeleteRepo(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "无效 ID")
		return
	}
	repo, err := s.repos.GetByID(r.Context(), id)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "仓库不存在")
		return
	}
	// 尽力解析 owner 名以删除磁盘目录
	ownerName := ""
	if u, uerr := s.users.GetByID(r.Context(), repo.OwnerID); uerr == nil {
		ownerName = u.Username
	} else if o, oerr := s.orgs.GetByID(r.Context(), repo.OwnerID); oerr == nil {
		ownerName = o.Name
	}
	if err := s.q.DeleteRepo(r.Context(), repo.ID); err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if ownerName != "" {
		_ = git.DeleteRepo(s.repos.DiskPath(ownerName, repo.Name))
		_ = git.DeleteRepo(s.wikiPath(ownerName, repo.Name))
	}
	http.Redirect(w, r, "/admin/repos", http.StatusSeeOther)
}

var _ = repos.VisibilityPublic
