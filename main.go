// SPDX-License-Identifier: AGPL-3.0
// Command gitli is a self-hosted, lightweight Git service platform.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LiStudioorg/gitli/internal/app"
	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/infra/backup"
	"github.com/LiStudioorg/gitli/internal/version"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}
	cmd := args[0]
	rest := args[1:]

	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	configPath := fs.String("config", "config.toml", "path to config.toml")

	switch cmd {
	case "help", "-h", "--help":
		usage()
		return nil
	case "version":
		info := version.Get()
		fmt.Printf("gitli %s (commit %s, built %s)\n", info.Version, info.Commit, info.Date)
		return nil
	case "hook":
		return runHookCommand(rest)
	case "backup":
		return runBackupCommand(rest)
	case "restore":
		return runRestoreCommand(rest)
	case "mirror":
		return runMirror(rest)
	case "import":
		return runImport(rest)
	case "runner":
		return runRunner(rest)
	case "search-index":
		return runSearchIndex(rest)
	case "admin":
		return runAdmin(rest)
	}

	if err := fs.Parse(rest); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	if err := refuseRoot(cfg); err != nil {
		return err
	}

	ctx := context.Background()

	switch cmd {
	case "serve":
		return runServe(ctx, cfg, *configPath)
	case "migrate":
		return runMigrate(ctx, cfg, *configPath)
	default:
		usage()
		return fmt.Errorf("unknown subcommand %q", cmd)
	}
}

// runBackupCommand creates a streaming backup archive. Usage:
//
//	gitli backup --out <file> [--source <dir>] [--encrypt] --config path
func runBackupCommand(args []string) error {
	fs := flag.NewFlagSet("backup", flag.ContinueOnError)
	configPath := fs.String("config", "config.toml", "path to config.toml")
	out := fs.String("out", "", "output archive file (.tar.gz or .tar.gz.enc)")
	source := fs.String("source", "", "source directory (default: data dir)")
	encrypt := fs.Bool("encrypt", false, "encrypt the archive (key from GITLI_BACKUP_KEY)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return fmt.Errorf("backup: --out is required")
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	if *source == "" {
		*source = dataDir(cfg)
	}
	opts := backup.Options{SourceDir: *source, OutputFile: *out, Encrypt: *encrypt}
	if *encrypt {
		key, err := backupKey()
		if err != nil {
			return err
		}
		opts.Key = key
	}
	if err := backup.Create(context.Background(), opts); err != nil {
		return err
	}
	fmt.Printf("backup written to %s\n", *out)
	return nil
}

// runRestoreCommand restores a backup archive. Usage:
//
//	gitli restore --in <file> [--dest <dir>] [--config path]
func runRestoreCommand(args []string) error {
	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	configPath := fs.String("config", "config.toml", "path to config.toml")
	in := fs.String("in", "", "input archive file")
	dest := fs.String("dest", "", "destination directory (default: data dir)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *in == "" {
		return fmt.Errorf("restore: --in is required")
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	if *dest == "" {
		*dest = dataDir(cfg)
	}
	key, _ := backupKey()
	if err := backup.Restore(context.Background(), *in, *dest, key); err != nil {
		return err
	}
	fmt.Printf("restored to %s\n", *dest)
	return nil
}

// dataDir derives the data directory from the SQLite DSN or storage paths.
func dataDir(cfg *config.Config) string {
	if cfg.Database.Driver == "sqlite" && cfg.Database.DSN != "" {
		if i := strings.LastIndex(cfg.Database.DSN, "/"); i > 0 {
			return cfg.Database.DSN[:i]
		}
	}
	if cfg.Storage.Repos != "" {
		return filepath.Dir(cfg.Storage.Repos)
	}
	return "./data"
}

// backupKey reads the backup encryption key from the configured environment
// variable (default GITLI_BACKUP_KEY) as hex or raw bytes.
func backupKey() ([]byte, error) {
	env := os.Getenv("GITLI_BACKUP_KEY_ENV")
	if env == "" {
		env = "GITLI_BACKUP_KEY"
	}
	raw := os.Getenv(env)
	if raw == "" {
		return nil, fmt.Errorf("backup: encryption key not set (env %s)", env)
	}
	if decoded, err := hex.DecodeString(raw); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if len(raw) == 32 {
		return []byte(raw), nil
	}
	sum := sha256.Sum256([]byte(raw))
	return sum[:], nil
}

func runServe(ctx context.Context, cfg *config.Config, configPath string) error {
	a, err := app.New(ctx, cfg, configPath)
	if err != nil {
		return err
	}
	defer func() { _ = a.Close() }()
	return a.Serve(ctx)
}

func runMigrate(ctx context.Context, cfg *config.Config, configPath string) error {
	a, err := app.New(ctx, cfg, configPath)
	if err != nil {
		return err
	}
	defer func() { _ = a.Close() }()
	return a.Migrate(ctx)
}

// runMirror synchronizes repositories that are due for a mirror update. With
// --once it runs a single pass; otherwise it loops every --interval seconds.
func runMirror(args []string) error {
	fs := flag.NewFlagSet("mirror", flag.ContinueOnError)
	configPath := fs.String("config", "config.toml", "path to config.toml")
	once := fs.Bool("once", false, "run one synchronization pass then exit")
	interval := fs.Int("interval", 300, "seconds between passes when not --once")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx := context.Background()
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	a, err := app.New(ctx, cfg, *configPath)
	if err != nil {
		return err
	}
	defer func() { _ = a.Close() }()

	runOnce := func() {
		mirrors, err := a.Deps.Mirrors.ListAll(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, "mirror: list:", err)
			return
		}
		now := time.Now()
		for _, m := range mirrors {
			if !m.Due(now) {
				continue
			}
			rp, err := a.Deps.Repos.ByID(ctx, m.RepoID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "mirror: repo %d: %v\n", m.RepoID, err)
				continue
			}
			owner := a.RepoOwnerSlug(ctx, rp)
			path := a.Deps.FS.Path(owner, rp.Name)
			if err := a.Deps.Mirrors.Sync(ctx, m.ID, path); err != nil {
				fmt.Fprintf(os.Stderr, "mirror %d: %v\n", m.ID, err)
				continue
			}
			fmt.Printf("mirror %d synced (%s)\n", m.ID, m.Direction)
		}
	}

	if *once {
		runOnce()
		return nil
	}
	ticker := time.NewTicker(time.Duration(*interval) * time.Second)
	defer ticker.Stop()
	runOnce()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			runOnce()
		}
	}
}

