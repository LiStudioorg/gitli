// SPDX-License-Identifier: AGPL-3.0
// Package cisecret stores per-repository CI secrets. Values are encrypted at
// rest with AES-256-GCM when a cipher is configured; when the cipher is nil
// (development only) values are stored as plaintext.
package cisecret

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/LiStudioorg/gitli/internal/infra/crypto"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates no secret exists for a repository/name pair.
var ErrNotFound = errors.New("cisecret: not found")

// encPrefix marks ciphertext written by this service.
const encPrefix = "enc:"

// Service manages CI secrets for repositories.
type Service struct {
	st     *store.Store
	cipher *crypto.Cipher
}

// New builds a CI secret Service. When cipher is nil, values are stored and
// read back as plaintext; this is intended only for development and logs a
// warning at call sites should never happen here.
func New(st *store.Store, cipher *crypto.Cipher) *Service {
	return &Service{st: st, cipher: cipher}
}

// Set inserts or updates a secret by (repo_id, name).
func (s *Service) Set(ctx context.Context, repoID int64, name, value string) error {
	if repoID <= 0 {
		return fmt.Errorf("cisecret: invalid repo id")
	}
	if name == "" {
		return fmt.Errorf("cisecret: name is required")
	}
	stored, err := s.encrypt(value)
	if err != nil {
		return err
	}
	var existing int64
	lookup := s.st.Rebind(`SELECT id FROM ci_secrets WHERE repo_id = ? AND name = ?`)
	err = s.st.DB.QueryRowContext(ctx, lookup, repoID, name).Scan(&existing)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		q := s.st.Rebind(`INSERT INTO ci_secrets (repo_id, name, ciphertext) VALUES (?, ?, ?)`)
		if _, err := s.st.DB.ExecContext(ctx, q, repoID, name, stored); err != nil {
			return fmt.Errorf("cisecret: insert: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("cisecret: lookup: %w", err)
	}
	q := s.st.Rebind(`UPDATE ci_secrets SET ciphertext = ? WHERE id = ?`)
	if _, err := s.st.DB.ExecContext(ctx, q, stored, existing); err != nil {
		return fmt.Errorf("cisecret: update: %w", err)
	}
	return nil
}

// List returns the secret names of a repository. Values are never exposed.
func (s *Service) List(ctx context.Context, repoID int64) ([]string, error) {
	q := s.st.Rebind(`SELECT name FROM ci_secrets WHERE repo_id = ? ORDER BY name`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID)
	if err != nil {
		return nil, fmt.Errorf("cisecret: list: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// Delete removes a secret. It returns ErrNotFound when nothing was deleted.
func (s *Service) Delete(ctx context.Context, repoID int64, name string) error {
	q := s.st.Rebind(`DELETE FROM ci_secrets WHERE repo_id = ? AND name = ?`)
	res, err := s.st.DB.ExecContext(ctx, q, repoID, name)
	if err != nil {
		return fmt.Errorf("cisecret: delete: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Get returns the decrypted value of a secret, or ErrNotFound.
func (s *Service) Get(ctx context.Context, repoID int64, name string) (string, error) {
	q := s.st.Rebind(`SELECT ciphertext FROM ci_secrets WHERE repo_id = ? AND name = ?`)
	var stored string
	err := s.st.DB.QueryRowContext(ctx, q, repoID, name).Scan(&stored)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("cisecret: get: %w", err)
	}
	return s.decrypt(stored)
}

func (s *Service) encrypt(plain string) (string, error) {
	if s.cipher == nil {
		return plain, nil
	}
	sealed, err := s.cipher.Encrypt([]byte(plain))
	if err != nil {
		return "", fmt.Errorf("cisecret: encrypt: %w", err)
	}
	return encPrefix + hex.EncodeToString(sealed), nil
}

func (s *Service) decrypt(stored string) (string, error) {
	if !strings.HasPrefix(stored, encPrefix) {
		return stored, nil
	}
	if s.cipher == nil {
		return "", errors.New("cisecret: no cipher configured for encrypted secret")
	}
	raw, err := hex.DecodeString(strings.TrimPrefix(stored, encPrefix))
	if err != nil {
		return "", fmt.Errorf("cisecret: decode: %w", err)
	}
	plain, err := s.cipher.Decrypt(raw)
	if err != nil {
		return "", fmt.Errorf("cisecret: decrypt: %w", err)
	}
	return string(plain), nil
}
