// SPDX-License-Identifier: AGPL-3.0
package org2fa

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
	ddl := `CREATE TABLE org_2fa_policies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		org_id INTEGER UNIQUE,
		required INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := database.Exec(ddl); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return New(store.New(database))
}

func TestGetDefault(t *testing.T) {
	s := newTestService(t)
	required, err := s.Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if required {
		t.Fatal("expected default false")
	}
}

func TestSetGetUpsert(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	if err := s.Set(ctx, 1, true); err != nil {
		t.Fatalf("set: %v", err)
	}
	required, err := s.Get(ctx, 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !required {
		t.Fatal("expected required true")
	}

	if err := s.Set(ctx, 1, false); err != nil {
		t.Fatalf("set upsert: %v", err)
	}
	required, err = s.Get(ctx, 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if required {
		t.Fatal("expected required false after upsert")
	}
}
