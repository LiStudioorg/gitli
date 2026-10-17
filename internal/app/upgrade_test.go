// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/migrations"
)

// newTestConfig returns an isolated SQLite configuration rooted in dir.
func newTestConfig(t *testing.T, dir string) *config.Config {
	t.Helper()
	cfg := config.Defaults()
	cfg.Database.Driver = "sqlite"
	cfg.Database.DSN = dir + "/gitli.db"
	cfg.Storage.Repos = dir + "/repos"
	cfg.Storage.LFS = dir + "/lfs"
	cfg.Storage.Packages = dir + "/packages"
	cfg.Storage.Avatars = dir + "/avatars"
	cfg.Storage.Attachments = dir + "/attachments"
	cfg.Server.SSHEnabled = false
	return cfg
}

func openTestDB(t *testing.T, cfg *config.Config) *db.DB {
	t.Helper()
	d, err := db.Open(&cfg.Database)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

// TestUpgradeFromPartialSchema simulates overwriting an older binary with a
// newer one: the database already has some tables and rows, and startup must
// apply only the missing migrations without losing existing data.
func TestUpgradeFromPartialSchema(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	cfg := newTestConfig(t, dir)

	paths, err := migrations.All(db.DialectSQLite)
	if err != nil {
		t.Fatal(err)
	}
	all, err := db.LoadMigrations(migrations.FS(), paths)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) < 3 {
		t.Fatalf("expected at least 3 migrations, got %d", len(all))
	}

	// "Old version": apply only the first half of the migration set.
	old := openTestDB(t, cfg)
	half := len(all) / 2
	if _, err := old.ApplyMigrations(ctx, all[:half]); err != nil {
		t.Fatalf("apply old migrations: %v", err)
	}
	if _, err := old.ExecContext(ctx, `INSERT INTO users (username, email, password_hash) VALUES ('legacy', 'legacy@example.com', 'x')`); err != nil {
		t.Fatalf("insert legacy user: %v", err)
	}
	var before int
	if err := old.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	_ = old.Close()

	// "New version": full startup migration path.
	a, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = a.Close() })
	if err := a.Migrate(ctx); err != nil {
		t.Fatalf("upgrade migrate: %v", err)
	}

	var after int
	if err := a.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != len(all) {
		t.Fatalf("schema_migrations = %d, want %d", after, len(all))
	}
	if after <= before {
		t.Fatalf("expected new migrations to be applied (%d -> %d)", before, after)
	}

	// New tables/columns from later migrations must exist.
	var n int
	if err := a.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM permissions`).Scan(&n); err != nil {
		t.Fatalf("permissions table missing after upgrade: %v", err)
	}
	var colCount int
	if err := a.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('activities') WHERE name = 'repo_name'`).Scan(&colCount); err != nil {
		t.Fatal(err)
	}
	if colCount != 1 {
		t.Fatal("activities.repo_name column missing after upgrade")
	}

	// Existing data survives the upgrade.
	var username string
	if err := a.DB.QueryRowContext(ctx, `SELECT username FROM users WHERE email = 'legacy@example.com'`).Scan(&username); err != nil {
		t.Fatalf("legacy user lost: %v", err)
	}
	if username != "legacy" {
		t.Fatalf("legacy user = %q, want legacy", username)
	}

	// Re-running is a no-op and must report zero newly applied migrations.
	applied, err := a.DB.ApplyMigrations(ctx, all)
	if err != nil {
		t.Fatalf("re-run migrations: %v", err)
	}
	if applied != 0 {
		t.Fatalf("re-run applied %d migrations, want 0", applied)
	}
}

// TestStartupCreatesDatabaseFromScratch covers the "double click and run"
// path: no database file exists, so startup must create and migrate it.
func TestStartupCreatesDatabaseFromScratch(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	cfg := newTestConfig(t, dir)

	a, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = a.Close() })
	if err := a.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var tables int
	if err := a.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table'`).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables < 20 {
		t.Fatalf("only %d tables created, expected the full schema", tables)
	}

	initd, err := a.Deps.Auth.IsInitialized(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if initd {
		t.Fatal("fresh database must not be initialized")
	}
}
