// SPDX-License-Identifier: AGPL-3.0
package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/LiStudioorg/gitli/internal/domain"
)

// ForkParams describes a repository fork request.
type ForkParams struct {
	Source     *domain.Repo
	TargetName string
	OwnerID    int64
	OwnerType  domain.OwnerType
	OrgID      *int64
}

// Fork creates a new repository copying the source's git data.
func (s *Service) Fork(ctx context.Context, p ForkParams) (*domain.Repo, error) {
	if p.TargetName == "" {
		p.TargetName = p.Source.Name
	}
	if p.OwnerType == "" {
		p.OwnerType = domain.OwnerUser
	}
	if err := validateRepoName(p.TargetName); err != nil {
		return nil, err
	}
	existing, err := s.ByOwnerName(ctx, p.OwnerID, p.TargetName)
	if err == nil && existing != nil {
		return nil, ErrExists
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	srcOwner := s.ownerSlug(ctx, p.Source.OwnerID, p.Source.OwnerType, p.Source.OrgID)
	dstOwner := s.ownerSlug(ctx, p.OwnerID, p.OwnerType, p.OrgID)
	srcPath := s.fs.Path(srcOwner, p.Source.Name)
	dstPath, err := s.fs.Ensure(dstOwner, p.TargetName)
	if err != nil {
		return nil, err
	}

	// Copy the bare repository on disk, then record the fork relationship.
	if err := s.git.CloneLocal(ctx, srcPath, dstPath); err != nil {
		_ = s.fs.Remove(dstOwner, p.TargetName)
		return nil, fmt.Errorf("repo: fork clone: %w", err)
	}
	_ = s.fs.InstallHooks(dstPath, dstOwner, p.TargetName, s.gitliBin)

	forkID := p.Source.ID
	q := s.st.Rebind(`INSERT INTO repos (owner_id, owner_type, org_id, name, description, default_branch, is_private, is_fork, fork_id) VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, p.OwnerID, string(p.OwnerType), p.OrgID, p.TargetName,
		p.Source.Description, p.Source.DefaultBranch, p.Source.IsPrivate, forkID)
	if err != nil {
		_ = s.fs.Remove(dstOwner, p.TargetName)
		return nil, fmt.Errorf("repo: insert fork: %w", err)
	}
	id, _ := res.LastInsertId()
	_, _ = s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE repos SET forks_count = forks_count + 1, updated_at = ? WHERE id = ?`), time.Now().UTC(), p.Source.ID)
	return s.ByID(ctx, id)
}
