// SPDX-License-Identifier: AGPL-3.0
package fsrepo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LiStudioorg/gitli/internal/platform"
)

func TestInstallHooks(t *testing.T) {
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "owner", "repo.git")
	if err := os.MkdirAll(repoPath, 0o750); err != nil {
		t.Fatal(err)
	}
	l := New(dir)
	// 可执行位只在 POSIX 平台有意义：Windows 上 os.Chmod 不会产生 0111。
	// AGENTS 约定「文件权限检查仅在 Unix 生效」，这里按平台能力决定是否断言。
	checkExec := platform.Current().SupportsPerms
	if err := l.InstallHooks(repoPath, "alice", "demo", "/usr/local/bin/gitli", "/etc/gitli/config.toml"); err != nil {
		t.Fatal(err)
	}
	for _, name := range KnownHooks {
		path := filepath.Join(repoPath, "hooks", name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("hook %s missing: %v", name, err)
		}
		if checkExec && info.Mode().Perm()&0o111 == 0 {
			t.Fatalf("hook %s not executable", name)
		}
		data, _ := os.ReadFile(path)
		s := string(data)
		if !strings.Contains(s, "gitli") || !strings.Contains(s, "alice") || !strings.Contains(s, "demo") {
			t.Fatalf("hook %s content unexpected: %s", name, s)
		}
	}
}

func TestRemoveHooks(t *testing.T) {
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "r.git")
	_ = os.MkdirAll(repoPath, 0o750)
	l := New(dir)
	if err := l.InstallHooks(repoPath, "o", "r", "gitli", "/etc/gitli/config.toml"); err != nil {
		t.Fatal(err)
	}
	if err := l.RemoveHooks(repoPath); err != nil {
		t.Fatal(err)
	}
	for _, name := range KnownHooks {
		if _, err := os.Stat(filepath.Join(repoPath, "hooks", name)); !os.IsNotExist(err) {
			t.Fatalf("hook %s should be removed", name)
		}
	}
}
