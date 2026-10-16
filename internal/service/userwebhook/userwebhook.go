// SPDX-License-Identifier: AGPL-3.0
// Package userwebhook manages per-user outbound webhook subscriptions.
package userwebhook

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/LiStudioorg/gitli/internal/infra/secutil"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates a missing user webhook.
var ErrNotFound = errors.New("userwebhook: not found")

// Hook is a user-scoped webhook subscription.
type Hook struct {
	ID     int64
	UserID int64
	URL    string
	Secret string
	Events []string
	Active bool
}

// Service manages user webhooks.
type Service struct {
	st *store.Store
}

// New builds a user webhook Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Create registers a webhook for a user after validating the outbound URL.
func (s *Service) Create(ctx context.Context, userID int64, url, secret string, events []string) (*Hook, error) {
	if err := secutil.ValidateOutboundURL(url, true); err != nil {
		return nil, err
	}
	q := s.st.Rebind(`INSERT INTO user_webhooks (user_id, url, secret, events, active) VALUES (?, ?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, userID, url, secret, strings.Join(events, ","), true)
	if err != nil {
		return nil, fmt.Errorf("userwebhook: insert: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, userID, id)
}

// ByID loads a webhook owned by userID.
func (s *Service) ByID(ctx context.Context, userID, id int64) (*Hook, error) {
	q := s.st.Rebind(`SELECT id, user_id, url, secret, events, active FROM user_webhooks WHERE id = ? AND user_id = ?`)
	var h Hook
	var events string
	err := s.st.DB.QueryRowContext(ctx, q, id, userID).Scan(&h.ID, &h.UserID, &h.URL, &h.Secret, &events, &h.Active)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("userwebhook: query: %w", err)
	}
	h.Events = splitCSV(events)
	return &h, nil
}

// List returns all webhooks owned by userID.
func (s *Service) List(ctx context.Context, userID int64) ([]Hook, error) {
	q := s.st.Rebind(`SELECT id, user_id, url, secret, events, active FROM user_webhooks WHERE user_id = ? ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("userwebhook: list: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Hook
	for rows.Next() {
		var h Hook
		var events string
		if err := rows.Scan(&h.ID, &h.UserID, &h.URL, &h.Secret, &events, &h.Active); err != nil {
			return nil, err
		}
		h.Events = splitCSV(events)
		out = append(out, h)
	}
	return out, rows.Err()
}

// Delete removes a webhook owned by userID.
func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	res, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM user_webhooks WHERE id = ? AND user_id = ?`), id, userID)
	if err != nil {
		return fmt.Errorf("userwebhook: delete: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("userwebhook: delete: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetActive toggles the active flag of a webhook owned by userID.
func (s *Service) SetActive(ctx context.Context, userID, id int64, active bool) error {
	res, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE user_webhooks SET active = ? WHERE id = ? AND user_id = ?`), active, id, userID)
	if err != nil {
		return fmt.Errorf("userwebhook: update: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("userwebhook: update: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
