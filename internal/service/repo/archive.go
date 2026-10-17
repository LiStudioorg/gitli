// SPDX-License-Identifier: AGPL-3.0
package repo

import (
	"context"
	"time"

	"github.com/LiStudioorg/gitli/internal/domain"
)

// Archive marks a repository read-only.
func (s *Service) Archive(ctx context.Context, repoID int64, archived bool) error {
	var archivedAt any
	if archived {
		archivedAt = time.Now().UTC()
	}
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE repos SET archived = ?, archived_at = ?, updated_at = ? WHERE id = ?`),
		archived, archivedAt, time.Now().UTC(), repoID)
	return err
}

// IsArchived reports whether a repository (or its owning org) is archived.
func (s *Service) IsArchived(ctx context.Context, r *domain.Repo) bool {
	if r.Archived {
		return true
	}
	if r.OrgID != nil {
		var archived bool
		if err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT archived FROM orgs WHERE id = ?`), *r.OrgID).Scan(&archived); err == nil {
			return archived
		}
	}
	return false
}
