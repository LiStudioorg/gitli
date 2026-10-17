// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"

	"github.com/LiStudioorg/gitli/internal/service/user"
)

// handleSAMLLogin redirects to the IdP (SP-initiated).
func (s *Server) handleSAMLLogin(w http.ResponseWriter, r *http.Request) {
	if s.deps.SAML == nil || !s.deps.SAML.Enabled() {
		writeError(w, http.StatusNotFound, "SAML 未启用")
		return
	}
	u, err := s.deps.SAML.LoginURL()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	http.Redirect(w, r, u, http.StatusFound)
}

// handleSAMLACS consumes the SAML assertion and establishes a session.
func (s *Server) handleSAMLACS(w http.ResponseWriter, r *http.Request) {
	if s.deps.SAML == nil || !s.deps.SAML.Enabled() {
		writeError(w, http.StatusNotFound, "SAML 未启用")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "无效的表单")
		return
	}
	name, email, err := s.deps.SAML.ParseResponse(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "SAML 断言无效")
		return
	}
	u, err := s.deps.Users.ByUsername(r.Context(), name)
	if err != nil {
		if email == "" {
			email = name + "@saml.local"
		}
		u, err = s.deps.Users.Create(r.Context(), user.CreateParams{Username: name, Email: email})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	raw, err := s.deps.Auth.CreateSession(r.Context(), u, clientIP(r), r.UserAgent())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: "gitli_session", Value: raw, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: s.deps.cookieSecure(r), MaxAge: 30 * 24 * 3600,
	})
	s.auditAction(r, u.ID, "auth.saml_login", "success")
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}
