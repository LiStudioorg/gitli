package web

import (
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"gitli/internal/auth"
	"gitli/internal/git"
	"gitli/internal/pulls"
)

type pullView struct {
	Number    int64
	Title     string
	State     string
	Author    string
	Head      string
	Base      string
	Merged    bool
	CreatedAt int64
	UpdatedAt int64
	Labels    []interface{}
}

type pullsData struct {
	OwnerName string
	RepoName  string
	State     string
	Pulls     []pullView
	Flash     Flash
}

type pullNewData struct {
	OwnerName string
	RepoName  string
	Branches  []string
	DefaultBase string
	Flash     Flash
}

type pullData struct {
	OwnerName string
	RepoName  string
	Number    int64
	Title     string
	State     string
	Author    string
	Head      string
	Base      string
	Merged    bool
	Closed    bool
	Body      string
	CreatedAt int64
	Comments  []commentView
	DiffHTML  template.HTML
	DiffLines []git.DiffLine
	HasDiff   bool
	Commits   int
	CanMerge  bool
	Mergeable bool
	Flash     Flash
}

// GET /{owner}/{repo}/pulls?state=open|closed|all
func (s *Server) handlePulls(w http.ResponseWriter, r *http.Request) {
	repo, _, ok := s.loadIssueCtx(w, r)
	if !ok {
		return
	}
	ownerName := r.PathValue("owner")
	repoName := r.PathValue("repo")
	state := r.URL.Query().Get("state")
	rows, err := s.pulls.List(r.Context(), repo.ID, state)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	views := make([]pullView, 0, len(rows))
	for _, row := range rows {
		stateStr := "open"
		if row.Closed == 1 {
			stateStr = "closed"
		}
		author := ""
		if u, uerr := s.users.GetByID(r.Context(), row.AuthorID); uerr == nil {
			author = u.Username
		}
		views = append(views, pullView{
			Number:    row.Number,
			Title:     row.Title,
			State:     stateStr,
			Author:    author,
			Head:      row.HeadBranch,
			Base:      row.BaseBranch,
			Merged:    row.Merged == 1,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}
	if state == "" {
		state = "open"
	}
	s.render(w, http.StatusOK, "pulls", pageData{
		Title: ownerName + "/" + repoName + " Pull Requests",
		User:  UserFromContext(r.Context()),
		CSRF:  CSRFFromContext(r.Context()),
		Data: pullsData{
			OwnerName: ownerName,
			RepoName:  repoName,
			State:     state,
			Pulls:     views,
		},
	})
}

// GET /{owner}/{repo}/pulls/new
func (s *Server) handlePullNew(w http.ResponseWriter, r *http.Request) {
	repo, repoPath, ok := s.loadIssueCtx(w, r)
	if !ok {
		return
	}
	user := UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	branches, err := git.ListBranches(r.Context(), repoPath)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	names := make([]string, 0, len(branches))
	for _, b := range branches {
		names = append(names, b.Name)
	}
	qbase := r.URL.Query().Get("base")
	qhead := r.URL.Query().Get("head")
	s.render(w, http.StatusOK, "pull_new", pageData{
		Title: "新建 Pull Request",
		User:  user,
		CSRF:  CSRFFromContext(r.Context()),
		Data: pullNewData{
			OwnerName:   r.PathValue("owner"),
			RepoName:    repo.Name,
			Branches:    names,
			DefaultBase: qbase,
			Flash:       Flash{},
		},
	})
	_ = qhead
}

// POST /{owner}/{repo}/pulls/new
func (s *Server) handlePullCreate(w http.ResponseWriter, r *http.Request) {
	repo, repoPath, ok := s.loadIssueCtx(w, r)
	if !ok {
		return
	}
	user := UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	base := strings.TrimSpace(r.PostFormValue("base"))
	head := strings.TrimSpace(r.PostFormValue("head"))
	title := strings.TrimSpace(r.PostFormValue("title"))
	body := r.PostFormValue("body")
	issue, err := s.pulls.Create(r.Context(), repo, repoPath, base, head, title, body, user.ID)
	if err != nil {
		msg := "创建失败"
		switch {
		case errors.Is(err, pulls.ErrSameBranch):
			msg = "base 与 head 分支不能相同"
		case errors.Is(err, pulls.ErrInvalidInput):
			msg = err.Error()
		default:
			if errors.Is(err, git.ErrMergeConflict) {
				msg = err.Error()
			} else {
				s.renderError(w, http.StatusInternalServerError, "internal error")
				return
			}
		}
		branches, _ := git.ListBranches(r.Context(), repoPath)
		names := make([]string, 0, len(branches))
		for _, b := range branches {
			names = append(names, b.Name)
		}
		s.render(w, http.StatusBadRequest, "pull_new", pageData{
			Title: "新建 Pull Request",
			User:  user,
			CSRF:  CSRFFromContext(r.Context()),
			Data: pullNewData{
				OwnerName: r.PathValue("owner"),
				RepoName:  repo.Name,
				Branches:  names,
				Flash:     Flash{Kind: "error", Message: msg},
			},
		})
		return
	}
	http.Redirect(w, r, "/"+r.PathValue("owner")+"/"+repo.Name+"/pulls/"+strconv.FormatInt(issue.Number, 10), http.StatusSeeOther)
}

// GET /{owner}/{repo}/pulls/{number}
func (s *Server) handlePull(w http.ResponseWriter, r *http.Request) {
	repo, repoPath, ok := s.loadIssueCtx(w, r)
	if !ok {
		return
	}
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "无效编号")
		return
	}
	detail, err := s.pulls.Get(r.Context(), repo.ID, number)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "PR 不存在")
		return
	}
	issue := detail.Issue
	pull := detail.Pull
	user := UserFromContext(r.Context())
	ownerName := r.PathValue("owner")
	repoName := r.PathValue("repo")

	// diff 与可合并性
	diffLines := []git.DiffLine{}
	hasDiff := false
	mergeable := false
	commitsAhead := 0
	if issue.Closed != 1 && pull.Merged != 1 {
		if lines, derr := s.pulls.Diff(r.Context(), repoPath, pull.BaseBranch, pull.HeadBranch); derr == nil {
			diffLines = lines
			hasDiff = true
		}
		mergeable = git.MergeAvailable(r.Context(), repoPath, pull.BaseBranch, pull.HeadBranch)
		if mb, merr := git.MergeBase(r.Context(), repoPath, pull.BaseBranch, pull.HeadBranch); merr == nil {
			if list, lerr := git.LogCommitsRange(r.Context(), repoPath, mb, pull.HeadBranch); lerr == nil {
				commitsAhead = len(list)
			}
		}
	}

	// 评论
	comments, _ := s.issues.ListComments(r.Context(), issue.ID)
	commentViews := make([]commentView, 0, len(comments))
	for _, c := range comments {
		author := ""
		if u, uerr := s.users.GetByID(r.Context(), c.AuthorID); uerr == nil {
			author = u.Username
		}
		commentViews = append(commentViews, commentView{ID: c.ID, Author: author, Body: c.Body, CreatedAt: c.CreatedAt})
	}

	state := "open"
	if pull.Merged == 1 {
		state = "merged"
	} else if issue.Closed == 1 {
		state = "closed"
	}

	s.render(w, http.StatusOK, "pull", pageData{
		Title: ownerName + "/" + repoName + " PR #" + strconv.FormatInt(number, 10),
		User:  user,
		CSRF:  CSRFFromContext(r.Context()),
		Data: pullData{
			OwnerName: ownerName,
			RepoName:  repoName,
			Number:    number,
			Title:     issue.Title,
			State:     state,
			Author:    s.usernameByID(r, issue.AuthorID),
			Head:      pull.HeadBranch,
			Base:      pull.BaseBranch,
			Merged:    pull.Merged == 1,
			Closed:    issue.Closed == 1,
			Body:      issue.Body,
			CreatedAt: issue.CreatedAt,
			Comments:  commentViews,
			DiffLines: diffLines,
			HasDiff:   hasDiff,
			Commits:   commitsAhead,
			CanMerge:  user != nil && auth.CanAccess(r.Context(), s.q, user, repo, true),
			Mergeable: mergeable,
		},
	})
}

