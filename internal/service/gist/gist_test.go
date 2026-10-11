// SPDX-License-Identifier: AGPL-3.0
package gist

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
	ddl := `CREATE TABLE gists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		description VARCHAR(255) NOT NULL DEFAULT '',
		is_public INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := database.Exec(ddl); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return New(store.New(database))
}

func TestCreateAndFetch(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	g, err := s.Create(ctx, 7, "hello", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if g.ID == 0 {
		t.Fatal("expected assigned id")
	}
	if g.UserID != 7 || g.Description != "hello" || !g.IsPublic {
		t.Fatalf("unexpected gist: %+v", g)
	}

	got, err := s.ByID(ctx, g.ID)
	if err != nil {
		t.Fatalf("by id: %v", err)
	}
	if got.Description != "hello" || !got.IsPublic {
		t.Fatalf("by id mismatch: %+v", got)
	}
}

func TestListByUser(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	if _, err := s.Create(ctx, 1, "first", false); err != nil {
		t.Fatalf("create first: %v", err)
	}
	if _, err := s.Create(ctx, 1, "second", true); err != nil {
		t.Fatalf("create second: %v", err)
	}
	if _, err := s.Create(ctx, 2, "other", true); err != nil {
		t.Fatalf("create other: %v", err)
	}
	got, err := s.ListByUser(ctx, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d want 2", len(got))
	}
	if got[0].Description != "second" || got[1].Description != "first" {
		t.Fatalf("unexpected order: %+v", got)
	}
}

func TestDelete(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	g, err := s.Create(ctx, 1, "x", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.Delete(ctx, 2, g.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete wrong owner = %v want ErrNotFound", err)
	}
	if err := s.Delete(ctx, 1, g.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.ByID(ctx, g.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("by id after delete = %v want ErrNotFound", err)
	}
}
