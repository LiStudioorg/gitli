// SPDX-License-Identifier: AGPL-3.0
package auth

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/LiStudioorg/gitli/internal/domain"
)

type ctxKey int

const (
	ctxKeyUser ctxKey = iota
	ctxKeyToken
)

// WithUser returns a context carrying the authenticated user.
func WithUser(ctx context.Context, u *domain.User) context.Context {
	return context.WithValue(ctx, ctxKeyUser, u)
}

// UserFrom returns the authenticated user from a context, or nil.
func UserFrom(ctx context.Context) *domain.User {
	u, _ := ctx.Value(ctxKeyUser).(*domain.User)
	return u
}

// WithToken attaches the PAT used for the request, if any.
func WithToken(ctx context.Context, t *domain.AccessToken) context.Context {
	return context.WithValue(ctx, ctxKeyToken, t)
}

// TokenFrom returns the PAT used for the request, if any.
func TokenFrom(ctx context.Context) *domain.AccessToken {
	t, _ := ctx.Value(ctxKeyToken).(*domain.AccessToken)
	return t
}

// TokenResolver resolves a raw PAT to a token; implemented by the pat service.
type TokenResolver interface {
	Resolve(ctx context.Context, raw string) (*domain.AccessToken, error)
}

// SetTokenResolver wires the PAT resolver used for bearer authentication.
func (s *Service) SetTokenResolver(r TokenResolver) { s.resolver = r }

// AuthenticateMiddleware resolves a session cookie or bearer PAT into a user.
// Requests without credentials proceed unauthenticated.
func (s *Service) AuthenticateMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := extractCredential(r)
		if raw == "" {
			next.ServeHTTP(w, r)
			return
		}
		ctx := r.Context()
		if isSessionToken(raw) {
			ip, agent := clientMeta(r)
			if u, _, err := s.AuthenticateRequest(ctx, raw, ip, agent); err == nil {
				next.ServeHTTP(w, r.WithContext(WithUser(ctx, u)))
				return
			}
		}
		if s.resolver != nil {
			if tok, err := s.resolver.Resolve(ctx, raw); err == nil {
				if u, err := s.users.ByID(ctx, tok.UserID); err == nil {
					ctx = WithToken(ctx, tok)
					next.ServeHTTP(w, r.WithContext(WithUser(ctx, u)))
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func extractCredential(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	if c, err := r.Cookie("gitli_session"); err == nil {
		return c.Value
	}
	return ""
}

func isSessionToken(raw string) bool {
	return strings.HasPrefix(raw, "gls_")
}

// clientMeta extracts the client IP and User-Agent for session binding checks.
func clientMeta(r *http.Request) (ip, agent string) {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			ip = strings.TrimSpace(xff[:i])
		} else {
			ip = strings.TrimSpace(xff)
		}
	} else if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		ip = host
	} else {
		ip = r.RemoteAddr
	}
	return ip, r.UserAgent()
}
