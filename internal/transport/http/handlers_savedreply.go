// SPDX-License-Identifier: AGPL-3.0
package http

import "net/http"

type savedReplyReq struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (s *Server) handleListSavedReplies(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	replies, err := s.deps.SavedReplies.List(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, replies)
}

func (s *Server) handleCreateSavedReply(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req savedReplyReq
	if !decodeJSON(w, r, &req) {
		return
	}
	reply, err := s.deps.SavedReplies.Create(r.Context(), u.ID, req.Title, req.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, reply)
}

func (s *Server) handleUpdateSavedReply(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	var req savedReplyReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.SavedReplies.Update(r.Context(), u.ID, id, req.Title, req.Body); err != nil {
		writeError(w, http.StatusNotFound, "保存的回复不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleDeleteSavedReply(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	if err := s.deps.SavedReplies.Delete(r.Context(), u.ID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
