// SPDX-License-Identifier: AGPL-3.0
// Command gitli is a self-hosted, lightweight Git service platform.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/LiStudioorg/gitli/internal/app"
	"github.com/LiStudioorg/gitli/internal/config"
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
		return runServe(ctx, cfg)
	case "migrate":
		return runMigrate(ctx, cfg)
	case "runner", "mirror", "import", "search-index", "admin", "backup", "restore":
		return runPlaceholder(cmd)
	default:
		usage()
		return fmt.Errorf("unknown subcommand %q", cmd)
	}
}

func runServe(ctx context.Context, cfg *config.Config) error {
	a, err := app.New(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = a.Close() }()
	return a.Serve(ctx)
}

func runMigrate(ctx context.Context, cfg *config.Config) error {
	a, err := app.New(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = a.Close() }()
	return a.Migrate(ctx)
}

func runPlaceholder(name string) error {
	return fmt.Errorf("subcommand %q is not implemented yet", name)
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
