// SPDX-License-Identifier: AGPL-3.0
// Package savedreply manages reusable comment snippets for a user.
package savedreply

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates a missing saved reply.
var ErrNotFound = errors.New("savedreply: not found")

// Reply is a saved reply.
type Reply struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// Service manages saved replies.
type Service struct {
	st *store.Store
}

// New builds a saved reply Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Create stores a new saved reply.
func (s *Service) Create(ctx context.Context, userID int64, title, body string) (*Reply, error) {
	if title == "" {
		return nil, fmt.Errorf("savedreply: title required")
	}
	q := s.st.Rebind(`INSERT INTO saved_replies (user_id, title, body) VALUES (?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, userID, title, body)
	if err != nil {
		return nil, fmt.Errorf("savedreply: insert: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

// ByID loads a saved reply.
func (s *Service) ByID(ctx context.Context, id int64) (*Reply, error) {
	q := s.st.Rebind(`SELECT id, user_id, title, body, created_at FROM saved_replies WHERE id = ?`)
	var r Reply
	err := s.st.DB.QueryRowContext(ctx, q, id).Scan(&r.ID, &r.UserID, &r.Title, &r.Body, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &r, err
}

// List returns a user's saved replies.
func (s *Service) List(ctx context.Context, userID int64) ([]*Reply, error) {
	q := s.st.Rebind(`SELECT id, user_id, title, body, created_at FROM saved_replies WHERE user_id = ? ORDER BY id DESC`)
	rows, err := s.st.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*Reply
	for rows.Next() {
		var r Reply
		if err := rows.Scan(&r.ID, &r.UserID, &r.Title, &r.Body, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

// Update edits a saved reply owned by the user.
func (s *Service) Update(ctx context.Context, userID, id int64, title, body string) error {
	res, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE saved_replies SET title = ?, body = ? WHERE id = ? AND user_id = ?`), title, body, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a saved reply owned by the user.
func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM saved_replies WHERE id = ? AND user_id = ?`), id, userID)
	return err
}
