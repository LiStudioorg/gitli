// SPDX-License-Identifier: AGPL-3.0
package pull

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
	ddl := `CREATE TABLE review_comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		review_id INTEGER NOT NULL,
		path VARCHAR(1024) NOT NULL DEFAULT '',
		line INTEGER NOT NULL DEFAULT 0,
		body TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := database.Exec(ddl); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return New(store.New(database))
}

func TestAddAndListReviewComments(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	c1, err := s.AddReviewComment(ctx, 5, "main.go", 12, "nit: rename")
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if c1.ID == 0 || c1.ReviewID != 5 || c1.Path != "main.go" || c1.Line != 12 {
		t.Fatalf("unexpected comment: %+v", c1)
	}
	if _, err := s.AddReviewComment(ctx, 5, "main.go", 40, "lgtm"); err != nil {
		t.Fatalf("add second: %v", err)
	}
	if _, err := s.AddReviewComment(ctx, 6, "other.go", 1, "different review"); err != nil {
		t.Fatalf("add other review: %v", err)
	}

	comments, err := s.ListReviewComments(ctx, 5)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("len = %d, want 2", len(comments))
	}
	if comments[0].Body != "nit: rename" || comments[1].Body != "lgtm" {
		t.Fatalf("unexpected order: %+v", comments)
	}
}
