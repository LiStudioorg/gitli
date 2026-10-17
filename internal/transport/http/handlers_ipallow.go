// SPDX-License-Identifier: AGPL-3.0
package http

import "net/http"

func (s *Server) handleListIPAllow(w http.ResponseWriter, r *http.Request) {
	o, err := s.deps.Orgs.ByName(r.Context(), urlParam(r, "org"))
	if err != nil {
		writeError(w, http.StatusNotFound, "organization not found")
		return
	}
	rules, err := s.deps.IPAllow.List(r.Context(), o.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	enabled, _ := s.deps.IPAllow.Enabled(r.Context(), o.ID)
	writeJSON(w, http.StatusOK, map[string]any{"rules": rules, "enabled": enabled})
}

type ipAllowReq struct {
	CIDR        string `json:"cidr"`
	Description string `json:"description"`
	Enabled     *bool  `json:"enabled"`
}

func (s *Server) handleAddIPAllow(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	o, err := s.deps.Orgs.ByName(r.Context(), urlParam(r, "org"))
	if err != nil {
		writeError(w, http.StatusNotFound, "organization not found")
		return
	}
	var req ipAllowReq
	if !decodeJSON(w, r, &req) {
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	rule, err := s.deps.IPAllow.Add(r.Context(), o.ID, req.CIDR, req.Description, enabled)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.auditAction(r, u.ID, "org.ip_allowlist_add", "success")
	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) handleDeleteIPAllow(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	o, err := s.deps.Orgs.ByName(r.Context(), urlParam(r, "org"))
	if err != nil {
		writeError(w, http.StatusNotFound, "organization not found")
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.deps.IPAllow.Delete(r.Context(), o.ID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "org.ip_allowlist_delete", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type ipAllowToggleReq struct {
	Enabled bool `json:"enabled"`
}

func (s *Server) handleToggleIPAllow(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	o, err := s.deps.Orgs.ByName(r.Context(), urlParam(r, "org"))
	if err != nil {
		writeError(w, http.StatusNotFound, "organization not found")
		return
	}
	var req ipAllowToggleReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.IPAllow.SetEnabled(r.Context(), o.ID, req.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "org.ip_allowlist_toggle", "success")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "enabled": req.Enabled})
}
