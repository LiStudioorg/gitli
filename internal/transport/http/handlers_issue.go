// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"
	"time"

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

type labelReq struct {
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

func (s *Server) handleListLabels(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.resolveRepo(w, r, r.URL.Query().Get("owner"), r.URL.Query().Get("repo"), domain.PermRead)
	if !ok {
		return
	}
	labels, err := s.deps.Issues.ListLabels(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, labels)
}

func (s *Server) handleCreateLabel(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req labelReq
	if !decodeJSON(w, r, &req) {
		return
	}
	rp, _, ok := s.resolveRepo(w, r, req.Owner, req.Repo, domain.PermWrite)
	if !ok {
		return
	}
	l, err := s.deps.Issues.CreateLabel(r.Context(), rp.ID, req.Name, req.Color, req.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

func (s *Server) handleDeleteLabel(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.resolveRepo(w, r, r.URL.Query().Get("owner"), r.URL.Query().Get("repo"), domain.PermWrite)
	if !ok {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.deps.Issues.DeleteLabel(r.Context(), rp.ID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type milestoneReq struct {
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	Title       string `json:"title"`
	Description string `json:"description"`
	DueAt       string `json:"due_at"`
}

func (s *Server) handleListMilestones(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.resolveRepo(w, r, r.URL.Query().Get("owner"), r.URL.Query().Get("repo"), domain.PermRead)
	if !ok {
		return
	}
	ms, err := s.deps.Issues.ListMilestones(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ms)
}

func (s *Server) handleCreateMilestone(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req milestoneReq
	if !decodeJSON(w, r, &req) {
		return
	}
	rp, _, ok := s.resolveRepo(w, r, req.Owner, req.Repo, domain.PermWrite)
	if !ok {
		return
	}
	var dueAt *time.Time
	if req.DueAt != "" {
		if t, err := time.Parse(time.RFC3339, req.DueAt); err == nil {
			dueAt = &t
		}
	}
	m, err := s.deps.Issues.CreateMilestone(r.Context(), rp.ID, req.Title, req.Description, dueAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, m)
}
