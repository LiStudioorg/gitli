// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"net/http"
	"strings"
)

// usernameRedirect issues a 301 to the canonical username when the requested
// owner segment is a former username recorded in user_redirects.
func (s *Server) usernameRedirect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// Only handle single-segment owner paths and owner/... paths, skipping
		// API, health, static assets and git protocol endpoints.
		if strings.HasPrefix(path, "/api/") || path == "/healthz" || path == "/metrics" ||
			strings.HasPrefix(path, "/assets/") || strings.Contains(path, ".git") {
			next.ServeHTTP(w, r)
			return
		}
		owner, rest := splitOwner(path)
		if owner == "" {
			next.ServeHTTP(w, r)
			return
		}
		// Skip if the user exists already (current username).
		if _, err := s.deps.Users.ByUsername(r.Context(), owner); err == nil {
			next.ServeHTTP(w, r)
			return
		}
		newUserID, err := s.deps.Users.ResolveRedirect(r.Context(), owner)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		target, err := s.deps.Users.ByID(r.Context(), newUserID)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		http.Redirect(w, r, "/"+target.Username+rest, http.StatusMovedPermanently)
	})
}

// splitOwner extracts the first path segment and the remaining path.
func splitOwner(path string) (owner, rest string) {
	p := strings.TrimPrefix(path, "/")
	if p == "" {
		return "", ""
	}
	if i := strings.IndexByte(p, '/'); i >= 0 {
		return p[:i], p[i:]
	}
	return p, ""
}
