// SPDX-License-Identifier: AGPL-3.0
package rbac

import (
	"context"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const testSchema = `
CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT);
CREATE TABLE orgs (id INTEGER PRIMARY KEY, owner_id INTEGER, name TEXT);
CREATE TABLE roles (
  id INTEGER PRIMARY KEY AUTOINCREMENT, org_id INTEGER, name TEXT,
  description TEXT DEFAULT '', is_builtin INTEGER DEFAULT 0, scope TEXT DEFAULT 'org'
);
CREATE TABLE permissions (
  id INTEGER PRIMARY KEY AUTOINCREMENT, resource TEXT, action TEXT, description TEXT DEFAULT '',
  UNIQUE(resource, action)
);
CREATE TABLE role_permissions (id INTEGER PRIMARY KEY AUTOINCREMENT, role_id INTEGER, permission_id INTEGER, UNIQUE(role_id, permission_id));
CREATE TABLE org_members (id INTEGER PRIMARY KEY AUTOINCREMENT, org_id INTEGER, user_id INTEGER, role_id INTEGER, membership TEXT DEFAULT 'member', UNIQUE(org_id, user_id));
CREATE TABLE teams (id INTEGER PRIMARY KEY AUTOINCREMENT, org_id INTEGER, name TEXT, description TEXT DEFAULT '', permission TEXT DEFAULT 'read', parent_team_id INTEGER, role_id INTEGER);
CREATE TABLE team_members (id INTEGER PRIMARY KEY AUTOINCREMENT, team_id INTEGER, user_id INTEGER, role TEXT DEFAULT 'member');
CREATE TABLE team_repos (id INTEGER PRIMARY KEY AUTOINCREMENT, team_id INTEGER, repo_id INTEGER, role_id INTEGER, permission TEXT DEFAULT 'read');
CREATE TABLE repo_collaborators (id INTEGER PRIMARY KEY AUTOINCREMENT, repo_id INTEGER, user_id INTEGER, role_id INTEGER, permission TEXT DEFAULT 'read');
CREATE TABLE branch_protection_rules (id INTEGER PRIMARY KEY AUTOINCREMENT, repo_id INTEGER, pattern TEXT, require_reviews INTEGER DEFAULT 0, require_signed INTEGER DEFAULT 0, require_ci_jobs TEXT DEFAULT '', push_roles TEXT DEFAULT '', merge_roles TEXT DEFAULT '');
CREATE TABLE permission_audit (id INTEGER PRIMARY KEY AUTOINCREMENT, actor_id INTEGER, target_type TEXT DEFAULT '', target_id INTEGER, action TEXT, old_value TEXT DEFAULT '', new_value TEXT DEFAULT '');
CREATE TABLE path_permissions (id INTEGER PRIMARY KEY AUTOINCREMENT, repo_id INTEGER, path_pattern TEXT, role_id INTEGER, permission TEXT DEFAULT 'read');
`

func testService(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(testSchema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	st := store.New(database)
	svc := New(st)
	if err := svc.Seed(context.Background()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return svc, st
}

func TestNestedTeamInheritance(t *testing.T) {
	svc, _ := testService(t)
	ctx := context.Background()

	adminRole := roleID(t, svc, "admin")
	readRole := roleID(t, svc, "read")

	parentID, err := svc.CreateTeam(ctx, CreateTeamParams{OrgID: 1, Name: "parent", RoleID: &readRole})
	if err != nil {
		t.Fatal(err)
	}
	childID, err := svc.CreateTeam(ctx, CreateTeamParams{OrgID: 1, Name: "child", ParentTeamID: &parentID, RoleID: &adminRole})
	if err != nil {
		t.Fatal(err)
	}
	// User is only a member of the child; should inherit parent's read role too.
	if err := svc.AddTeamMember(ctx, 0, childID, 100, "member"); err != nil {
		t.Fatal(err)
	}
	perms, err := svc.EffectivePermissions(ctx, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !perms[Permission{"repo", "read"}] {
		t.Fatal("expected inherited read permission from parent team")
	}
	if !perms[Permission{"settings", "delete"}] {
		t.Fatal("expected admin permission from child team")
	}
}

func TestCycleDetection(t *testing.T) {
	svc, _ := testService(t)
	ctx := context.Background()
	a, _ := svc.CreateTeam(ctx, CreateTeamParams{OrgID: 1, Name: "a"})
	b, _ := svc.CreateTeam(ctx, CreateTeamParams{OrgID: 1, Name: "b", ParentTeamID: &a})
	c, _ := svc.CreateTeam(ctx, CreateTeamParams{OrgID: 1, Name: "c", ParentTeamID: &b})
	// Reparent a under c would form a cycle a->? c->b->a.
	if err := svc.ReparentTeam(ctx, a, &c); err != ErrCycle {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestMaxDepth(t *testing.T) {
	svc, _ := testService(t)
	ctx := context.Background()
	var last int64
	first, _ := svc.CreateTeam(ctx, CreateTeamParams{OrgID: 1, Name: "t0"})
	last = first
	for i := 1; i <= MaxNestDepth; i++ {
		id, err := svc.CreateTeam(ctx, CreateTeamParams{OrgID: 1, Name: name(i), ParentTeamID: &last})
		if err != nil && i < MaxNestDepth {
			t.Fatalf("depth %d unexpected error: %v", i, err)
		}
		if err == ErrMaxDepth {
			return
		}
		last = id
	}
	extra := name(MaxNestDepth + 1)
	_, err := svc.CreateTeam(ctx, CreateTeamParams{OrgID: 1, Name: extra, ParentTeamID: &last})
	if err != ErrMaxDepth {
		t.Fatalf("expected ErrMaxDepth, got %v", err)
	}
}

func TestPermissionCacheInvalidation(t *testing.T) {
	svc, _ := testService(t)
	ctx := context.Background()
	readRole := roleID(t, svc, "read")
	team, _ := svc.CreateTeam(ctx, CreateTeamParams{OrgID: 1, Name: "t", RoleID: &readRole})
	if err := svc.AddTeamMember(ctx, 0, team, 200, "member"); err != nil {
		t.Fatal(err)
	}
	if !svc.Check(ctx, 200, false, Requirement{Resource: "repo", Action: "read"}) {
		t.Fatal("expected read permission")
	}
	if svc.Check(ctx, 200, false, Requirement{Resource: "settings", Action: "write"}) {
		t.Fatal("did not expect settings:write")
	}
	// Upgrade the user's team role and invalidate.
	adminRole := roleID(t, svc, "admin")
	svc.cache.invalidateAll()
	if _, err := svc.st.DB.ExecContext(ctx, svc.st.Rebind(`UPDATE teams SET role_id = ? WHERE id = ?`), adminRole, team); err != nil {
		t.Fatal(err)
	}
	if !svc.Check(ctx, 200, false, Requirement{Resource: "settings", Action: "write"}) {
		t.Fatal("expected settings:write after role upgrade")
	}
}

func name(i int) string {
	return "t" + string(rune('0'+i))
}

func roleID(t *testing.T, svc *Service, name string) int64 {
	t.Helper()
	var id int64
	if err := svc.st.DB.QueryRowContext(context.Background(), svc.st.Rebind(`SELECT id FROM roles WHERE name = ? AND org_id IS NULL`), name).Scan(&id); err != nil {
		t.Fatalf("role %s: %v", name, err)
	}
	return id
}
