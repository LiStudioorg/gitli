// SPDX-License-Identifier: AGPL-3.0
package repo

import (
	"context"
	"fmt"
	"time"
)

// Rename changes a repository's name and records a redirect from the old path.
func (s *Service) Rename(ctx context.Context, repoID int64, newName string) error {
	if err := validateRepoName(newName); err != nil {
		return err
	}
	r, err := s.ByID(ctx, repoID)
	if err != nil {
		return err
	}
	if r.Name == newName {
		return nil
	}
	owner := s.ownerSlug(ctx, r.OwnerID, r.OwnerType, r.OrgID)
	if existing, err := s.ByOwnerName(ctx, r.OwnerID, newName); err == nil && existing != nil {
		return ErrExists
	}
	tx, err := s.st.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, s.st.Rebind(`INSERT INTO repo_redirects (old_owner, old_name, repo_id) VALUES (?, ?, ?) ON CONFLICT (old_owner, old_name) DO UPDATE SET repo_id = excluded.repo_id`), owner, r.Name, repoID); err != nil {
		// MySQL lacks ON CONFLICT; fall back to a simple insert-ignore.
		if _, err2 := tx.ExecContext(ctx, s.st.Rebind(`INSERT IGNORE INTO repo_redirects (old_owner, old_name, repo_id) VALUES (?, ?, ?)`), owner, r.Name, repoID); err2 != nil {
			return fmt.Errorf("repo: rename redirect: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, s.st.Rebind(`UPDATE repos SET name = ?, updated_at = ? WHERE id = ?`), newName, time.Now().UTC(), repoID); err != nil {
		return fmt.Errorf("repo: rename update: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return s.fs.Move(owner, r.Name, owner, newName)
}

// ResolveRedirect returns the repo id for a former owner/name, if recorded.
func (s *Service) ResolveRedirect(ctx context.Context, owner, name string) (int64, error) {
	q := s.st.Rebind(`SELECT repo_id FROM repo_redirects WHERE old_owner = ? AND old_name = ?`)
	var id int64
	err := s.st.DB.QueryRowContext(ctx, q, owner, name).Scan(&id)
	if err != nil {
		return 0, ErrNotFound
	}
	return id, nil
}
