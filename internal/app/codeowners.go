// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"

	"github.com/LiStudioorg/gitli/internal/service/codeowners"
)

// codeownersPaths lists the candidate locations of a CODEOWNERS file.
var codeownersPaths = []string{"CODEOWNERS", ".github/CODEOWNERS", "docs/CODEOWNERS"}

// LoadCodeOwners reads and parses the repository's CODEOWNERS at ref.
func (a *App) LoadCodeOwners(ctx context.Context, repoPath, ref string) ([]codeowners.Rule, error) {
	for _, p := range codeownersPaths {
		if data, ok := a.Deps.Git.ShowFile(ctx, repoPath, ref, p); ok {
			return codeowners.Parse(data)
		}
	}
	return nil, nil
}

// RequestCodeOwnerReviews resolves owners for the files changed between base
// and head and creates review requests for the matching users. It returns the
// set of requested usernames.
func (a *App) RequestCodeOwnerReviews(ctx context.Context, repoID int64, repoPath, base, head string, pullID int64) ([]string, error) {
	rules, err := a.LoadCodeOwners(ctx, repoPath, base)
	if err != nil || len(rules) == 0 {
		return nil, err
	}
	files, err := a.Deps.Git.ChangedFiles(ctx, repoPath, base, head)
	if err != nil || len(files) == 0 {
		return nil, err
	}
	seen := map[string]bool{}
	var requested []string
	for _, f := range files {
		for _, owner := range codeowners.OwnersFor(rules, f) {
			if seen[owner] {
				continue
			}
			seen[owner] = true
			u, err := a.Deps.Users.ByUsername(ctx, owner)
			if err != nil {
				continue
			}
			if _, err := a.Deps.Pulls.AddReview(ctx, pullID, u.ID, "requested", "CODEOWNERS"); err != nil {
				continue
			}
			a.Notify(ctx, u.ID, "pull", pullID, "你被请求审查（CODEOWNERS）", "")
			requested = append(requested, owner)
		}
	}
	return requested, nil
}
