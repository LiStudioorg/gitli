// SPDX-License-Identifier: AGPL-3.0

// Package test contains end-to-end integration tests.
//
// These tests exercise real components (config loading, database migration,
// RBAC seeding and git repository provisioning) without spinning up the HTTP
// listener. They are guarded by a build tag so that plain `go test ./...`
// skips them and does not depend on a local git binary or writable temp state.
//
// Run with:
//
//	go test -tags integration ./test/
//
// SHA-256 is stubbed in the sandbox, so no token/hash/crypto value is asserted;
// only structural facts (row counts, filesystem existence, RBAC evaluation)
// are checked.
package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/LiStudioorg/gitli/internal/app"
	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/rbac"
	"github.com/LiStudioorg/gitli/internal/service/repo"
	"github.com/LiStudioorg/gitli/internal/service/user"
)

const testConfig = `[server]
http_listen = ["127.0.0.1:0"]
http_external_url = "http://127.0.0.1"
ssh_enabled = false
allow_root = true

[database]
driver = "sqlite"
dsn = "%s"

[storage]
repos = "%s"
lfs = "%s"
packages = "%s"
avatars = "%s"
attachments = "%s"

[log]
level = "error"
format = "text"
`

func TestEndToEnd(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()

	cfgPath := filepath.Join(root, "config.toml")
	body := fmt.Sprintf(testConfig,
		filepath.Join(root, "gitli.db"),
		filepath.Join(root, "repos"),
		filepath.Join(root, "lfs"),
		filepath.Join(root, "packages"),
		filepath.Join(root, "avatars"),
		filepath.Join(root, "attachments"),
	)
	if err := os.WriteFile(cfgPath, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	a, err := app.New(ctx, cfg, cfgPath)
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	t.Cleanup(func() { _ = a.Close() })

	if err := a.Migrate(ctx); err != nil {
		t.Fatalf("app.Migrate: %v", err)
	}

	var permCount int
	if err := a.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM permissions`).Scan(&permCount); err != nil {
		t.Fatalf("count permissions: %v", err)
	}
	if permCount == 0 {
		t.Fatal("expected seeded permissions, got 0")
	}
	t.Logf("seeded permissions: %d", permCount)

	owner, err := a.Deps.Users.Create(ctx, user.CreateParams{
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "stubbed",
	})
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	member, err := a.Deps.Users.Create(ctx, user.CreateParams{
		Username:     "bob",
		Email:        "bob@example.com",
		PasswordHash: "stubbed",
	})
	if err != nil {
		t.Fatalf("create member: %v", err)
	}

	o, err := a.Deps.Orgs.Create(ctx, "acme", "Acme Corp", owner.ID)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	if _, err := a.Deps.Orgs.CreateTeam(ctx, o.ID, "engineering", domain.PermRead); err != nil {
		t.Fatalf("create org team: %v", err)
	}

	readRole := builtinRoleID(t, a, "read")
	parent, err := a.Deps.RBAC.CreateTeam(ctx, rbac.CreateTeamParams{
		OrgID:  o.ID,
		Name:   "engineering-rbac",
		RoleID: &readRole,
	})
	if err != nil {
		t.Fatalf("create rbac parent team: %v", err)
	}
	if _, err := a.Deps.RBAC.CreateTeam(ctx, rbac.CreateTeamParams{
		OrgID:        o.ID,
		Name:         "backend",
		ParentTeamID: &parent,
		Permission:   "read",
	}); err != nil {
		t.Fatalf("create nested team: %v", err)
	}
	if err := a.Deps.RBAC.AddTeamMember(ctx, owner.ID, parent, member.ID, "member"); err != nil {
		t.Fatalf("add team member: %v", err)
	}

	perms, err := a.Deps.RBAC.EffectivePermissions(ctx, member.ID, 0)
	if err != nil {
		t.Fatalf("effective permissions: %v", err)
	}
	if !perms[rbac.Permission{Resource: "repo", Action: "read"}] {
		t.Fatalf("expected repo:read for team member, got %v", perms)
	}

	r, err := a.Deps.Repos.Create(ctx, repo.CreateParams{
		OwnerID:       owner.ID,
		OwnerType:     domain.OwnerUser,
		Name:          "demo",
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	repoPath := a.Deps.FS.Path("alice", "demo")
	info, err := os.Stat(repoPath)
	if err != nil {
		t.Fatalf("bare repo dir missing at %s: %v", repoPath, err)
	}
	if !info.IsDir() {
		t.Fatalf("expected directory for bare repo, got file: %s", repoPath)
	}
	if !a.Deps.FS.Exists("alice", "demo") {
		t.Fatalf("layout reports repo missing: %s", repoPath)
	}
	hook := filepath.Join(repoPath, "hooks", "pre-receive")
	if _, err := os.Stat(hook); err != nil {
		t.Fatalf("pre-receive hook not installed at %s: %v", hook, err)
	}
	t.Logf("repo %d provisioned at %s", r.ID, repoPath)
}

func builtinRoleID(t *testing.T, a *app.App, name string) int64 {
	t.Helper()
	var id int64
	q := a.DB.Rebind(`SELECT id FROM roles WHERE name = ? AND org_id IS NULL`)
	if err := a.DB.QueryRowContext(context.Background(), q, name).Scan(&id); err != nil {
		t.Fatalf("lookup role %s: %v", name, err)
	}
	return id
}
