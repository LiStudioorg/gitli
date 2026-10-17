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
		filepath.Join(root, "alice", "..", "..", "etc", "passwd"),
		filepath.Join(root, "alice", "~", "x"),
		root,
		root + "x" + string(filepath.Separator) + "evil",
		filepath.Join(string(filepath.Separator)+"etc", "passwd"),
	}
	if os.PathSeparator != '\\' {
		outside = append(outside, `\data\repos\alice\..\..\etc`, `alice\..\demo`)
	}
	for _, p := range outside {
		if _, err := ValidateRepoPath(root, p); err == nil {
			t.Fatalf("unsafe path accepted: %q", p)
		}
	}
}
