// SPDX-License-Identifier: AGPL-3.0
package mirror

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
	ddl := `CREATE TABLE repo_mirrors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_id INTEGER NOT NULL,
		remote_url VARCHAR(1024) NOT NULL,
		direction VARCHAR(16) NOT NULL DEFAULT 'pull',
		interval_minutes INTEGER NOT NULL DEFAULT 60,
		last_synced_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := database.Exec(ddl); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return New(store.New(database), gitcmd.New())
}

func TestCreateRejectsBadDirection(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	if _, err := s.Create(ctx, 1, "https://example.com/repo.git", "sideways", 30); err == nil {
		t.Fatal("expected invalid direction to be rejected")
	}
	if _, err := s.Create(ctx, 1, "", DirectionPull, 30); err == nil {
		t.Fatal("expected empty remote url to be rejected")
	}
}

func TestCreateAndListByRepo(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	created, err := s.Create(ctx, 42, "https://example.com/repo.git", DirectionPull, 15)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected non-zero id")
	}
	if created.Direction != DirectionPull {
		t.Fatalf("direction = %q want %q", created.Direction, DirectionPull)
	}

	if _, err := s.Create(ctx, 7, "https://example.com/other.git", DirectionPush, 0); err != nil {
		t.Fatalf("create other: %v", err)
	}

	list, err := s.ListByRepo(ctx, 42)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d want 1", len(list))
	}
	if list[0].RepoID != 42 || list[0].RemoteURL != "https://example.com/repo.git" {
		t.Fatalf("unexpected mirror: %+v", list[0])
	}
	if list[0].IntervalMinutes != 15 {
		t.Fatalf("interval = %d want 15", list[0].IntervalMinutes)
	}
	if list[0].LastSyncedAt != nil {
		t.Fatal("last_synced_at should be nil before sync")
	}

	empty, err := s.ListByRepo(ctx, 999)
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("list len = %d want 0", len(empty))
	}
}
