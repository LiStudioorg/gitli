// SPDX-License-Identifier: AGPL-3.0
package http

import "net/http"

type blockReq struct {
	Username string `json:"username"`
}

type notifPrefReq struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type org2FAPolicyReq struct {
	Required bool `json:"required"`
}

// handleBlockUser blocks a user by username.
func (s *Server) handleBlockUser(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req blockReq
	if !decodeJSON(w, r, &req) {
		return
	}
	target, err := s.deps.Users.ByUsername(r.Context(), req.Username)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err := s.deps.Blocks.Block(r.Context(), u.ID, target.ID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"blocked": target.Username})
}

// handleUnblockUser removes a block by username.
func (s *Server) handleUnblockUser(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	target, err := s.deps.Users.ByUsername(r.Context(), urlParam(r, "username"))
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err := s.deps.Blocks.Unblock(r.Context(), u.ID, target.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"unblocked": target.Username})
}

// handleListBlockedUsers returns the usernames the caller has blocked.
func (s *Server) handleListBlockedUsers(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	ids, err := s.deps.Blocks.ListBlocked(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		blocked, err := s.deps.Users.ByID(r.Context(), id)
		if err != nil {
			continue
		}
		names = append(names, blocked.Username)
	}
	writeJSON(w, http.StatusOK, map[string]any{"blocked": names})
}

// handleListNotifPrefs returns all notification preferences for the caller.
func (s *Server) handleListNotifPrefs(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	prefs, err := s.deps.NotifPrefs.All(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, prefs)
}

// handleSetNotifPref upserts a notification preference for the caller.
func (s *Server) handleSetNotifPref(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req notifPrefReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Key == "" {
		writeError(w, http.StatusBadRequest, "key required")
		return
	}
	if err := s.deps.NotifPrefs.Set(r.Context(), u.ID, req.Key, req.Value); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"key": req.Key, "value": req.Value})
}

// orgAdmin checks that the caller is the org owner or a site admin.
func (s *Server) orgAdmin(w http.ResponseWriter, r *http.Request) (int64, bool) {
	u := requireUser(w, r)
	if u == nil {
		return 0, false
	}
	o, err := s.deps.Orgs.ByName(r.Context(), urlParam(r, "org"))
	if err != nil {
		writeError(w, http.StatusNotFound, "organization not found")
		return 0, false
	}
	if o.OwnerID != u.ID && !u.IsAdmin {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return 0, false
	}
	return o.ID, true
}

// handleGetOrg2FAPolicy returns the org's 2FA policy.
func (s *Server) handleGetOrg2FAPolicy(w http.ResponseWriter, r *http.Request) {
	orgID, ok := s.orgAdmin(w, r)
	if !ok {
		return
	}
	required, err := s.deps.Org2FA.Get(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"required": required})
}

// handleSetOrg2FAPolicy upserts the org's 2FA policy.
func (s *Server) handleSetOrg2FAPolicy(w http.ResponseWriter, r *http.Request) {
	orgID, ok := s.orgAdmin(w, r)
	if !ok {
		return
	}
	var req org2FAPolicyReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.Org2FA.Set(r.Context(), orgID, req.Required); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"required": req.Required})
}
