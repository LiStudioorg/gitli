// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/audit"
	"github.com/LiStudioorg/gitli/internal/service/git"
)

// emptySHA is the all-zero object id git uses for deletions.
const emptySHA = "0000000000000000000000000000000000000000"

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
	if a.Deps.Repos.IsArchived(ctx, rp) {
		fmt.Fprintln(os.Stderr, "gitli: 仓库已归档，只读，禁止推送")
		a.auditPushEvent(ctx, rp.ID, "git.push", owner, repoName, updates, audit.ResultDenied, "repository archived")
		return fmt.Errorf("repository archived")
	}
	repoPath := a.Deps.FS.Path(owner, repoName)
	if err := git.EnforceProtections(ctx, checker, rp.ID, updates, isAdmin, repoPath, a.Deps.Signatures); err != nil {
		fmt.Fprintf(os.Stderr, "gitli: 推送被拒绝：%v\n", err)
		a.auditPushEvent(ctx, rp.ID, "git.protected_branch.push.failure", owner, repoName, updates, audit.ResultDenied, err.Error())
		return err
	}
	a.auditPushEvent(ctx, rp.ID, "git.push", owner, repoName, updates, audit.ResultSuccess, "")
	return nil
}

// auditPushEvent records git push activity for the repository's owner.
func (a *App) auditPushEvent(ctx context.Context, repoID int64, event, owner, repoName string, updates []git.RefUpdate, result, errMsg string) {
	if a.Deps.Audit == nil {
		return
	}
	refs := make([]string, 0, len(updates))
	for _, u := range updates {
		refs = append(refs, u.Ref)
	}
	actorName := os.Getenv("GITLI_HOOK_ACTOR")
	tid := repoID
	a.Deps.Audit.Log(audit.Entry{
		ActorType:    "user",
		Event:        event,
		Resource:     "git",
		Action:       event,
		TargetType:   "repo",
		TargetID:     &tid,
		TargetName:   owner + "/" + repoName,
		Result:       result,
		ErrorMessage: errMsg,
		Metadata:     `{"refs":"` + strings.Join(refs, ",") + `","actor":"` + actorName + `"}`,
	})
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
	repoPath := a.Deps.FS.Path(owner, repoName)
	a.recordPushActivity(ctx, rp, owner, repoName, repoPath, updates)
	for _, u := range updates {
		if u.IsDelete() {
			continue
		}
		_, _ = a.CreateWorkflowJobs(ctx, rp.ID, u.New, u.Ref)
		if _, err := a.RefreshDependencies(ctx, rp.ID, repoPath, u.New); err != nil {
			a.Log.Debug("refresh dependencies failed", "error", err)
		}
	}
	return nil
}

// recordPushActivity writes one timeline entry per pushed branch or tag.
func (a *App) recordPushActivity(ctx context.Context, rp *domain.Repo, owner, repoName, repoPath string, updates []git.RefUpdate) {
	if a.Deps == nil || a.Deps.Activities == nil {
		return
	}
	actorName := os.Getenv("GITLI_HOOK_ACTOR")
	actorID := int64(0)
	if u, err := a.Deps.Users.ByUsername(ctx, actorName); err == nil {
		actorID = u.ID
	}
	repoID := rp.ID
	fullName := owner + "/" + repoName
	for _, u := range updates {
		if u.IsDelete() || u.New == emptySHA {
			continue
		}
		if strings.HasPrefix(u.Ref, "refs/tags/") {
			if actorID == 0 {
				continue
			}
			_ = a.Deps.Activities.Record(ctx, &domain.Activity{
				ActorID:    actorID,
				Verb:       "published_tag",
				ObjectType: "tag",
				ObjectID:   repoID,
				RepoID:     &repoID,
				RepoName:   fullName,
				Summary:    strings.TrimPrefix(u.Ref, "refs/tags/"),
			})
			continue
		}
		if actorID == 0 {
			continue
		}
		summary := u.New[:8]
		count := 1
		if u.Old != "" && u.Old != emptySHA {
			if out, err := a.Deps.GitCmd.Run(ctx, repoPath, "rev-list", "--count", u.Old+".."+u.New); err == nil {
				if n, err := strconv.Atoi(strings.TrimSpace(out)); err == nil && n > 0 {
					count = n
				}
			}
		}
		if out, err := a.Deps.GitCmd.Run(ctx, repoPath, "log", "-1", "--format=%s", u.New); err == nil {
			if msg := strings.TrimSpace(out); msg != "" {
				summary += " " + truncateSummary(msg, 80)
			}
		}
		if count > 1 {
			summary += fmt.Sprintf(" (%d commits)", count)
		}
		_ = a.Deps.Activities.Record(ctx, &domain.Activity{
			ActorID:    actorID,
			Verb:       "pushed",
			ObjectType: "branch",
			ObjectID:   repoID,
			RepoID:     &repoID,
			RepoName:   fullName,
			Summary:    summary,
		})
	}
}

func truncateSummary(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
