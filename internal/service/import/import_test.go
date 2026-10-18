// SPDX-License-Identifier: AGPL-3.0
package importer

import (
	"context"
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
	ddl := `CREATE TABLE repo_imports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_id INTEGER NOT NULL,
		source VARCHAR(64) NOT NULL,
		source_url VARCHAR(1024) NOT NULL,
		status VARCHAR(32) NOT NULL DEFAULT 'pending',
		message TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := database.Exec(ddl); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return New(store.New(database), gitcmd.New())
}

func TestCreateRejectsInvalidSource(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	if _, err := s.Create(ctx, 1, "bitbucket", "https://example.com/repo.git"); err == nil {
		t.Fatal("expected invalid source to be rejected")
	}
	if _, err := s.Create(ctx, 1, SourceHosted, ""); err == nil {
		t.Fatal("expected empty source url to be rejected")
	}
}

func TestCreateValidPending(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	for _, src := range []string{SourceHosted, SourceGeneric} {
		job, err := s.Create(ctx, 5, src, "https://example.com/repo.git")
		if err != nil {
			t.Fatalf("create %s: %v", src, err)
		}
		if job.Status != StatusPending {
			t.Fatalf("status = %q want %q", job.Status, StatusPending)
		}
		if job.ID == 0 {
			t.Fatal("expected non-zero id")
		}
	}

	list, err := s.ListByRepo(ctx, 5)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list len = %d want 2", len(list))
	}

	got, err := s.ByID(ctx, list[0].ID)
	if err != nil {
		t.Fatalf("by id: %v", err)
	}
	if got.Source != SourceHosted || got.Status != StatusPending {
		t.Fatalf("unexpected job: %+v", got)
	}
}

// TestCreateRejectsSSRFTargets ensures server-side fetches cannot be aimed at
// loopback, private or cloud metadata addresses.
func TestCreateRejectsSSRFTargets(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	blocked := []string{
		"http://127.0.0.1/repo.git",
		"http://localhost/repo.git",
		"http://169.254.169.254/latest/meta-data",
		"http://10.0.0.5/repo.git",
		"http://192.168.1.10/repo.git",
		"http://172.16.0.1/repo.git",
		"http://[::1]/repo.git",
	}
	for _, u := range blocked {
		if _, err := s.Create(ctx, 1, SourceGeneric, u); err == nil {
			t.Errorf("SSRF target accepted: %s", u)
		}
	}

	if _, err := s.Create(ctx, 1, SourceGeneric, "http://127.0.0.1/repo.git'); DROP TABLE repo_imports;--"); err == nil {
		t.Error("expected injection-shaped URL to be rejected")
	}
}
