// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/infra/oidc"
	"github.com/LiStudioorg/gitli/internal/service/audit"
	"github.com/LiStudioorg/gitli/internal/service/user"
)

// handleOIDCLogin redirects the user to the identity provider.
func (s *Server) handleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if s.deps.OIDC == nil || !s.deps.OIDC.Enabled() {
		writeError(w, http.StatusNotFound, "OIDC 未启用")
		return
	}
	state, err := oidc.RandomState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "状态生成失败")
		return
	}
	verifier, err := oidc.GenerateVerifier()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "PKCE 生成失败")
		return
	}
	nonce, _ := oidc.RandomState()
	// Persist state/verifier/nonce in short-lived cookies for the callback.
	secure := s.deps.cookieSecure(r)
	setShortCookie(w, "gitli_oidc_state", state, secure)
	setShortCookie(w, "gitli_oidc_verifier", verifier, secure)
	setShortCookie(w, "gitli_oidc_nonce", nonce, secure)
	url, err := s.deps.OIDC.AuthCodeURL(state, nonce, oidc.Challenge(verifier))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

// handleOIDCCallback completes the authorization-code flow.
func (s *Server) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	if s.deps.OIDC == nil || !s.deps.OIDC.Enabled() {
		writeError(w, http.StatusNotFound, "OIDC 未启用")
		return
	}
	stateCookie, err := r.Cookie("gitli_oidc_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		writeError(w, http.StatusBadRequest, "无效的状态参数")
		return
	}
	verifierCookie, err := r.Cookie("gitli_oidc_verifier")
	if err != nil {
		writeError(w, http.StatusBadRequest, "缺少 PKCE 校验串")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "缺少授权码")
		return
	}
	tok, err := s.deps.OIDC.Exchange(r.Context(), code, verifierCookie.Value)
	if err != nil {
		writeError(w, http.StatusBadGateway, "令牌交换失败")
		return
	}
	info, err := s.deps.OIDC.UserInfo(r.Context(), tok.AccessToken)
	if err != nil {
		writeError(w, http.StatusBadGateway, "获取用户信息失败")
		return
	}
	u, err := s.upsertOIDCUser(r, info)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	clearCookie(w, "gitli_oidc_state")
	clearCookie(w, "gitli_oidc_verifier")
	clearCookie(w, "gitli_oidc_nonce")

	// Establish a normal session for the resolved user.
	res, err := s.deps.Auth.CreateSession(r.Context(), u, clientIP(r), r.UserAgent())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "gitli_session",
		Value:    res,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.deps.cookieSecure(r),
		MaxAge:   30 * 24 * 3600,
	})
	uid := u.ID
	_ = s.deps.Audit.Record(r.Context(), audit.Entry{
		ActorID: &uid, Action: "auth.oidc_login", TargetType: "user",
		IP: clientIP(r), Detail: "success",
	})
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

// upsertOIDCUser finds or provisions a local user for the OIDC identity.
func (s *Server) upsertOIDCUser(r *http.Request, info *oidc.UserInfo) (*domain.User, error) {
	if info.Email != "" {
		if u, err := s.deps.Users.ByEmail(r.Context(), info.Email); err == nil {
			return u, nil
		}
	}
	username := info.PreferredUsername
	if username == "" {
		username = sanitizeUsername(info.Email)
	}
	if username == "" {
		username = "oidc-" + info.Subject
	}
	if _, err := s.deps.Users.ByUsername(r.Context(), username); err == nil {
		username = username + "-" + info.Subject
	}
	return s.deps.Users.Create(r.Context(), user.CreateParams{
		Username: username,
		Email:    info.Email,
	})
}

func sanitizeUsername(email string) string {
	out := make([]rune, 0, len(email))
	for _, c := range email {
		if c == '@' {
			break
		}
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' {
			out = append(out, c)
		}
	}
	return string(out)
}

func setShortCookie(w http.ResponseWriter, name, value string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: value, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: secure, MaxAge: 600,
	})
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1})
}
