// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"errors"
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/audit"
	"github.com/LiStudioorg/gitli/internal/service/auth"
)

type registerReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := validateUsername(req.Username); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	u, err := s.deps.Auth.Register(r.Context(), auth.RegisterParams{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		s.auditLogin(r, nil, "register", false)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = s.deps.Audit.Record(r.Context(), audit.Entry{
		ActorID: &u.ID, Action: "user.register", TargetType: "user", TargetID: &u.ID,
		IP: clientIP(r), Detail: "ok",
	})
	writeJSON(w, http.StatusCreated, map[string]any{"id": u.ID, "username": u.Username})
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	TOTPCode string `json:"totp_code"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if !decodeJSON(w, r, &req) {
		return
	}
	res, err := s.deps.Auth.Login(r.Context(), auth.LoginParams{
		Username: req.Username,
		Password: req.Password,
		TOTPCode: req.TOTPCode,
		IP:       clientIP(r),
		Agent:    r.UserAgent(),
	})
	if err != nil {
		s.auditLogin(r, nil, "login", false)
		if errors.Is(err, auth.ErrTOTPRequired) {
			writeError(w, http.StatusUnauthorized, "totp_required")
			return
		}
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	s.auditLogin(r, &res.User.ID, "login", true)
	http.SetCookie(w, &http.Cookie{
		Name:     "gitli_session",
		Value:    res.RawToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.deps.HTTPSecure(),
		MaxAge:   30 * 24 * 3600,
	})
	writeJSON(w, http.StatusOK, map[string]any{"id": res.User.ID, "username": res.User.Username})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("gitli_session"); err == nil {
		_ = s.deps.Auth.Logout(r.Context(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "gitli_session", Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) auditLogin(r *http.Request, userID *int64, action string, ok bool) {
	result := "failure"
	if ok {
		result = "success"
	}
	_ = s.deps.Audit.Record(r.Context(), audit.Entry{
		ActorID: userID, Action: "auth." + action, TargetType: "user",
		IP: clientIP(r), Detail: result + " agent=" + r.UserAgent(),
	})
}

// reauth verifies the current password (and TOTP when enabled) for sensitive ops.
func (s *Server) reauth(r *http.Request, u *domain.User, password, totpCode string) bool {
	if _, err := s.deps.Auth.Login(r.Context(), auth.LoginParams{
		Username: u.Username,
		Password: password,
		TOTPCode: totpCode,
		IP:       clientIP(r),
		Agent:    r.UserAgent(),
	}); err != nil {
		return false
	}
	return true
}

func validateUsername(name string) error {
	return validateName("username", name)
}
