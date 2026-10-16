// SPDX-License-Identifier: AGPL-3.0
package lfs

import (
	"context"
	"errors"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

func newLockService(t *testing.T) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	ddl := `CREATE TABLE lfs_locks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_id INTEGER NOT NULL,
		path VARCHAR(1024) NOT NULL,
		user_id INTEGER NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := database.Exec(ddl); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return New(store.New(database), t.TempDir())
}

func TestLockValidation(t *testing.T) {
	s := newLockService(t)
	ctx := context.Background()
	for _, path := range []string{"", "a/../b", "..", "a\x00b", "a\nb"} {
		if _, err := s.Lock(ctx, 1, 1, path); !errors.Is(err, ErrLockInvalid) {
			t.Errorf("path %q: err = %v, want ErrLockInvalid", path, err)
		}
	}
}

func TestLockDoubleLockRejected(t *testing.T) {
	s := newLockService(t)
	ctx := context.Background()

	l1, err := s.Lock(ctx, 1, 10, "assets/logo.png")
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	if l1.ID == 0 || l1.UserID != 10 {
		t.Fatalf("unexpected lock: %+v", l1)
	}

	again, err := s.Lock(ctx, 1, 10, "assets/logo.png")
	if err != nil {
		t.Fatalf("idempotent lock: %v", err)
	}
	if again.ID != l1.ID {
		t.Fatalf("idempotent lock created new row: %d != %d", again.ID, l1.ID)
	}

	if _, err := s.Lock(ctx, 1, 20, "assets/logo.png"); !errors.Is(err, ErrLocked) {
		t.Fatalf("other user lock err = %v, want ErrLocked", err)
	}
}

func TestUnlockOwnerCheck(t *testing.T) {
	s := newLockService(t)
	ctx := context.Background()

	l, err := s.Lock(ctx, 1, 10, "a.bin")
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	if err := s.Unlock(ctx, 1, 99, l.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("non-owner unlock err = %v, want ErrNotFound", err)
	}
	if err := s.Unlock(ctx, 1, 10, l.ID); err != nil {
		t.Fatalf("owner unlock: %v", err)
	}
	if err := s.Unlock(ctx, 1, 10, l.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("double unlock err = %v, want ErrNotFound", err)
	}
}

func TestListLocks(t *testing.T) {
	s := newLockService(t)
	ctx := context.Background()
	if _, err := s.Lock(ctx, 1, 10, "a"); err != nil {
		t.Fatalf("lock a: %v", err)
	}
	if _, err := s.Lock(ctx, 1, 20, "b"); err != nil {
		t.Fatalf("lock b: %v", err)
	}
	if _, err := s.Lock(ctx, 2, 10, "c"); err != nil {
		t.Fatalf("lock c: %v", err)
	}
	locks, err := s.ListLocks(ctx, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(locks) != 2 {
		t.Fatalf("len = %d, want 2", len(locks))
	}
}
