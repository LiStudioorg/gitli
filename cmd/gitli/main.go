package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"gitli/internal/config"
	"gitli/internal/db"
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

	srv, err := web.NewServer(queries)
	if err != nil {
		return fmt.Errorf("init web server: %w", err)
	}

	// M4: 在此启动内置 SSH server 与 git:// 匿名协议 server。

	slog.Info("gitli serving", "http", cfg.Server.HTTPAddr, "data", cfg.App.DataDir)
	return http.ListenAndServe(cfg.Server.HTTPAddr, srv)
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
