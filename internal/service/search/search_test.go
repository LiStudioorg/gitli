// SPDX-License-Identifier: AGPL-3.0
package search

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/LiStudioorg/gitli/internal/infra/gitcmd"
)

func requireGit(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git binary not found in PATH; skipping")
	}
	return path
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

func newTestRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)
	dir := t.TempDir()
	runGit(t, dir, "init", "--initial-branch=main")
	if err := os.WriteFile(filepath.Join(dir, "hello.go"), []byte("package main\n\nfunc main() {\n\tprintln(\"needle\")\n}\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "other.txt"), []byte("needle one\nneedle two\nneedle three\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")
	return dir
}

func TestGrepFindsMatch(t *testing.T) {
	dir := newTestRepo(t)
	svc := New(gitcmd.New())

	results, err := svc.Grep(context.Background(), dir, "needle", "HEAD", 0)
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one match, got none")
	}
	found := false
	for _, r := range results {
		if r.Path == "hello.go" && r.Line == 4 {
			found = true
			if r.Language != "Go" {
				t.Errorf("language = %q, want Go", r.Language)
			}
		}
	}
	if !found {
		t.Errorf("expected match in hello.go:4, results = %+v", results)
	}
}

func TestGrepRespectsLimit(t *testing.T) {
	dir := newTestRepo(t)
	svc := New(gitcmd.New())

	all, err := svc.Grep(context.Background(), dir, "needle", "HEAD", 0)
	if err != nil {
		t.Fatalf("Grep all: %v", err)
	}
	if len(all) < 2 {
		t.Fatalf("expected multiple matches, got %d", len(all))
	}

	limited, err := svc.Grep(context.Background(), dir, "needle", "HEAD", 1)
	if err != nil {
		t.Fatalf("Grep limited: %v", err)
	}
	if len(limited) != 1 {
		t.Fatalf("limit = 1, got %d results", len(limited))
	}
}

func TestGrepNoMatch(t *testing.T) {
	dir := newTestRepo(t)
	svc := New(gitcmd.New())

	results, err := svc.Grep(context.Background(), dir, "does-not-exist-xyz", "HEAD", 0)
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no matches, got %d", len(results))
	}
}

func TestGrepPaths(t *testing.T) {
	dir := newTestRepo(t)
	svc := New(gitcmd.New())

	paths, err := svc.GrepPaths(context.Background(), dir, "needle", "HEAD", 0)
	if err != nil {
		t.Fatalf("GrepPaths: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 distinct paths, got %v", paths)
	}
}
