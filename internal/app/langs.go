// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/LiStudioorg/gitli/internal/service/repo"
)

// LanguageRefreshKind is the queue job kind for refreshing repo languages.
const LanguageRefreshKind = "refresh-languages"

// ParseLanguageRefreshPayload parses a refresh-languages job payload into a repo id.
func ParseLanguageRefreshPayload(payload string) (int64, error) {
	id, err := strconv.ParseInt(payload, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse repo id %q: %w", payload, err)
	}
	if id <= 0 {
		return 0, fmt.Errorf("invalid repo id %d", id)
	}
	return id, nil
}

// RefreshLanguages recomputes and persists the language breakdown for a repo.
func (a *App) RefreshLanguages(ctx context.Context, repoID int64) error {
	rp, err := a.Deps.Repos.ByID(ctx, repoID)
	if err != nil {
		return fmt.Errorf("load repo %d: %w", repoID, err)
	}
	owner := a.RepoOwnerSlug(ctx, rp)
	path := a.Deps.FS.Path(owner, rp.Name)
	detected, err := a.Deps.Git.DetectLanguages(ctx, path, rp.DefaultBranch)
	if err != nil {
		// A push to a non-default branch (or a fresh repo) can land before the
		// default branch exists; treat that as "nothing to detect" rather than
		// an error so the queue stays clean.
		if strings.Contains(err.Error(), "Not a valid object name") {
			return nil
		}
		return fmt.Errorf("detect languages for repo %d: %w", repoID, err)
	}
	stats := make([]repo.LanguageStat, 0, len(detected))
	for _, d := range detected {
		stats = append(stats, repo.LanguageStat{Language: d.Language, Bytes: d.Bytes, Percentage: d.Percentage})
	}
	if err := a.Deps.Repos.ReplaceLanguages(ctx, repoID, stats); err != nil {
		return fmt.Errorf("store languages for repo %d: %w", repoID, err)
	}
	return nil
}
