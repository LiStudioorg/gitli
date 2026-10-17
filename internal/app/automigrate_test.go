// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/service/auth"
)

// TestServeAutoMigratesFreshDatabase ensures a brand-new database gets its
// schema and RBAC seed without running `gitli migrate` first.
func TestServeAutoMigratesFreshDatabase(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Defaults()
	cfg.Database.DSN = dir + "/fresh.db"
	cfg.Storage.Repos = dir + "/repos"
	cfg.Storage.LFS = dir + "/lfs"
	cfg.Storage.Packages = dir + "/packages"
	cfg.Storage.Avatars = dir + "/avatars"
	cfg.Storage.Attachments = dir + "/attachments"
	cfg.Server.SSHEnabled = false

	a, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = a.Close() })

	// Serve performs the migration before wiring listeners; exercising just
	// that prefix is enough to prove a fresh database becomes usable.
	if err := a.Migrate(context.Background()); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}

	var count int
	if err := a.DB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM permissions").Scan(&count); err != nil {
		t.Fatalf("permissions table missing: %v", err)
	}
	if count == 0 {
		t.Fatal("expected seeded permissions")
	}

	initd, err := a.Deps.Auth.IsInitialized(context.Background())
	if err != nil {
		t.Fatalf("IsInitialized: %v", err)
	}
	if initd {
		t.Fatal("fresh database must not be initialized")
	}

	u, err := a.Deps.Auth.Setup(context.Background(), authSetupParams("admin", "admin@example.com", "adminpass123"))
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if !u.IsAdmin {
		t.Fatal("first user must be an administrator")
	}
	if _, err := a.Deps.Auth.Setup(context.Background(), authSetupParams("second", "s@example.com", "adminpass123")); err == nil {
		t.Fatal("second Setup must fail")
	}
}

func authSetupParams(username, email, password string) auth.SetupParams {
	return auth.SetupParams{Username: username, Email: email, Password: password}
}
