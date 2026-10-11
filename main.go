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
	case "runner", "mirror", "import", "search-index", "admin":
		return runPlaceholder(cmd)
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

func runPlaceholder(name string) error {
	return fmt.Errorf("subcommand %q is not implemented yet", name)
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
  version        print version
  help           show this help
`)
}
