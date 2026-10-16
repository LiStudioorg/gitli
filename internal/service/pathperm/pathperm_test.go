// SPDX-License-Identifier: AGPL-3.0
package pathperm

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
	ddl := `CREATE TABLE path_permissions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_id INTEGER NOT NULL,
		path_pattern VARCHAR(1024) NOT NULL,
		role_id INTEGER,
		permission VARCHAR(16) NOT NULL DEFAULT 'read',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := database.Exec(ddl); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return New(store.New(database))
}

func TestCreateValidation(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	if _, err := s.Create(ctx, 1, "src/*", "superuser"); err == nil {
		t.Fatal("expected invalid permission to be rejected")
	}
	if _, err := s.Create(ctx, 1, "../etc/passwd", "read"); err == nil {
		t.Fatal("expected traversal pattern to be rejected")
	}
	if _, err := s.Create(ctx, 1, "src/\x00bad", "read"); err == nil {
		t.Fatal("expected control char pattern to be rejected")
	}
	if _, err := s.Create(ctx, 1, "   ", "read"); err == nil {
		t.Fatal("expected blank pattern to be rejected")
	}

	r, err := s.Create(ctx, 1, "src/*", "write")
	if err != nil {
		t.Fatalf("create valid: %v", err)
	}
	if r.ID == 0 || r.Permission != "write" {
		t.Fatalf("unexpected rule: %+v", r)
	}
	if _, err := s.Create(ctx, 1, "docs/*", ""); err != nil {
		t.Fatalf("default permission should be accepted: %v", err)
	}
}

func TestListAndDelete(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	created, err := s.Create(ctx, 5, "src/*", "write")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.Create(ctx, 5, "docs/*", "read"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.Create(ctx, 9, "other/*", "read"); err != nil {
		t.Fatalf("create: %v", err)
	}

	list, err := s.List(ctx, 5)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list len = %d want 2", len(list))
	}

	if err := s.Delete(ctx, 5, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := s.Delete(ctx, 5, created.ID); err == nil {
		t.Fatal("expected second delete to fail")
	}
	list, err = s.List(ctx, 5)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d want 1", len(list))
	}
}

func TestMatchSpecificity(t *testing.T) {
	rules := []Rule{
		{Pattern: "src/*", Permission: "read"},
		{Pattern: "src/secret/*", Permission: "admin"},
	}

	perm, ok := Match(rules, "src/secret/key.txt")
	if !ok || perm != "admin" {
		t.Fatalf("Match = %q,%v want admin,true", perm, ok)
	}

	perm, ok = Match(rules, "src/main.go")
	if !ok || perm != "read" {
		t.Fatalf("Match = %q,%v want read,true", perm, ok)
	}

	if _, ok := Match(rules, "docs/readme.md"); ok {
		t.Fatal("expected no match for docs path")
	}
}

func TestMatchGlobBehavior(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"*", "anything/at/all", true},
		{"*.go", "main.go", true},
		{"*.go", "main.rs", false},
		{"src/?.go", "src/a.go", true},
		{"src/?.go", "src/ab.go", false},
		{"a*b*c", "axxbxxc", true},
		{"a*b*c", "abc", true},
		{"src/*", "src/nested/deep/file.go", true},
		{"src/*", "docs/file.go", false},
	}
	for _, tc := range cases {
		got := globMatch(tc.pattern, tc.path)
		if got != tc.want {
			t.Errorf("globMatch(%q, %q) = %v want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
}
