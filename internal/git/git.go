package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var nameRe = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// ValidateRepoName 校验仓库名：只允许 [A-Za-z0-9_.-]，禁止 ".."，禁止 "." 开头隐藏名。
func ValidateRepoName(name string) error {
	if name == "" || len(name) > 100 {
		return fmt.Errorf("invalid repo name length")
	}
	if !nameRe.MatchString(name) {
		return fmt.Errorf("invalid characters in repo name")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("repo name must not contain '..'")
	}
	if strings.HasSuffix(name, ".git") || strings.HasSuffix(name, ".wiki") {
		return fmt.Errorf("reserved name suffix")
	}
	switch name {
	case ".", "api", "static", "login", "logout", "register", "explore", "admin", "assets":
		return fmt.Errorf("reserved name")
	}
	return nil
}

// ValidateUsername 校验路径中的用户名（同仓库名规则，但允许更少字符也行，保持一致）。
func ValidateUsername(name string) error {
	if name == "" || len(name) > 100 {
		return fmt.Errorf("invalid username length")
	}
	if !nameRe.MatchString(name) {
		return fmt.Errorf("invalid characters in username")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("username must not contain '..'")
	}
	return nil
}

// RepoPath 返回裸仓库的磁盘路径：<reposDir>/<owner>/<name>.git。
func RepoPath(reposDir, owner, name string) string {
	return filepath.Join(reposDir, owner, name+".git")
}

// InitRepo 初始化裸仓库（含 HEAD 指向 main、接收所有 object 类型）。
func InitRepo(ctx context.Context, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create repo dir: %w", err)
	}
	if err := run(ctx, "", "init", "--bare", path); err != nil {
		return fmt.Errorf("git init: %w", err)
	}
	// 默认分支 main；允许 push 任意分支
	if err := run(ctx, path, "symbolic-ref", "HEAD", "refs/heads/main"); err != nil {
		return fmt.Errorf("set HEAD: %w", err)
	}
	return run(ctx, path, "config", "http.receivepack", "true")
}

// DeleteRepo 删除仓库目录。
func DeleteRepo(path string) error {
	return os.RemoveAll(path)
}

// RepoExists 判断裸仓库是否已存在。
func RepoExists(path string) bool {
	_, err := os.Stat(filepath.Join(path, "HEAD"))
	return err == nil
}

// run 执行 git 子命令（参数数组，绝不走 shell）。
func run(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
