package web

import (
	"context"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"gitli/internal/assets"
	"gitli/internal/db"
	"gitli/internal/issues"
	"gitli/internal/orgs"
	"gitli/internal/pulls"
	"gitli/internal/repos"
	"gitli/internal/users"
)

// Server 装配路由与依赖。
type Server struct {
	q        db.Querier
	users    *users.Service
	repos    *repos.Service
	orgs     *orgs.Service
	issues   *issues.Service
	pulls    *pulls.Service
	reposDir string
	rootURL  string
	r        *Renderer
	mux      *chi.Mux
}

func NewServer(q db.Querier, reposDir, rootURL string) (*Server, error) {
	r, err := NewRenderer()
	if err != nil {
		return nil, err
	}
	s := &Server{
		q:       q,
		users:   users.NewService(q),
		repos:   repos.NewService(q, reposDir),
		orgs:    orgs.NewService(q),
		issues:  issues.NewService(q),
		rootURL: strings.TrimSuffix(rootURL, "/"),
		r:       r,
		mux:     chi.NewRouter(),
	}
	s.reposDir = reposDir
	s.pulls = pulls.NewService(q, s.issues, s.repoWithPath)
	s.routes()
	return s, nil
}

// repoWithPath 解析 owner/repo 并返回磁盘路径（pulls.Service 用）。
func (s *Server) repoWithPath(ctx context.Context, ownerName, name string) (db.Repo, string, error) {
	oi, err := s.repos.GetWithKind(ctx, ownerName, name)
	if err != nil {
		return db.Repo{}, "", err
	}
	return oi.Repo, s.repos.DiskPath(oi.OwnerName, name), nil
}

func (s *Server) routes() {
	m := s.mux
	m.Use(middleware.RealIP)
	m.Use(middleware.RequestID)
	m.Use(middleware.Recoverer)
	m.Use(SecureHeaders)
	m.Use(LoadSession(s.q))
	m.Use(VerifyCSRF)

	staticFS, _ := fs.Sub(assets.StaticFS(), "static")
	m.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	m.Get("/", s.handleHome)
	m.Get("/login", s.showLogin)
	m.Post("/login", s.doLogin)
	m.Get("/register", s.showRegister)
	m.Post("/register", s.doRegister)
	m.Post("/logout", s.doLogout)

	// 用户主页与仓库
	m.Get("/{owner}", s.showUser)
	m.Post("/{owner}/repos/create", s.doCreateRepo)
	m.Get("/{owner}/{repo}", s.showRepo)

	// M3 代码浏览
	m.Get("/{owner}/{repo}/branches", s.handleRefs)
	m.Get("/{owner}/{repo}/tags", s.handleRefs)
	m.Get("/{owner}/{repo}/tree/{ref}", s.handleTree)
	m.Get("/{owner}/{repo}/tree/{ref}/*", s.handleTree)
	m.Get("/{owner}/{repo}/blob/{ref}/*", s.handleBlob)
	m.Get("/{owner}/{repo}/raw/{ref}/*", s.handleRaw)
	m.Get("/{owner}/{repo}/blame/{ref}/*", s.handleBlame)
	m.Get("/{owner}/{repo}/commits/{ref}", s.handleCommits)
	m.Get("/{owner}/{repo}/commit/{sha}", s.handleCommit)

	// M5 组织
	m.Get("/orgs/new", s.showOrgNew)
	m.Post("/orgs/create", s.doOrgCreate)
	m.Get("/orgs/{name}", s.showOrg)
	m.Post("/orgs/{name}/members/add", s.doOrgAddMember)
	m.Post("/orgs/{name}/repos/create", s.doOrgCreateRepo)

	// M5 Issue
	m.Get("/{owner}/{repo}/issues", s.handleIssues)
	m.Get("/{owner}/{repo}/issues/new", s.handleIssueNew)
	m.Post("/{owner}/{repo}/issues/new", s.handleIssueCreate)
	m.Get("/{owner}/{repo}/issues/{number}", s.handleIssue)
	m.Post("/{owner}/{repo}/issues/{number}/comment", s.handleIssueComment)
	m.Post("/{owner}/{repo}/issues/{number}/state", s.handleIssueState)
	m.Post("/{owner}/{repo}/issues/{number}/labels", s.handleIssueLabels)

	// M6 PR
	m.Get("/{owner}/{repo}/pulls", s.handlePulls)
	m.Get("/{owner}/{repo}/pulls/new", s.handlePullNew)
	m.Post("/{owner}/{repo}/pulls/new", s.handlePullCreate)
	m.Get("/{owner}/{repo}/pulls/{number}", s.handlePull)
	m.Post("/{owner}/{repo}/pulls/{number}/comment", s.handleIssueComment)
	m.Post("/{owner}/{repo}/pulls/{number}/merge", s.handlePullMerge)

	// M7 Wiki
	m.Get("/{owner}/{repo}/wiki", s.handleWiki)
	m.Get("/{owner}/{repo}/wiki/new", s.handleWikiNew)
	m.Post("/{owner}/{repo}/wiki/new", s.handleWikiCreate)
	m.Get("/{owner}/{repo}/wiki/{page}", s.handleWikiPage)
	m.Get("/{owner}/{repo}/wiki/{page}/edit", s.handleWikiEdit)
	m.Post("/{owner}/{repo}/wiki/{page}/edit", s.handleWikiSave)

	// M7 管理后台
	m.Get("/admin", s.handleAdmin)
	m.Get("/admin/users", s.handleAdminUsers)
	m.Post("/admin/users/{id}/delete", s.handleAdminDeleteUser)
	m.Get("/admin/repos", s.handleAdminRepos)
	m.Post("/admin/repos/{id}/delete", s.handleAdminDeleteRepo)

	// M7 OAuth2/OIDC
	m.Get("/oauth2/start", s.handleOAuthStart)
	m.Get("/oauth2/callback", s.handleOAuthCallback)

	// Git smart HTTP：/{owner}/{repo}.git/*
	m.Mount("/{owner}/{repo}.git", s.gitHTTP())

	m.NotFound(func(w http.ResponseWriter, r *http.Request) {
		s.renderError(w, http.StatusNotFound, "页面不存在")
	})
	m.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		s.renderError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
	})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
