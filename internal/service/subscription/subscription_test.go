// SPDX-License-Identifier: AGPL-3.0
package subscription

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
	ddl := `CREATE TABLE subscriptions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		target_type VARCHAR(32),
		target_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := database.Exec(ddl); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return New(store.New(database))
}

func TestSubscribeIdempotentAndUnsubscribe(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	if err := s.Subscribe(ctx, 1, "issue", 10); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if err := s.Subscribe(ctx, 1, "issue", 10); err != nil {
		t.Fatalf("subscribe duplicate: %v", err)
	}
	ok, err := s.IsSubscribed(ctx, 1, "issue", 10)
	if err != nil {
		t.Fatalf("is subscribed: %v", err)
	}
	if !ok {
		t.Fatal("expected subscribed")
	}

	if err := s.Unsubscribe(ctx, 1, "issue", 10); err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}
	ok, err = s.IsSubscribed(ctx, 1, "issue", 10)
	if err != nil {
		t.Fatalf("is subscribed after: %v", err)
	}
	if ok {
		t.Fatal("expected not subscribed")
	}
}

func TestListByUser(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	for _, id := range []int64{3, 1, 2} {
		if err := s.Subscribe(ctx, 7, "pull", id); err != nil {
			t.Fatalf("subscribe %d: %v", id, err)
		}
	}
	ids, err := s.ListByUser(ctx, 7, "pull")
	if err != nil {
		t.Fatalf("list by user: %v", err)
	}
	if len(ids) != 3 || ids[0] != 1 || ids[2] != 3 {
		t.Fatalf("ids = %v want [1 2 3]", ids)
	}
}

func TestInvalidTarget(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	if err := s.Subscribe(ctx, 1, "bogus", 1); err == nil {
		t.Fatal("expected error for invalid target type")
	}
	if _, err := s.IsSubscribed(ctx, 1, "bogus", 1); err == nil {
		t.Fatal("expected error for invalid target type")
	}
}
