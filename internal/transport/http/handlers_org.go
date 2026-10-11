// SPDX-License-Identifier: AGPL-3.0
package http

import "net/http"

type createOrgReq struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

func (s *Server) handleCreateOrg(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	var req createOrgReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := validateName("org", req.Name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	o, err := s.deps.Orgs.Create(r.Context(), req.Name, req.DisplayName, u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.auditAction(r, u.ID, "org.create", "success")
	writeJSON(w, http.StatusCreated, o)
}
