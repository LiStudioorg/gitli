// SPDX-License-Identifier: AGPL-3.0
package release

import (
	"context"
	"errors"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	ddl := []string{
		`CREATE TABLE releases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER NOT NULL,
			tag VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL DEFAULT '',
			body TEXT NOT NULL DEFAULT '',
			is_draft INTEGER NOT NULL DEFAULT 1,
			is_prerelease INTEGER NOT NULL DEFAULT 0,
			author_id INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE release_assets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			release_id INTEGER NOT NULL,
			name VARCHAR(255) NOT NULL,
			size_bytes INTEGER NOT NULL DEFAULT 0,
			download_count INTEGER NOT NULL DEFAULT 0,
			stored_path VARCHAR(1024) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, stmt := range ddl {
		if _, err := database.Exec(stmt); err != nil {
			t.Fatalf("create table: %v", err)
		}
	}
	return New(store.New(database))
}

func TestCreateAndPublish(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	r, err := s.Create(ctx, 1, 42, "v1.0.0", "First", "notes")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if r.ID == 0 || !r.IsDraft {
		t.Fatalf("expected draft with id: %+v", r)
	}

	got, err := s.ByTag(ctx, 1, "v1.0.0")
	if err != nil {
		t.Fatalf("by tag: %v", err)
	}
	if got.Name != "First" || !got.IsDraft {
		t.Fatalf("by tag mismatch: %+v", got)
	}

	if err := s.Publish(ctx, r.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	published, err := s.ByTag(ctx, 1, "v1.0.0")
	if err != nil {
		t.Fatalf("by tag after publish: %v", err)
	}
	if published.IsDraft {
		t.Fatal("release still draft after publish")
	}

	if err := s.Publish(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("publish missing = %v want ErrNotFound", err)
	}
}

func TestList(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	for _, tag := range []string{"v1", "v2", "v3"} {
		if _, err := s.Create(ctx, 1, 1, tag, tag, ""); err != nil {
			t.Fatalf("create %s: %v", tag, err)
		}
	}
	if _, err := s.Create(ctx, 2, 1, "v9", "v9", ""); err != nil {
		t.Fatalf("create other repo: %v", err)
	}
	got, err := s.List(ctx, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d want 3", len(got))
	}
	if got[0].Tag != "v3" || got[2].Tag != "v1" {
		t.Fatalf("unexpected order: %+v", got)
	}
}

func TestAssets(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	r, err := s.Create(ctx, 1, 1, "v1", "v1", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	a, err := s.AddAsset(ctx, r.ID, "binary.tar.gz", 2048, "/data/assets/abc")
	if err != nil {
		t.Fatalf("add asset: %v", err)
	}
	if a.ID == 0 || a.ReleaseID != r.ID || a.Name != "binary.tar.gz" || a.SizeBytes != 2048 || a.StoredPath != "/data/assets/abc" {
		t.Fatalf("unexpected asset: %+v", a)
	}
	assets, err := s.ListAssets(ctx, r.ID)
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}
	if len(assets) != 1 || assets[0].ID != a.ID {
		t.Fatalf("unexpected assets: %+v", assets)
	}
}

func TestByTagNotFound(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	if _, err := s.ByTag(ctx, 1, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("by tag missing = %v want ErrNotFound", err)
	}
}
