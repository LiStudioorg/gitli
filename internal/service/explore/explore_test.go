// SPDX-License-Identifier: AGPL-3.0
package explore

import (
	"context"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const schema = `
CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT);
CREATE TABLE orgs (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT);
CREATE TABLE repos (id INTEGER PRIMARY KEY AUTOINCREMENT, owner_id INTEGER, owner_type TEXT DEFAULT 'user', name TEXT, description TEXT DEFAULT '', is_private INTEGER DEFAULT 0, archived INTEGER DEFAULT 0, stars_count INTEGER DEFAULT 0, forks_count INTEGER DEFAULT 0, deleted_at DATETIME);
CREATE TABLE repo_topics (id INTEGER PRIMARY KEY AUTOINCREMENT, repo_id INTEGER, topic TEXT);
CREATE TABLE repo_languages (id INTEGER PRIMARY KEY AUTOINCREMENT, repo_id INTEGER, language TEXT, bytes INTEGER DEFAULT 0);
CREATE TABLE trending_cache (repo_id INTEGER PRIMARY KEY, score REAL DEFAULT 0, computed_at DATETIME);
`

func newTestService(t *testing.T) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: "file:explore?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(schema); err != nil {
		t.Fatal(err)
	}
	return New(store.New(database))
}

func TestTrendingAndTopics(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	_, _ = s.st.DB.Exec(`INSERT INTO users (id, username) VALUES (1, 'alice')`)
	_, _ = s.st.DB.Exec(`INSERT INTO repos (id, owner_id, name, description, stars_count, forks_count) VALUES (1, 1, 'hot', 'popular', 100, 10)`)
	_, _ = s.st.DB.Exec(`INSERT INTO repos (id, owner_id, name, stars_count) VALUES (2, 1, 'cold', 1)`)
	_, _ = s.st.DB.Exec(`INSERT INTO repo_topics (repo_id, topic) VALUES (1, 'golang')`)
	_, _ = s.st.DB.Exec(`INSERT INTO repo_languages (repo_id, language, bytes) VALUES (1, 'Go', 100)`)

	n, err := s.RecomputeTrending(ctx)
	if err != nil || n != 2 {
		t.Fatalf("recompute = %d %v", n, err)
	}
	trending, err := s.Trending(ctx, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(trending) != 2 || trending[0].Name != "hot" || trending[0].Owner != "alice" {
		t.Fatalf("trending = %+v", trending)
	}
	// Language filter.
	goOnly, _ := s.Trending(ctx, "Go", 10)
	if len(goOnly) != 1 || goOnly[0].Name != "hot" {
		t.Fatalf("Go filter = %+v", goOnly)
	}
	// Topic.
	byTopic, _ := s.ByTopic(ctx, "golang", "", 10)
	if len(byTopic) != 1 || byTopic[0].Name != "hot" {
		t.Fatalf("topic = %+v", byTopic)
	}
	topics, _ := s.TopTopics(ctx, 10)
	if len(topics) != 1 || topics[0].Topic != "golang" || topics[0].Count != 1 {
		t.Fatalf("top topics = %+v", topics)
	}
}
