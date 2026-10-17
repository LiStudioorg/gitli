// SPDX-License-Identifier: AGPL-3.0
// Package deploykey manages repository-scoped SSH deploy keys.
package deploykey

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Errors returned by the deploy key service.
var (
	ErrNotFound = errors.New("deploykey: not found")
)

// Service manages deploy keys.
type Service struct {
	st *store.Store
}

// New builds a deploy key Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Add validates and stores a deploy key. readOnly controls push permission.
func (s *Service) Add(ctx context.Context, repoID int64, title, pubKey string, readOnly bool) (*domain.DeployKey, error) {
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pubKey))
	if err != nil {
		return nil, fmt.Errorf("deploykey: parse key: %w", err)
	}
	fp := ssh.FingerprintSHA256(pk)
	q := s.st.Rebind(`INSERT INTO deploy_keys (repo_id, title, fingerprint, public_key, read_only) VALUES (?, ?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, repoID, title, fp, pubKey, readOnly)
	if err != nil {
		return nil, fmt.Errorf("deploykey: insert: %w", err)
	}
	id, _ := res.LastInsertId()
	return &domain.DeployKey{ID: id, RepoID: repoID, Title: title, Fingerprint: fp, PublicKey: pubKey, ReadOnly: readOnly}, nil
}

// List returns a repository's deploy keys.
func (s *Service) List(ctx context.Context, repoID int64) ([]*domain.DeployKey, error) {
	q := s.st.Rebind(`SELECT id, repo_id, title, fingerprint, public_key, read_only, created_at, last_used_at FROM deploy_keys WHERE repo_id = ? ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.DeployKey
	for rows.Next() {
		var k domain.DeployKey
		if err := rows.Scan(&k.ID, &k.RepoID, &k.Title, &k.Fingerprint, &k.PublicKey, &k.ReadOnly, &k.CreatedAt, &k.LastUsedAt); err != nil {
			return nil, err
		}
		out = append(out, &k)
	}
	return out, rows.Err()
}

// SetReadOnly toggles a deploy key's permission.
func (s *Service) SetReadOnly(ctx context.Context, repoID, keyID int64, readOnly bool) error {
	res, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE deploy_keys SET read_only = ? WHERE id = ? AND repo_id = ?`), readOnly, keyID, repoID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a deploy key scoped to a repository.
func (s *Service) Delete(ctx context.Context, repoID, keyID int64) error {
	return s.st.ExecAffectedOr(ctx, ErrNotFound, s.st.Rebind(`DELETE FROM deploy_keys WHERE id = ? AND repo_id = ?`), keyID, repoID)
}

// Resolve maps a public key fingerprint to its repository and permission.
func (s *Service) Resolve(ctx context.Context, fingerprint string) (repoID int64, readOnly bool, err error) {
	q := s.st.Rebind(`SELECT repo_id, read_only FROM deploy_keys WHERE fingerprint = ?`)
	err = s.st.DB.QueryRowContext(ctx, q, fingerprint).Scan(&repoID, &readOnly)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, ErrNotFound
	}
	return repoID, readOnly, err
}

// Touch records last usage of a deploy key.
func (s *Service) Touch(ctx context.Context, fingerprint string) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE deploy_keys SET last_used_at = ? WHERE fingerprint = ?`), time.Now().UTC(), fingerprint)
	return err
}
