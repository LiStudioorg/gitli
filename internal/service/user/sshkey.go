// SPDX-License-Identifier: AGPL-3.0
package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/LiStudioorg/gitli/internal/domain"
)

// AddSSHKey validates and stores an SSH public key.
func (s *Service) AddSSHKey(ctx context.Context, userID int64, title, pubKey string) (*domain.SSHKey, error) {
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pubKey))
	if err != nil {
		return nil, fmt.Errorf("user: parse ssh key: %w", err)
	}
	fp := ssh.FingerprintSHA256(pk)
	q := s.st.Rebind(`INSERT INTO ssh_keys (user_id, title, fingerprint, public_key) VALUES (?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, userID, title, fp, pubKey)
	if err != nil {
		return nil, fmt.Errorf("user: insert ssh key: %w", err)
	}
	id, _ := res.LastInsertId()
	return &domain.SSHKey{ID: id, UserID: userID, Title: title, Fingerprint: fp, PublicKey: pubKey}, nil
}

// ListSSHKeys returns a user's keys.
func (s *Service) ListSSHKeys(ctx context.Context, userID int64) ([]*domain.SSHKey, error) {
	q := s.st.Rebind(`SELECT id, user_id, title, fingerprint, public_key, verified, last_used_at, created_at FROM ssh_keys WHERE user_id = ? ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.SSHKey
	for rows.Next() {
		var k domain.SSHKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.Title, &k.Fingerprint, &k.PublicKey, &k.Verified, &k.LastUsedAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &k)
	}
	return out, rows.Err()
}

// DeleteSSHKey removes a key owned by the user.
func (s *Service) DeleteSSHKey(ctx context.Context, userID, keyID int64) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM ssh_keys WHERE id = ? AND user_id = ?`), keyID, userID)
	return err
}

// UserBySSHFingerprint resolves the owner of a public key.
func (s *Service) UserBySSHFingerprint(ctx context.Context, fingerprint string) (*domain.User, error) {
	q := s.st.Rebind(`SELECT u.id, u.username, u.email, u.password_hash, u.full_name, u.bio, u.avatar_url, u.website, u.location, u.is_admin, u.is_active, u.totp_enabled, u.totp_secret, u.created_at, u.updated_at FROM users u JOIN ssh_keys k ON k.user_id = u.id WHERE k.fingerprint = ?`)
	row := s.st.DB.QueryRowContext(ctx, q, fingerprint)
	u, err := scanUserRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

// TouchSSHKey records last usage.
func (s *Service) TouchSSHKey(ctx context.Context, fingerprint string) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE ssh_keys SET last_used_at = ? WHERE fingerprint = ?`), time.Now().UTC(), fingerprint)
	return err
}
