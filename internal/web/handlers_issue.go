package web

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"gitli/internal/auth"
	"gitli/internal/db"
	"gitli/internal/issues"
)

type issueView struct {
	Number    int64
	Title     string
	Body      string
	State     string
	Author    string
	CreatedAt int64
	UpdatedAt int64
	Labels    []db.Label
}

type commentView struct {
	ID        int64
	Author    string
	Body      string
	CreatedAt int64
}

type issuesData struct {
	OwnerName string
	RepoName  string
	State     string
	Issues    []issueView
	Flash     Flash
}

type issueData struct {
	OwnerName string
	RepoName  string
	Issue     issueView
	Comments  []commentView
	IsPull    bool
	CanAct    bool
	Labels    []db.Label
	Flash     Flash
}

type issueNewData struct {
	OwnerName string
	RepoName  string
	Flash     Flash
}

// loadIssueCtx 解析 owner/repo 并校验读权限；返回 repo、磁盘路径。
func (s *Server) loadIssueCtx(w http.ResponseWriter, r *http.Request) (db.Repo, string, bool) {
	ownerName := r.PathValue("owner")
	repoName := r.PathValue("repo")
	oi, err := s.repos.GetWithKind(r.Context(), ownerName, repoName)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "仓库不存在")
		return db.Repo{}, "", false
	}
	user := UserFromContext(r.Context())
	if !auth.CanAccess(r.Context(), s.q, user, oi.Repo, false) {
		s.renderError(w, http.StatusNotFound, "仓库不存在")
		return db.Repo{}, "", false
	}
	return oi.Repo, s.repos.DiskPath(oi.OwnerName, repoName), true
}

// issueToView 构造 issue 列表/详情视图（r 已在闭包内可用）。
func (s *Server) issueToView(r *http.Request, issue db.Issue, labels []db.Label) issueView {
	state := "open"
	if issue.Closed == 1 {
		state = "closed"
	}
	author := ""
	if u, err := s.users.GetByID(r.Context(), issue.AuthorID); err == nil {
		author = u.Username
	}
	return issueView{
		Number:    issue.Number,
		Title:     issue.Title,
		Body:      issue.Body,
		State:     state,
		Author:    author,
		CreatedAt: issue.CreatedAt,
		UpdatedAt: issue.UpdatedAt,
		Labels:    labels,
	}
}

// GET /{owner}/{repo}/issues?state=open|closed|all
func (s *Server) handleIssues(w http.ResponseWriter, r *http.Request) {
	repo, _, ok := s.loadIssueCtx(w, r)
	if !ok {
		return
	}
	ownerName := r.PathValue("owner")
	repoName := r.PathValue("repo")
	state := r.URL.Query().Get("state")
	list, err := s.issues.List(r.Context(), repo.ID, 0, state)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	views := make([]issueView, 0, len(list))
	for _, is := range list {
		views = append(views, s.issueToView(r, is, nil))
	}
	if state == "" {
		state = "open"
	}
	s.render(w, http.StatusOK, "issues", pageData{
		Title: ownerName + "/" + repoName + " Issues",
		User:  UserFromContext(r.Context()),
		CSRF:  CSRFFromContext(r.Context()),
		Data: issuesData{
			OwnerName: ownerName,
			RepoName:  repoName,
			State:     state,
			Issues:    views,
		},
	})
}

// GET /{owner}/{repo}/issues/new
func (s *Server) handleIssueNew(w http.ResponseWriter, r *http.Request) {
	repo, _, ok := s.loadIssueCtx(w, r)
	if !ok {
		return
	}
	user := UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	_ = repo
	s.render(w, http.StatusOK, "issue_new", pageData{
		Title: "新建 Issue",
		User:  user,
		CSRF:  CSRFFromContext(r.Context()),
		Data: issueNewData{
			OwnerName: r.PathValue("owner"),
			RepoName:  r.PathValue("repo"),
		},
	})
}

// POST /{owner}/{repo}/issues/new
func (s *Server) handleIssueCreate(w http.ResponseWriter, r *http.Request) {
	repo, _, ok := s.loadIssueCtx(w, r)
	if !ok {
		return
	}
	user := UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	title := r.PostFormValue("title")
	body := r.PostFormValue("body")
	issue, err := s.issues.Create(r.Context(), repo.ID, user.ID, title, body, 0, sql.NullInt64{})
	if err != nil {
		if errors.Is(err, issues.ErrInvalidInput) {
			s.render(w, http.StatusBadRequest, "issue_new", pageData{
				Title: "新建 Issue",
				User:  user,
				CSRF:  CSRFFromContext(r.Context()),
				Data: issueNewData{
					OwnerName: r.PathValue("owner"),
					RepoName:  r.PathValue("repo"),
					Flash:     Flash{Kind: "error", Message: "标题不能为空"},
				},
			})
			return
		}
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	base := "/" + r.PathValue("owner") + "/" + r.PathValue("repo")
	http.Redirect(w, r, base+"/issues/"+strconv.FormatInt(issue.Number, 10), http.StatusSeeOther)
}

// GET /{owner}/{repo}/issues/{number}
func (s *Server) handleIssue(w http.ResponseWriter, r *http.Request) {
	repo, _, ok := s.loadIssueCtx(w, r)
	if !ok {
		return
	}
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "无效编号")
		return
	}
	issue, err := s.issues.Get(r.Context(), repo.ID, number)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "Issue 不存在")
		return
	}
	// PR 型 issue 重定向到 pulls 页面
	if issue.IsPull == 1 {
		http.Redirect(w, r, "/"+r.PathValue("owner")+"/"+r.PathValue("repo")+"/pulls/"+r.PathValue("number"), http.StatusSeeOther)
		return
	}
	s.renderIssueDetail(w, r, repo, issue, false)
}

