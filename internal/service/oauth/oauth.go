// SPDX-License-Identifier: AGPL-3.0
// Package oauth implements an OAuth2 authorization-code provider.
package oauth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	infratoken "github.com/LiStudioorg/gitli/internal/infra/token"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Errors returned by the OAuth2 service.
var (
	ErrNotFound      = errors.New("oauth: not found")
	ErrInvalidClient = errors.New("oauth: invalid client")
	ErrInvalidGrant  = errors.New("oauth: invalid grant")
	ErrInvalidScope  = errors.New("oauth: invalid scope")
)

// App is a registered OAuth2 application.
type App struct {
	ID           int64     `json:"id"`
	OwnerID      int64     `json:"owner_id"`
	Name         string    `json:"name"`
	ClientID     string    `json:"client_id"`
	RedirectURIs []string  `json:"redirect_uris"`
	Scopes       []string  `json:"scopes"`
	CreatedAt    time.Time `json:"created_at"`
}

// Service implements the OAuth2 flow.
type Service struct {
	st *store.Store
}

// New builds the OAuth2 Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// CreateApp registers an application and returns the one-time plaintext secret.
func (s *Service) CreateApp(ctx context.Context, ownerID int64, name string, redirectURIs, scopes []string) (*App, string, error) {
	if name == "" {
		return nil, "", fmt.Errorf("oauth: name required")
	}
	if len(redirectURIs) == 0 {
		return nil, "", fmt.Errorf("oauth: at least one redirect uri required")
	}
	clientID, err := randID("glo_")
	if err != nil {
		return nil, "", err
	}
	secret, err := randID("glsec_")
	if err != nil {
		return nil, "", err
	}
	q := s.st.Rebind(`INSERT INTO oauth_apps (owner_id, name, client_id, client_secret_hash, redirect_uris, scopes) VALUES (?, ?, ?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, ownerID, name, clientID, infratoken.Hash(secret), strings.Join(redirectURIs, "\n"), strings.Join(scopes, " "))
	if err != nil {
		return nil, "", fmt.Errorf("oauth: insert app: %w", err)
	}
	id, _ := res.LastInsertId()
	app, err := s.AppByID(ctx, id)
	return app, secret, err
}

// AppByID loads an app.
func (s *Service) AppByID(ctx context.Context, id int64) (*App, error) {
	return s.scanApp(ctx, s.st.Rebind(`SELECT id, owner_id, name, client_id, redirect_uris, scopes, created_at FROM oauth_apps WHERE id = ?`), id)
}

// AppByClientID loads an app by client id.
func (s *Service) AppByClientID(ctx context.Context, clientID string) (*App, error) {
	return s.scanApp(ctx, s.st.Rebind(`SELECT id, owner_id, name, client_id, redirect_uris, scopes, created_at FROM oauth_apps WHERE client_id = ?`), clientID)
}

func (s *Service) scanApp(ctx context.Context, query string, arg any) (*App, error) {
	var a App
	var uris, scopes string
	err := s.st.DB.QueryRowContext(ctx, query, arg).Scan(&a.ID, &a.OwnerID, &a.Name, &a.ClientID, &uris, &scopes, &a.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if uris != "" {
		a.RedirectURIs = strings.Split(uris, "\n")
	}
	if scopes != "" {
		a.Scopes = strings.Fields(scopes)
	}
	return &a, nil
}

// ListAppsByOwner lists apps owned by a user.
func (s *Service) ListAppsByOwner(ctx context.Context, ownerID int64) ([]*App, error) {
	rows, err := s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT id FROM oauth_apps WHERE owner_id = ? ORDER BY id DESC`), ownerID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*App
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if a, err := s.AppByID(ctx, id); err == nil {
			out = append(out, a)
		}
	}
	return out, rows.Err()
}

// DeleteApp removes an app (owner only).
func (s *Service) DeleteApp(ctx context.Context, ownerID, appID int64) error {
	return s.st.ExecAffectedOr(ctx, ErrNotFound, s.st.Rebind(`DELETE FROM oauth_apps WHERE id = ? AND owner_id = ?`), appID, ownerID)
}

// VerifyClient validates client_id + secret.
func (s *Service) VerifyClient(ctx context.Context, clientID, secret string) (*App, error) {
	var id int64
	var hash string
	err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT id, client_secret_hash FROM oauth_apps WHERE client_id = ?`), clientID).Scan(&id, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidClient
	}
	if err != nil {
		return nil, err
	}
	if !infratoken.Verify(secret, hash) {
		return nil, ErrInvalidClient
	}
	return s.AppByID(ctx, id)
}
