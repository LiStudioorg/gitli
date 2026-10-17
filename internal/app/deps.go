// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"

	"github.com/LiStudioorg/gitli/internal/service/deps"
)

// RefreshDependencies parses dependency manifests at ref and stores the graph.
func (a *App) RefreshDependencies(ctx context.Context, repoID int64, repoPath, ref string) (int, error) {
	if a.Deps.Deps == nil || ref == "" {
		return 0, nil
	}
	read := func(name string) []byte {
		if data, ok := a.Deps.Git.ShowFile(ctx, repoPath, ref, name); ok {
			return data
		}
		return nil
	}
	list := deps.ParseManifests(
		read("go.mod"),
		read("package.json"),
		read("pom.xml"),
		read("requirements.txt"),
	)
	if err := a.Deps.Deps.Replace(ctx, repoID, list); err != nil {
		return 0, err
	}
	return len(list), nil
}
