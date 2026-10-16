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

const sshKeyColumns = `id, user_id, title, fingerprint, public_key, can_authenticate, can_sign, verified, last_used_at, created_at`

// AddSSHKey validates and stores an SSH public key with its purposes.
// At least one of canAuthenticate / canSign must be true.
func (s *Service) AddSSHKey(ctx context.Context, userID int64, title, pubKey string, canAuthenticate, canSign bool) (*domain.SSHKey, error) {
	if !canAuthenticate && !canSign {
		return nil, fmt.Errorf("user: ssh key must allow authentication or signing")
	}
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pubKey))
	if err != nil {
		return nil, fmt.Errorf("user: parse ssh key: %w", err)
	}
	fp := ssh.FingerprintSHA256(pk)
	q := s.st.Rebind(`INSERT INTO ssh_keys (user_id, title, fingerprint, public_key, can_authenticate, can_sign) VALUES (?, ?, ?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, userID, title, fp, pubKey, canAuthenticate, canSign)
	if err != nil {
		return nil, fmt.Errorf("user: insert ssh key: %w", err)
	}
	id, _ := res.LastInsertId()
	return &domain.SSHKey{ID: id, UserID: userID, Title: title, Fingerprint: fp, PublicKey: pubKey, CanAuthenticate: canAuthenticate, CanSign: canSign}, nil
}

// ListSSHKeys returns a user's keys.
func (s *Service) ListSSHKeys(ctx context.Context, userID int64) ([]*domain.SSHKey, error) {
	q := s.st.Rebind(`SELECT ` + sshKeyColumns + ` FROM ssh_keys WHERE user_id = ? ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanSSHKeys(rows)
}

// UpdateSSHKeyPurposes changes a key's authentication/signing flags.
// At least one must remain true.
func (s *Service) UpdateSSHKeyPurposes(ctx context.Context, userID, keyID int64, canAuthenticate, canSign bool) error {
	if !canAuthenticate && !canSign {
		return fmt.Errorf("user: ssh key must allow authentication or signing")
	}
	res, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE ssh_keys SET can_authenticate = ?, can_sign = ? WHERE id = ? AND user_id = ?`),
		canAuthenticate, canSign, keyID, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteSSHKey removes a key owned by the user.
func (s *Service) DeleteSSHKey(ctx context.Context, userID, keyID int64) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM ssh_keys WHERE id = ? AND user_id = ?`), keyID, userID)
	return err
}

// UserBySSHFingerprint resolves the owner of a public key that is allowed to
// authenticate (used by the SSH git transport).
func (s *Service) UserBySSHFingerprint(ctx context.Context, fingerprint string) (*domain.User, error) {
	q := s.st.Rebind(`SELECT u.id, u.username, u.email, u.password_hash, u.full_name, u.bio, u.avatar_url, u.website, u.location, u.is_admin, u.is_active, u.totp_enabled, u.totp_secret, u.created_at, u.updated_at FROM users u JOIN ssh_keys k ON k.user_id = u.id WHERE k.fingerprint = ? AND k.can_authenticate = ?`)
	row := s.st.DB.QueryRowContext(ctx, q, fingerprint, true)
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

// --- GPG / signing keys ---

// AddSigningKey stores a GPG (or other) public key used for signature
// verification. GPG keys are signing-only by default.
func (s *Service) AddSigningKey(ctx context.Context, userID int64, keyType, title, pubKey string) (*domain.SigningKey, error) {
	if keyType == "" {
		keyType = "gpg"
	}
	fp, err := fingerprintSigningKey(pubKey)
	if err != nil {
		return nil, err
	}
	q := s.st.Rebind(`INSERT INTO signing_keys (user_id, key_type, title, fingerprint, public_key) VALUES (?, ?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, userID, keyType, title, fp, pubKey)
	if err != nil {
		return nil, fmt.Errorf("user: insert signing key: %w", err)
	}
	id, _ := res.LastInsertId()
	return &domain.SigningKey{ID: id, UserID: userID, KeyType: keyType, Title: title, Fingerprint: fp, PublicKey: pubKey}, nil
}

// ListSigningKeys returns a user's GPG/signing keys.
func (s *Service) ListSigningKeys(ctx context.Context, userID int64) ([]*domain.SigningKey, error) {
	q := s.st.Rebind(`SELECT id, user_id, key_type, title, fingerprint, public_key, source_key_id, verified, created_at FROM signing_keys WHERE user_id = ? ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.SigningKey
	for rows.Next() {
		var k domain.SigningKey
		var src sql.NullInt64
		if err := rows.Scan(&k.ID, &k.UserID, &k.KeyType, &k.Title, &k.Fingerprint, &k.PublicKey, &src, &k.Verified, &k.CreatedAt); err != nil {
			return nil, err
		}
		if src.Valid {
			v := src.Int64
			k.SourceKeyID = &v
		}
		out = append(out, &k)
	}
	return out, rows.Err()
}

// DeleteSigningKey removes a signing key owned by the user.
func (s *Service) DeleteSigningKey(ctx context.Context, userID, keyID int64) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM signing_keys WHERE id = ? AND user_id = ?`), keyID, userID)
	return err
}

// SigningPublicKeys returns all signing-capable public keys for a user, drawn
// from both GPG keys and SSH keys flagged with CanSign.
func (s *Service) SigningPublicKeys(ctx context.Context, userID int64) ([]string, error) {
	q := s.st.Rebind(`SELECT public_key FROM signing_keys WHERE user_id = ? AND key_type = 'gpg'
		UNION ALL
		SELECT public_key FROM ssh_keys WHERE user_id = ? AND can_sign = ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, userID, userID, true)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// SigningKeyOwner resolves the user id that owns a signing key with the given
// fingerprint, searching GPG keys and signing-capable SSH keys.
func (s *Service) SigningKeyOwner(ctx context.Context, fingerprint string) (int64, string, error) {
	var userID int64
	q := s.st.Rebind(`SELECT user_id FROM signing_keys WHERE fingerprint = ?`)
	if err := s.st.DB.QueryRowContext(ctx, q, fingerprint).Scan(&userID); err == nil {
		return userID, "gpg", nil
	}
	q2 := s.st.Rebind(`SELECT user_id FROM ssh_keys WHERE fingerprint = ? AND can_sign = ?`)
	if err := s.st.DB.QueryRowContext(ctx, q2, fingerprint, true).Scan(&userID); err == nil {
		return userID, "ssh", nil
	}
	return 0, "", ErrNotFound
}

func scanSSHKeys(rows *sql.Rows) ([]*domain.SSHKey, error) {
	var out []*domain.SSHKey
	for rows.Next() {
		var k domain.SSHKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.Title, &k.Fingerprint, &k.PublicKey, &k.CanAuthenticate, &k.CanSign, &k.Verified, &k.LastUsedAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &k)
	}
	return out, rows.Err()
}
