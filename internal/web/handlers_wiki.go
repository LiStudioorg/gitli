package web

import (
	"errors"
	"html/template"
	"net/http"

	"gitli/internal/auth"
	"gitli/internal/wiki"
)

type wikiData struct {
	OwnerName string
	RepoName  string
	Pages     []string
	Page      string
	Content   template.HTML
	HasPage   bool
	CanWrite  bool
	Flash     Flash
}

type wikiEditData struct {
	OwnerName string
	RepoName  string
	Page      string
	Content   string
	IsNew     bool
	Flash     Flash
}

// loadWikiCtx 解析仓库 + 权限，返回写权限。
func (s *Server) loadWikiCtx(w http.ResponseWriter, r *http.Request) (ownerName, repoName, repoPath string, canWrite bool, ok bool) {
	ownerName = r.PathValue("owner")
	repoName = r.PathValue("repo")
	oi, err := s.repos.GetWithKind(r.Context(), ownerName, repoName)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "仓库不存在")
		return "", "", "", false, false
	}
	user := UserFromContext(r.Context())
	if !auth.CanAccess(r.Context(), s.q, user, oi.Repo, false) {
		s.renderError(w, http.StatusNotFound, "仓库不存在")
		return "", "", "", false, false
	}
	canWrite = user != nil && auth.CanAccess(r.Context(), s.q, user, oi.Repo, true)
	return ownerName, repoName, s.repos.DiskPath(ownerName, repoName), canWrite, true
}

func (s *Server) wikiPath(owner, repo string) string {
	return wiki.WikiPath(s.reposDir, owner, repo)
}

// GET /{owner}/{repo}/wiki → 默认首页 Home
func (s *Server) handleWiki(w http.ResponseWriter, r *http.Request) {
	owner, repo, _, canWrite, ok := s.loadWikiCtx(w, r)
	if !ok {
		return
	}
	wikiPath := s.wikiPath(owner, repo)
	pages, _ := wiki.ListPages(r.Context(), wikiPath)
	data := wikiData{
		OwnerName: owner,
		RepoName:  repo,
		Pages:     pages,
		CanWrite:  canWrite,
	}
	if wiki.HasPage(r.Context(), wikiPath, "Home") {
		content, err := wiki.GetPage(r.Context(), wikiPath, "Home")
		if err == nil {
			data.Page = "Home"
			data.Content = RenderMarkdown(content)
			data.HasPage = true
		}
	}
	s.render(w, http.StatusOK, "wiki", pageData{
		Title: owner + "/" + repo + " Wiki",
		User:  UserFromContext(r.Context()),
		CSRF:  CSRFFromContext(r.Context()),
		Data:  data,
	})
}

// GET /{owner}/{repo}/wiki/{page}
func (s *Server) handleWikiPage(w http.ResponseWriter, r *http.Request) {
	owner, repo, _, canWrite, ok := s.loadWikiCtx(w, r)
	if !ok {
		return
	}
	page := r.PathValue("page")
	wikiPath := s.wikiPath(owner, repo)
	pages, _ := wiki.ListPages(r.Context(), wikiPath)
	content, err := wiki.GetPage(r.Context(), wikiPath, page)
	if err != nil {
		if errors.Is(err, wiki.ErrPageNotFound) {
			// 不存在 → 引导创建
			s.render(w, http.StatusNotFound, "wiki_page", pageData{
				Title: owner + "/" + repo + " Wiki: " + page,
				User:  UserFromContext(r.Context()),
				CSRF:  CSRFFromContext(r.Context()),
				Data: wikiData{
					OwnerName: owner, RepoName: repo, Pages: pages,
					Page: page, CanWrite: canWrite,
					Flash: Flash{Kind: "info", Message: "页面不存在，可以创建它"},
				},
			})
			return
		}
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	s.render(w, http.StatusOK, "wiki_page", pageData{
		Title: owner + "/" + repo + " Wiki: " + page,
		User:  UserFromContext(r.Context()),
		CSRF:  CSRFFromContext(r.Context()),
		Data: wikiData{
			OwnerName: owner,
			RepoName:  repo,
			Pages:     pages,
			Page:      page,
			Content:   RenderMarkdown(content),
			HasPage:   true,
			CanWrite:  canWrite,
		},
	})
}

// GET /{owner}/{repo}/wiki/new
func (s *Server) handleWikiNew(w http.ResponseWriter, r *http.Request) {
	owner, repo, _, canWrite, ok := s.loadWikiCtx(w, r)
	if !ok {
		return
	}
	user := UserFromContext(r.Context())
	if !canWrite {
		s.renderError(w, http.StatusForbidden, "需要写权限")
		return
	}
	_ = user
	s.render(w, http.StatusOK, "wiki_edit", pageData{
		Title: "新建 Wiki 页面",
		User:  UserFromContext(r.Context()),
		CSRF:  CSRFFromContext(r.Context()),
		Data:  wikiEditData{OwnerName: owner, RepoName: repo, IsNew: true},
	})
}

// POST /{owner}/{repo}/wiki/new
func (s *Server) handleWikiCreate(w http.ResponseWriter, r *http.Request) {
	s.handleWikiSaveForm(w, r)
}

// GET /{owner}/{repo}/wiki/{page}/edit
func (s *Server) handleWikiEdit(w http.ResponseWriter, r *http.Request) {
	owner, repo, _, canWrite, ok := s.loadWikiCtx(w, r)
	if !ok {
		return
	}
	if !canWrite {
		s.renderError(w, http.StatusForbidden, "需要写权限")
		return
	}
	page := r.PathValue("page")
	content, err := wiki.GetPage(r.Context(), s.wikiPath(owner, repo), page)
	isNew := err != nil
	body := ""
	if !isNew {
		body = string(content)
	}
	s.render(w, http.StatusOK, "wiki_edit", pageData{
		Title: "编辑 Wiki: " + page,
		User:  UserFromContext(r.Context()),
		CSRF:  CSRFFromContext(r.Context()),
		Data:  wikiEditData{OwnerName: owner, RepoName: repo, Page: page, Content: body, IsNew: isNew},
	})
}

// POST /{owner}/{repo}/wiki/{page}/edit
func (s *Server) handleWikiSave(w http.ResponseWriter, r *http.Request) {
	s.handleWikiSaveForm(w, r)
}

func (s *Server) handleWikiSaveForm(w http.ResponseWriter, r *http.Request) {
	owner, repo, _, canWrite, ok := s.loadWikiCtx(w, r)
	if !ok {
		return
	}
	user := UserFromContext(r.Context())
	if !canWrite || user == nil {
		s.renderError(w, http.StatusForbidden, "需要写权限")
		return
	}
	page := r.PostFormValue("page")
	if page == "" {
		page = r.PathValue("page")
	}
	content := r.PostFormValue("content")
	msg := r.PostFormValue("msg")
	if msg == "" {
		msg = "update " + page
	}
	if err := wiki.SavePage(r.Context(), s.wikiPath(owner, repo), page, []byte(content), msg); err != nil {
		if errors.Is(err, wiki.ErrInvalidPage) {
			s.render(w, http.StatusBadRequest, "wiki_edit", pageData{
				Title: "Wiki",
				User:  user,
				CSRF:  CSRFFromContext(r.Context()),
				Data: wikiEditData{
					OwnerName: owner, RepoName: repo, Page: page, Content: content,
					Flash: Flash{Kind: "error", Message: "页面名只允许字母、数字、下划线、连字符（1-50 字符）"},
				},
			})
			return
		}
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	http.Redirect(w, r, "/"+owner+"/"+repo+"/wiki/"+page, http.StatusSeeOther)
}
