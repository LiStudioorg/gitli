// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"

	"github.com/LiStudioorg/gitli/internal/service/license"
)

// DetectLicense reads the repository's LICENSE file at ref and stores the
// detected SPDX id. Safe to call repeatedly.
func (a *App) DetectLicense(ctx context.Context, repoID int64, repoPath, ref string) string {
	if len(ref) == 0 {
		return ""
	}
	for _, p := range license.CandidatePaths {
		data, ok := a.Deps.Git.ShowFile(ctx, repoPath, ref, p)
		if !ok || len(data) < 16 {
			continue
		}
		if id, ok := license.Detect(data); ok {
			_ = a.Deps.Repos.SetLicense(ctx, repoID, id)
			return id
		}
	}
	return ""
}
