// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"errors"
	"net/http"

	"github.com/LiStudioorg/gitli/internal/service/userwebhook"
)

type createUserWebhookReq struct {
	URL    string   `json:"url"`
	Secret string   `json:"secret"`
	Events []string `json:"events"`
}

type updateUserWebhookReq struct {
	Active bool `json:"active"`
}

func (s *Server) handleCreateUserWebhook(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req createUserWebhookReq
	if !decodeJSON(w, r, &req) {
		return
	}
	h, err := s.deps.UserWebhooks.Create(r.Context(), u.ID, req.URL, req.Secret, req.Events)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, h)
}

func (s *Server) handleListUserWebhooks(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	hooks, err := s.deps.UserWebhooks.List(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for i := range hooks {
		hooks[i].Secret = ""
	}
	writeJSON(w, http.StatusOK, hooks)
}

func (s *Server) handleDeleteUserWebhook(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	if err := s.deps.UserWebhooks.Delete(r.Context(), u.ID, id); err != nil {
		if errors.Is(err, userwebhook.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Webhook 不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleUpdateUserWebhook(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	var req updateUserWebhookReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.UserWebhooks.SetActive(r.Context(), u.ID, id, req.Active); err != nil {
		if errors.Is(err, userwebhook.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Webhook 不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
