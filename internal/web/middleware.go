package web

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"gitli/internal/auth"
	"gitli/internal/db"
)

type contextKey int

const (
	ctxUser contextKey = iota
	ctxCSRF
)

// UserFromContext 从 context 取当前登录用户（未登录为 nil）。
func UserFromContext(ctx context.Context) *db.User {
	u, _ := ctx.Value(ctxUser).(*db.User)
	return u
}

// CSRFFromContext 从 context 取当前会话 CSRF token。
func CSRFFromContext(ctx context.Context) string {
	s, _ := ctx.Value(ctxCSRF).(string)
	return s
}

const sessionCookie = "gitli_session"

// LoadSession 解析 session cookie，加载用户与 CSRF 进 context。
func LoadSession(q db.Querier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(sessionCookie)
			if err == nil && c.Value != "" {
				if user, csrf, err := auth.SessionUserWithCSRF(r.Context(), q, c.Value); err == nil {
					u := user
					ctx := context.WithValue(r.Context(), ctxUser, &u)
					ctx = context.WithValue(ctx, ctxCSRF, csrf)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireLogin 未登录则跳转 /login。
func RequireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if UserFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// VerifyCSRF 对所有非安全方法校验 CSRF token（仅对已登录会话有意义）。
func VerifyCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			want := CSRFFromContext(r.Context())
			if want != "" {
				got := r.PostFormValue("csrf_token")
				if got == "" {
					got = r.Header.Get("X-CSRF-Token")
				}
				if subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
					http.Error(w, "CSRF token mismatch", http.StatusForbidden)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// SecureHeaders 附加基础安全响应头。
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		if strings.HasPrefix(r.URL.Path, "/static/") {
			h.Set("Cache-Control", "public, max-age=3600")
		}
		next.ServeHTTP(w, r)
	})
}
