// SPDX-License-Identifier: AGPL-3.0
package ci

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ErrNotFound indicates a missing cache entry.
var ErrNotFound = errors.New("ci: not found")

// CacheEntry is a build cache blob scoped to a repository and key.
type CacheEntry struct {
	ID         int64
	RepoID     int64
	Key        string
	StoredPath string
	SizeBytes  int64
}

// PutCache stores or replaces the cache entry identified by repo_id + key.
func (s *Service) PutCache(ctx context.Context, repoID int64, key, storedPath string, size int64) error {
	sel := s.st.Rebind(`SELECT id FROM ci_cache WHERE repo_id = ? AND key = ?`)
	var id int64
	err := s.st.DB.QueryRowContext(ctx, sel, repoID, key).Scan(&id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		ins := s.st.Rebind(`INSERT INTO ci_cache (repo_id, key, stored_path, size_bytes) VALUES (?, ?, ?, ?)`)
		if _, err := s.st.DB.ExecContext(ctx, ins, repoID, key, storedPath, size); err != nil {
			return fmt.Errorf("insert cache: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("select cache: %w", err)
	default:
		upd := s.st.Rebind(`UPDATE ci_cache SET stored_path = ?, size_bytes = ? WHERE id = ?`)
		if _, err := s.st.DB.ExecContext(ctx, upd, storedPath, size, id); err != nil {
			return fmt.Errorf("update cache: %w", err)
		}
		return nil
	}
}

// GetCache returns the cache entry for repo_id + key, or ErrNotFound.
func (s *Service) GetCache(ctx context.Context, repoID int64, key string) (*CacheEntry, error) {
	q := s.st.Rebind(`SELECT id, repo_id, key, stored_path, size_bytes FROM ci_cache WHERE repo_id = ? AND key = ?`)
	var e CacheEntry
	err := s.st.DB.QueryRowContext(ctx, q, repoID, key).Scan(&e.ID, &e.RepoID, &e.Key, &e.StoredPath, &e.SizeBytes)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get cache: %w", err)
	}
	return &e, nil
}

// ListCache returns all cache entries for a repository in insertion order.
func (s *Service) ListCache(ctx context.Context, repoID int64) ([]CacheEntry, error) {
	q := s.st.Rebind(`SELECT id, repo_id, key, stored_path, size_bytes FROM ci_cache WHERE repo_id = ? ORDER BY id ASC`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID)
	if err != nil {
		return nil, fmt.Errorf("list cache: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []CacheEntry
	for rows.Next() {
		var e CacheEntry
		if err := rows.Scan(&e.ID, &e.RepoID, &e.Key, &e.StoredPath, &e.SizeBytes); err != nil {
			return nil, fmt.Errorf("scan cache: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// DeleteCache removes the cache entry for repo_id + key.
func (s *Service) DeleteCache(ctx context.Context, repoID int64, key string) error {
	q := s.st.Rebind(`DELETE FROM ci_cache WHERE repo_id = ? AND key = ?`)
	if _, err := s.st.DB.ExecContext(ctx, q, repoID, key); err != nil {
		return fmt.Errorf("delete cache: %w", err)
	}
	return nil
}
