// SPDX-License-Identifier: AGPL-3.0
package repo

import (
	"context"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/org"
)

// PermissionFor resolves the effective permission a user has on a repo.
// Resolution order: admin > owner > org team > collaborator.
func (s *Service) PermissionFor(ctx context.Context, r *domain.Repo, userID int64, isAdmin bool, orgs *org.Service) (domain.Perm, error) {
	if userID == 0 {
		return domain.PermNone, nil
	}
	if isAdmin {
		return domain.PermAdmin, nil
	}
	if r.OwnerType == domain.OwnerUser && r.OwnerID == userID {
		return domain.PermAdmin, nil
	}
	if r.OwnerType == domain.OwnerOrg && orgs != nil && r.OrgID != nil {
		perm, err := orgs.TeamPermission(ctx, *r.OrgID, userID)
		if err != nil {
			return domain.PermNone, err
		}
		if perm > domain.PermNone {
			return perm, nil
		}
	}
	return s.CollaboratorPermission(ctx, r.ID, userID)
}
