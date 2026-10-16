// SPDX-License-Identifier: AGPL-3.0
// Package notifpref stores per-user notification preferences.
package notifpref

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Service manages notification preferences.
type Service struct {
	st *store.Store
}

// New builds a notifpref Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Get returns the value for a preference key and whether it exists.
func (s *Service) Get(ctx context.Context, userID int64, key string) (string, bool, error) {
	q := s.st.Rebind(`SELECT value FROM notification_prefs WHERE user_id = ? AND key = ?`)
	var v string
	err := s.st.DB.QueryRowContext(ctx, q, userID, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("notifpref: get: %w", err)
	}
	return v, true, nil
}

// Set upserts a preference value.
func (s *Service) Set(ctx context.Context, userID int64, key, value string) error {
	if err := s.Delete(ctx, userID, key); err != nil {
		return err
	}
	q := s.st.Rebind(`INSERT INTO notification_prefs (user_id, key, value) VALUES (?, ?, ?)`)
	if _, err := s.st.DB.ExecContext(ctx, q, userID, key, value); err != nil {
		return fmt.Errorf("notifpref: insert: %w", err)
	}
	return nil
}

// All returns every preference for a user.
func (s *Service) All(ctx context.Context, userID int64) (map[string]string, error) {
	q := s.st.Rebind(`SELECT key, value FROM notification_prefs WHERE user_id = ? ORDER BY key`)
	rows, err := s.st.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("notifpref: all: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// Delete removes a preference key.
func (s *Service) Delete(ctx context.Context, userID int64, key string) error {
	q := s.st.Rebind(`DELETE FROM notification_prefs WHERE user_id = ? AND key = ?`)
	if _, err := s.st.DB.ExecContext(ctx, q, userID, key); err != nil {
		return fmt.Errorf("notifpref: delete: %w", err)
	}
	return nil
}
