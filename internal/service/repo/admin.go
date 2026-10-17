// SPDX-License-Identifier: AGPL-3.0
package repo

import (
	"context"
	"os"
	"path/filepath"

	"github.com/LiStudioorg/gitli/internal/domain"
)

// ListAll returns every repository (used by CLI/admin tooling).
func (s *Service) ListAll(ctx context.Context) ([]*domain.Repo, error) {
	q := s.st.Rebind(`SELECT ` + repoColumns + ` FROM repos ORDER BY owner_id, name`)
	rows, err := s.st.DB.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return s.scanMany(rows)
}

// RecalcSize computes the on-disk size of a repository and stores it in KiB.
func (s *Service) RecalcSize(ctx context.Context, repoID int64) (int64, error) {
	r, err := s.ByID(ctx, repoID)
	if err != nil {
		return 0, err
	}
	owner := s.ownerSlug(ctx, r.OwnerID, r.OwnerType, r.OrgID)
	root := s.fs.Path(owner, r.Name)
	var total int64
	_ = filepath.Walk(root, func(_ string, info os.FileInfo, werr error) error {
		if werr != nil || info == nil || info.IsDir() {
			return nil
		}
		total += info.Size()
		return nil
	})
	kb := total / 1024
	if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE repos SET size_kb = ? WHERE id = ?`), kb, repoID); err != nil {
		return 0, err
	}
	return kb, nil
}

// Count returns the total number of repositories.
func (s *Service) Count(ctx context.Context) (int, error) {
	var n int
	err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT COUNT(*) FROM repos`)).Scan(&n)
	return n, err
}
