// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"log/slog"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/infra/fsrepo"
	"github.com/LiStudioorg/gitli/internal/infra/gitcmd"
	"github.com/LiStudioorg/gitli/internal/infra/queue"
	"github.com/LiStudioorg/gitli/internal/service/activity"
	"github.com/LiStudioorg/gitli/internal/service/audit"
	"github.com/LiStudioorg/gitli/internal/service/auth"
	"github.com/LiStudioorg/gitli/internal/service/ci"
	"github.com/LiStudioorg/gitli/internal/service/comment"
	"github.com/LiStudioorg/gitli/internal/service/gist"
	"github.com/LiStudioorg/gitli/internal/service/git"
	"github.com/LiStudioorg/gitli/internal/service/ide"
	importer "github.com/LiStudioorg/gitli/internal/service/import"
	"github.com/LiStudioorg/gitli/internal/service/issue"
	"github.com/LiStudioorg/gitli/internal/service/lfs"
	"github.com/LiStudioorg/gitli/internal/service/metrics"
	"github.com/LiStudioorg/gitli/internal/service/mirror"
	"github.com/LiStudioorg/gitli/internal/service/notification"
	"github.com/LiStudioorg/gitli/internal/service/org"
	pkgstore "github.com/LiStudioorg/gitli/internal/service/package"
	"github.com/LiStudioorg/gitli/internal/service/pat"
	"github.com/LiStudioorg/gitli/internal/service/pkgsvc"
	"github.com/LiStudioorg/gitli/internal/service/pull"
	"github.com/LiStudioorg/gitli/internal/service/rbac"
	"github.com/LiStudioorg/gitli/internal/service/registry"
	"github.com/LiStudioorg/gitli/internal/service/release"
	"github.com/LiStudioorg/gitli/internal/service/repo"
	"github.com/LiStudioorg/gitli/internal/service/search"
	"github.com/LiStudioorg/gitli/internal/service/user"
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
	Webhooks      *webhook.Service
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
	CI       *ci.Service
	IDE      *ide.Service
	PkgSvc   *pkgsvc.Service
	Registry *registry.Service
	Mirrors  *mirror.Service
	Imports  *importer.Service
}

// ExternalBase returns the externally visible HTTP base URL.
func (d *Deps) ExternalBase() string { return d.External.HTTPBase }

// HTTPSecure reports whether the external URL uses https.
func (d *Deps) HTTPSecure() bool { return isHTTPS(d.External.HTTPBase) }
