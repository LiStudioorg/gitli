// SPDX-License-Identifier: AGPL-3.0
package userwebhook

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
	ddl := `CREATE TABLE user_webhooks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		url VARCHAR(1024) NOT NULL,
		secret VARCHAR(255) NOT NULL DEFAULT '',
		events TEXT NOT NULL DEFAULT '',
		active INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := database.Exec(ddl); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return New(store.New(database))
}

func TestCreateRejectsPrivateURL(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	if _, err := s.Create(ctx, 1, "http://127.0.0.1/x", "s", []string{"push"}); err == nil {
		t.Fatal("expected SSRF validation error for loopback address")
	}
}

func TestCreateAcceptsPublicURL(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	h, err := s.Create(ctx, 1, "https://example.com/hook", "sekret", []string{"push", "issues"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if h.ID == 0 || h.UserID != 1 || !h.Active {
		t.Fatalf("unexpected hook: %+v", h)
	}
	if len(h.Events) != 2 || h.Events[0] != "push" {
		t.Fatalf("unexpected events: %+v", h.Events)
	}
}

func TestListDeleteSetActive(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	h1, err := s.Create(ctx, 1, "https://example.com/a", "", []string{"push"})
	if err != nil {
		t.Fatalf("create a: %v", err)
	}
	if _, err := s.Create(ctx, 1, "https://example.com/b", "", nil); err != nil {
		t.Fatalf("create b: %v", err)
	}
	if _, err := s.Create(ctx, 2, "https://example.com/c", "", nil); err != nil {
		t.Fatalf("create c: %v", err)
	}

	hooks, err := s.List(ctx, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(hooks) != 2 {
		t.Fatalf("len = %d, want 2", len(hooks))
	}

	if err := s.SetActive(ctx, 1, h1.ID, false); err != nil {
		t.Fatalf("set active: %v", err)
	}
	got, err := s.ByID(ctx, 1, h1.ID)
	if err != nil {
		t.Fatalf("by id: %v", err)
	}
	if got.Active {
		t.Fatal("hook should be inactive")
	}

	if err := s.Delete(ctx, 2, h1.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-user delete err = %v, want ErrNotFound", err)
	}
	if err := s.Delete(ctx, 1, h1.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := s.Delete(ctx, 1, h1.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("double delete err = %v, want ErrNotFound", err)
	}
}
