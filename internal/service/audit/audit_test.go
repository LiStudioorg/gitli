// SPDX-License-Identifier: AGPL-3.0
package audit

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const auditSchema = `
CREATE TABLE audit_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  actor_id INTEGER, action VARCHAR(64) NOT NULL DEFAULT '',
  target_type VARCHAR(32) NOT NULL DEFAULT '', target_id INTEGER,
  ip VARCHAR(64) NOT NULL DEFAULT '', detail TEXT NOT NULL DEFAULT '',
  actor_type VARCHAR(16) NOT NULL DEFAULT 'user', actor_ip VARCHAR(64) NOT NULL DEFAULT '',
  actor_user_agent VARCHAR(512) NOT NULL DEFAULT '', event VARCHAR(128) NOT NULL DEFAULT '',
  resource VARCHAR(64) NOT NULL DEFAULT '', target_name VARCHAR(512) NOT NULL DEFAULT '',
  old_value TEXT NOT NULL DEFAULT '', new_value TEXT NOT NULL DEFAULT '',
  result VARCHAR(16) NOT NULL DEFAULT 'success', error_message TEXT NOT NULL DEFAULT '',
  request_id VARCHAR(64) NOT NULL DEFAULT '', metadata TEXT NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE audit_logs_archive (
  id INTEGER PRIMARY KEY AUTOINCREMENT, orig_id INTEGER,
  actor_id INTEGER, actor_type VARCHAR(16) DEFAULT 'user', actor_ip VARCHAR(64) DEFAULT '',
  actor_user_agent VARCHAR(512) DEFAULT '', event VARCHAR(128) DEFAULT '', resource VARCHAR(64) DEFAULT '',
  action VARCHAR(64) DEFAULT '', target_type VARCHAR(32) DEFAULT '', target_id INTEGER,
  target_name VARCHAR(512) DEFAULT '', old_value TEXT DEFAULT '', new_value TEXT DEFAULT '',
  result VARCHAR(16) DEFAULT 'success', error_message TEXT DEFAULT '', request_id VARCHAR(64) DEFAULT '',
  metadata TEXT DEFAULT '', created_at DATETIME, archived_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE audit_alert_rules (
  id INTEGER PRIMARY KEY AUTOINCREMENT, name VARCHAR(255), event VARCHAR(128),
  threshold INTEGER DEFAULT 10, window_seconds INTEGER DEFAULT 300,
  channel VARCHAR(16) DEFAULT 'email', target VARCHAR(1024) DEFAULT '',
  enabled INTEGER DEFAULT 1, last_fired_at DATETIME, created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

func newTestService(t *testing.T) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(auditSchema); err != nil {
		t.Fatal(err)
	}
	return New(store.New(database))
}

func TestRecordAndQuery(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	uid := int64(7)
	if err := s.Record(ctx, Entry{ActorID: &uid, Event: "repo.delete", TargetType: "repo", IP: "1.2.3.4", Result: ResultSuccess}); err != nil {
		t.Fatal(err)
	}
	if err := s.Record(ctx, Entry{ActorID: &uid, Event: "auth.login.failure", Result: ResultFailure, IP: "1.2.3.5"}); err != nil {
		t.Fatal(err)
	}
	logs, err := s.Query(ctx, Query{Event: "repo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].Event != "repo.delete" {
		t.Fatalf("query by event: %+v", logs)
	}
	if logs[0].Resource != "repo" {
		t.Fatalf("resource derived wrong: %q", logs[0].Resource)
	}
	denied, err := s.Query(ctx, Query{Result: ResultFailure})
	if err != nil || len(denied) != 1 {
		t.Fatalf("query by result: %v %d", err, len(denied))
	}
	byActor, err := s.Query(ctx, Query{ActorID: &uid})
	if err != nil || len(byActor) != 2 {
		t.Fatalf("query by actor: %v %d", err, len(byActor))
	}
}

func TestExport(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_ = s.Record(ctx, Entry{Event: "user.create", Result: ResultSuccess})
	}
	var csv bytes.Buffer
	n, err := s.Stream(ctx, Query{}, "csv", &csv)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 || !strings.Contains(csv.String(), "user.create") {
		t.Fatalf("csv export n=%d body=%s", n, csv.String())
	}
	var js bytes.Buffer
	if _, err := s.Stream(ctx, Query{}, "json", &js); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(js.String(), "[\n") || !strings.Contains(js.String(), "user.create") {
		t.Fatalf("json export=%s", js.String())
	}
}

func TestStatsAndArchive(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	_ = s.Record(ctx, Entry{Event: "git.push"})
	stats, err := s.StatsSince(ctx, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.ByEvent) == 0 {
		t.Fatal("expected by_event stats")
	}
	// Move records into the past, then archive with a 1-day retention.
	if _, err := s.st.DB.ExecContext(ctx, `UPDATE audit_logs SET created_at = ?`, time.Now().AddDate(0, 0, -10)); err != nil {
		t.Fatal(err)
	}
	n, err := s.Archive(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("expected rows archived")
	}
	after, _ := s.Query(ctx, Query{})
	if len(after) != 0 {
		t.Fatalf("expected live table empty, got %d", len(after))
	}
}

func TestAlertRules(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	rule, err := s.CreateAlertRule(ctx, AlertRule{Name: "bruteforce", Event: "auth.login.failure", Threshold: 2, WindowSec: 300, Channel: "email", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		_ = s.Record(ctx, Entry{Event: "auth.login.failure", Result: ResultFailure})
	}
	fired, err := s.EvaluateAlerts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(fired) != 1 || fired[0].Rule.ID != rule.ID || fired[0].Count < 2 {
		t.Fatalf("expected rule to fire: %+v", fired)
	}
	if err := s.DeleteAlertRule(ctx, rule.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AlertRuleByID(ctx, rule.ID); err != ErrAlertNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}
