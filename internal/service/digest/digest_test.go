// SPDX-License-Identifier: AGPL-3.0
package digest

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const schema = `
CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT, email TEXT);
CREATE TABLE user_notification_prefs (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER UNIQUE, frequency TEXT DEFAULT 'realtime', last_sent_at DATETIME, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE notifications (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER, title TEXT, url TEXT, is_read INTEGER DEFAULT 0);
`

func newTestService(t *testing.T) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: "file:digest?mode=memory&cache=shared"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(schema); err != nil {
		t.Fatal(err)
	}
	_, _ = database.Exec(`INSERT INTO users (id, username, email) VALUES (1, 'alice', 'alice@example.com')`)
	return New(store.New(database))
}

func TestFrequency(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	if f, _ := s.GetFrequency(ctx, 1); f != Realtime {
		t.Fatalf("default frequency = %s", f)
	}
	if err := s.SetFrequency(ctx, 1, Daily); err != nil {
		t.Fatal(err)
	}
	if f, _ := s.GetFrequency(ctx, 1); f != Daily {
		t.Fatalf("frequency = %s", f)
	}
	if err := s.SetFrequency(ctx, 1, "hourly"); err == nil {
		t.Fatal("invalid frequency should fail")
	}
}

func TestDueAndCandidates(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	now := time.Now()
	if !Due(Daily, nil, now) {
		t.Fatal("first daily digest should be due")
	}
	recent := now.Add(-time.Hour)
	if Due(Daily, &recent, now) {
		t.Fatal("daily digest should not be due within 24h")
	}
	if Due(Realtime, nil, now) || Due(Off, nil, now) {
		t.Fatal("realtime/off should never be due")
	}
	_ = s.SetFrequency(ctx, 1, Daily)
	cands, err := s.DueCandidates(ctx, now)
	if err != nil || len(cands) != 1 || cands[0].Email != "alice@example.com" {
		t.Fatalf("candidates = %+v %v", cands, err)
	}
	_ = s.MarkSent(ctx, 1, now)
	if c2, _ := s.DueCandidates(ctx, now); len(c2) != 0 {
		t.Fatalf("after mark sent, candidates = %+v", c2)
	}
}

func TestRenderHTML(t *testing.T) {
	html := RenderHTML("alice", "https://git.example.com", []Notification{{Title: "New issue", URL: "/o/r/issues/1"}}, "https://git.example.com/settings")
	if !strings.Contains(html, "alice") || !strings.Contains(html, "New issue") {
		t.Fatalf("html missing content: %s", html)
	}
	if !strings.Contains(html, "https://git.example.com/o/r/issues/1") {
		t.Fatalf("html missing absolute link: %s", html)
	}
}
