// SPDX-License-Identifier: AGPL-3.0
package oauth

import (
	"context"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const schema = `
CREATE TABLE oauth_apps (id INTEGER PRIMARY KEY AUTOINCREMENT, owner_id INTEGER, name TEXT, client_id TEXT UNIQUE, client_secret_hash TEXT, redirect_uris TEXT DEFAULT '', scopes TEXT DEFAULT '', created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE oauth_authorizations (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER, app_id INTEGER, scopes TEXT DEFAULT '', created_at DATETIME DEFAULT CURRENT_TIMESTAMP, UNIQUE (user_id, app_id));
CREATE TABLE oauth_codes (id INTEGER PRIMARY KEY AUTOINCREMENT, code_hash TEXT UNIQUE, app_id INTEGER, user_id INTEGER, redirect_uri TEXT, scopes TEXT, expires_at DATETIME, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE oauth_tokens (id INTEGER PRIMARY KEY AUTOINCREMENT, token_hash TEXT UNIQUE, app_id INTEGER, user_id INTEGER, scopes TEXT, expires_at DATETIME, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
`

func newTestService(t *testing.T) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: "file:oauth?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(schema); err != nil {
		t.Fatal(err)
	}
	return New(store.New(database))
}

func TestAuthorizationCodeFlow(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	app, secret, err := s.CreateApp(ctx, 1, "Test App", []string{"https://app.example/cb"}, []string{"read:user"})
	if err != nil {
		t.Fatal(err)
	}
	if !ValidateRedirectURI(app, "https://app.example/cb") {
		t.Fatal("redirect should be valid")
	}
	if ValidateRedirectURI(app, "https://evil.example/cb") {
		t.Fatal("unregistered redirect must be rejected")
	}
	// Issue code + consent.
	code, err := s.IssueCode(ctx, AuthorizeParams{AppID: app.ID, UserID: 1, RedirectURI: "https://app.example/cb", Scopes: []string{"read:user"}})
	if err != nil {
		t.Fatal(err)
	}
	// Exchange.
	tok, err := s.ExchangeCode(ctx, app.ClientID, secret, code, "https://app.example/cb")
	if err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken == "" {
		t.Fatal("expected access token")
	}
	// Code is one-time.
	if _, err := s.ExchangeCode(ctx, app.ClientID, secret, code, "https://app.example/cb"); err == nil {
		t.Fatal("code reuse should fail")
	}
	// Token resolves to user.
	uid, scopes, err := s.TokenUser(ctx, tok.AccessToken)
	if err != nil || uid != 1 || len(scopes) != 1 {
		t.Fatalf("token user = %d %v %v", uid, scopes, err)
	}
	// Consent recorded.
	auths, _ := s.ListAuthorizations(ctx, 1)
	if len(auths) != 1 || auths[0].AppName != "Test App" {
		t.Fatalf("authorizations = %+v", auths)
	}
	// Revoke.
	if err := s.RevokeAuthorization(ctx, 1, app.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.TokenUser(ctx, tok.AccessToken); err == nil {
		t.Fatal("token should be revoked")
	}
}

func TestInvalidClient(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	app, _, _ := s.CreateApp(ctx, 1, "A", []string{"https://x/cb"}, nil)
	if _, err := s.VerifyClient(ctx, app.ClientID, "wrong"); err != ErrInvalidClient {
		t.Fatalf("expected invalid client, got %v", err)
	}
}
