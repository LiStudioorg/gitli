// SPDX-License-Identifier: AGPL-3.0
package ide

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/LiStudioorg/gitli/internal/infra/gitcmd"
)

func gitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// setupRepo creates a temp git repo with a committed Go and JS file.
func setupRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init", "--initial-branch=main")
	goFile := filepath.Join(dir, "hello.go")
	if err := os.WriteFile(goFile, []byte("package main\n\nfunc HelloWorld() {}\n"), 0o644); err != nil {
		t.Fatalf("write go: %v", err)
	}
	jsFile := filepath.Join(dir, "foo.js")
	if err := os.WriteFile(jsFile, []byte("class Foo {}\n"), 0o644); err != nil {
		t.Fatalf("write js: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "init")
	return dir
}

func TestListDir(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}
	dir := setupRepo(t)
	s := New(gitcmd.New())
	entries, err := s.ListDir(context.Background(), dir, "HEAD", "")
	if err != nil {
		t.Fatalf("ListDir: %v", err)
	}
	found := map[string]string{}
	for _, e := range entries {
		found[e.Path] = e.Type
	}
	if found["hello.go"] != "file" {
		t.Fatalf("hello.go not listed as file: %+v", entries)
	}
	if found["foo.js"] != "file" {
		t.Fatalf("foo.js not listed as file: %+v", entries)
	}
}

func TestReadFile(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}
	dir := setupRepo(t)
	s := New(gitcmd.New())
	data, err := s.ReadFile(context.Background(), dir, "HEAD", "hello.go", 0)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) == "" {
		t.Fatal("expected non-empty content")
	}
	if _, err := s.ReadFile(context.Background(), dir, "HEAD", "hello.go", 4); err == nil {
		t.Fatal("expected error when file exceeds maxBytes")
	}
}

func TestSymbols(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available")
	}
	dir := setupRepo(t)
	s := New(gitcmd.New())
	syms, err := s.Symbols(context.Background(), dir, "HEAD", "HelloWorld", 0)
	if err != nil {
		t.Fatalf("Symbols: %v", err)
	}
	if len(syms) == 0 {
		t.Fatal("expected at least one symbol")
	}
	if syms[0].Name != "HelloWorld" || syms[0].Kind != "func" {
		t.Fatalf("unexpected symbol: %+v", syms[0])
	}
	if syms[0].File != "hello.go" || syms[0].Line != 3 {
		t.Fatalf("unexpected location: %+v", syms[0])
	}

	jsSyms, err := s.Symbols(context.Background(), dir, "HEAD", "Foo", 0)
	if err != nil {
		t.Fatalf("Symbols(js): %v", err)
	}
	if len(jsSyms) == 0 || jsSyms[0].Kind != "class" {
		t.Fatalf("expected class Foo, got %+v", jsSyms)
	}

	empty, err := s.Symbols(context.Background(), dir, "HEAD", "", 0)
	if err != nil {
		t.Fatalf("Symbols(empty): %v", err)
	}
	if empty != nil {
		t.Fatalf("empty query should return nil, got %+v", empty)
	}
}
