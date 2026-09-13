package web

import (
	"errors"
	"net/http"

	"gitli/internal/auth"
	"gitli/internal/db"
	"gitli/internal/users"
)

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	s.render(w, http.StatusOK, "home", pageData{
		Title: "Home",
		User:  UserFromContext(r.Context()),
		CSRF:  CSRFFromContext(r.Context()),
	})
}

func (s *Server) showLogin(w http.ResponseWriter, r *http.Request) {
	s.render(w, http.StatusOK, "login", pageData{
		Title:        "登录",
		User:         UserFromContext(r.Context()),
		Data:         Flash{},
		OAuthEnabled: auth.OAuthEnabled(),
	})
}

func (s *Server) doLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderError(w, http.StatusBadRequest, "invalid form")
		return
	}
	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	sid, _, err := auth.Login(r.Context(), s.q, username, password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			s.render(w, http.StatusUnauthorized, "login", pageData{
				Title: "登录",
				Data:  Flash{Kind: "error", Message: "用户名或密码错误"},
			})
			return
		}
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	// TOTP 两步验证：已启用用户先不建 session，进入二次验证
	if u, err := s.users.GetByUsername(r.Context(), auth.NormalizeUsername(username)); err == nil && u.TotpEnabled == 1 {
		s.handleLogin2FA(w, r, u.ID, nil)
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

func (s *Server) doLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		_ = auth.Logout(r.Context(), s.q, c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) showRegister(w http.ResponseWriter, r *http.Request) {
	s.render(w, http.StatusOK, "register", pageData{
		Title: "注册",
		User:  UserFromContext(r.Context()),
		Data:  Flash{},
	})
}

func (s *Server) doRegister(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderError(w, http.StatusBadRequest, "invalid form")
		return
	}
	username := r.PostFormValue("username")
	email := r.PostFormValue("email")
	password := r.PostFormValue("password")

	_, err := s.users.Register(r.Context(), username, email, password)
	if err != nil {
		msg := "注册失败，请检查输入"
		switch {
		case errors.Is(err, users.ErrUsernameTaken):
			msg = "用户名已被占用"
		case errors.Is(err, users.ErrEmailTaken):
			msg = "邮箱已被占用"
		case errors.Is(err, users.ErrInvalidInput):
			msg = err.Error()
		default:
			s.renderError(w, http.StatusInternalServerError, "internal error")
			return
		}
		s.render(w, http.StatusBadRequest, "register", pageData{
			Title: "注册",
			Data:  Flash{Kind: "error", Message: msg},
		})
		return
	}

	// 注册成功即登录
	sid, _, err := auth.Login(r.Context(), s.q, auth.NormalizeUsername(username), password)
	if err == nil {
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookie,
			Value:    sid,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   14 * 24 * 3600,
		})
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

var _ = db.User{} // keep import stable
