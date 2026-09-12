package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"gitli/internal/auth"
	"gitli/internal/config"
	"gitli/internal/db"
	"gitli/internal/git"
	"gitli/internal/web"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "admin" {
		fmt.Fprintln(os.Stderr, "admin 子命令尚未实现（M7）")
		os.Exit(1)
	}
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "serve" {
		args = args[1:]
	}
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", args[0])
		os.Exit(2)
	}

	if err := runServe(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func runServe() error {
	configPath := flag.String("config", "config.toml", "配置文件路径")
	flag.Parse()

	logger := newLogger()
	slog.SetDefault(logger)

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := os.MkdirAll(cfg.App.DataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	if err := os.MkdirAll(cfg.ReposDir(), 0o755); err != nil {
		return fmt.Errorf("create repos dir: %w", err)
	}

	conn, err := db.Open(cfg.DBPath())
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer conn.Close()
	queries := db.NewQuerier(conn)

	// M7: OAuth2/OIDC 配置注入
	auth.SetOAuthConfig(&cfg.OAuth2)

	srv, err := web.NewServer(queries, cfg.ReposDir(), cfg.Server.RootURL)
	if err != nil {
		return fmt.Errorf("init web server: %w", err)
	}

	// M4: 在此启动内置 SSH server 与 git:// 匿名协议 server。
	startGitServers(queries, cfg)

	slog.Info("gitli serving", "http", cfg.Server.HTTPAddr, "data", cfg.App.DataDir)
	return http.ListenAndServe(cfg.Server.HTTPAddr, srv)
}

// startGitServers 以 goroutine 启动 SSH 与 git:// 匿名协议 server（失败仅记日志，不影响 HTTP）。
func startGitServers(q db.Querier, cfg *config.Config) {
	reposDir := cfg.ReposDir()
	authorize := func(repo db.Repo, user *db.User, write bool) bool {
		return auth.CanAccess(context.Background(), q, user, repo, write)
	}
	go func() {
		sshSrv := git.NewSSHServer(cfg.Server.SSHAddr, reposDir, q, authorize)
		slog.Info("ssh server listening", "addr", cfg.Server.SSHAddr)
		if err := sshSrv.ListenAndServe(); err != nil {
			slog.Error("ssh server stopped", "err", err)
		}
	}()
	go func() {
		gitSrv := git.NewGitProtocolServer(cfg.Server.GitAddr, reposDir, q)
		slog.Info("git protocol server listening", "addr", cfg.Server.GitAddr)
		if err := gitSrv.ListenAndServe(); err != nil {
			slog.Error("git protocol server stopped", "err", err)
		}
	}()
}

func newLogger() *slog.Logger {
	level := slog.LevelInfo
	switch os.Getenv("GITLI_LOG_LEVEL") {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	return slog.New(h)
}
