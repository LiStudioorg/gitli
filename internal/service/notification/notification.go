// SPDX-License-Identifier: AGPL-3.0
// Package notification manages in-app notifications.
package notification

import (
	"context"
	"fmt"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Service manages notifications.
type Service struct {
	st *store.Store
}

// New builds a notification Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Create inserts a notification.
func (s *Service) Create(ctx context.Context, n *domain.Notification) error {
	q := s.st.Rebind(`INSERT INTO notifications (user_id, subject_type, subject_id, title, url) VALUES (?, ?, ?, ?, ?)`)
	_, err := s.st.DB.ExecContext(ctx, q, n.UserID, n.SubjectType, n.SubjectID, n.Title, n.URL)
	if err != nil {
		return fmt.Errorf("notification: insert: %w", err)
	}
	return nil
}

// List returns a user's notifications.
func (s *Service) List(ctx context.Context, userID int64, unreadOnly bool, limit int) ([]*domain.Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	base := `SELECT id, user_id, subject_type, subject_id, title, url, is_read, created_at FROM notifications WHERE user_id = ?`
	if unreadOnly {
		base += ` AND is_read = ?`
	}
	base += ` ORDER BY id DESC LIMIT ?`
	args := []any{userID}
	if unreadOnly {
		args = append(args, false)
	}
	args = append(args, limit)
	rows, err := s.st.DB.QueryContext(ctx, s.st.Rebind(base), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.Notification
	for rows.Next() {
		var n domain.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.SubjectType, &n.SubjectID, &n.Title, &n.URL, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &n)
	}
	return out, rows.Err()
}

// MarkRead marks a notification read.
func (s *Service) MarkRead(ctx context.Context, userID, id int64) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE notifications SET is_read = ? WHERE id = ? AND user_id = ?`), true, id, userID)
	return err
}

// UnreadCount returns the number of unread notifications.
func (s *Service) UnreadCount(ctx context.Context, userID int64) (int, error) {
	var n int
	err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT COUNT(*) FROM notifications WHERE user_id = ? AND is_read = ?`), userID, false).Scan(&n)
	return n, err
}

// MarkAllRead marks every notification for a user as read.
func (s *Service) MarkAllRead(ctx context.Context, userID int64) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE notifications SET is_read = ? WHERE user_id = ? AND is_read = ?`), true, userID, false)
	return err
}
