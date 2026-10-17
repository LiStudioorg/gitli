// SPDX-License-Identifier: AGPL-3.0
// Package follow manages user follow relationships.
package follow

import (
	"context"
	"fmt"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Service manages follows.
type Service struct {
	st *store.Store
}

// New builds a follow Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Follow creates a follow relationship. Rejects self-follow; idempotent.
func (s *Service) Follow(ctx context.Context, followerID, followingID int64) error {
	if followerID == followingID {
		return fmt.Errorf("follow: cannot follow yourself")
	}
	q := s.st.Rebind(`INSERT INTO follows (follower_id, following_id) VALUES (?, ?) ON CONFLICT (follower_id, following_id) DO NOTHING`)
	if _, err := s.st.DB.ExecContext(ctx, q, followerID, followingID); err != nil {
		// MySQL fallback.
		if _, err2 := s.st.DB.ExecContext(ctx, s.st.Rebind(`INSERT IGNORE INTO follows (follower_id, following_id) VALUES (?, ?)`), followerID, followingID); err2 != nil {
			return fmt.Errorf("follow: insert: %w", err)
		}
	}
	return nil
}

// Unfollow removes a follow relationship.
func (s *Service) Unfollow(ctx context.Context, followerID, followingID int64) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM follows WHERE follower_id = ? AND following_id = ?`), followerID, followingID)
	return err
}

// IsFollowing reports whether followerID follows followingID.
func (s *Service) IsFollowing(ctx context.Context, followerID, followingID int64) (bool, error) {
	var x int
	err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT 1 FROM follows WHERE follower_id = ? AND following_id = ?`), followerID, followingID).Scan(&x)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// Counts returns (followers, following) for a user.
func (s *Service) Counts(ctx context.Context, userID int64) (int, int, error) {
	var followers, following int
	if err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT COUNT(*) FROM follows WHERE following_id = ?`), userID).Scan(&followers); err != nil {
		return 0, 0, err
	}
	if err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT COUNT(*) FROM follows WHERE follower_id = ?`), userID).Scan(&following); err != nil {
		return 0, 0, err
	}
	return followers, following, nil
}

// ListFollowers returns user ids that follow userID.
func (s *Service) ListFollowers(ctx context.Context, userID int64, limit int) ([]int64, error) {
	return s.list(ctx, `SELECT follower_id FROM follows WHERE following_id = ? ORDER BY id DESC LIMIT ?`, userID, limit)
}

// ListFollowing returns user ids that userID follows.
func (s *Service) ListFollowing(ctx context.Context, userID int64, limit int) ([]int64, error) {
	return s.list(ctx, `SELECT following_id FROM follows WHERE follower_id = ? ORDER BY id DESC LIMIT ?`, userID, limit)
}

func (s *Service) list(ctx context.Context, query string, userID int64, limit int) ([]int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.st.DB.QueryContext(ctx, s.st.Rebind(query), userID, limit)
	if err != nil {
		return nil, err
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
