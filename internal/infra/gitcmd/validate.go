// SPDX-License-Identifier: AGPL-3.0
package gitcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateRepoPath ensures a repository path is safe to pass to git:
// no traversal tokens, no null bytes, and still inside root after cleaning.
func ValidateRepoPath(root, repoPath string) (string, error) {
	if repoPath == "" {
		return "", fmt.Errorf("gitcmd: empty repo path")
	}
	if strings.ContainsRune(repoPath, 0) {
		return "", fmt.Errorf("gitcmd: repo path contains NUL byte")
	}
	if strings.Contains(repoPath, "..") {
		return "", fmt.Errorf("gitcmd: repo path contains forbidden characters")
	}
	// 反斜杠与 ~ 只在 POSIX 上才是可疑字符。Windows 上 \ 是原生路径分隔符，
	// 一律禁止会让本机所有 git 操作（分支、文件树、clone/push）全部失败，
	// 与 README 宣称的 windows 支持矩阵矛盾。真正的越界防护是下面的
	// Clean 之后的 root 前缀包含检查，owner/repo 名字另由 secutil.ValidateName 过滤。
	if os.PathSeparator != '\\' && strings.ContainsAny(repoPath, "~\\") {
		return "", fmt.Errorf("gitcmd: repo path contains forbidden characters")
	}
	cleanRoot := filepath.Clean(root)
	cleanRepo := filepath.Clean(repoPath)
	if cleanRepo == cleanRoot {
		return "", fmt.Errorf("gitcmd: repo path is the root directory")
	}
	if !strings.HasPrefix(cleanRepo, cleanRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("gitcmd: repo path %q escapes root %q", cleanRepo, cleanRoot)
	}
	return cleanRepo, nil
}
