// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
)

type createWebhookReq struct {
	URL    string   `json:"url"`
	Secret string   `json:"secret"`
	Events []string `json:"events"`
}

func (s *Server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	rp, u, ok := s.loadRepoAndPerm(w, r, domain.PermAdmin)
	if !ok {
		return
	}
	var req createWebhookReq
	if !decodeJSON(w, r, &req) {
		return
	}
	wh, err := s.deps.Webhooks.Create(r.Context(), &rp.ID, nil, req.URL, req.Secret, req.Events)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.auditAction(r, u.ID, "webhook.create", "success")
	writeJSON(w, http.StatusCreated, wh)
}

func (s *Server) handleListWebhooks(w http.ResponseWriter, r *http.Request) {
	rp, _, ok := s.loadRepoAndPerm(w, r, domain.PermAdmin)
	if !ok {
		return
	}
	hooks, err := s.deps.Webhooks.ListForRepo(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, h := range hooks {
		h.Secret = ""
	}
	writeJSON(w, http.StatusOK, hooks)
}

func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	unreadOnly := r.URL.Query().Get("unread") == "1"
	items, err := s.deps.Notifications.List(r.Context(), u.ID, unreadOnly, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	count, _ := s.deps.Notifications.UnreadCount(r.Context(), u.ID)
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "unread": count})
}

func (s *Server) handleMarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的 ID")
		return
	}
	if err := s.deps.Notifications.MarkRead(r.Context(), u.ID, id); err != nil {
		writeServiceError(w, err, "通知不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleMarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	if err := s.deps.Notifications.MarkAllRead(r.Context(), u.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
