// SPDX-License-Identifier: AGPL-3.0
package gitcmd

import (
	"fmt"
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
	if strings.Contains(repoPath, "..") || strings.ContainsAny(repoPath, "~\\") {
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
