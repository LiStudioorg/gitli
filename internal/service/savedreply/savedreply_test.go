// SPDX-License-Identifier: AGPL-3.0
package savedreply

import (
	"context"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const schema = `CREATE TABLE saved_replies (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER, title TEXT, body TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);`

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

func TestSavedReplyCRUD(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	r, err := s.Create(ctx, 1, "LGTM", "Looks good to me")
	if err != nil {
		t.Fatal(err)
	}
	if r.Title != "LGTM" {
		t.Fatalf("title = %q", r.Title)
	}
	if _, err := s.Create(ctx, 1, "", "x"); err == nil {
		t.Fatal("empty title should fail")
	}
	list, err := s.List(ctx, 1)
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %v %v", list, err)
	}
	if err := s.Update(ctx, 1, r.ID, "LGTM2", "updated"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.ByID(ctx, r.ID)
	if got.Title != "LGTM2" || got.Body != "updated" {
		t.Fatalf("after update = %+v", got)
	}
	// Other user cannot update.
	if err := s.Update(ctx, 2, r.ID, "x", "y"); err != ErrNotFound {
		t.Fatalf("cross-user update = %v", err)
	}
	if err := s.Delete(ctx, 1, r.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ByID(ctx, r.ID); err != ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}
