// SPDX-License-Identifier: AGPL-3.0
// Package pat manages personal access tokens (classic and fine-grained).
package pat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/LiStudioorg/gitli/internal/domain"
	infratoken "github.com/LiStudioorg/gitli/internal/infra/token"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates a missing token.
var ErrNotFound = errors.New("pat: not found")

// Service manages access tokens.
type Service struct {
	st *store.Store
}

// New builds a PAT Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// CreateParams describes a token creation request.
type CreateParams struct {
	UserID    int64
	Name      string
	Kind      domain.TokenKind
	Scopes    []string
	Grants    []domain.TokenRepoGrant
	ExpiresAt *time.Time
}

// Created is the result of token creation; Raw is shown only once.
type Created struct {
	Token *domain.AccessToken
	Raw   string
}

// Create issues a new token.
func (s *Service) Create(ctx context.Context, p CreateParams) (*Created, error) {
	if p.Kind == "" {
		p.Kind = domain.TokenClassic
	}
	tok, err := infratoken.New("gitli")
	if err != nil {
		return nil, err
	}
	tx, err := s.st.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, s.st.Rebind(`INSERT INTO access_tokens (user_id, name, token_hash, kind, expires_at) VALUES (?, ?, ?, ?, ?)`),
		p.UserID, p.Name, tok.Hash, string(p.Kind), p.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("pat: insert: %w", err)
	}
	id, _ := res.LastInsertId()
	if err := s.insertScopes(ctx, tx, id, string(p.Kind), p.Scopes, p.Grants); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &Created{
		Token: &domain.AccessToken{ID: id, UserID: p.UserID, Name: p.Name, Kind: p.Kind, Scopes: p.Scopes, RepoGrants: p.Grants, ExpiresAt: p.ExpiresAt},
		Raw:   tok.Plain,
	}, nil
}

func (s *Service) insertScopes(ctx context.Context, tx *sql.Tx, tokenID int64, kind string, scopes []string, grants []domain.TokenRepoGrant) error {
	if kind == string(domain.TokenFineGrained) {
		for _, sc := range scopes {
			if _, err := tx.ExecContext(ctx, s.st.Rebind(`INSERT INTO token_fine_grained_scopes (token_id, scope) VALUES (?, ?)`), tokenID, sc); err != nil {
				return err
			}
		}
		for _, g := range grants {
			if _, err := tx.ExecContext(ctx, s.st.Rebind(`INSERT INTO token_repositories (token_id, repo_id, permission) VALUES (?, ?, ?)`), tokenID, g.RepoID, g.Permission); err != nil {
				return err
			}
		}
		return nil
	}
	for _, sc := range scopes {
		if _, err := tx.ExecContext(ctx, s.st.Rebind(`INSERT INTO token_classic_scopes (token_id, scope) VALUES (?, ?)`), tokenID, sc); err != nil {
			return err
		}
	}
	return nil
}

// List returns a user's tokens (without secrets).
func (s *Service) List(ctx context.Context, userID int64) ([]*domain.AccessToken, error) {
	q := s.st.Rebind(`SELECT id, user_id, name, token_hash, kind, last_used_at, expires_at, created_at FROM access_tokens WHERE user_id = ? ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.AccessToken
	for rows.Next() {
		var t domain.AccessToken
		var kind string
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &kind, &t.LastUsedAt, &t.ExpiresAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.Kind = domain.TokenKind(kind)
		out = append(out, &t)
	}
	return out, rows.Err()
}

// Revoke deletes a token.
func (s *Service) Revoke(ctx context.Context, userID, tokenID int64) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM access_tokens WHERE id = ? AND user_id = ?`), tokenID, userID)
	return err
}

// Resolve validates a raw token and returns it with scopes and grants loaded.
func (s *Service) Resolve(ctx context.Context, raw string) (*domain.AccessToken, error) {
	hash := infratoken.Hash(raw)
	q := s.st.Rebind(`SELECT id, user_id, name, token_hash, kind, last_used_at, expires_at, created_at FROM access_tokens WHERE token_hash = ?`)
	var t domain.AccessToken
	var kind string
	err := s.st.DB.QueryRowContext(ctx, q, hash).Scan(&t.ID, &t.UserID, &t.Name, &t.TokenHash, &kind, &t.LastUsedAt, &t.ExpiresAt, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	t.Kind = domain.TokenKind(kind)
	if t.ExpiresAt != nil && t.ExpiresAt.Before(time.Now()) {
		return nil, ErrNotFound
	}
	if err := s.loadScopes(ctx, &t); err != nil {
		return nil, err
	}
	_, _ = s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE access_tokens SET last_used_at = ? WHERE id = ?`), time.Now().UTC(), t.ID)
	return &t, nil
}

func (s *Service) loadScopes(ctx context.Context, t *domain.AccessToken) error {
	var table string
	if t.Kind == domain.TokenFineGrained {
		table = "token_fine_grained_scopes"
	} else {
		table = "token_classic_scopes"
	}
	rows, err := s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT scope FROM `+table+` WHERE token_id = ?`), t.ID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var sc string
		if err := rows.Scan(&sc); err != nil {
			return err
		}
		t.Scopes = append(t.Scopes, sc)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if t.Kind != domain.TokenFineGrained {
		return nil
	}
	grows, err := s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT repo_id, permission FROM token_repositories WHERE token_id = ?`), t.ID)
	if err != nil {
		return err
	}
	defer func() { _ = grows.Close() }()
	for grows.Next() {
		var g domain.TokenRepoGrant
		if err := grows.Scan(&g.RepoID, &g.Permission); err != nil {
			return err
		}
		t.RepoGrants = append(t.RepoGrants, g)
	}
	return grows.Err()
}

// HasRepoPermission checks whether a token grants access to a repo.
func HasRepoPermission(t *domain.AccessToken, repoID int64, need domain.Perm) bool {
	if t.Kind == domain.TokenClassic {
		for _, sc := range t.Scopes {
			if sc == "repo" || sc == "*" {
				return true
			}
		}
		return false
	}
	for _, g := range t.RepoGrants {
		if g.RepoID == repoID {
			return domain.ParsePerm(g.Permission) >= need
		}
	}
	return false
}
