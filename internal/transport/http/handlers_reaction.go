// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"
	"strconv"
)

type reactionReq struct {
	TargetType string `json:"target_type"`
	TargetID   int64  `json:"target_id"`
	Emoji      string `json:"emoji"`
}

func (s *Server) handleAddReaction(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req reactionReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.Reactions.Add(r.Context(), u.ID, req.TargetType, req.TargetID, req.Emoji); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleRemoveReaction(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req reactionReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.Reactions.Remove(r.Context(), u.ID, req.TargetType, req.TargetID, req.Emoji); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleListReactions(w http.ResponseWriter, r *http.Request) {
	targetType := r.URL.Query().Get("target_type")
	targetID, err := strconv.ParseInt(r.URL.Query().Get("target_id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid target_id")
		return
	}
	counts, err := s.deps.Reactions.List(r.Context(), targetType, targetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := map[string]any{"counts": counts}
	if u := userFrom(r.Context()); u != nil {
		mine, err := s.deps.Reactions.ListByUser(r.Context(), u.ID, targetType, targetID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if mine == nil {
			mine = []string{}
		}
		resp["mine"] = mine
	}
	writeJSON(w, http.StatusOK, resp)
}

type subscriptionReq struct {
	TargetType string `json:"target_type"`
	TargetID   int64  `json:"target_id"`
}

func (s *Server) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req subscriptionReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.Subscriptions.Subscribe(r.Context(), u.ID, req.TargetType, req.TargetID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req subscriptionReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.Subscriptions.Unsubscribe(r.Context(), u.ID, req.TargetType, req.TargetID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleGetSubscription(w http.ResponseWriter, r *http.Request) {
	targetType := r.URL.Query().Get("target_type")
	targetID, err := strconv.ParseInt(r.URL.Query().Get("target_id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid target_id")
		return
	}
	subscribed := false
	if u := userFrom(r.Context()); u != nil {
		subscribed, err = s.deps.Subscriptions.IsSubscribed(r.Context(), u.ID, targetType, targetID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"subscribed": subscribed})
}
