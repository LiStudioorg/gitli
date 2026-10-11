// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"log/slog"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/infra/fsrepo"
	"github.com/LiStudioorg/gitli/internal/infra/gitcmd"
	"github.com/LiStudioorg/gitli/internal/service/activity"
	"github.com/LiStudioorg/gitli/internal/service/audit"
	"github.com/LiStudioorg/gitli/internal/service/auth"
	"github.com/LiStudioorg/gitli/internal/service/comment"
	"github.com/LiStudioorg/gitli/internal/service/git"
	"github.com/LiStudioorg/gitli/internal/service/issue"
	"github.com/LiStudioorg/gitli/internal/service/notification"
	"github.com/LiStudioorg/gitli/internal/service/org"
	"github.com/LiStudioorg/gitli/internal/service/pat"
	"github.com/LiStudioorg/gitli/internal/service/pull"
	"github.com/LiStudioorg/gitli/internal/service/rbac"
	"github.com/LiStudioorg/gitli/internal/service/repo"
	"github.com/LiStudioorg/gitli/internal/service/user"
	"github.com/LiStudioorg/gitli/internal/service/webhook"
)

// Deps holds every dependency the HTTP transport needs.
type Deps struct {
	Cfg    *config.Config
	DB     *db.DB
	Log    *slog.Logger
	FS     *fsrepo.Layout
	GitCmd *gitcmd.Runner

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
}

// ExternalBase returns the externally visible HTTP base URL.
func (d *Deps) ExternalBase() string { return d.External.HTTPBase }

// HTTPSecure reports whether the external URL uses https.
func (d *Deps) HTTPSecure() bool { return isHTTPS(d.External.HTTPBase) }
