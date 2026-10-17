// SPDX-License-Identifier: AGPL-3.0
package repo

import (
	"context"
	"time"
)

// SoftDelete marks a repository as deleted without removing its data.
func (s *Service) SoftDelete(ctx context.Context, id int64) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE repos SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`),
		time.Now().UTC(), time.Now().UTC(), id)
	return err
}

// Restore clears the deleted marker.
func (s *Service) Restore(ctx context.Context, id int64) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE repos SET deleted_at = NULL, updated_at = ? WHERE id = ?`), time.Now().UTC(), id)
	return err
}

// IsDeleted reports whether a repository is soft-deleted.
func (s *Service) IsDeleted(ctx context.Context, id int64) (bool, error) {
	var deleted *time.Time
	err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT deleted_at FROM repos WHERE id = ?`), id).Scan(&deleted)
	if err != nil {
		return false, ErrNotFound
	}
	return deleted != nil, nil
}

// Purge permanently removes a repository (data + on-disk). Irreversible.
func (s *Service) Purge(ctx context.Context, id int64) error {
	r, err := s.ByID(ctx, id)
	if err != nil {
		return err
	}
	owner := s.ownerSlug(ctx, r.OwnerID, r.OwnerType, r.OrgID)
	if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM repos WHERE id = ?`), id); err != nil {
		return err
	}
	return s.fs.Remove(owner, r.Name)
}

// ListDeletedByOwner returns soft-deleted repositories for an owner.
func (s *Service) ListDeletedByOwner(ctx context.Context, ownerID int64) ([]*RepoRef, error) {
	q := s.st.Rebind(`SELECT id, name, deleted_at FROM repos WHERE owner_id = ? AND deleted_at IS NOT NULL ORDER BY deleted_at DESC`)
	rows, err := s.st.DB.QueryContext(ctx, q, ownerID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*RepoRef
	for rows.Next() {
		var r RepoRef
		if err := rows.Scan(&r.ID, &r.Name, &r.DeletedAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

// PurgeExpired permanently deletes repos soft-deleted longer than retention days.
func (s *Service) PurgeExpired(ctx context.Context, retentionDays int) (int, error) {
	if retentionDays <= 0 {
		retentionDays = 30
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	q := s.st.Rebind(`SELECT id FROM repos WHERE deleted_at IS NOT NULL AND deleted_at < ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, cutoff)
	if err != nil {
		return 0, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	_ = rows.Close()
	n := 0
	for _, id := range ids {
		if err := s.Purge(ctx, id); err == nil {
			n++
		}
	}
	return n, nil
}

// RepoRef is a lightweight reference used by list endpoints.
type RepoRef struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}