// POST /{owner}/{repo}/pulls/{number}/merge
func (s *Server) handlePullMerge(w http.ResponseWriter, r *http.Request) {
	repo, repoPath, ok := s.loadIssueCtx(w, r)
	if !ok {
		return
	}
	user := UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if !auth.CanAccess(r.Context(), s.q, user, repo, true) {
		s.renderError(w, http.StatusForbidden, "无合并权限")
		return
	}
	number, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "无效编号")
		return
	}
	detail, err := s.pulls.Get(r.Context(), repo.ID, number)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "PR 不存在")
		return
	}
	method := r.PostFormValue("method")
	msg := r.PostFormValue("msg")
	commit, err := s.pulls.Merge(r.Context(), repo, repoPath, detail, method, msg)
	if err != nil {
		msg := "合并失败"
		if errors.Is(err, pulls.ErrConflict) || errors.Is(err, git.ErrMergeConflict) {
			msg = "存在冲突，无法合并"
		} else if errors.Is(err, pulls.ErrAlreadyMerged) {
			msg = "PR 已合并或关闭"
		}
		http.Redirect(w, r, "/"+r.PathValue("owner")+"/"+repo.Name+"/pulls/"+strconv.FormatInt(number, 10), http.StatusSeeOther)
		_ = msg
		return
	}
	_ = commit
	http.Redirect(w, r, "/"+r.PathValue("owner")+"/"+repo.Name+"/pulls/"+strconv.FormatInt(number, 10), http.StatusSeeOther)
}

func (s *Server) usernameByID(r *http.Request, id int64) string {
	if u, err := s.users.GetByID(r.Context(), id); err == nil {
		return u.Username
	}
	return ""
}