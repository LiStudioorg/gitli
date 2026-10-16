// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/pull"
)

type createPullReq struct {
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	HeadBranch string `json:"head_branch"`
	BaseBranch string `json:"base_branch"`
}

func (s *Server) handleCreatePull(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req createPullReq
	if !decodeJSON(w, r, &req) {
		return
	}
	rp, _, ok := s.resolveRepo(w, r, req.Owner, req.Repo, domain.PermWrite)
	if !ok {
		return
	}
	p, err := s.deps.Pulls.Create(r.Context(), pull.CreateParams{
		RepoID: rp.ID, AuthorID: u.ID, Title: req.Title, Body: req.Body,
		HeadRepoID: &rp.ID, HeadBranch: req.HeadBranch, BaseBranch: req.BaseBranch,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "pull.create", "success")
	s.dispatch(r, "pull_request", rp.ID, u.ID, map[string]any{
		"action": "opened", "number": p.Number, "title": p.Title, "pull_id": p.ID,
	})
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleListPulls(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.resolveRepo(w, r, r.URL.Query().Get("owner"), r.URL.Query().Get("repo"), domain.PermRead)
	if !ok {
		return
	}
	items, err := s.deps.Pulls.List(r.Context(), rp.ID, r.URL.Query().Get("state"), 30, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleGetPull(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.resolveRepo(w, r, r.URL.Query().Get("owner"), r.URL.Query().Get("repo"), domain.PermRead)
	if !ok {
		return
	}
	number, err := pathInt(r, "number")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid number")
		return
	}
	p, err := s.deps.Pulls.ByNumber(r.Context(), rp.ID, int(number))
	if err != nil {
		writeError(w, http.StatusNotFound, "pull request not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleMergePull(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	rp, _, ok := s.resolveRepo(w, r, r.URL.Query().Get("owner"), r.URL.Query().Get("repo"), domain.PermWrite)
	if !ok {
		return
	}
	number, err := pathInt(r, "number")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid number")
		return
	}
	p, err := s.deps.Pulls.ByNumber(r.Context(), rp.ID, int(number))
	if err != nil {
		writeError(w, http.StatusNotFound, "pull request not found")
		return
	}
	if err := s.deps.Pulls.MarkMerged(r.Context(), p.ID, u.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "pull.merge", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "merged"})
}

func (s *Server) handleListReviews(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.resolveRepo(w, r, r.URL.Query().Get("owner"), r.URL.Query().Get("repo"), domain.PermRead)
	if !ok {
		return
	}
	number, err := pathInt(r, "number")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid number")
		return
	}
	p, err := s.deps.Pulls.ByNumber(r.Context(), rp.ID, int(number))
	if err != nil {
		writeError(w, http.StatusNotFound, "pull request not found")
		return
	}
	reviews, err := s.deps.Pulls.ListReviews(r.Context(), p.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, reviews)
}

type reviewReq struct {
	State string `json:"state"`
	Body  string `json:"body"`
}

func (s *Server) handleCreateReview(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	rp, _, ok := s.resolveRepo(w, r, r.URL.Query().Get("owner"), r.URL.Query().Get("repo"), domain.PermRead)
	if !ok {
		return
	}
	number, err := pathInt(r, "number")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid number")
		return
	}
	p, err := s.deps.Pulls.ByNumber(r.Context(), rp.ID, int(number))
	if err != nil {
		writeError(w, http.StatusNotFound, "pull request not found")
		return
	}
	var req reviewReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.State == "" {
		req.State = "commented"
	}
	rev, err := s.deps.Pulls.AddReview(r.Context(), p.ID, u.ID, req.State, req.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rev)
}
