// SPDX-License-Identifier: AGPL-3.0
package user

import (
	"context"
	"fmt"
	"time"

	"github.com/LiStudioorg/gitli/internal/domain"
)

// SetAdmin grants or revokes administrator rights.
func (s *Service) SetAdmin(ctx context.Context, userID int64, admin bool) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE users SET is_admin = ?, updated_at = ? WHERE id = ?`), admin, time.Now().UTC(), userID)
	return err
}

// SetActive enables or disables an account.
func (s *Service) SetActive(ctx context.Context, userID int64, active bool) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE users SET is_active = ?, updated_at = ? WHERE id = ?`), active, time.Now().UTC(), userID)
	return err
}

// SetEmail updates a user's email address.
func (s *Service) SetEmail(ctx context.Context, userID int64, email string) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE users SET email = ?, updated_at = ? WHERE id = ?`), email, time.Now().UTC(), userID)
	return err
}

// ResetTOTP clears a user's TOTP configuration.
func (s *Service) ResetTOTP(ctx context.Context, userID int64) error {
	if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE users SET totp_secret = '', totp_enabled = ?, updated_at = ? WHERE id = ?`), false, time.Now().UTC(), userID); err != nil {
		return err
	}
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM user_secrets WHERE user_id = ? AND name = 'recovery'`), userID)
	return err
}

// ListAll returns every user (used by CLI/admin tooling).
func (s *Service) ListAll(ctx context.Context) ([]*domain.User, error) {
	q := s.st.Rebind(`SELECT ` + userColumns + ` FROM users ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return s.scanMany(rows)
}

// CountAdmins returns the number of active administrators.
func (s *Service) CountAdmins(ctx context.Context) (int, error) {
	var n int
	err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT COUNT(*) FROM users WHERE is_admin = ? AND is_active = ?`), true, true).Scan(&n)
	return n, err
}

// Delete removes a user row. Callers must purge owned data separately.
func (s *Service) Delete(ctx context.Context, userID int64) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM users WHERE id = ?`), userID)
	return err
}

// ErrLastAdmin indicates an operation would remove the last active admin.
var ErrLastAdmin = fmt.Errorf("user: cannot remove the last active administrator")
