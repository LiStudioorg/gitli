// SPDX-License-Identifier: AGPL-3.0
// Package rbac implements the resource×action permission model, nested teams,
// custom roles and a cached permission checker.
package rbac

import "fmt"

// Permission identifies a resource and an action.
type Permission struct {
	Resource string
	Action   string
}

// String renders a permission as "resource:action".
func (p Permission) String() string { return p.Resource + ":" + p.Action }

// ParsePermission parses "resource:action".
func ParsePermission(s string) (Permission, error) {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return Permission{Resource: s[:i], Action: s[i+1:]}, nil
		}
	}
	return Permission{}, fmt.Errorf("rbac: invalid permission %q", s)
}

// MatchPattern performs simple glob matching supporting '*' and '?'.
func MatchPattern(pattern, name string) bool {
	px, sx := 0, 0
	nextPx, nextSx := -1, 0
	for sx < len(name) {
		if px < len(pattern) && (pattern[px] == '?' || pattern[px] == name[sx]) {
			px++
			sx++
		} else if px < len(pattern) && pattern[px] == '*' {
			nextPx = px
			nextSx = sx + 1
			px++
		} else if nextPx != -1 {
			px = nextPx + 1
			sx = nextSx
			nextSx++
		} else {
			return false
		}
	}
	for px < len(pattern) && pattern[px] == '*' {
		px++
	}
	return px == len(pattern)
}

// Requirement is a permission check request against a repo/org scope.
type Requirement struct {
	Resource string
	Action   string
	RepoID   int64
	OrgID    int64
}

// BuiltinPermissions is the canonical list of system permissions.
var BuiltinPermissions = []Permission{
	{"repo", "read"}, {"repo", "write"}, {"repo", "admin"}, {"repo", "maintain"}, {"repo", "triage"},
	{"contents", "read"}, {"contents", "write"},
	{"issues", "read"}, {"issues", "write"}, {"issues", "triage"},
	{"pulls", "read"}, {"pulls", "write"}, {"pulls", "review"}, {"pulls", "merge"},
	{"actions", "read"}, {"actions", "write"}, {"actions", "run"},
	{"packages", "read"}, {"packages", "write"}, {"packages", "delete"},
	{"wiki", "read"}, {"wiki", "write"},
	{"settings", "read"}, {"settings", "write"}, {"settings", "delete"},
	{"webhooks", "read"}, {"webhooks", "write"},
	{"members", "read"}, {"members", "invite"}, {"members", "remove"},
	{"org", "read"}, {"org", "write"}, {"org", "admin"}, {"org", "billing"},
	{"team", "read"}, {"team", "write"}, {"team", "manage_members"},
	{"audit", "read"},
}

// BuiltinRole is a named default role.
type BuiltinRole struct {
	Name        string
	Description string
	Permissions []Permission
}

// RoleNone, RoleRead, RoleTriage, RoleWrite, RoleMaintain, RoleAdmin mirror the
// conventional repository permission ladder.
var (
	RoleNone     = BuiltinRole{Name: "none", Description: "无权限"}
	RoleRead     = BuiltinRole{Name: "read", Description: "只读", Permissions: readPerms()}
	RoleTriage   = BuiltinRole{Name: "triage", Description: "分诊", Permissions: union(readPerms(), triagePerms())}
	RoleWrite    = BuiltinRole{Name: "write", Description: "读写", Permissions: union(readPerms(), triagePerms(), writePerms())}
	RoleMaintain = BuiltinRole{Name: "maintain", Description: "维护", Permissions: union(readPerms(), triagePerms(), writePerms(), maintainPerms())}
	RoleAdmin    = BuiltinRole{Name: "admin", Description: "管理", Permissions: allPerms()}
)

// BuiltinRoles is the ordered ladder used for seeding and comparisons.
var BuiltinRoles = []BuiltinRole{RoleRead, RoleTriage, RoleWrite, RoleMaintain, RoleAdmin}

// RankOf returns the index of a builtin role name in the ladder, or -1.
func RankOf(name string) int {
	switch name {
	case "read":
		return 0
	case "triage":
		return 1
	case "write":
		return 2
	case "maintain":
		return 3
	case "admin":
		return 4
	default:
		return -1
	}
}

func readPerms() []Permission {
	return []Permission{
		{"repo", "read"}, {"contents", "read"}, {"issues", "read"}, {"pulls", "read"},
		{"actions", "read"}, {"packages", "read"}, {"wiki", "read"},
		{"members", "read"}, {"org", "read"}, {"team", "read"},
	}
}

func triagePerms() []Permission {
	return []Permission{{"issues", "triage"}, {"pulls", "review"}}
}

func writePerms() []Permission {
	return []Permission{
		{"repo", "write"}, {"contents", "write"}, {"issues", "write"}, {"pulls", "write"},
		{"actions", "write"}, {"actions", "run"}, {"packages", "write"}, {"wiki", "write"},
		{"webhooks", "read"}, {"webhooks", "write"},
	}
}

func maintainPerms() []Permission {
	return []Permission{
		{"repo", "maintain"}, {"pulls", "merge"}, {"settings", "read"}, {"members", "invite"},
	}
}

func allPerms() []Permission {
	out := make([]Permission, len(BuiltinPermissions))
	copy(out, BuiltinPermissions)
	return out
}

func union(sets ...[]Permission) []Permission {
	seen := map[Permission]bool{}
	var out []Permission
	for _, s := range sets {
		for _, p := range s {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	return out
}
