// SPDX-License-Identifier: AGPL-3.0
package globalsearch

import (
	"context"
	"reflect"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const schema = `
CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT, full_name TEXT DEFAULT '');
CREATE TABLE orgs (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, display_name TEXT DEFAULT '');
CREATE TABLE repos (id INTEGER PRIMARY KEY AUTOINCREMENT, owner_id INTEGER, owner_type TEXT DEFAULT 'user', name TEXT, description TEXT DEFAULT '', is_private INTEGER DEFAULT 0, stars_count INTEGER DEFAULT 0, deleted_at DATETIME);
CREATE TABLE repo_languages (id INTEGER PRIMARY KEY AUTOINCREMENT, repo_id INTEGER, language TEXT, bytes INTEGER DEFAULT 0);
CREATE TABLE issues (id INTEGER PRIMARY KEY AUTOINCREMENT, repo_id INTEGER, number INTEGER, title TEXT, state TEXT DEFAULT 'open');
CREATE TABLE pulls (id INTEGER PRIMARY KEY AUTOINCREMENT, repo_id INTEGER, number INTEGER, title TEXT, state TEXT DEFAULT 'open');
`

func newTestService(t *testing.T) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: "file:gsearch?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(schema); err != nil {
		t.Fatal(err)
	}
	return New(store.New(database))
}

func TestParseQuery(t *testing.T) {
	q := Parse("golang server language:go is:open author:alice org:acme stars:>10")
	if !reflect.DeepEqual(q.Terms, []string{"golang", "server"}) {
		t.Fatalf("terms = %v", q.Terms)
	}
	if q.Language != "go" || q.State != "open" || q.Author != "alice" || q.Org != "acme" || q.StarsGt != 10 {
		t.Fatalf("qualifiers = %+v", q)
	}
}

func TestSearch(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	_, _ = s.st.DB.Exec(`INSERT INTO users (id, username, full_name) VALUES (1, 'alice', 'Alice')`)
	_, _ = s.st.DB.Exec(`INSERT INTO repos (id, owner_id, name, description, stars_count) VALUES (1, 1, 'gitli', 'a git service', 42)`)
	_, _ = s.st.DB.Exec(`INSERT INTO issues (repo_id, number, title, state) VALUES (1, 1, 'bug in search', 'open')`)
	_, _ = s.st.DB.Exec(`INSERT INTO pulls (repo_id, number, title, state) VALUES (1, 1, 'add search', 'open')`)

	res, err := s.Search(ctx, "search", 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Issues) != 2 {
		t.Fatalf("expected 2 issues/pulls, got %d", len(res.Issues))
	}
	if len(res.Users) != 0 {
		t.Fatalf("expected 0 users for 'search', got %d", len(res.Users))
	}

	// Repo search by name.
	res, _ = s.Search(ctx, "gitli", 30)
	if len(res.Repos) != 1 || res.Repos[0].Owner != "alice" {
		t.Fatalf("repo search = %+v", res.Repos)
	}

	// stars filter.
	res, _ = s.Search(ctx, "gitli stars:>10", 30)
	if len(res.Repos) != 1 {
		t.Fatalf("stars filter should match, got %d", len(res.Repos))
	}
	res, _ = s.Search(ctx, "gitli stars:>100", 30)
	if len(res.Repos) != 0 {
		t.Fatalf("stars filter too high should match nothing, got %d", len(res.Repos))
	}
}
