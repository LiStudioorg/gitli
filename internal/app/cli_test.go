// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Defaults()
	cfg.Database.DSN = dir + "/test.db"
	cfg.Storage.Repos = dir + "/repos"
	cfg.Storage.LFS = dir + "/lfs"
	cfg.Storage.Packages = dir + "/packages"
	cfg.Storage.Avatars = dir + "/avatars"
	cfg.Storage.Attachments = dir + "/attachments"
	cfg.Server.SSHEnabled = false
	a, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	return a
}

func TestCLIUserLifecycle(t *testing.T) {
	a := newTestApp(t)
	ctx := context.Background()

	u, pw, err := a.CLIUserCreate(ctx, "alice", "alice@example.com", "supersecret123", true)
	if err != nil {
		t.Fatal(err)
	}
	if pw != "" {
		t.Fatal("explicit password should not be echoed")
	}
	if !u.IsAdmin {
		t.Fatal("expected admin")
	}

	// Last-admin protection: cannot revoke the only admin.
	if err := a.CLIUserSetAdmin(ctx, "alice", false); err == nil {
		t.Fatal("should refuse to remove the last admin")
	}
	if err := a.CLIUserDisable(ctx, "alice"); err == nil {
		t.Fatal("should refuse to disable the last admin")
	}

	// Create a second admin; now revocation is allowed.
	if _, _, err := a.CLIUserCreate(ctx, "bob", "bob@example.com", "supersecret123", true); err != nil {
		t.Fatal(err)
	}
	if err := a.CLIUserSetAdmin(ctx, "alice", false); err != nil {
		t.Fatalf("revoke with another admin should succeed: %v", err)
	}

	// Password change + email + TOTP reset.
	if err := a.CLIUserChangePassword(ctx, "alice", "anothersecret123"); err != nil {
		t.Fatal(err)
	}
	if err := a.CLIUserChangeEmail(ctx, "alice", "new@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := a.CLIUserResetTOTP(ctx, "alice"); err != nil {
		t.Fatal(err)
	}

	users, err := a.CLIUserList(ctx)
	if err != nil || len(users) != 2 {
		t.Fatalf("users = %d %v", len(users), err)
	}
}

func TestCLIRepoAndOrg(t *testing.T) {
	a := newTestApp(t)
	ctx := context.Background()
	if _, _, err := a.CLIUserCreate(ctx, "alice", "alice@example.com", "supersecret123", false); err != nil {
		t.Fatal(err)
	}
	o, err := a.CLIOrgCreate(ctx, "acme", "alice")
	if err != nil || o.Name != "acme" {
		t.Fatalf("org create: %v", err)
	}
	if err := a.CLIOrgAddMember(ctx, "acme", "alice", "read"); err != nil {
		t.Fatal(err)
	}
	if err := a.CLIOrgRemoveMember(ctx, "acme", "alice"); err != nil {
		t.Fatal(err)
	}
	if err := a.CLIOrgDelete(ctx, "acme"); err != nil {
		t.Fatal(err)
	}
	orgs, _ := a.Deps.Orgs.ListAll(ctx)
	if len(orgs) != 0 {
		t.Fatalf("orgs after delete = %d", len(orgs))
	}
}

func TestCLIConfigHelpers(t *testing.T) {
	secret, err := (&App{}).CLIGenerateSecret(16)
	if err != nil || len(secret) != 32 {
		t.Fatalf("secret = %q %v", secret, err)
	}
}