// runImport performs a one-shot repository import for an existing repo record.
func runImport(args []string) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	configPath := fs.String("config", "config.toml", "path to config.toml")
	repoID := fs.Int64("repo-id", 0, "existing repository id to import into")
	source := fs.String("source", "generic", "source: github|gitlab|gitea|generic")
	url := fs.String("url", "", "source repository URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *repoID == 0 || *url == "" {
		return fmt.Errorf("import: --repo-id and --url are required")
	}
	ctx := context.Background()
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	a, err := app.New(ctx, cfg, *configPath)
	if err != nil {
		return err
	}
	defer func() { _ = a.Close() }()
	job, err := a.Deps.Imports.Create(ctx, *repoID, *source, *url)
	if err != nil {
		return err
	}
	rp, err := a.Deps.Repos.ByID(ctx, *repoID)
	if err != nil {
		return err
	}
	path := a.Deps.FS.Path(a.RepoOwnerSlug(ctx, rp), rp.Name)
	if err := a.Deps.Imports.Run(ctx, job.ID, path); err != nil {
		return err
	}
	fmt.Printf("import job %d completed\n", job.ID)
	return nil
}

// runRunner polls the CI queue and executes claimed jobs in-process using the
// host shell (gated by the configured concurrency).
func runRunner(args []string) error {
	fs := flag.NewFlagSet("runner", flag.ContinueOnError)
	configPath := fs.String("config", "config.toml", "path to config.toml")
	runnerID := fs.Int64("id", 1, "runner id")
	once := fs.Bool("once", false, "process one batch then exit")
	interval := fs.Int("interval", 10, "seconds between polls")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx := context.Background()
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	a, err := app.New(ctx, cfg, *configPath)
	if err != nil {
		return err
	}
	defer func() { _ = a.Close() }()

	poll := func() bool {
		job, err := a.Deps.CI.ClaimJob(ctx, *runnerID)
		if err != nil || job == nil {
			return false
		}
		fmt.Printf("runner: claimed job %d (%s)\n", job.ID, job.Name)
		err = a.Deps.CI.RunWith(ctx, job.ID, func(ctx context.Context) error {
			return a.RunCIJob(ctx, job)
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "runner: job %d: %v\n", job.ID, err)
			_ = a.Deps.CI.CompleteJob(ctx, job.ID, "failed", "")
			return true
		}
		_ = a.Deps.CI.CompleteJob(ctx, job.ID, "success", "")
		return true
	}
	if *once {
		poll()
		return nil
	}
	ticker := time.NewTicker(time.Duration(*interval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			for poll() {
			}
		}
	}
}

