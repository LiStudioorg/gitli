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
	// Bare execution (no args, or only --config) starts the server.
	if len(args) == 0 || args[0] == "--config" || args[0] == "-config" || strings.HasPrefix(args[0], "--config=") {
		fs := flag.NewFlagSet("gitli", flag.ContinueOnError)
		configPath := fs.String("config", "config.toml", "配置文件路径")
		if err := fs.Parse(args); err != nil {
			return err
		}
		created, err := ensureConfigFile(*configPath)
		if err != nil {
			return err
		}
		if created {
			fmt.Printf("未找到配置文件 %s，已自动生成默认配置（含随机加密密钥）。\n", *configPath)
		} else if err := ensureConfiguredSecrets(*configPath); err != nil {
			return err
		}
		cfg, err := config.Load(*configPath)
		if err != nil {
			return err
		}
		if err := refuseRoot(cfg); err != nil {
			return err
		}
		if err := checkWeakSecrets(cfg); err != nil {
			return err
		}
		return runServe(context.Background(), cfg, *configPath)
	}
	cmd := args[0]
	rest := args[1:]

	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	configPath := fs.String("config", "config.toml", "配置文件路径")

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
	case "install":
		return runInstall(rest)
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

	if created, err := ensureConfigFile(*configPath); err != nil {
		return err
	} else if created {
		fmt.Printf("未找到配置文件 %s，已自动生成默认配置（含随机加密密钥）。\n", *configPath)
	} else if err := ensureConfiguredSecrets(*configPath); err != nil {
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
	case "serve", "server":
		if err := checkWeakSecrets(cfg); err != nil {
			return err
		}
		return runServe(ctx, cfg, *configPath)
	case "migrate":
		return runMigrate(ctx, cfg, *configPath)
	default:
		usage()
		return fmt.Errorf("未知的子命令 %q", cmd)
	}
}

// runBackupCommand creates a streaming backup archive. Usage:
//
//	gitli backup --out <file> [--source <dir>] [--encrypt] --config path
func runBackupCommand(args []string) error {
	fs := flag.NewFlagSet("backup", flag.ContinueOnError)
	configPath := fs.String("config", "config.toml", "配置文件路径")
	out := fs.String("out", "", "输出归档文件（.tar.gz 或 .tar.gz.enc）")
	source := fs.String("source", "", "源目录（默认数据目录）")
	encrypt := fs.Bool("encrypt", false, "加密归档（密钥来自 GITLI_BACKUP_KEY）")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return fmt.Errorf("backup：必须指定 --out")
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
	configPath := fs.String("config", "config.toml", "配置文件路径")
	in := fs.String("in", "", "输入归档文件")
	dest := fs.String("dest", "", "目标目录（默认数据目录）")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *in == "" {
		return fmt.Errorf("restore：必须指定 --in")
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
	configPath := fs.String("config", "config.toml", "配置文件路径")
	once := fs.Bool("once", false, "只执行一轮同步后退出")
	interval := fs.Int("interval", 300, "非 --once 时的轮询间隔（秒）")
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
	configPath := fs.String("config", "config.toml", "配置文件路径")
	repoID := fs.Int64("repo-id", 0, "导入目标仓库 ID")
	source := fs.String("source", "generic", "来源类型：hosted|generic")
	url := fs.String("url", "", "来源仓库地址")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *repoID == 0 || *url == "" {
		return fmt.Errorf("import：必须指定 --repo-id 与 --url")
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
	configPath := fs.String("config", "config.toml", "配置文件路径")
	runnerID := fs.Int64("id", 1, "Runner ID")
	once := fs.Bool("once", false, "处理一批后退出")
	interval := fs.Int("interval", 10, "轮询间隔（秒）")
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
	fmt.Fprintln(os.Stderr, "search-index：gitli 使用按需 git grep，无需构建索引")
	return nil
}

// runHookCommand executes a server-side git hook. Usage:
//
//	gitli hook <name> --owner <owner> --repo <repo> [--config path]
func runHookCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("hook：缺少钩子名称")
	}
	name := args[0]
	fs := flag.NewFlagSet("hook", flag.ContinueOnError)
	configPath := fs.String("config", "config.toml", "配置文件路径")
	owner := fs.String("owner", "", "仓库所有者")
	repo := fs.String("repo", "", "仓库名称")
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
	fmt.Fprint(os.Stderr, `gitli — 轻量级自研 Git 服务平台

用法：
  gitli [--config config.toml]          直接启动服务（等同 serve）
  gitli <命令> [参数] [--config 路径]

命令：
  serve, server  启动 HTTP + SSH 服务
  migrate        手动同步数据库结构（正常启动会自动迁移）
  runner         内置 CI Runner
  mirror         镜像同步调度
  import         仓库导入工具
  search-index   搜索索引构建（按需 git grep，无索引）
  admin          管理命令（最高权限，Web 后台仅为其子集）
  backup         备份数据
  restore        恢复数据
  hook           执行服务端 Git 钩子（由 git 调用）
  install        安装为 systemd 系统服务（需 root；--print 仅输出 unit）
  version        显示版本信息
  help           显示本帮助

提示：
  裸执行 gitli 会直接启动服务；首次访问会进入初始化向导创建管理员。
`)
}
