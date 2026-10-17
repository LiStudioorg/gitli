// SPDX-License-Identifier: AGPL-3.0
package project

import (
	"context"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const schema = `
CREATE TABLE projects (id INTEGER PRIMARY KEY AUTOINCREMENT, owner_id INTEGER, repo_id INTEGER, name TEXT, description TEXT DEFAULT '', created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE project_columns (id INTEGER PRIMARY KEY AUTOINCREMENT, project_id INTEGER, name TEXT, position INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE project_cards (id INTEGER PRIMARY KEY AUTOINCREMENT, column_id INTEGER, issue_id INTEGER, pull_id INTEGER, title TEXT DEFAULT '', position INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
`

func newTestService(t *testing.T) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: "file:project?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(schema); err != nil {
		t.Fatal(err)
	}
	return New(store.New(database))
}

func TestBoardLifecycle(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	repoID := int64(1)
	p, err := s.Create(ctx, 5, &repoID, "Sprint", "sprint board")
	if err != nil {
		t.Fatal(err)
	}
	board, cols, err := s.Board(ctx, p.ID)
	if err != nil || board.Name != "Sprint" {
		t.Fatalf("board = %v %v", board, err)
	}
	if len(cols) != 3 || cols[0].Name != "To Do" || cols[2].Name != "Done" {
		t.Fatalf("columns = %+v", cols)
	}
	// Add a card to To Do.
	card, err := s.AddCard(ctx, cols[0].ID, ptr(42), nil, "Issue 42")
	if err != nil {
		t.Fatal(err)
	}
	// Move to Done.
	if err := s.MoveCard(ctx, card.ID, cols[2].ID, 0); err != nil {
		t.Fatal(err)
	}
	_, cols2, _ := s.Board(ctx, p.ID)
	if len(cols2[0].Cards) != 0 || len(cols2[2].Cards) != 1 {
		t.Fatalf("after move: todo=%d done=%d", len(cols2[0].Cards), len(cols2[2].Cards))
	}
	if err := s.DeleteCard(ctx, card.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ByID(ctx, p.ID); err != ErrNotFound {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func ptr(v int64) *int64 { return &v }
