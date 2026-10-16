// SPDX-License-Identifier: AGPL-3.0
package topic

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
	if _, err := database.Exec(`CREATE TABLE repo_topics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_id INTEGER NOT NULL,
		topic VARCHAR(64) NOT NULL
	)`); err != nil {
		t.Fatalf("ddl: %v", err)
	}
	return New(store.New(database))
}

func TestValidate(t *testing.T) {
	valid := []string{"go", "web-framework", "a.b", "v1.2", "x"}
	for _, name := range valid {
		if err := Validate(name); err != nil {
			t.Errorf("Validate(%q) unexpected error: %v", name, err)
		}
	}
	invalid := []string{"", "-leading", "UPPER", "with space", "under_score", "ümlaut"}
	for _, name := range invalid {
		if err := Validate(name); err == nil {
			t.Errorf("Validate(%q) expected error", name)
		}
	}
	long := ""
	for i := 0; i < MaxLength+1; i++ {
		long += "a"
	}
	if err := Validate(long); err == nil {
		t.Errorf("Validate(long) expected error")
	}
}

func TestAddListRemoveDedup(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	if err := s.Add(ctx, 1, "Go"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := s.Add(ctx, 1, "go"); err != nil {
		t.Fatalf("add dup: %v", err)
	}
	if err := s.Add(ctx, 1, "web-framework"); err != nil {
		t.Fatalf("add second: %v", err)
	}
	if err := s.Add(ctx, 2, "go"); err != nil {
		t.Fatalf("add other repo: %v", err)
	}

	got, err := s.List(ctx, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 topics (dedup), got %d: %v", len(got), got)
	}
	if got[0] != "go" || got[1] != "web-framework" {
		t.Fatalf("unexpected topics: %v", got)
	}

	if err := s.Remove(ctx, 1, "go"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	got, err = s.List(ctx, 1)
	if err != nil {
		t.Fatalf("list after remove: %v", err)
	}
	if len(got) != 1 || got[0] != "web-framework" {
		t.Fatalf("unexpected topics after remove: %v", got)
	}

	other, err := s.List(ctx, 2)
	if err != nil {
		t.Fatalf("list repo 2: %v", err)
	}
	if len(other) != 1 || other[0] != "go" {
		t.Fatalf("repo 2 should be unaffected: %v", other)
	}
}

func TestAddInvalid(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	if err := s.Add(ctx, 1, "Bad Name"); err == nil {
		t.Fatalf("expected validation error")
	}
	if err := s.Add(ctx, 0, "go"); err == nil {
		t.Fatalf("expected invalid repo id error")
	}
}