// runSearchIndex is a no-op placeholder for the optional search index builder.
func runSearchIndex(_ []string) error {
	fmt.Fprintln(os.Stderr, "search-index: gitli uses on-demand `git grep`; no index is required")
	return nil
}

// runAdmin provides administrative subcommands.
func runAdmin(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("admin: usage: gitli admin <user-list|user-promote|user-reset-password>")
	}
	sub := args[0]
	fs := flag.NewFlagSet("admin "+sub, flag.ContinueOnError)
	configPath := fs.String("config", "config.toml", "path to config.toml")
	username := fs.String("username", "", "target username")
	password := fs.String("password", "", "new password (user-reset-password)")
	// Backward-compatible: also accept the config flag only.
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	ctx := context.Background()
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	a, err := app.New(ctx, cfg, *configPath)
	if err != nil {
		return err
	}
	defer func() { _ = a.Close() }()

	switch sub {
	case "user-list":
		users, err := a.Deps.Users.List(ctx, 100, 0)
		if err != nil {
			return err
		}
		for _, u := range users {
			fmt.Printf("%d\t%s\t%s\tadmin=%v\n", u.ID, u.Username, u.Email, u.IsAdmin)
		}
		return nil
	case "user-promote":
		if *username == "" {
			return fmt.Errorf("admin: --username required")
		}
		u, err := a.Deps.Users.ByUsername(ctx, *username)
		if err != nil {
			return err
		}
		if err := a.PromoteAdmin(ctx, u.ID); err != nil {
			return err
		}
		fmt.Printf("user %s promoted to admin\n", u.Username)
		return nil
	case "user-reset-password":
		if *username == "" || len(*password) < 12 {
			return fmt.Errorf("admin: --username and a >=12 char --password are required")
		}
		u, err := a.Deps.Users.ByUsername(ctx, *username)
		if err != nil {
			return err
		}
		if err := a.Deps.Auth.UpdatePassword(ctx, u, *password); err != nil {
			return err
		}
		fmt.Printf("password reset for %s\n", u.Username)
		return nil
	default:
		return fmt.Errorf("admin: unknown subcommand %q", sub)
	}
}

// runHookCommand executes a server-side git hook. Usage:
//
//	gitli hook <name> --owner <owner> --repo <repo> [--config path]
func runHookCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("hook: missing hook name")
	}
	name := args[0]
	fs := flag.NewFlagSet("hook", flag.ContinueOnError)
	configPath := fs.String("config", "config.toml", "path to config.toml")
	owner := fs.String("owner", "", "repository owner")
	repo := fs.String("repo", "", "repository name")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	a, err := app.New(context.Background(), cfg, *configPath)
	if err != nil {
		return err
	}
	defer func() { _ = a.Close() }()
	return a.RunHook(context.Background(), name, *owner, *repo)
}

func usage() {
	fmt.Fprint(os.Stderr, `gitli - lightweight self-hosted Git service

Usage:
  gitli <command> [--config path/to/config.toml]

Commands:
  serve          start HTTP + SSH service
  migrate        apply database migrations
  runner         CI runner
  mirror         mirror synchronization scheduler
  import         repository import tool
  search-index   optional search index builder
  admin          administrative commands
  backup         back up data
  restore        restore data
  hook           run a server-side git hook (invoked by git)
  version        print version
  help           show this help
`)
}
