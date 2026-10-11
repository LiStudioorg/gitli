// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/LiStudioorg/gitli/internal/domain"
)

type ctxKey int

const (
	ctxKeyUser ctxKey = iota
	ctxKeySessionToken
	ctxKeyToken
)

// userFrom returns the authenticated user, or nil.
func userFrom(ctx context.Context) *domain.User {
	u, _ := ctx.Value(ctxKeyUser).(*domain.User)
	return u
}

// requireUser writes 401 and returns nil when unauthenticated.
func requireUser(w http.ResponseWriter, r *http.Request) *domain.User {
	u := userFrom(r.Context())
	if u == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return nil
	}
	return u
}

// bearerToken extracts a bearer token or the session cookie value.
func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	if c, err := r.Cookie("gitli_session"); err == nil {
		return c.Value
	}
	return ""
}
