// SPDX-License-Identifier: AGPL-3.0
package wiki

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
	ddl := `CREATE TABLE wiki_pages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_id INTEGER NOT NULL,
		slug VARCHAR(255) NOT NULL,
		title VARCHAR(255) NOT NULL DEFAULT '',
		body TEXT NOT NULL DEFAULT '',
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

	page, err := s.Create(ctx, 1, "home", "Home", "welcome")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if page.ID == 0 {
		t.Fatal("expected assigned id")
	}
	if page.Slug != "home" || page.Title != "Home" || page.Body != "welcome" {
		t.Fatalf("unexpected page: %+v", page)
	}

	bySlug, err := s.BySlug(ctx, 1, "home")
	if err != nil {
		t.Fatalf("by slug: %v", err)
	}
	if bySlug.ID != page.ID || bySlug.Body != "welcome" {
		t.Fatalf("by slug mismatch: %+v", bySlug)
	}

	byID, err := s.ByID(ctx, page.ID)
	if err != nil {
		t.Fatalf("by id: %v", err)
	}
	if byID.Slug != "home" {
		t.Fatalf("by id mismatch: %+v", byID)
	}
}

func TestList(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	if _, err := s.Create(ctx, 1, "b", "B", ""); err != nil {
		t.Fatalf("create b: %v", err)
	}
	if _, err := s.Create(ctx, 1, "a", "A", ""); err != nil {
		t.Fatalf("create a: %v", err)
	}
	if _, err := s.Create(ctx, 2, "c", "C", ""); err != nil {
		t.Fatalf("create c: %v", err)
	}
	pages, err := s.List(ctx, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("len = %d want 2", len(pages))
	}
	if pages[0].Slug != "a" || pages[1].Slug != "b" {
		t.Fatalf("unexpected order: %+v", pages)
	}
}

func TestUpdate(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	page, err := s.Create(ctx, 1, "home", "Home", "old")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	page.Title = "New"
	page.Body = "new body"
	if err := s.Update(ctx, page); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := s.BySlug(ctx, 1, "home")
	if err != nil {
		t.Fatalf("by slug: %v", err)
	}
	if got.Title != "New" || got.Body != "new body" {
		t.Fatalf("update not applied: %+v", got)
	}

	if err := s.Update(ctx, &Page{ID: 999, Title: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update missing = %v want ErrNotFound", err)
	}
}

func TestDelete(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	if _, err := s.Create(ctx, 1, "home", "Home", "x"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.Delete(ctx, 1, "home"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.BySlug(ctx, 1, "home"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("by slug after delete = %v want ErrNotFound", err)
	}
	if err := s.Delete(ctx, 1, "home"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete missing = %v want ErrNotFound", err)
	}
}
