// SPDX-License-Identifier: AGPL-3.0
package deploykey

import (
	"context"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const schema = `
CREATE TABLE deploy_keys (id INTEGER PRIMARY KEY AUTOINCREMENT, repo_id INTEGER, title TEXT, fingerprint TEXT UNIQUE, public_key TEXT, read_only INTEGER DEFAULT 1, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, last_used_at DATETIME);
`

const testPub = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIIxSM99xAJU7rC3M4/am9dGWvRzJtRiZbw+3UaYMiqac deploy@example"

func newTestService(t *testing.T) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(schema); err != nil {
		t.Fatal(err)
	}
	return New(store.New(database))
}

func TestDeployKeyLifecycle(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	k, err := s.Add(ctx, 1, "ci", testPub, true)
	if err != nil {
		t.Fatal(err)
	}
	if !k.ReadOnly {
		t.Fatal("expected read-only key")
	}
	// Resolve maps fingerprint to repo + permission.
	repoID, readOnly, err := s.Resolve(ctx, k.Fingerprint)
	if err != nil || repoID != 1 || !readOnly {
		t.Fatalf("resolve = %d %v %v", repoID, readOnly, err)
	}
	// Toggle to read-write.
	if err := s.SetReadOnly(ctx, 1, k.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, ro, _ := s.Resolve(ctx, k.Fingerprint); ro {
		t.Fatal("expected read-write after toggle")
	}
	// List.
	keys, err := s.List(ctx, 1)
	if err != nil || len(keys) != 1 {
		t.Fatalf("list = %v %v", keys, err)
	}
	// Delete.
	if err := s.Delete(ctx, 1, k.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Resolve(ctx, k.Fingerprint); err == nil {
		t.Fatal("expected not found after delete")
	}
}

func TestDeployKeyRejectsBadKey(t *testing.T) {
	s := newTestService(t)
	if _, err := s.Add(context.Background(), 1, "bad", "not-a-key", false); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestSetReadOnlyWrongRepo(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	k, _ := s.Add(ctx, 1, "k", testPub, true)
	if err := s.SetReadOnly(ctx, 2, k.ID, false); err == nil {
		t.Fatal("expected not found for wrong repo")
	}
}
