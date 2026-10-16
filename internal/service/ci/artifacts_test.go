// SPDX-License-Identifier: AGPL-3.0
package ci

import (
	"context"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// newCITestService spins up an in-memory sqlite database with the CI tables.
func newCITestService(t *testing.T) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	ddl := []string{
		`CREATE TABLE ci_pipelines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER NOT NULL,
			commit_sha VARCHAR(64) NOT NULL,
			ref VARCHAR(255) NOT NULL,
			status VARCHAR(32) NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE ci_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pipeline_id INTEGER NOT NULL,
			name VARCHAR(255) NOT NULL,
			stage VARCHAR(255) NOT NULL,
			status VARCHAR(32) NOT NULL,
			runner_id INTEGER,
			script TEXT,
			log_path VARCHAR(512),
			started_at DATETIME,
			finished_at DATETIME,
			created_at DATETIME
		)`,
		`CREATE TABLE ci_artifacts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id INTEGER NOT NULL,
			name VARCHAR(255) NOT NULL DEFAULT '',
			stored_path VARCHAR(1024) NOT NULL DEFAULT '',
			size_bytes INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE ci_cache (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER NOT NULL,
			key VARCHAR(255) NOT NULL,
			stored_path VARCHAR(1024) NOT NULL DEFAULT '',
			size_bytes INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE ci_runners (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(255) NOT NULL,
			token_hash VARCHAR(255) NOT NULL UNIQUE,
			tags TEXT NOT NULL DEFAULT '',
			status VARCHAR(32) NOT NULL DEFAULT 'offline',
			last_seen_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, q := range ddl {
		if _, err := database.Exec(q); err != nil {
			t.Fatalf("ddl: %v", err)
		}
	}
	return New(store.New(database), nil, 1)
}

func TestAddAndListArtifacts(t *testing.T) {
	s := newCITestService(t)
	ctx := context.Background()

	a, err := s.AddArtifact(ctx, 10, "build.zip", "/data/ci/10/build.zip", 2048)
	if err != nil {
		t.Fatalf("add artifact: %v", err)
	}
	if a.ID == 0 || a.JobID != 10 || a.Name != "build.zip" || a.SizeBytes != 2048 {
		t.Fatalf("unexpected artifact: %+v", a)
	}
	if _, err := s.AddArtifact(ctx, 10, "logs.txt", "/data/ci/10/logs.txt", 12); err != nil {
		t.Fatalf("add second artifact: %v", err)
	}
	if _, err := s.AddArtifact(ctx, 11, "other.zip", "/data/ci/11/other.zip", 5); err != nil {
		t.Fatalf("add other job artifact: %v", err)
	}

	arts, err := s.ListArtifacts(ctx, 10)
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if len(arts) != 2 {
		t.Fatalf("want 2 artifacts, got %d", len(arts))
	}
	if arts[0].Name != "build.zip" || arts[1].Name != "logs.txt" {
		t.Fatalf("unexpected order: %+v", arts)
	}
}

func TestDeleteArtifact(t *testing.T) {
	s := newCITestService(t)
	ctx := context.Background()

	a, err := s.AddArtifact(ctx, 1, "a.bin", "/p/a.bin", 1)
	if err != nil {
		t.Fatalf("add artifact: %v", err)
	}
	if err := s.DeleteArtifact(ctx, a.ID); err != nil {
		t.Fatalf("delete artifact: %v", err)
	}
	arts, err := s.ListArtifacts(ctx, 1)
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if len(arts) != 0 {
		t.Fatalf("want 0 artifacts after delete, got %d", len(arts))
	}
}
