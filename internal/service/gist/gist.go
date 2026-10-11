// SPDX-License-Identifier: AGPL-3.0
// Package gist manages user-owned code snippets.
package gist

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates a missing gist.
var ErrNotFound = errors.New("gist: not found")

// Service manages gists.
type Service struct {
	st *store.Store
}

// New builds a gist Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Gist is a user-owned snippet.
type Gist struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
}

// Create inserts a new gist.
func (s *Service) Create(ctx context.Context, userID int64, description string, isPublic bool) (*Gist, error) {
	q := s.st.Rebind(`INSERT INTO gists (user_id, description, is_public, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`)
	now := time.Now().UTC()
	res, err := s.st.DB.ExecContext(ctx, q, userID, description, isPublic, now, now)
	if err != nil {
		return nil, fmt.Errorf("gist: insert: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

// ByID fetches a gist by id.
func (s *Service) ByID(ctx context.Context, id int64) (*Gist, error) {
	q := s.st.Rebind(`SELECT id, user_id, description, is_public FROM gists WHERE id = ?`)
	var g Gist
	err := s.st.DB.QueryRowContext(ctx, q, id).Scan(&g.ID, &g.UserID, &g.Description, &g.IsPublic)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("gist: by id: %w", err)
	}
	return &g, nil
}

// ListByUser returns gists owned by a user ordered by newest first.
func (s *Service) ListByUser(ctx context.Context, userID int64) ([]*Gist, error) {
	q := s.st.Rebind(`SELECT id, user_id, description, is_public FROM gists WHERE user_id = ? ORDER BY id DESC`)
	rows, err := s.st.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("gist: list: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []*Gist
	for rows.Next() {
		var g Gist
		if err := rows.Scan(&g.ID, &g.UserID, &g.Description, &g.IsPublic); err != nil {
			return nil, fmt.Errorf("gist: scan: %w", err)
		}
		out = append(out, &g)
	}
	return out, rows.Err()
}

// Delete removes a gist owned by userID.
func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	q := s.st.Rebind(`DELETE FROM gists WHERE id = ? AND user_id = ?`)
	res, err := s.st.DB.ExecContext(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("gist: delete: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
