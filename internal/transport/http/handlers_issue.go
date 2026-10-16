// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/comment"
)

type createIssueReq struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (s *Server) handleCreateIssue(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req createIssueReq
	if !decodeJSON(w, r, &req) {
		return
	}
	rp, _, ok := s.resolveRepo(w, r, req.Owner, req.Repo, domain.PermWrite)
	if !ok {
		return
	}
	is, err := s.deps.Issues.Create(r.Context(), rp.ID, u.ID, req.Title, req.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "issue.create", "success")
	s.dispatch(r, "issues", rp.ID, u.ID, map[string]any{
		"action": "opened", "number": is.Number, "title": is.Title, "issue_id": is.ID,
	})
	writeJSON(w, http.StatusCreated, is)
}

func (s *Server) handleListIssues(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	repoName := r.URL.Query().Get("repo")
	rp, _, ok := s.resolveRepo(w, r, owner, repoName, domain.PermRead)
	if !ok {
		return
	}
	state := r.URL.Query().Get("state")
	items, err := s.deps.Issues.List(r.Context(), rp.ID, state, 30, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleGetIssue(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	repoName := r.URL.Query().Get("repo")
	rp, _, ok := s.resolveRepo(w, r, owner, repoName, domain.PermRead)
	if !ok {
		return
	}
	number, err := pathInt(r, "number")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid number")
		return
	}
	is, err := s.deps.Issues.ByNumber(r.Context(), rp.ID, int(number))
	if err != nil {
		writeError(w, http.StatusNotFound, "issue not found")
		return
	}
	writeJSON(w, http.StatusOK, is)
}

type stateReq struct {
	State string `json:"state"`
}

func (s *Server) handleSetIssueState(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	owner := r.URL.Query().Get("owner")
	repoName := r.URL.Query().Get("repo")
	rp, _, ok := s.resolveRepo(w, r, owner, repoName, domain.PermWrite)
	if !ok {
		return
	}
	number, err := pathInt(r, "number")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid number")
		return
	}
	is, err := s.deps.Issues.ByNumber(r.Context(), rp.ID, int(number))
	if err != nil {
		writeError(w, http.StatusNotFound, "issue not found")
		return
	}
	var req stateReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.Issues.SetState(r.Context(), is.ID, req.State); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleListIssueComments(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	repoName := r.URL.Query().Get("repo")
	rp, _, ok := s.resolveRepo(w, r, owner, repoName, domain.PermRead)
	if !ok {
		return
	}
	number, err := pathInt(r, "number")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid number")
		return
	}
	is, err := s.deps.Issues.ByNumber(r.Context(), rp.ID, int(number))
	if err != nil {
		writeError(w, http.StatusNotFound, "issue not found")
		return
	}
	comments, err := s.deps.Comments.ListForIssue(r.Context(), is.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, comments)
}

type commentReq struct {
	Body string `json:"body"`
}

func (s *Server) handleCreateIssueComment(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	owner := r.URL.Query().Get("owner")
	repoName := r.URL.Query().Get("repo")
	rp, _, ok := s.resolveRepo(w, r, owner, repoName, domain.PermRead)
	if !ok {
		return
	}
	number, err := pathInt(r, "number")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid number")
		return
	}
	is, err := s.deps.Issues.ByNumber(r.Context(), rp.ID, int(number))
	if err != nil {
		writeError(w, http.StatusNotFound, "issue not found")
		return
	}
	var req commentReq
	if !decodeJSON(w, r, &req) {
		return
	}
	c, err := s.deps.Comments.Create(r.Context(), comment.CreateParams{
		RepoID: rp.ID, IssueID: &is.ID, AuthorID: u.ID, Body: req.Body,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.dispatch(r, "issue_comment", rp.ID, u.ID, map[string]any{
		"action": "created", "issue_number": is.Number, "comment_id": c.ID,
	})
	writeJSON(w, http.StatusCreated, c)
}
