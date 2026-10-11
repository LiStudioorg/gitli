// SPDX-License-Identifier: AGPL-3.0
package middleware

import "net/http"

// Body limits in bytes for different request classes.
const (
	BodyLimitTiny   = 1 << 10  // 1 KiB   (login, sensitive actions)
	BodyLimitSmall  = 16 << 10 // 16 KiB  (labels, metadata)
	BodyLimitMedium = 64 << 10 // 64 KiB  (comments, issues, pulls)
	BodyLimitLarge  = 1 << 20  // 1 MiB   (default API JSON)
	BodyLimitHuge   = 10 << 20 // 10 MiB  (bulk)
)

// MaxBody returns a middleware that caps the request body size.
func MaxBody(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}
			next.ServeHTTP(w, r)
		})
	}
}
