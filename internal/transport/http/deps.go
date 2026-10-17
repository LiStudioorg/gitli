// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"context"
	"log/slog"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/infra/fsrepo"
	"github.com/LiStudioorg/gitli/internal/infra/gitcmd"
	"github.com/LiStudioorg/gitli/internal/infra/oidc"
	"github.com/LiStudioorg/gitli/internal/infra/queue"
	"github.com/LiStudioorg/gitli/internal/service/activity"
	"github.com/LiStudioorg/gitli/internal/service/audit"
	"github.com/LiStudioorg/gitli/internal/service/auth"
	"github.com/LiStudioorg/gitli/internal/service/block"
	"github.com/LiStudioorg/gitli/internal/service/ci"
	"github.com/LiStudioorg/gitli/internal/service/cisecret"
	"github.com/LiStudioorg/gitli/internal/service/comment"
	"github.com/LiStudioorg/gitli/internal/service/deploykey"
	"github.com/LiStudioorg/gitli/internal/service/gist"
	"github.com/LiStudioorg/gitli/internal/service/git"
	"github.com/LiStudioorg/gitli/internal/service/ide"
	importer "github.com/LiStudioorg/gitli/internal/service/import"
	"github.com/LiStudioorg/gitli/internal/service/issue"
	"github.com/LiStudioorg/gitli/internal/service/lfs"
	"github.com/LiStudioorg/gitli/internal/service/metrics"
	"github.com/LiStudioorg/gitli/internal/service/mirror"
	"github.com/LiStudioorg/gitli/internal/service/notification"
	"github.com/LiStudioorg/gitli/internal/service/notifpref"
	"github.com/LiStudioorg/gitli/internal/service/org"
	"github.com/LiStudioorg/gitli/internal/service/org2fa"
	pkgstore "github.com/LiStudioorg/gitli/internal/service/package"
	"github.com/LiStudioorg/gitli/internal/service/pat"
	"github.com/LiStudioorg/gitli/internal/service/pathperm"
	"github.com/LiStudioorg/gitli/internal/service/pkgsvc"
	"github.com/LiStudioorg/gitli/internal/service/pull"
	"github.com/LiStudioorg/gitli/internal/service/rbac"
	"github.com/LiStudioorg/gitli/internal/service/reaction"
	"github.com/LiStudioorg/gitli/internal/service/registry"
	"github.com/LiStudioorg/gitli/internal/service/release"
	"github.com/LiStudioorg/gitli/internal/service/repo"
	"github.com/LiStudioorg/gitli/internal/service/runner"
	"github.com/LiStudioorg/gitli/internal/service/search"
	"github.com/LiStudioorg/gitli/internal/service/signature"
	"github.com/LiStudioorg/gitli/internal/service/subscription"
	"github.com/LiStudioorg/gitli/internal/service/template"
	"github.com/LiStudioorg/gitli/internal/service/topic"
	"github.com/LiStudioorg/gitli/internal/service/user"
	"github.com/LiStudioorg/gitli/internal/service/userwebhook"
	"github.com/LiStudioorg/gitli/internal/service/webhook"
	"github.com/LiStudioorg/gitli/internal/service/wiki"
)

// Deps holds every dependency the HTTP transport needs.
type Deps struct {
	Cfg    *config.Config
	DB     *db.DB
	Log    *slog.Logger
	FS     *fsrepo.Layout
	GitCmd *gitcmd.Runner
	Queue  *queue.Queue

	External config.ExternalURLs

	Auth          *auth.Service
	Users         *user.Service
	Tokens        *pat.Service
	Orgs          *org.Service
	RBAC          *rbac.Service
	Repos         *repo.Service
	Git           *git.Service
	Issues        *issue.Service
	Pulls         *pull.Service
	Comments      *comment.Service
	Reactions     *reaction.Service
	Subscriptions *subscription.Service
	Webhooks      *webhook.Service
	UserWebhooks  *userwebhook.Service
	Notifications *notification.Service
	Activities    *activity.Service
	Audit         *audit.Service
	Wiki          *wiki.Service
	Gists         *gist.Service
	Releases      *release.Service

	LFS          *lfs.Service
	Search       *search.Service
	Blobs        *pkgstore.BlobStore
	PackagesRoot string

	Metrics  *metrics.Registry
	OIDC     *oidc.Provider
	CI       *ci.Service
	IDE      *ide.Service
	PkgSvc   *pkgsvc.Service
	Registry *registry.Service
	Mirrors  *mirror.Service
	Imports  *importer.Service

	PathPerms  *pathperm.Service
	Signatures *signature.Service
	Topics     *topic.Service
	CISecrets  *cisecret.Service

	Blocks     *block.Service
	NotifPrefs *notifpref.Service
	Org2FA     *org2fa.Service

	Runners    *runner.Service
	DeployKeys *deploykey.Service

	// Dispatch delivers a repository event to webhooks. Injected by the app
	// layer to avoid an import cycle.
	Dispatch func(ctx context.Context, action string, repoID, actorID int64, payload map[string]any)
	Notify   func(ctx context.Context, userID int64, subjectType string, subjectID int64, title, url string)

	// RequestCodeOwnerReviews creates review requests for CODEOWNERS matches.
	RequestCodeOwnerReviews func(ctx context.Context, repoID int64, repoPath, base, head string, pullID int64) []string

	// ListIssueTemplates returns parsed issue templates for a repo.
	ListIssueTemplates func(ctx context.Context, repoPath, ref string) []*template.IssueTemplate
	// PullRequestTemplate returns the repo's PR template body.
	PullRequestTemplate func(ctx context.Context, repoPath, ref string) string
}

// ExternalBase returns the externally visible HTTP base URL.
func (d *Deps) ExternalBase() string { return d.External.HTTPBase }

// HTTPSecure reports whether the external URL uses https.
func (d *Deps) HTTPSecure() bool { return isHTTPS(d.External.HTTPBase) }
