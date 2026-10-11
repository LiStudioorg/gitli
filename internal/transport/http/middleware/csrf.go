// SPDX-License-Identifier: AGPL-3.0
package middleware

import (
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
			token := ensureCSRFCookie(w, r)
			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			if r.Header.Get(csrfHeaderName) != token {
				http.Error(w, "invalid csrf token", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
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
