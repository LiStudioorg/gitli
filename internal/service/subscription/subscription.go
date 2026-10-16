// SPDX-License-Identifier: AGPL-3.0
// Package subscription manages user subscriptions to issues, pull requests and repositories.
package subscription

import (
	"context"
	"fmt"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

var validTargets = map[string]bool{
	"issue": true,
	"pull":  true,
	"repo":  true,
}

// Service manages subscriptions.
type Service struct {
	st *store.Store
}

// New builds a subscription Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Subscribe records a subscription, skipping duplicates.
func (s *Service) Subscribe(ctx context.Context, userID int64, targetType string, targetID int64) error {
	if !validTargets[targetType] {
		return fmt.Errorf("subscription: invalid target type %q", targetType)
	}
	var n int
	q := s.st.Rebind(`SELECT COUNT(*) FROM subscriptions WHERE user_id = ? AND target_type = ? AND target_id = ?`)
	if err := s.st.DB.QueryRowContext(ctx, q, userID, targetType, targetID).Scan(&n); err != nil {
		return fmt.Errorf("subscription: lookup: %w", err)
	}
	if n > 0 {
		return nil
	}
	insert := s.st.Rebind(`INSERT INTO subscriptions (user_id, target_type, target_id) VALUES (?, ?, ?)`)
	if _, err := s.st.DB.ExecContext(ctx, insert, userID, targetType, targetID); err != nil {
		return fmt.Errorf("subscription: insert: %w", err)
	}
	return nil
}

// Unsubscribe removes a subscription.
func (s *Service) Unsubscribe(ctx context.Context, userID int64, targetType string, targetID int64) error {
	if !validTargets[targetType] {
		return fmt.Errorf("subscription: invalid target type %q", targetType)
	}
	q := s.st.Rebind(`DELETE FROM subscriptions WHERE user_id = ? AND target_type = ? AND target_id = ?`)
	if _, err := s.st.DB.ExecContext(ctx, q, userID, targetType, targetID); err != nil {
		return fmt.Errorf("subscription: delete: %w", err)
	}
	return nil
}

// IsSubscribed reports whether a user is subscribed to a target.
func (s *Service) IsSubscribed(ctx context.Context, userID int64, targetType string, targetID int64) (bool, error) {
	if !validTargets[targetType] {
		return false, fmt.Errorf("subscription: invalid target type %q", targetType)
	}
	var n int
	q := s.st.Rebind(`SELECT COUNT(*) FROM subscriptions WHERE user_id = ? AND target_type = ? AND target_id = ?`)
	if err := s.st.DB.QueryRowContext(ctx, q, userID, targetType, targetID).Scan(&n); err != nil {
		return false, fmt.Errorf("subscription: lookup: %w", err)
	}
	return n > 0, nil
}

// ListByUser lists the target IDs a user is subscribed to for a target type.
func (s *Service) ListByUser(ctx context.Context, userID int64, targetType string) ([]int64, error) {
	if !validTargets[targetType] {
		return nil, fmt.Errorf("subscription: invalid target type %q", targetType)
	}
	q := s.st.Rebind(`SELECT target_id FROM subscriptions WHERE user_id = ? AND target_type = ? ORDER BY target_id`)
	rows, err := s.st.DB.QueryContext(ctx, q, userID, targetType)
	if err != nil {
		return nil, fmt.Errorf("subscription: list by user: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
