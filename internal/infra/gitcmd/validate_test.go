// SPDX-License-Identifier: AGPL-3.0
package gitcmd

import (
	"os"
	"path/filepath"
	"testing"
)

// Layout.Path 的返回值必然带平台原生分隔符。此前 ValidateRepoPath 无条件禁止
// '\'，Windows 上任何仓库路径都会在进入 git 之前被判为非法，等于 Git 托管在该
// 平台整体不可用；这里锁住「原生路径必须放行」这条行为。
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

func TestValidateRepoPathRejectsEscapeAttempts(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repos")
	cases := []struct {
		name string
		path string
	}{
		{"empty", ""},
		{"root itself", root},
		// filepath.Join 会先把 ".." 化简掉，越界靠 Clean + root 前缀检查兜住。
		{"joined escape", filepath.Join(root, "..", "outside")},
		{"literal traversal", root + string(filepath.Separator) + ".." + string(filepath.Separator) + "outside"},
		{"nul byte", filepath.Join(root, "a\x00b")},
	}
	for _, c := range cases {
		if _, err := ValidateRepoPath(root, c.path); err == nil {
			t.Errorf("%s: expected rejection", c.name)
		}
	}
}

// 反斜杠与 ~ 作为「可疑字符」只在 POSIX 上有意义：Windows 上它们是合法分隔符 /
// 短文件名组成部分。该用例保证这次放宽不会把 POSIX 侧的防护一起放宽。
func TestValidateRepoPathSuspiciousCharsArePOSIXOnly(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repos")
	if os.PathSeparator == '\\' {
		return
	}
	if _, err := ValidateRepoPath(root, filepath.Join(root, "demo")+"\\evil"); err == nil {
		t.Error("posix: backslash must be rejected")
	}
	if _, err := ValidateRepoPath(root, filepath.Join(root, "~home")); err == nil {
		t.Error("posix: tilde must be rejected")
	}
}
