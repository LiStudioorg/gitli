// SPDX-License-Identifier: AGPL-3.0
// Package user provides user account operations.
package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates a missing user.
var ErrNotFound = errors.New("user: not found")

// Service manages user accounts.
type Service struct {
	st *store.Store
}

// New creates a user Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// CreateParams describes a new user.
type CreateParams struct {
	Username     string
	Email        string
	PasswordHash string
	IsAdmin      bool
}

// Create inserts a new user and returns it.
func (s *Service) Create(ctx context.Context, p CreateParams) (*domain.User, error) {
	q := s.st.Rebind(`INSERT INTO users (username, email, password_hash, is_admin) VALUES (?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, p.Username, p.Email, p.PasswordHash, p.IsAdmin)
	if err != nil {
		return nil, fmt.Errorf("user: insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("user: last insert id: %w", err)
	}
	return s.ByID(ctx, id)
}

const userColumns = `id, username, email, password_hash, full_name, bio, avatar_url, website, location, is_admin, is_active, totp_enabled, totp_secret, created_at, updated_at`

// ByID fetches a user by primary key.
func (s *Service) ByID(ctx context.Context, id int64) (*domain.User, error) {
	q := s.st.Rebind(`SELECT ` + userColumns + ` FROM users WHERE id = ?`)
	return s.scanOne(ctx, q, id)
}

// ByUsername fetches a user by username.
func (s *Service) ByUsername(ctx context.Context, username string) (*domain.User, error) {
	q := s.st.Rebind(`SELECT ` + userColumns + ` FROM users WHERE username = ?`)
	return s.scanOne(ctx, q, username)
}

// ByEmail fetches a user by email.
func (s *Service) ByEmail(ctx context.Context, email string) (*domain.User, error) {
	q := s.st.Rebind(`SELECT ` + userColumns + ` FROM users WHERE email = ?`)
	return s.scanOne(ctx, q, email)
}

// Exists reports whether a username is already taken.
func (s *Service) Exists(ctx context.Context, username string) (bool, error) {
	q := s.st.Rebind(`SELECT 1 FROM users WHERE username = ?`)
	var x int
	err := s.st.DB.QueryRowContext(ctx, q, username).Scan(&x)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// List returns users with pagination.
func (s *Service) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	q := s.st.Rebind(`SELECT ` + userColumns + ` FROM users ORDER BY id LIMIT ? OFFSET ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanMany(rows)
}

// UpdateProfile updates mutable profile fields.
func (s *Service) UpdateProfile(ctx context.Context, u *domain.User) error {
	q := s.st.Rebind(`UPDATE users SET full_name = ?, bio = ?, website = ?, location = ?, updated_at = ? WHERE id = ?`)
	_, err := s.st.DB.ExecContext(ctx, q, u.FullName, u.Bio, u.Website, u.Location, time.Now().UTC(), u.ID)
	return err
}

// UpdatePassword sets a new password hash.
func (s *Service) UpdatePassword(ctx context.Context, userID int64, hash string) error {
	q := s.st.Rebind(`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`)
	_, err := s.st.DB.ExecContext(ctx, q, hash, time.Now().UTC(), userID)
	return err
}

// UpdateTOTP sets the encrypted TOTP secret and enabled flag.
func (s *Service) UpdateTOTP(ctx context.Context, userID int64, secret string, enabled bool) error {
	q := s.st.Rebind(`UPDATE users SET totp_secret = ?, totp_enabled = ?, updated_at = ? WHERE id = ?`)
	_, err := s.st.DB.ExecContext(ctx, q, secret, enabled, time.Now().UTC(), userID)
	return err
}

// Rename changes a username and records a redirect from the old name.
func (s *Service) Rename(ctx context.Context, userID int64, newName string) error {
	tx, err := s.st.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var oldName string
	if err := tx.QueryRowContext(ctx, s.st.Rebind(`SELECT username FROM users WHERE id = ?`), userID).Scan(&oldName); err != nil {
		return fmt.Errorf("user: read old name: %w", err)
	}
	if _, err := tx.ExecContext(ctx, s.st.Rebind(`UPDATE users SET username = ?, updated_at = ? WHERE id = ?`), newName, time.Now().UTC(), userID); err != nil {
		return fmt.Errorf("user: rename: %w", err)
	}
	if _, err := tx.ExecContext(ctx, s.st.Rebind(`INSERT INTO user_redirects (old_username, user_id) VALUES (?, ?)`), oldName, userID); err != nil {
		return fmt.Errorf("user: redirect: %w", err)
	}
	return tx.Commit()
}

// ResolveRedirect returns the user id for a former username, if any.
func (s *Service) ResolveRedirect(ctx context.Context, oldName string) (int64, error) {
	q := s.st.Rebind(`SELECT user_id FROM user_redirects WHERE old_username = ?`)
	var id int64
	err := s.st.DB.QueryRowContext(ctx, q, oldName).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return id, err
}

func (s *Service) scanOne(ctx context.Context, query string, args ...any) (*domain.User, error) {
	row := s.st.DB.QueryRowContext(ctx, query, args...)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (s *Service) scanMany(rows *sql.Rows) ([]*domain.User, error) {
	var out []*domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanUser(sc scanner) (*domain.User, error) {
	var u domain.User
	err := sc.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.FullName, &u.Bio,
		&u.AvatarURL, &u.Website, &u.Location, &u.IsAdmin, &u.IsActive, &u.TOTPEnabled,
		&u.TOTPSecret, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func scanUserRow(row *sql.Row) (*domain.User, error) {
	return scanUser(row)
}
