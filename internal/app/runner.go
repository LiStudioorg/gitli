// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"strings"
	"time"

	runnersvc "github.com/LiStudioorg/gitli/internal/service/runner"
)

// runnerOfflineSweeper periodically marks stale runners offline.
func (a *App) runnerOfflineSweeper(ctx context.Context) {
	interval := time.Duration(a.Cfg.Runner.OfflineSweepIntervalSec) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second
	}
	timeout := time.Duration(a.Cfg.Runner.HeartbeatTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n, err := a.Deps.Runners.SweepOffline(ctx, timeout); err == nil && n > 0 {
				a.Log.Info("runners marked offline", "count", n)
			}
		}
	}
}

// CreateWorkflowJobs scans .gitli/workflows/ at the given commit and enqueues a
// runner job for each workflow file found. Returns the number of jobs created.
func (a *App) CreateWorkflowJobs(ctx context.Context, repoID int64, commitSHA, ref string) (int, error) {
	if a.Deps == nil || a.Deps.Runners == nil || !a.Cfg.Runner.Enabled {
		return 0, nil
	}
	repo, err := a.Deps.Repos.ByID(ctx, repoID)
	if err != nil {
		return 0, err
	}
	owner := a.RepoOwnerSlug(ctx, repo)
	repoPath := a.Deps.FS.Path(owner, repo.Name)

	// List workflow files under .gitli/workflows at the commit.
	out, err := a.Deps.GitCmd.Run(ctx, repoPath, "ls-tree", "-r", "--name-only", commitSHA, "--", ".gitli/workflows")
	if err != nil {
		return 0, nil // no workflows or missing dir: nothing to do
	}
	count := 0
	for _, line := range strings.Split(out, "\n") {
		name := strings.TrimSpace(line)
		if name == "" || (!strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml")) {
			continue
		}
		content, err := a.Deps.GitCmd.Run(ctx, repoPath, "show", commitSHA+":"+name)
		if err != nil {
			continue
		}
		if _, err := a.Deps.Runners.CreateJob(ctx, runnersvc.CreateJobParams{
			RepoID:          repoID,
			CommitSHA:       commitSHA,
			Ref:             ref,
			WorkflowPath:    name,
			WorkflowContent: content,
		}); err != nil {
			a.Log.Warn("create runner job", "workflow", name, "error", err)
			continue
		}
		count++
	}
	if count > 0 {
		a.Log.Info("workflow jobs enqueued", "repo", repo.Name, "count", count, "ref", ref)
	}
	return count, nil
}
