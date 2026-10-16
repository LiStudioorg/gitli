// SPDX-License-Identifier: AGPL-3.0
package notifpref

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
	ddl := `CREATE TABLE notification_prefs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		key VARCHAR(64),
		value VARCHAR(64) DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := database.Exec(ddl); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return New(store.New(database))
}

func TestGetMissing(t *testing.T) {
	s := newTestService(t)
	v, ok, err := s.Get(context.Background(), 1, "email")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if ok || v != "" {
		t.Fatalf("got %q,%v want empty,false", v, ok)
	}
}

func TestSetGetUpsert(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	if err := s.Set(ctx, 1, "email", "on"); err != nil {
		t.Fatalf("set: %v", err)
	}
	v, ok, err := s.Get(ctx, 1, "email")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok || v != "on" {
		t.Fatalf("got %q,%v want on,true", v, ok)
	}

	if err := s.Set(ctx, 1, "email", "off"); err != nil {
		t.Fatalf("set upsert: %v", err)
	}
	v, ok, err = s.Get(ctx, 1, "email")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok || v != "off" {
		t.Fatalf("got %q,%v want off,true", v, ok)
	}
}

func TestAllAndDelete(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	if err := s.Set(ctx, 1, "email", "on"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := s.Set(ctx, 1, "push", "off"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := s.Set(ctx, 2, "email", "off"); err != nil {
		t.Fatalf("set: %v", err)
	}

	all, err := s.All(ctx, 1)
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if len(all) != 2 || all["email"] != "on" || all["push"] != "off" {
		t.Fatalf("all = %v", all)
	}

	if err := s.Delete(ctx, 1, "email"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok, err := s.Get(ctx, 1, "email"); err != nil || ok {
		t.Fatalf("expected email deleted, got ok=%v err=%v", ok, err)
	}
}
