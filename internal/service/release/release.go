// SPDX-License-Identifier: AGPL-3.0
// Package release manages repository releases and their download assets.
package release

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates a missing release.
var ErrNotFound = errors.New("release: not found")

// Service manages releases.
type Service struct {
	st *store.Store
}

// New builds a release Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Release is a tagged, optionally published, set of assets.
type Release struct {
	ID           int64  `json:"id"`
	RepoID       int64  `json:"repo_id"`
	AuthorID     int64  `json:"author_id"`
	Tag          string `json:"tag"`
	Name         string `json:"name"`
	Body         string `json:"body"`
	IsDraft      bool   `json:"is_draft"`
	IsPrerelease bool   `json:"is_prerelease"`
	CreatedAt    string `json:"created_at"`
}

// Asset is a downloadable file attached to a release.
type Asset struct {
	ID         int64  `json:"id"`
	ReleaseID  int64  `json:"release_id"`
	Name       string `json:"name"`
	SizeBytes  int64  `json:"size_bytes"`
	StoredPath string `json:"stored_path"`
}

// Create inserts a new draft release.
func (s *Service) Create(ctx context.Context, repoID, authorID int64, tag, name, body string) (*Release, error) {
	q := s.st.Rebind(`INSERT INTO releases (repo_id, tag, name, body, is_draft, is_prerelease, author_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	now := time.Now().UTC()
	res, err := s.st.DB.ExecContext(ctx, q, repoID, tag, name, body, true, false, authorID, now, now)
	if err != nil {
		return nil, fmt.Errorf("release: insert: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.byID(ctx, id)
}

// ByTag fetches a release by repository and tag.
func (s *Service) ByTag(ctx context.Context, repoID int64, tag string) (*Release, error) {
	q := s.st.Rebind(`SELECT id, repo_id, author_id, tag, name, body, is_draft, is_prerelease FROM releases WHERE repo_id = ? AND tag = ?`)
	var r Release
	err := s.st.DB.QueryRowContext(ctx, q, repoID, tag).Scan(&r.ID, &r.RepoID, &r.AuthorID, &r.Tag, &r.Name, &r.Body, &r.IsDraft, &r.IsPrerelease)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("release: by tag: %w", err)
	}
	return &r, nil
}

// List returns all releases for a repository ordered newest first.
func (s *Service) List(ctx context.Context, repoID int64) ([]*Release, error) {
	q := s.st.Rebind(`SELECT id, repo_id, author_id, tag, name, body, is_draft, is_prerelease, created_at FROM releases WHERE repo_id = ? ORDER BY id DESC`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID)
	if err != nil {
		return nil, fmt.Errorf("release: list: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []*Release
	for rows.Next() {
		var r Release
		if err := rows.Scan(&r.ID, &r.RepoID, &r.AuthorID, &r.Tag, &r.Name, &r.Body, &r.IsDraft, &r.IsPrerelease, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("release: scan: %w", err)
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

// Publish clears the draft flag on a release.
func (s *Service) Publish(ctx context.Context, id int64) error {
	q := s.st.Rebind(`UPDATE releases SET is_draft = ?, updated_at = ? WHERE id = ?`)
	res, err := s.st.DB.ExecContext(ctx, q, false, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("release: publish: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// AddAsset attaches a download asset to a release.
func (s *Service) AddAsset(ctx context.Context, releaseID int64, name string, size int64, storedPath string) (*Asset, error) {
	q := s.st.Rebind(`INSERT INTO release_assets (release_id, name, size_bytes, stored_path) VALUES (?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, releaseID, name, size, storedPath)
	if err != nil {
		return nil, fmt.Errorf("release: add asset: %w", err)
	}
	id, _ := res.LastInsertId()
	return &Asset{ID: id, ReleaseID: releaseID, Name: name, SizeBytes: size, StoredPath: storedPath}, nil
}

// ListAssets returns assets attached to a release ordered by id.
func (s *Service) ListAssets(ctx context.Context, releaseID int64) ([]*Asset, error) {
	q := s.st.Rebind(`SELECT id, release_id, name, size_bytes, stored_path FROM release_assets WHERE release_id = ? ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q, releaseID)
	if err != nil {
		return nil, fmt.Errorf("release: list assets: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []*Asset
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.ID, &a.ReleaseID, &a.Name, &a.SizeBytes, &a.StoredPath); err != nil {
			return nil, fmt.Errorf("release: scan asset: %w", err)
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

func (s *Service) byID(ctx context.Context, id int64) (*Release, error) {
	q := s.st.Rebind(`SELECT id, repo_id, author_id, tag, name, body, is_draft, is_prerelease FROM releases WHERE id = ?`)
	var r Release
	err := s.st.DB.QueryRowContext(ctx, q, id).Scan(&r.ID, &r.RepoID, &r.AuthorID, &r.Tag, &r.Name, &r.Body, &r.IsDraft, &r.IsPrerelease)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("release: by id: %w", err)
	}
	return &r, nil
}
