// SPDX-License-Identifier: AGPL-3.0
// Package wiki manages per-repository wiki pages.
package wiki

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates a missing wiki page.
var ErrNotFound = errors.New("wiki: not found")

// Service manages wiki pages.
type Service struct {
	st *store.Store
}

// New builds a wiki Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Page is a single wiki page belonging to a repository.
type Page struct {
	ID     int64  `json:"id"`
	RepoID int64  `json:"repo_id"`
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// Create inserts a new wiki page.
func (s *Service) Create(ctx context.Context, repoID int64, slug, title, body string) (*Page, error) {
	q := s.st.Rebind(`INSERT INTO wiki_pages (repo_id, slug, title, body, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`)
	now := time.Now().UTC()
	res, err := s.st.DB.ExecContext(ctx, q, repoID, slug, title, body, now, now)
	if err != nil {
		return nil, fmt.Errorf("wiki: insert: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

// BySlug fetches a page by repository and slug.
func (s *Service) BySlug(ctx context.Context, repoID int64, slug string) (*Page, error) {
	q := s.st.Rebind(`SELECT id, repo_id, slug, title, body FROM wiki_pages WHERE repo_id = ? AND slug = ?`)
	var p Page
	err := s.st.DB.QueryRowContext(ctx, q, repoID, slug).Scan(&p.ID, &p.RepoID, &p.Slug, &p.Title, &p.Body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("wiki: by slug: %w", err)
	}
	return &p, nil
}

// ByID fetches a page by id.
func (s *Service) ByID(ctx context.Context, id int64) (*Page, error) {
	q := s.st.Rebind(`SELECT id, repo_id, slug, title, body FROM wiki_pages WHERE id = ?`)
	var p Page
	err := s.st.DB.QueryRowContext(ctx, q, id).Scan(&p.ID, &p.RepoID, &p.Slug, &p.Title, &p.Body)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("wiki: by id: %w", err)
	}
	return &p, nil
}

// List returns all pages for a repository ordered by slug.
func (s *Service) List(ctx context.Context, repoID int64) ([]*Page, error) {
	q := s.st.Rebind(`SELECT id, repo_id, slug, title, body FROM wiki_pages WHERE repo_id = ? ORDER BY slug`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID)
	if err != nil {
		return nil, fmt.Errorf("wiki: list: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []*Page
	for rows.Next() {
		var p Page
		if err := rows.Scan(&p.ID, &p.RepoID, &p.Slug, &p.Title, &p.Body); err != nil {
			return nil, fmt.Errorf("wiki: scan: %w", err)
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// Update persists title and body for an existing page identified by id.
func (s *Service) Update(ctx context.Context, page *Page) error {
	q := s.st.Rebind(`UPDATE wiki_pages SET title = ?, body = ?, updated_at = ? WHERE id = ?`)
	res, err := s.st.DB.ExecContext(ctx, q, page.Title, page.Body, time.Now().UTC(), page.ID)
	if err != nil {
		return fmt.Errorf("wiki: update: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a page by repository and slug.
func (s *Service) Delete(ctx context.Context, repoID int64, slug string) error {
	q := s.st.Rebind(`DELETE FROM wiki_pages WHERE repo_id = ? AND slug = ?`)
	res, err := s.st.DB.ExecContext(ctx, q, repoID, slug)
	if err != nil {
		return fmt.Errorf("wiki: delete: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
