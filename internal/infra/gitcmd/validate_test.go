// SPDX-License-Identifier: AGPL-3.0
package gitcmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestValidateRepoPathRejectsTraversal pins the security invariants that must
// hold on every platform, including the POSIX backslash and tilde checks.
func TestValidateRepoPathRejectsTraversal(t *testing.T) {
	root := filepath.Join(string(filepath.Separator)+"data", "repos")
	inside := filepath.Join(root, "alice", "demo")

	if _, err := ValidateRepoPath(root, inside); err != nil {
		t.Fatalf("valid path rejected: %v", err)
	}

	outside := []string{
		"",
		filepath.Join(root, "alice", "..", "..", "etc", "passwd"),
		// `~` 是家目录展开符号，与平台无关，任何平台都必须拒绝。
		filepath.Join(root, "alice", "~", "x"),
		root,
		root + "x" + string(filepath.Separator) + "evil",
		filepath.Join(string(filepath.Separator)+"etc", "passwd"),
		filepath.Join(root, "a\x00b"),
	}
	// 反斜杠只在非 Windows 平台是可疑字符：Windows 上它是原生分隔符，
	// 禁止它会让所有 git 调用在进入 git 之前就被拒。
	if os.PathSeparator != '\\' {
		outside = append(outside, `\data\repos\alice\..\..\etc`, `alice\..\demo`)
	}
	for _, p := range outside {
		if _, err := ValidateRepoPath(root, p); err == nil {
			t.Fatalf("unsafe path accepted: %q", p)
		}
	}
}

// TestValidateRepoPathAcceptsNativeLayout 锁住「原生分隔符必须放行」：
// Layout.Path 的返回值必然带平台原生分隔符，Windows 上一度因此全站 git 不可用。
func TestValidateRepoPathAcceptsNativeLayout(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repos")
	if err := os.MkdirAll(root, 0o750); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, "alice", "demo.git")
	got, err := ValidateRepoPath(root, p)
	if err != nil {
		t.Fatalf("native repo path rejected: %v", err)
	}
	if want := filepath.Clean(p); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
