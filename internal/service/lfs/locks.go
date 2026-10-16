// SPDX-License-Identifier: AGPL-3.0
package lfs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrLocked indicates a path is already locked by another user.
var ErrLocked = errors.New("lfs: already locked")

// ErrLockInvalid indicates a malformed lock path.
var ErrLockInvalid = errors.New("lfs: invalid lock path")

// Lock is a Git LFS file lock held by a single user for a repository.
type Lock struct {
	ID        int64
	RepoID    int64
	Path      string
	UserID    int64
	CreatedAt time.Time
}

// validLockPath rejects empty paths, traversal sequences and control chars.
func validLockPath(path string) bool {
	if path == "" {
		return false
	}
	if strings.Contains(path, "..") {
		return false
	}
	for _, r := range path {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

// Lock acquires a lock on path for userID. Acquiring an existing lock owned by
// the same user is idempotent; a lock owned by another user returns ErrLocked.
func (s *Service) Lock(ctx context.Context, repoID, userID int64, path string) (*Lock, error) {
	if !validLockPath(path) {
		return nil, ErrLockInvalid
	}
	existing, err := s.lockByPath(ctx, repoID, path)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.UserID == userID {
			return existing, nil
		}
		return nil, ErrLocked
	}
	q := s.st.Rebind(`INSERT INTO lfs_locks (repo_id, path, user_id) VALUES (?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, repoID, path, userID)
	if err != nil {
		return nil, fmt.Errorf("lfs: insert lock: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.lockByID(ctx, id)
}

// Unlock releases a lock owned by userID.
func (s *Service) Unlock(ctx context.Context, repoID, userID, lockID int64) error {
	res, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM lfs_locks WHERE id = ? AND repo_id = ? AND user_id = ?`), lockID, repoID, userID)
	if err != nil {
		return fmt.Errorf("lfs: delete lock: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("lfs: delete lock: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListLocks returns all locks for a repository.
func (s *Service) ListLocks(ctx context.Context, repoID int64) ([]Lock, error) {
	q := s.st.Rebind(`SELECT id, repo_id, path, user_id, created_at FROM lfs_locks WHERE repo_id = ? ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID)
	if err != nil {
		return nil, fmt.Errorf("lfs: list locks: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Lock
	for rows.Next() {
		var l Lock
		if err := rows.Scan(&l.ID, &l.RepoID, &l.Path, &l.UserID, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Service) lockByPath(ctx context.Context, repoID int64, path string) (*Lock, error) {
	q := s.st.Rebind(`SELECT id, repo_id, path, user_id, created_at FROM lfs_locks WHERE repo_id = ? AND path = ?`)
	return s.scanLock(ctx, q, repoID, path)
}

func (s *Service) lockByID(ctx context.Context, id int64) (*Lock, error) {
	q := s.st.Rebind(`SELECT id, repo_id, path, user_id, created_at FROM lfs_locks WHERE id = ?`)
	l, err := s.scanLock(ctx, q, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, ErrNotFound
	}
	return l, nil
}

func (s *Service) scanLock(ctx context.Context, q string, args ...any) (*Lock, error) {
	var l Lock
	err := s.st.DB.QueryRowContext(ctx, q, args...).Scan(&l.ID, &l.RepoID, &l.Path, &l.UserID, &l.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lfs: query lock: %w", err)
	}
	return &l, nil
}
