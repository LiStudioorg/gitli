// SPDX-License-Identifier: AGPL-3.0
package block

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
	ddl := `CREATE TABLE user_blocks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		blocker_id INTEGER,
		blocked_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := database.Exec(ddl); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return New(store.New(database))
}

func TestSelfBlockRejected(t *testing.T) {
	s := newTestService(t)
	if err := s.Block(context.Background(), 1, 1); err == nil {
		t.Fatal("expected self-block to be rejected")
	}
}

func TestBlockIsBlockedList(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	if err := s.Block(ctx, 1, 2); err != nil {
		t.Fatalf("block: %v", err)
	}
	if err := s.Block(ctx, 1, 3); err != nil {
		t.Fatalf("block: %v", err)
	}

	ok, err := s.IsBlocked(ctx, 1, 2)
	if err != nil {
		t.Fatalf("is blocked: %v", err)
	}
	if !ok {
		t.Fatal("expected 1->2 to be blocked")
	}
	ok, err = s.IsBlocked(ctx, 2, 1)
	if err != nil {
		t.Fatalf("is blocked: %v", err)
	}
	if ok {
		t.Fatal("expected 2->1 not to be blocked")
	}

	list, err := s.ListBlocked(ctx, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 || list[0] != 2 || list[1] != 3 {
		t.Fatalf("list = %v want [2 3]", list)
	}
}

func TestBlockIdempotentAndUnblock(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	if err := s.Block(ctx, 1, 2); err != nil {
		t.Fatalf("block: %v", err)
	}
	if err := s.Block(ctx, 1, 2); err != nil {
		t.Fatalf("block duplicate: %v", err)
	}
	list, err := s.ListBlocked(ctx, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d want 1", len(list))
	}

	if err := s.Unblock(ctx, 1, 2); err != nil {
		t.Fatalf("unblock: %v", err)
	}
	ok, err := s.IsBlocked(ctx, 1, 2)
	if err != nil {
		t.Fatalf("is blocked: %v", err)
	}
	if ok {
		t.Fatal("expected block removed")
	}
}