// 渲染 issue 详情（issues 与 PR 评论共用）。
func (s *Server) renderIssueDetail(w http.ResponseWriter, r *http.Request, repo db.Repo, issue db.Issue, isPull bool) {
	ownerName := r.PathValue("owner")
	repoName := r.PathValue("repo")
	user := UserFromContext(r.Context())

	labels, _ := s.issues.ListLabels(r.Context(), issue.ID)
	comments, err := s.issues.ListComments(r.Context(), issue.ID)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	commentViews := make([]commentView, 0, len(comments))
	for _, c := range comments {
		author := ""
		if u, uerr := s.users.GetByID(r.Context(), c.AuthorID); uerr == nil {
			author = u.Username
		}
		commentViews = append(commentViews, commentView{ID: c.ID, Author: author, Body: c.Body, CreatedAt: c.CreatedAt})
	}

	// 评论/关闭/打标签需登录 + 可读
	canAct := user != nil && auth.CanAccess(r.Context(), s.q, user, repo, false)

	name := "issue"
	if isPull {
		name = "pull"
	}
	s.render(w, http.StatusOK, name, pageData{
		Title: ownerName + "/" + repoName + " #" + strconv.FormatInt(issue.Number, 10),
		User:  user,
		CSRF:  CSRFFromContext(r.Context()),
		Data: issueData{
			OwnerName: ownerName,
			RepoName:  repoName,
			Issue:     s.issueToView(r, issue, labels),
			Comments:  commentViews,
			IsPull:    isPull,
			CanAct:    canAct,
		},
	})
}

// POST /{owner}/{repo}/issues/{number}/comment
func (s *Server) handleIssueComment(w http.ResponseWriter, r *http.Request) {
	repo, _, ok := s.loadIssueCtx(w, r)
	if !ok {
		return
	}
	user := UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "无效编号")
		return
	}
	issue, err := s.issues.Get(r.Context(), repo.ID, number)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "Issue 不存在")
		return
	}
	if _, err := s.issues.Comment(r.Context(), issue.ID, user.ID, r.PostFormValue("body")); err != nil {
		s.renderError(w, http.StatusBadRequest, "评论内容不能为空")
		return
	}
	base := "/" + r.PathValue("owner") + "/" + r.PathValue("repo")
	kind := "issues"
	if issue.IsPull == 1 {
		kind = "pulls"
	}
	http.Redirect(w, r, base+"/"+kind+"/"+strconv.FormatInt(number, 10), http.StatusSeeOther)
}

// POST /{owner}/{repo}/issues/{number}/state
func (s *Server) handleIssueState(w http.ResponseWriter, r *http.Request) {
	repo, _, ok := s.loadIssueCtx(w, r)
	if !ok {
		return
	}
	user := UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "无效编号")
		return
	}
	issue, err := s.issues.Get(r.Context(), repo.ID, number)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "Issue 不存在")
		return
	}
	switch r.PostFormValue("state") {
	case "close":
		err = s.issues.Close(r.Context(), issue.ID)
	case "reopen":
		err = s.issues.Reopen(r.Context(), issue.ID)
	default:
		err = issues.ErrInvalidInput
	}
	if err != nil {
		s.renderError(w, http.StatusBadRequest, "无效状态")
		return
	}
	base := "/" + r.PathValue("owner") + "/" + r.PathValue("repo")
	kind := "issues"
	if issue.IsPull == 1 {
		kind = "pulls"
	}
	http.Redirect(w, r, base+"/"+kind+"/"+strconv.FormatInt(number, 10), http.StatusSeeOther)
}

// POST /{owner}/{repo}/issues/{number}/labels
func (s *Server) handleIssueLabels(w http.ResponseWriter, r *http.Request) {
	repo, _, ok := s.loadIssueCtx(w, r)
	if !ok {
		return
	}
	user := UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "无效编号")
		return
	}
	issue, err := s.issues.Get(r.Context(), repo.ID, number)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "Issue 不存在")
		return
	}
	labelName := r.PostFormValue("label")
	action := r.PostFormValue("action")
	label, lerr := s.issues.GetLabelByName(r.Context(), repo.ID, labelName)
	if lerr != nil {
		// 不存在则自动创建
		label, lerr = s.issues.CreateLabel(r.Context(), repo.ID, labelName, "")
		if lerr != nil {
			s.renderError(w, http.StatusInternalServerError, "internal error")
			return
		}
	}
	if action == "remove" {
		err = s.issues.RemoveLabel(r.Context(), issue.ID, label.ID)
	} else {
		err = s.issues.AddLabel(r.Context(), issue.ID, label.ID)
	}
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	base := "/" + r.PathValue("owner") + "/" + r.PathValue("repo")
	kind := "issues"
	if issue.IsPull == 1 {
		kind = "pulls"
	}
	http.Redirect(w, r, base+"/"+kind+"/"+strconv.FormatInt(number, 10), http.StatusSeeOther)
}
