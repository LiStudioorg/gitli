// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"
	"strconv"

	"github.com/LiStudioorg/gitli/internal/domain"
)

func (s *Server) handleListDiscussionCategories(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	_ = s.deps.Discussions.EnsureCategories(r.Context(), rp.ID)
	cats, err := s.deps.Discussions.ListCategories(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cats)
}

func (s *Server) handleListDiscussions(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	var catID *int64
	if v := r.URL.Query().Get("category_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			catID = &n
		}
	}
	list, err := s.deps.Discussions.List(r.Context(), rp.ID, catID, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

type createDiscussionReq struct {
	Title      string `json:"title"`
	Body       string `json:"body"`
	CategoryID *int64 `json:"category_id"`
}

func (s *Server) handleCreateDiscussion(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermWrite)
	if !ok {
		return
	}
	_ = s.deps.Discussions.EnsureCategories(r.Context(), rp.ID)
	var req createDiscussionReq
	if !decodeJSON(w, r, &req) {
		return
	}
	d, err := s.deps.Discussions.Create(r.Context(), rp.ID, u.ID, req.CategoryID, req.Title, req.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.dispatch(r, "discussion", rp.ID, u.ID, map[string]any{"action": "created", "id": d.ID, "title": d.Title})
	writeJSON(w, http.StatusCreated, d)
}

func (s *Server) handleGetDiscussion(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	d, err := s.deps.Discussions.ByID(r.Context(), id)
	if err != nil || d.RepoID != rp.ID {
		writeError(w, http.StatusNotFound, "讨论不存在")
		return
	}
	comments, _ := s.deps.Discussions.ListComments(r.Context(), id)
	writeJSON(w, http.StatusOK, map[string]any{"discussion": d, "comments": comments})
}

type discussionCommentReq struct {
	Body string `json:"body"`
}

func (s *Server) handleCreateDiscussionComment(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermRead)
	if !ok {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	d, err := s.deps.Discussions.ByID(r.Context(), id)
	if err != nil || d.RepoID != rp.ID {
		writeError(w, http.StatusNotFound, "讨论不存在")
		return
	}
	var req discussionCommentReq
	if !decodeJSON(w, r, &req) {
		return
	}
	c, err := s.deps.Discussions.AddComment(r.Context(), id, u.ID, req.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleMarkDiscussionAnswer(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermWrite)
	if !ok {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	d, err := s.deps.Discussions.ByID(r.Context(), id)
	if err != nil || d.RepoID != rp.ID {
		writeError(w, http.StatusNotFound, "讨论不存在")
		return
	}
	var req struct {
		CommentID int64 `json:"comment_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.Discussions.MarkAnswer(r.Context(), id, req.CommentID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.auditAction(r, u.ID, "discussion.mark_answer", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
