package web

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"gitli/internal/assets"
	"gitli/internal/db"
)

var funcMap = template.FuncMap{
	"timefmt": func(unix int64) string {
		return time.Unix(unix, 0).UTC().Format("2006-01-02 15:04")
	},
}

// Renderer 渲染页面模板。
type Renderer struct {
	tmpl map[string]*template.Template
}

// NewRenderer 解析 layout + 页面模板。
func NewRenderer() (*Renderer, error) {
	r := &Renderer{tmpl: map[string]*template.Template{}}
	pages := []string{
		"home",
		"login",
		"register",
		"error",
		"user",
		"repo",
		"repo_tree",
		"repo_blob",
		"repo_commits",
		"repo_commit",
		"repo_blame",
		"repo_refs",
		"org_new",
		"org",
		"issues",
		"issue",
		"issue_new",
		"pulls",
		"pull",
		"pull_new",
		"wiki",
		"wiki_page",
		"wiki_edit",
		"admin",
		"admin_users",
		"admin_repos",
		"tokens",
		"security",
		"totp_setup",
		"totp_recovery",
		"totp_verify",
	}
	for _, name := range pages {
		t, err := template.New("layout.html").Funcs(funcMap).ParseFS(
			assets.Templates(),
			"templates/layouts/layout.html",
			"templates/partials/*.html",
			fmt.Sprintf("templates/pages/%s.html", name),
		)
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", name, err)
		}
		r.tmpl[name] = t
	}
	return r, nil
}

// pageData 传给布局模板的公共数据。
type pageData struct {
	Title        string
	User         *db.User
	CSRF         string
	Data         any
	OAuthEnabled bool
}

func (r *Renderer) Render(w http.ResponseWriter, status int, name string, data pageData) {
	t, ok := r.tmpl[name]
	if !ok {
		http.Error(w, fmt.Sprintf("template %q not found", name), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := t.ExecuteTemplate(w, "layout.html", data); err != nil {
		// 响应头已发出，只能记录；handler 层面无法再改状态码
		_ = err
	}
}

// render 由 Server 调用的统一渲染入口。
func (s *Server) render(w http.ResponseWriter, status int, name string, data pageData) {
	s.r.Render(w, status, name, data)
}

// renderError 渲染统一错误页。
func (s *Server) renderError(w http.ResponseWriter, status int, msg string) {
	s.r.RenderError(w, status, msg)
}

// Flash 简单闪存消息（页面内联，不做 cookie flash，M1 从简）。
type Flash struct {
	Kind    string // "error" | "info"
	Message string
}

// RenderError 渲染统一错误页。
func (r *Renderer) RenderError(w http.ResponseWriter, status int, msg string) {
	r.Render(w, status, "error", pageData{
		Title: "Error",
		Data:  Flash{Kind: "error", Message: msg},
	})
}
