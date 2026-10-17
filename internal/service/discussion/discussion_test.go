// SPDX-License-Identifier: AGPL-3.0
package discussion

import (
	"context"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const schema = `
CREATE TABLE discussion_categories (id INTEGER PRIMARY KEY AUTOINCREMENT, repo_id INTEGER, name TEXT, description TEXT DEFAULT '', is_answerable INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE discussions (id INTEGER PRIMARY KEY AUTOINCREMENT, repo_id INTEGER, category_id INTEGER, title TEXT, body TEXT DEFAULT '', author_id INTEGER, is_answered INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE discussion_comments (id INTEGER PRIMARY KEY AUTOINCREMENT, discussion_id INTEGER, author_id INTEGER, body TEXT DEFAULT '', is_answer INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
`

func newTestService(t *testing.T) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: "file:discussion?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(schema); err != nil {
		t.Fatal(err)
	}
	return New(store.New(database))
}

func TestCategoriesAndDiscussion(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	if err := s.EnsureCategories(ctx, 1); err != nil {
		t.Fatal(err)
	}
	cats, err := s.ListCategories(ctx, 1)
	if err != nil || len(cats) != 4 {
		t.Fatalf("categories = %v %v", cats, err)
	}
	var qaID int64
	for _, c := range cats {
		if c.Name == "Q&A" {
			qaID = c.ID
		}
	}
	d, err := s.Create(ctx, 1, 5, &qaID, "How do I deploy?", "question body")
	if err != nil {
		t.Fatal(err)
	}
	list, _ := s.List(ctx, 1, &qaID, 10)
	if len(list) != 1 || list[0].Title != "How do I deploy?" {
		t.Fatalf("list = %+v", list)
	}
	c, err := s.AddComment(ctx, d.ID, 6, "Use docker compose")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkAnswer(ctx, d.ID, c.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := s.ByID(ctx, d.ID)
	if !got.IsAnswered {
		t.Fatal("expected answered")
	}
	comments, _ := s.ListComments(ctx, d.ID)
	if len(comments) != 1 || !comments[0].IsAnswer {
		t.Fatalf("comments = %+v", comments)
	}
}

func TestMarkAnswerRequiresQA(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	_ = s.EnsureCategories(ctx, 1)
	cats, _ := s.ListCategories(ctx, 1)
	var generalID int64
	for _, c := range cats {
		if c.Name == "General" {
			generalID = c.ID
		}
	}
	d, _ := s.Create(ctx, 1, 1, &generalID, "chat", "x")
	c, _ := s.AddComment(ctx, d.ID, 1, "reply")
	if err := s.MarkAnswer(ctx, d.ID, c.ID); err == nil {
		t.Fatal("non-Q&A discussion must not be answerable")
	}
}
