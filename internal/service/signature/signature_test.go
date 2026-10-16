// SPDX-License-Identifier: AGPL-3.0
package signature

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/infra/gitcmd"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	ddl := `CREATE TABLE commit_signatures (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_id INTEGER NOT NULL,
		commit_sha VARCHAR(64) NOT NULL,
		signer_id INTEGER,
		signature_type VARCHAR(16) NOT NULL DEFAULT 'gpg',
		verified INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := database.Exec(ddl); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return New(store.New(database), gitcmd.New())
}

func TestValidSHA(t *testing.T) {
	bad := []string{
		"",
		"-abc123",
		"not-a-sha",
		"abc",
		strings.Repeat("g", 40),
		"abc123\n",
		"abc 123",
	}
	for _, sha := range bad {
		if validSHA(sha) {
			t.Errorf("validSHA(%q) = true want false", sha)
		}
	}
	good := []string{
		strings.Repeat("a", 40),
		strings.Repeat("0", 64),
		"0123456789abcdef0123456789abcdef01234567",
	}
	for _, sha := range good {
		if !validSHA(sha) {
			t.Errorf("validSHA(%q) = false want true", sha)
		}
	}
}

func TestVerifyRejectsBadSHA(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	if _, err := s.Verify(ctx, t.TempDir(), "-bad"); err == nil {
		t.Fatal("expected bad sha to be rejected")
	}
	if _, err := s.Verify(ctx, t.TempDir(), "zzzz"); err == nil {
		t.Fatal("expected non-hex sha to be rejected")
	}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found in PATH; skipping")
	}
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

func newTestRepo(t *testing.T) (string, string) {
	t.Helper()
	requireGit(t)
	dir := t.TempDir()
	runGit(t, dir, "init", "--initial-branch=main")
	f := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(f, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	return dir, strings.TrimSpace(string(out))
}

func TestVerifyUnsignedCommit(t *testing.T) {
	dir, sha := newTestRepo(t)
	s := newTestService(t)
	ctx := context.Background()

	sig, err := s.Verify(ctx, dir, sha)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if sig.Verified {
		t.Fatal("expected unsigned commit to be unverified")
	}
	if sig.CommitSHA != sha {
		t.Fatalf("sha = %q want %q", sig.CommitSHA, sha)
	}

	sig.RepoID = 7
	if err := s.Save(ctx, sig); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := s.Get(ctx, 7, sha)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil || got.Verified {
		t.Fatalf("unexpected cached signature: %+v", got)
	}

	missing, err := s.Get(ctx, 7, strings.Repeat("a", 40))
	if err != nil {
		t.Fatalf("get missing: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil for uncached sha, got %+v", missing)
	}
}
