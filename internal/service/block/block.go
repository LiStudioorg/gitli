// SPDX-License-Identifier: AGPL-3.0
// Package block manages user-to-user blocking relationships.
package block

import (
	"context"
	"fmt"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Service manages user blocks.
type Service struct {
	st *store.Store
}

// New builds a block Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Block records that blockerID blocks blockedID. It is idempotent and rejects
// self-blocks.
func (s *Service) Block(ctx context.Context, blockerID, blockedID int64) error {
	if blockerID == blockedID {
		return fmt.Errorf("block: cannot block yourself")
	}
	var n int
	q := s.st.Rebind(`SELECT COUNT(*) FROM user_blocks WHERE blocker_id = ? AND blocked_id = ?`)
	if err := s.st.DB.QueryRowContext(ctx, q, blockerID, blockedID).Scan(&n); err != nil {
		return fmt.Errorf("block: lookup: %w", err)
	}
	if n > 0 {
		return nil
	}
	ins := s.st.Rebind(`INSERT INTO user_blocks (blocker_id, blocked_id) VALUES (?, ?)`)
	if _, err := s.st.DB.ExecContext(ctx, ins, blockerID, blockedID); err != nil {
		return fmt.Errorf("block: insert: %w", err)
	}
	return nil
}

// Unblock removes a block.
func (s *Service) Unblock(ctx context.Context, blockerID, blockedID int64) error {
	q := s.st.Rebind(`DELETE FROM user_blocks WHERE blocker_id = ? AND blocked_id = ?`)
	if _, err := s.st.DB.ExecContext(ctx, q, blockerID, blockedID); err != nil {
		return fmt.Errorf("block: delete: %w", err)
	}
	return nil
}

// IsBlocked reports whether blockerID blocks blockedID.
func (s *Service) IsBlocked(ctx context.Context, blockerID, blockedID int64) (bool, error) {
	var n int
	q := s.st.Rebind(`SELECT COUNT(*) FROM user_blocks WHERE blocker_id = ? AND blocked_id = ?`)
	if err := s.st.DB.QueryRowContext(ctx, q, blockerID, blockedID).Scan(&n); err != nil {
		return false, fmt.Errorf("block: is blocked: %w", err)
	}
	return n > 0, nil
}

// ListBlocked returns the ids of users blocked by blockerID.
func (s *Service) ListBlocked(ctx context.Context, blockerID int64) ([]int64, error) {
	q := s.st.Rebind(`SELECT blocked_id FROM user_blocks WHERE blocker_id = ? ORDER BY blocked_id`)
	rows, err := s.st.DB.QueryContext(ctx, q, blockerID)
	if err != nil {
		return nil, fmt.Errorf("block: list: %w", err)
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
