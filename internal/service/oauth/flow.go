// SPDX-License-Identifier: AGPL-3.0
package oauth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	infratoken "github.com/LiStudioorg/gitli/internal/infra/token"
)

// randID generates a random URL-safe identifier with the given prefix.
func randID(prefix string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(buf), nil
}

// AuthorizeParams describes an authorization request.
type AuthorizeParams struct {
	AppID       int64
	UserID      int64
	RedirectURI string
	Scopes      []string
}

// IssueCode issues a short-lived authorization code and records the user's
// consent for the app's scopes.
func (s *Service) IssueCode(ctx context.Context, p AuthorizeParams) (string, error) {
	code, err := randID("gla_")
	if err != nil {
		return "", err
	}
	tx, err := s.st.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()

	ins := s.st.Rebind(`INSERT INTO oauth_codes (code_hash, app_id, user_id, redirect_uri, scopes, expires_at) VALUES (?, ?, ?, ?, ?, ?)`)
	if _, err := tx.ExecContext(ctx, ins, infratoken.Hash(code), p.AppID, p.UserID, p.RedirectURI, strings.Join(p.Scopes, " "), time.Now().Add(10*time.Minute)); err != nil {
		return "", err
	}
	consent := s.st.Rebind(`INSERT INTO oauth_authorizations (user_id, app_id, scopes) VALUES (?, ?, ?) ON CONFLICT (user_id, app_id) DO UPDATE SET scopes = excluded.scopes`)
	if _, err := tx.ExecContext(ctx, consent, p.UserID, p.AppID, strings.Join(p.Scopes, " ")); err != nil {
		if _, err2 := tx.ExecContext(ctx, s.st.Rebind(`INSERT INTO oauth_authorizations (user_id, app_id, scopes) VALUES (?, ?, ?)`), p.UserID, p.AppID, strings.Join(p.Scopes, " ")); err2 != nil {
			return "", err
		}
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return code, nil
}

// TokenResponse is the response of the token endpoint.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// ExchangeCode validates an authorization code and issues an access token.
func (s *Service) ExchangeCode(ctx context.Context, clientID, secret, code, redirectURI string) (*TokenResponse, error) {
	app, err := s.VerifyClient(ctx, clientID, secret)
	if err != nil {
		return nil, err
	}
	var (
		appID   int64
		userID  int64
		redir   string
		scopes  string
		expires time.Time
	)
	q := s.st.Rebind(`SELECT app_id, user_id, redirect_uri, scopes, expires_at FROM oauth_codes WHERE code_hash = ?`)
	err = s.st.DB.QueryRowContext(ctx, q, infratoken.Hash(code)).Scan(&appID, &userID, &redir, &scopes, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidGrant
	}
	if err != nil {
		return nil, err
	}
	if app.ID != appID {
		return nil, ErrInvalidGrant
	}
	if time.Now().After(expires) {
		_, _ = s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM oauth_codes WHERE code_hash = ?`), infratoken.Hash(code))
		return nil, ErrInvalidGrant
	}
	if redirectURI != "" && redirectURI != redir {
		return nil, ErrInvalidGrant
	}
	// One-time code.
	_, _ = s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM oauth_codes WHERE code_hash = ?`), infratoken.Hash(code))

	access, err := randID("glat_")
	if err != nil {
		return nil, err
	}
	tokExpiry := time.Now().Add(365 * 24 * time.Hour)
	ins := s.st.Rebind(`INSERT INTO oauth_tokens (token_hash, app_id, user_id, scopes, expires_at) VALUES (?, ?, ?, ?, ?)`)
	if _, err := s.st.DB.ExecContext(ctx, ins, infratoken.Hash(access), appID, userID, scopes, tokExpiry); err != nil {
		return nil, err
	}
	return &TokenResponse{
		AccessToken: access,
		TokenType:   "bearer",
		ExpiresIn:   int((365 * 24 * time.Hour).Seconds()),
		Scope:       scopes,
	}, nil
}

// TokenUser resolves an access token to a user id and granted scopes.
func (s *Service) TokenUser(ctx context.Context, accessToken string) (userID int64, scopes []string, err error) {
	var scopeStr string
	var expires sql.NullTime
	q := s.st.Rebind(`SELECT user_id, scopes, expires_at FROM oauth_tokens WHERE token_hash = ?`)
	err = s.st.DB.QueryRowContext(ctx, q, infratoken.Hash(accessToken)).Scan(&userID, &scopeStr, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, ErrInvalidGrant
	}
	if err != nil {
		return 0, nil, err
	}
	if expires.Valid && time.Now().After(expires.Time) {
		return 0, nil, ErrInvalidGrant
	}
	if scopeStr != "" {
		scopes = strings.Fields(scopeStr)
	}
	return userID, scopes, nil
}

// Authorization is a user's consent record for an app.
type Authorization struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	AppID     int64     `json:"app_id"`
	AppName   string    `json:"app_name"`
	ClientID  string    `json:"client_id"`
	Scopes    []string  `json:"scopes"`
	CreatedAt time.Time `json:"created_at"`
}

// ListAuthorizations lists a user's app consents.
func (s *Service) ListAuthorizations(ctx context.Context, userID int64) ([]*Authorization, error) {
	q := s.st.Rebind(`SELECT a.id, a.user_id, a.app_id, COALESCE(p.name,''), COALESCE(p.client_id,''), a.scopes, a.created_at
		FROM oauth_authorizations a LEFT JOIN oauth_apps p ON p.id = a.app_id
		WHERE a.user_id = ? ORDER BY a.id DESC`)
	rows, err := s.st.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*Authorization
	for rows.Next() {
		var a Authorization
		var scopes string
		if err := rows.Scan(&a.ID, &a.UserID, &a.AppID, &a.AppName, &a.ClientID, &scopes, &a.CreatedAt); err != nil {
			return nil, err
		}
		if scopes != "" {
			a.Scopes = strings.Fields(scopes)
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

// RevokeAuthorization removes a user's consent for an app and its tokens.
func (s *Service) RevokeAuthorization(ctx context.Context, userID, appID int64) error {
	if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM oauth_authorizations WHERE user_id = ? AND app_id = ?`), userID, appID); err != nil {
		return err
	}
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM oauth_tokens WHERE user_id = ? AND app_id = ?`), userID, appID)
	return err
}

// ValidateRedirectURI reports whether a redirect URI is registered for an app.
func ValidateRedirectURI(app *App, uri string) bool {
	if app == nil {
		return false
	}
	for _, r := range app.RedirectURIs {
		if r == uri {
			return true
		}
	}
	return false
}

// SupportedScopes lists the scopes the provider understands.
var SupportedScopes = map[string]string{
	"read:user":  "读取用户资料",
	"read:repo":  "读取仓库信息",
	"write:repo": "写入仓库信息",
}

// FilterScopes keeps only supported scopes.
func FilterScopes(scopes []string) []string {
	out := make([]string, 0, len(scopes))
	for _, sc := range scopes {
		if _, ok := SupportedScopes[sc]; ok {
			out = append(out, sc)
		}
	}
	return out
}
