// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"fmt"
	"os"

	"github.com/LiStudioorg/gitli/internal/service/git"
)

// RunHook executes a server-side git hook. It reads the hook's stdin (the
// ref-update lines) and enforces repository protections.
func (a *App) RunHook(ctx context.Context, name, owner, repoName string) error {
	switch name {
	case "pre-receive":
		return a.runPreReceive(ctx, owner, repoName)
	case "post-receive":
		return a.runPostReceive(ctx, owner, repoName)
	case "update":
		return nil
	default:
		return fmt.Errorf("unknown hook %q", name)
	}
}

func (a *App) runPreReceive(ctx context.Context, owner, repoName string) error {
	updates, err := git.ParsePreReceive(os.Stdin)
	if err != nil {
		return err
	}
	if len(updates) == 0 {
		return nil
	}
	ownerUser, err := a.Deps.Users.ByUsername(ctx, owner)
	if err != nil {
		return fmt.Errorf("hook: owner %q not found", owner)
	}
	rp, err := a.Deps.Repos.ByOwnerName(ctx, ownerUser.ID, repoName)
	if err != nil {
		return fmt.Errorf("hook: repo %q not found", repoName)
	}
	checker := &protectionChecker{a: a}
	isAdmin := os.Getenv("GITLI_HOOK_IS_ADMIN") == "1"
	if err := git.EnforceProtections(ctx, checker, rp.ID, updates, isAdmin); err != nil {
		fmt.Fprintf(os.Stderr, "gitli: 推送被拒绝：%v\n", err)
		return err
	}
	return nil
}

// protectionChecker adapts the repo service to git.ProtectionChecker.
type protectionChecker struct{ a *App }

func (p *protectionChecker) BranchProtections(ctx context.Context, repoID int64) ([]git.ProtectionRule, error) {
	rows, err := p.a.Deps.Repos.ListBranchProtections(ctx, repoID)
	if err != nil {
		return nil, err
	}
	out := make([]git.ProtectionRule, 0, len(rows))
	for _, r := range rows {
		out = append(out, git.ProtectionRule{
			Pattern:       r.Pattern,
			RequireReview: r.RequireReview,
			RequireSigned: r.RequireSigned,
		})
	}
	return out, nil
}

func (p *protectionChecker) TagProtections(ctx context.Context, repoID int64) ([]string, error) {
	return p.a.Deps.Repos.ListTagProtectionPatterns(ctx, repoID)
}

// runPostReceive creates runner jobs for workflow files on push.
func (a *App) runPostReceive(ctx context.Context, owner, repoName string) error {
	updates, err := git.ParsePreReceive(os.Stdin)
	if err != nil {
		return nil // post-receive failures must not block the push
	}
	ownerUser, err := a.Deps.Users.ByUsername(ctx, owner)
	if err != nil {
		return nil
	}
	rp, err := a.Deps.Repos.ByOwnerName(ctx, ownerUser.ID, repoName)
	if err != nil {
		return nil
	}
	for _, u := range updates {
		if u.IsDelete() {
			continue
		}
		_, _ = a.CreateWorkflowJobs(ctx, rp.ID, u.New, u.Ref)
	}
	return nil
}
