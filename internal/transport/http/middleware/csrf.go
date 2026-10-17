// SPDX-License-Identifier: AGPL-3.0
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

const csrfCookieName = "gitli_csrf"
const csrfHeaderName = "X-CSRF-Token"

// CSRF implements a double-submit cookie CSRF protection scheme.
// Safe methods are passed through; unsafe methods require the header to
// match the cookie value.
func CSRF() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, _ := r.Context().Value(csrfContextKey{}).(string)
			if token == "" {
				token = ensureCSRFCookie(w, r)
			}
			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			if r.Header.Get(csrfHeaderName) != token {
				http.Error(w, "CSRF 校验失败，请刷新页面后重试", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// csrfContextKey stores the token issued for this request so the outer issuer
// and the per-API validator agree on a single value.
type csrfContextKey struct{}

// IssueCSRF makes sure every page load receives a CSRF cookie, so the SPA can
// perform its first state-changing request without a prior API round-trip.
// It never validates; the per-API CSRF middleware still enforces the token.
// The issued token is stashed on the request context so a downstream CSRF
// middleware reuses it instead of generating a second, conflicting cookie.
func IssueCSRF() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, _ := r.Context().Value(csrfContextKey{}).(string)
			if token == "" {
				token = ensureCSRFCookie(w, r)
			}
			ctx := context.WithValue(r.Context(), csrfContextKey{}, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ensureCSRFCookie(w http.ResponseWriter, r *http.Request) string {
	if c, err := r.Cookie(csrfCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	token := base64.RawURLEncoding.EncodeToString(buf)
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})
	return token
}

func isSafeMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}
