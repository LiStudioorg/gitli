package web

import (
	"net/http"

	"gitli/internal/auth"
)

// GET /oauth2/start：配置了则跳 provider；未配置返回 404。
func (s *Server) handleOAuthStart(w http.ResponseWriter, r *http.Request) {
	cfg, err := auth.OAuthConfigOrError()
	if err != nil {
		s.renderError(w, http.StatusNotFound, "OAuth2 未配置："+err.Error())
		return
	}
	state := auth.RandomState()
	http.SetCookie(w, &http.Cookie{
		Name:     "gitli_oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})
	http.Redirect(w, r, auth.OAuthAuthCodeURL(cfg, state), http.StatusSeeOther)
}

// GET /oauth2/callback：code → token → userinfo → 注册/登录 → 跳首页。
func (s *Server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	cfg, err := auth.OAuthConfigOrError()
	if err != nil {
		s.renderError(w, http.StatusNotFound, "OAuth2 未配置")
		return
	}
	// 校验 state
	st, err := r.Cookie("gitli_oauth_state")
	if err != nil || st.Value == "" || st.Value != r.URL.Query().Get("state") {
		s.renderError(w, http.StatusBadRequest, "OAuth2 state 校验失败")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		s.renderError(w, http.StatusBadRequest, "缺少 code")
		return
	}
	token, err := auth.OAuthExchange(r.Context(), cfg, code)
	if err != nil {
		s.renderError(w, http.StatusBadGateway, "OAuth2 token 交换失败")
		return
	}
	sub, username, email, err := auth.OAuthUserinfo(r.Context(), cfg, token)
	if err != nil {
		s.renderError(w, http.StatusBadGateway, "OAuth2 userinfo 获取失败")
		return
	}
	// username 兜底：sub 前缀
	if username == "" {
		username = "oauth-" + sub[:8]
	}
	if email == "" {
		email = username + "@oauth.local"
	}
	// 用户不存在则自动注册
	user, err := auth.EnsureOAuthUser(r.Context(), s.q, username, email)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "OAuth2 用户落地失败")
		return
	}
	// 建 session 登录
	sid, _, err := auth.SessionForUser(r.Context(), s.q, user)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    sid,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   14 * 24 * 3600,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
