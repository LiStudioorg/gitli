// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"errors"
	"net/http"

	"github.com/LiStudioorg/gitli/internal/service/audit"
	"github.com/LiStudioorg/gitli/internal/service/auth"
)

type setupReq struct {
	SiteName string `json:"site_name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// handleSetupStatus reports whether the instance still needs its first
// administrator. It never exposes authentication details.
func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	initialized, err := s.deps.Auth.IsInitialized(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read setup state")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"initialized": initialized})
}

// handleSetup creates the first administrator. The endpoint is single-use:
// once a user exists it returns 409 Conflict.
func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	var req setupReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := validateUsername(req.Username); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	u, err := s.deps.Auth.Setup(r.Context(), auth.SetupParams{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrAlreadySetUp):
			writeError(w, http.StatusConflict, "instance already initialized")
		case errors.Is(err, auth.ErrWeakPassword):
			writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	_ = s.deps.Audit.Record(r.Context(), audit.Entry{
		ActorID: &u.ID, Action: "user.setup", TargetType: "user", TargetID: &u.ID,
		IP: clientIP(r), Detail: "initial administrator",
	})
	writeJSON(w, http.StatusCreated, map[string]any{"id": u.ID, "username": u.Username})
}
