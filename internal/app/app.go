// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/infra/crypto"
	"github.com/LiStudioorg/gitli/internal/infra/fsrepo"
	"github.com/LiStudioorg/gitli/internal/infra/gitcmd"
	"github.com/LiStudioorg/gitli/internal/infra/secutil"
	"github.com/LiStudioorg/gitli/internal/infra/weakpass"
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
	"github.com/LiStudioorg/gitli/internal/service/store"
	"github.com/LiStudioorg/gitli/internal/service/user"
	"github.com/LiStudioorg/gitli/internal/service/webhook"
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
func New(ctx context.Context, cfg *config.Config) (*App, error) {
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
	if err := rbacSvc.Seed(ctx); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("seed rbac: %w", err)
	}
	repos := repo.New(st, layout, runner)
	if exe, err := os.Executable(); err == nil {
		repos.SetGitliBinary(exe)
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
		Webhooks:      webhook.New(st),
		Notifications: notification.New(st),
		Activities:    activity.New(st),
		Audit:         audit.New(st),
	}
	return &App{Cfg: cfg, DB: database, Log: log, Deps: deps}, nil
}

func enforcePerms(cfg *config.Config) error {
	return secutil.EnforceDirPerms(map[string]os.FileMode{
		cfg.Storage.Repos: 0o750,
	})
}
