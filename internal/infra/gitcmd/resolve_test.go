// SPDX-License-Identifier: AGPL-3.0
package gitcmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestResolveExplicitBin verifies an explicit [git] bin is used verbatim and
// never routed through PATH lookup.
func TestResolveExplicitBin(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "git")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := resolveOnce(bin)
	if err != nil {
		t.Fatalf("resolveOnce: %v", err)
	}
	if got != bin {
		t.Fatalf("got %q want %q", got, bin)
	}
}

// TestResolveMissingExplicitBin reports a clear stat error instead of probing PATH.
func TestResolveMissingExplicitBin(t *testing.T) {
	if _, err := resolveOnce(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected error for missing explicit bin")
	}
}

// TestFindTermuxGitMissing ensures the Termux probe does not panic and
// reports absence on non-Termux hosts.
func TestFindTermuxGitMissing(t *testing.T) {
	if _, ok := findTermuxGit(); ok {
		t.Skip("host actually has a Termux git path")
	}
}
