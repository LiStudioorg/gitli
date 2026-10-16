// SPDX-License-Identifier: AGPL-3.0
//
// Package app wires together configuration, services and transports and
// manages the application lifecycle.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/infra/crypto"
	"github.com/LiStudioorg/gitli/internal/infra/fsrepo"
	"github.com/LiStudioorg/gitli/internal/infra/gitcmd"
	"github.com/LiStudioorg/gitli/internal/infra/queue"
	"github.com/LiStudioorg/gitli/internal/infra/secutil"
	"github.com/LiStudioorg/gitli/internal/infra/weakpass"
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
	"github.com/LiStudioorg/gitli/internal/service/pathperm"
	"github.com/LiStudioorg/gitli/internal/service/pkgsvc"
	"github.com/LiStudioorg/gitli/internal/service/pull"
	"github.com/LiStudioorg/gitli/internal/service/rbac"
	"github.com/LiStudioorg/gitli/internal/service/reaction"
	"github.com/LiStudioorg/gitli/internal/service/registry"
	"github.com/LiStudioorg/gitli/internal/service/release"
	"github.com/LiStudioorg/gitli/internal/service/repo"
	runnersvc "github.com/LiStudioorg/gitli/internal/service/runner"
	"github.com/LiStudioorg/gitli/internal/service/search"
	"github.com/LiStudioorg/gitli/internal/service/signature"
	"github.com/LiStudioorg/gitli/internal/service/store"
	"github.com/LiStudioorg/gitli/internal/service/subscription"
	"github.com/LiStudioorg/gitli/internal/service/user"
	"github.com/LiStudioorg/gitli/internal/service/webhook"
	"github.com/LiStudioorg/gitli/internal/service/wiki"
	httptransport "github.com/LiStudioorg/gitli/internal/transport/http"
)

// App is the top-level application runtime.
type App struct {
	Cfg  *config.Config
	DB   *db.DB
	Log  *slog.Logger
	Deps *httptransport.Deps
}

// New builds an App from configuration, opening the database and wiring services.
func New(ctx context.Context, cfg *config.Config, configPath ...string) (*App, error) {
	log := NewLogger(cfg.Log)

	if err := ensureDirs(cfg); err != nil {
		return nil, err
	}
	if err := enforcePerms(cfg); err != nil {
		return nil, err
	}

	database, err := db.Open(&cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	var cipher *crypto.Cipher
	if cfg.Security.Crypto.KeyHex != "" {
		cipher, err = crypto.NewFromHex(cfg.Security.Crypto.KeyHex)
		if err != nil {
			_ = database.Close()
			return nil, fmt.Errorf("init cipher: %w", err)
		}
	}

	runner, err := gitcmd.Configure(gitcmd.Options{
		Bin:            cfg.Git.Bin,
		TimeoutSeconds: cfg.Git.TimeoutSeconds,
		CheckVersion:   cfg.Git.CheckVersion,
		MinVersion:     cfg.Git.MinVersion,
		Log:            log,
	})
	if err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("init git: %w", err)
	}

	st := store.New(database)
	layout := fsrepo.New(cfg.Storage.Repos)

	users := user.New(st)
	authSvc := auth.New(st, auth.Options{
		Users:             users,
		Cipher:            cipher,
		Log:               log,
		MaxAttempt:        cfg.Security.Login.MaxAttempts,
		Lockout:           cfg.Security.Login.LockoutDuration(),
		MinPasswordLength: cfg.Security.Password.MinLength,
		BindIP:            cfg.Security.Session.BindIP,
		BindAgent:         cfg.Security.Session.BindAgent,
	})
	authSvc.SetPepper(cfg.Security.Password.Pepper)
	if cfg.Security.Password.HIBP {
		authSvc.SetHIBP(weakpass.NewHIBP())
	}
	tokens := pat.New(st)
	authSvc.SetTokenResolver(tokens)

	orgs := org.New(st)
	rbacSvc := rbac.New(st)
	repos := repo.New(st, layout, runner)
	if exe, err := os.Executable(); err == nil {
		repos.SetGitliBinary(exe)
	}
	if len(configPath) > 0 && configPath[0] != "" {
		if abs, err := filepath.Abs(configPath[0]); err == nil {
			repos.SetConfigPath(abs)
		}
	}

	deps := &httptransport.Deps{
		Cfg:           cfg,
		DB:            database,
		Log:           log,
		FS:            layout,
		GitCmd:        runner,
		External:      config.ResolveExternalURLs(cfg),
		Auth:          authSvc,
		Users:         users,
		Tokens:        tokens,
		Orgs:          orgs,
		RBAC:          rbacSvc,
		Repos:         repos,
		Git:           git.New(runner),
		Issues:        issue.New(st),
		Pulls:         pull.New(st),
		Comments:      comment.New(st),
		Reactions:     reaction.New(st),
		Subscriptions: subscription.New(st),
		Webhooks:      webhook.New(st),
		Notifications: notification.New(st),
		Activities:    activity.New(st),
		Audit:         audit.New(st),
		Wiki:          wiki.New(st),
		Gists:         gist.New(st),
		Releases:      release.New(st),
		LFS:           lfs.New(st, cfg.Storage.LFS),
		Search:        search.New(runner),
		Blobs:         pkgstore.NewBlobStore(st, cfg.Storage.Packages),
		PackagesRoot:  cfg.Storage.Packages,
		Metrics:       metrics.New(),
		OIDC:          oidcProvider(cfg, log),
		CI:            ci.New(st, log, cfg.CI.MaxConcurrency),
		IDE:           ide.New(runner),
		PkgSvc:        pkgsvc.New(st, pkgstore.NewBlobStore(st, cfg.Storage.Packages)),
		Registry:      registry.New(st, pkgstore.NewBlobStore(st, cfg.Storage.Packages), log),
		Mirrors:       mirror.New(st, runner),
		Imports:       importer.New(st, runner),
		PathPerms:     pathperm.New(st),
		Signatures:    signature.New(st, runner),
		Runners:       runnersvc.New(st, cfg.Runner.LogMaxMB),
	}
	a := &App{Cfg: cfg, DB: database, Log: log, Deps: deps}
	deps.Dispatch = func(ctx context.Context, action string, repoID, actorID int64, payload map[string]any) {
		a.DispatchRepoEvent(ctx, Event{Action: action, RepoID: repoID, ActorID: actorID, Payload: payload})
	}
	deps.Notify = a.Notify
	q := queue.New(2, 256, log)
	q.Register(LanguageRefreshKind, func(ctx context.Context, payload string) error {
		repoID, err := ParseLanguageRefreshPayload(payload)
		if err != nil {
			return err
		}
		return a.RefreshLanguages(ctx, repoID)
	})
	q.Register("webhook-deliver", func(ctx context.Context, payload string) error {
		return a.RunWebhookDeliver(ctx, payload)
	})
	deps.Queue = q
	return a, nil
}

func enforcePerms(cfg *config.Config) error {
	return secutil.EnforceDirPerms(map[string]os.FileMode{
		cfg.Storage.Repos: 0o750,
	})
}
