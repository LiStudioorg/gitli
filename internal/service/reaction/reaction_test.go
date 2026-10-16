// SPDX-License-Identifier: AGPL-3.0
package reaction

import (
	"context"
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
	ddl := `CREATE TABLE reactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		target_type VARCHAR(32),
		target_id INTEGER,
		emoji VARCHAR(32),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := database.Exec(ddl); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return New(store.New(database))
}

func TestAddIdempotentAndList(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	if err := s.Add(ctx, 1, "issue", 10, "+1"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := s.Add(ctx, 1, "issue", 10, "+1"); err != nil {
		t.Fatalf("add duplicate: %v", err)
	}
	if err := s.Add(ctx, 2, "issue", 10, "+1"); err != nil {
		t.Fatalf("add second user: %v", err)
	}

	counts, err := s.List(ctx, "issue", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if counts["+1"] != 2 {
		t.Fatalf("count = %d want 2", counts["+1"])
	}

	mine, err := s.ListByUser(ctx, 1, "issue", 10)
	if err != nil {
		t.Fatalf("list by user: %v", err)
	}
	if len(mine) != 1 || mine[0] != "+1" {
		t.Fatalf("mine = %v want [+1]", mine)
	}
}

func TestRemove(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	if err := s.Add(ctx, 1, "comment", 5, "heart"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := s.Remove(ctx, 1, "comment", 5, "heart"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	counts, err := s.List(ctx, "comment", 5)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(counts) != 0 {
		t.Fatalf("counts = %v want empty", counts)
	}
}

func TestInvalidTarget(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	if err := s.Add(ctx, 1, "bogus", 1, "+1"); err == nil {
		t.Fatal("expected error for invalid target type")
	}
	if err := s.Remove(ctx, 1, "bogus", 1, "+1"); err == nil {
		t.Fatal("expected error for invalid target type")
	}
}
