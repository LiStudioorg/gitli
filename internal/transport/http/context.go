// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"context"
	"net/http"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/auth"
)

// userFrom returns the authenticated user, or nil. It reads the value stored
// by auth.AuthenticateMiddleware so both layers share the same context key.
func userFrom(ctx context.Context) *domain.User {
	return auth.UserFrom(ctx)
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
