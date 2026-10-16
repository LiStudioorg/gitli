// SPDX-License-Identifier: AGPL-3.0
// Package reaction manages emoji reactions on issues, pull requests and comments.
package reaction

import (
	"context"
	"fmt"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

var validTargets = map[string]bool{
	"issue":   true,
	"pull":    true,
	"comment": true,
}

// Service manages reactions.
type Service struct {
	st *store.Store
}

// New builds a reaction Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Add records a reaction, skipping duplicates.
func (s *Service) Add(ctx context.Context, userID int64, targetType string, targetID int64, emoji string) error {
	if !validTargets[targetType] {
		return fmt.Errorf("reaction: invalid target type %q", targetType)
	}
	if emoji == "" {
		return fmt.Errorf("reaction: emoji required")
	}
	var n int
	q := s.st.Rebind(`SELECT COUNT(*) FROM reactions WHERE user_id = ? AND target_type = ? AND target_id = ? AND emoji = ?`)
	if err := s.st.DB.QueryRowContext(ctx, q, userID, targetType, targetID, emoji).Scan(&n); err != nil {
		return fmt.Errorf("reaction: lookup: %w", err)
	}
	if n > 0 {
		return nil
	}
	insert := s.st.Rebind(`INSERT INTO reactions (user_id, target_type, target_id, emoji) VALUES (?, ?, ?, ?)`)
	if _, err := s.st.DB.ExecContext(ctx, insert, userID, targetType, targetID, emoji); err != nil {
		return fmt.Errorf("reaction: insert: %w", err)
	}
	return nil
}

// Remove deletes a user's reaction.
func (s *Service) Remove(ctx context.Context, userID int64, targetType string, targetID int64, emoji string) error {
	if !validTargets[targetType] {
		return fmt.Errorf("reaction: invalid target type %q", targetType)
	}
	q := s.st.Rebind(`DELETE FROM reactions WHERE user_id = ? AND target_type = ? AND target_id = ? AND emoji = ?`)
	if _, err := s.st.DB.ExecContext(ctx, q, userID, targetType, targetID, emoji); err != nil {
		return fmt.Errorf("reaction: delete: %w", err)
	}
	return nil
}

// List returns emoji counts for a target.
func (s *Service) List(ctx context.Context, targetType string, targetID int64) (map[string]int, error) {
	q := s.st.Rebind(`SELECT emoji, COUNT(*) FROM reactions WHERE target_type = ? AND target_id = ? GROUP BY emoji`)
	rows, err := s.st.DB.QueryContext(ctx, q, targetType, targetID)
	if err != nil {
		return nil, fmt.Errorf("reaction: list: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := make(map[string]int)
	for rows.Next() {
		var emoji string
		var count int
		if err := rows.Scan(&emoji, &count); err != nil {
			return nil, err
		}
		out[emoji] = count
	}
	return out, rows.Err()
}

// ListByUser returns the emojis a user reacted with on a target.
func (s *Service) ListByUser(ctx context.Context, userID int64, targetType string, targetID int64) ([]string, error) {
	q := s.st.Rebind(`SELECT emoji FROM reactions WHERE user_id = ? AND target_type = ? AND target_id = ? ORDER BY emoji`)
	rows, err := s.st.DB.QueryContext(ctx, q, userID, targetType, targetID)
	if err != nil {
		return nil, fmt.Errorf("reaction: list by user: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var emoji string
		if err := rows.Scan(&emoji); err != nil {
			return nil, err
		}
		out = append(out, emoji)
	}
	return out, rows.Err()
}
