// SPDX-License-Identifier: AGPL-3.0
package middleware

import (
	"context"
	"net/http"
	"strings"
)

// SetupGuard blocks API traffic until the instance has an administrator.
// Paths listed in allow always pass through so the setup wizard can work.
func SetupGuard(isInitialized func(context.Context) (bool, error), allow []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allow))
	for _, p := range allow {
		allowed[p] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := allowed[r.URL.Path]; ok || strings.HasPrefix(r.URL.Path, "/api/v1/setup/") {
				next.ServeHTTP(w, r)
				return
			}
			initd, err := isInitialized(r.Context())
			if err != nil {
				http.Error(w, "setup state unavailable", http.StatusInternalServerError)
				return
			}
			if !initd {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusPreconditionFailed)
				_, _ = w.Write([]byte(`{"error":"instance not initialized","code":"setup_required"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
