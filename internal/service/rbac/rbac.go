// SPDX-License-Identifier: AGPL-3.0
package rbac

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Errors returned by the service.
var (
	ErrNotFound     = errors.New("rbac: not found")
	ErrCycle        = errors.New("rbac: team nesting would create a cycle")
	ErrMaxDepth     = errors.New("rbac: team nesting exceeds maximum depth")
	ErrBuiltinRole  = errors.New("rbac: builtin roles cannot be modified or deleted")
	ErrSelfEscalate = errors.New("rbac: cannot escalate own privileges")
)

// MaxNestDepth bounds team nesting.
const MaxNestDepth = 5

// Service evaluates and manages permissions.
type Service struct {
	st    *store.Store
	cache *permCache
}

// New builds the RBAC service and seeds builtin permissions/roles.
func New(st *store.Store) *Service {
	return &Service{st: st, cache: newPermCache(10000, 5*time.Minute)}
}

// Seed inserts builtin permissions and roles if absent. Safe to call repeatedly.
func (s *Service) Seed(ctx context.Context) error {
	for _, p := range BuiltinPermissions {
		q := s.st.Rebind(`INSERT INTO permissions (resource, action, description) VALUES (?, ?, '') ON CONFLICT (resource, action) DO NOTHING`)
		if _, err := s.st.DB.ExecContext(ctx, q, p.Resource, p.Action); err != nil {
			// MySQL lacks ON CONFLICT; fall back to ignore-style insert.
			if _, err2 := s.st.DB.ExecContext(ctx, s.st.Rebind(`INSERT IGNORE INTO permissions (resource, action, description) VALUES (?, ?, '')`), p.Resource, p.Action); err2 != nil {
				return fmt.Errorf("rbac: seed permission %s: %w", p.String(), err)
			}
		}
	}
	for _, r := range BuiltinRoles {
		if err := s.seedRole(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) seedRole(ctx context.Context, r BuiltinRole) error {
	var id int64
	q := s.st.Rebind(`SELECT id FROM roles WHERE org_id IS NULL AND name = ?`)
	err := s.st.DB.QueryRowContext(ctx, q, r.Name).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		ins := s.st.Rebind(`INSERT INTO roles (org_id, name, description, is_builtin, scope) VALUES (NULL, ?, ?, 1, 'repo')`)
		res, ierr := s.st.DB.ExecContext(ctx, ins, r.Name, r.Description)
		if ierr != nil {
			return fmt.Errorf("rbac: seed role %s: %w", r.Name, ierr)
		}
		id, _ = res.LastInsertId()
	} else if err != nil {
		return err
	}
	for _, p := range r.Permissions {
		var pid int64
		pq := s.st.Rebind(`SELECT id FROM permissions WHERE resource = ? AND action = ?`)
		if err := s.st.DB.QueryRowContext(ctx, pq, p.Resource, p.Action).Scan(&pid); err != nil {
			continue
		}
		link := s.st.Rebind(`INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)`)
		if _, err := s.st.DB.ExecContext(ctx, link, id, pid); err != nil {
			// ignore duplicate
			continue
		}
	}
	return nil
}

// --- team nesting ---

// CreateTeamParams describes team creation.
type CreateTeamParams struct {
	OrgID        int64
	Name         string
	Description  string
	ParentTeamID *int64
	RoleID       *int64
	Permission   string
}

// CreateTeam creates a team, validating nesting depth and cycles.
func (s *Service) CreateTeam(ctx context.Context, p CreateTeamParams) (int64, error) {
	if p.ParentTeamID != nil {
		if err := s.validateNesting(ctx, p.OrgID, 0, *p.ParentTeamID); err != nil {
			return 0, err
		}
	}
	perm := p.Permission
	if perm == "" {
		perm = "read"
	}
	q := s.st.Rebind(`INSERT INTO teams (org_id, name, description, permission, parent_team_id, role_id) VALUES (?, ?, ?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, p.OrgID, p.Name, p.Description, perm, p.ParentTeamID, p.RoleID)
	if err != nil {
		return 0, fmt.Errorf("rbac: create team: %w", err)
	}
	id, _ := res.LastInsertId()
	s.cache.invalidateAll()
	return id, nil
}

// ReparentTeam changes a team's parent, rejecting cycles and excessive depth.
func (s *Service) ReparentTeam(ctx context.Context, teamID int64, newParent *int64) error {
	if newParent != nil {
		if *newParent == teamID {
			return ErrCycle
		}
		orgID, err := s.teamOrg(ctx, teamID)
		if err != nil {
			return err
		}
		// Reject if newParent is a descendant of teamID (would form a cycle).
		desc, err := s.isDescendant(ctx, *newParent, teamID)
		if err != nil {
			return err
		}
		if desc {
			return ErrCycle
		}
		if err := s.validateNesting(ctx, orgID, teamID, *newParent); err != nil {
			return err
		}
	}
	q := s.st.Rebind(`UPDATE teams SET parent_team_id = ?, updated_at = ? WHERE id = ?`)
	if _, err := s.st.DB.ExecContext(ctx, q, newParent, time.Now().UTC(), teamID); err != nil {
		return err
	}
	s.cache.invalidateAll()
	return nil
}

// validateNesting ensures adding childTeam under parentTeam stays within an org
// and under MaxNestDepth.
func (s *Service) validateNesting(ctx context.Context, orgID, childTeam, parentTeam int64) error {
	depth := 1
	cur := parentTeam
	visited := map[int64]struct{}{}
	for cur != 0 {
		if _, seen := visited[cur]; seen {
			return ErrCycle
		}
		visited[cur] = struct{}{}
		depth++
		if depth > MaxNestDepth {
			return ErrMaxDepth
		}
		mOrg, parent, err := s.teamParent(ctx, cur)
		if err != nil {
			return err
		}
		if orgID != 0 && mOrg != orgID {
			return fmt.Errorf("rbac: parent team belongs to another org")
		}
		cur = parent
	}
	_ = childTeam
	return nil
}

// isDescendant reports whether candidate is a descendant of ancestor.
func (s *Service) isDescendant(ctx context.Context, candidate, ancestor int64) (bool, error) {
	visited := map[int64]struct{}{}
	cur := candidate
	for cur != 0 {
		if cur == ancestor {
			return true, nil
		}
		if _, seen := visited[cur]; seen {
			return false, ErrCycle
		}
		visited[cur] = struct{}{}
		_, parent, err := s.teamParent(ctx, cur)
		if err != nil {
			return false, err
		}
		cur = parent
	}
	return false, nil
}

func (s *Service) teamParent(ctx context.Context, teamID int64) (orgID int64, parent int64, err error) {
	var p sql.NullInt64
	q := s.st.Rebind(`SELECT org_id, parent_team_id FROM teams WHERE id = ?`)
	err = s.st.DB.QueryRowContext(ctx, q, teamID).Scan(&orgID, &p)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, ErrNotFound
	}
	if err != nil {
		return 0, 0, err
	}
	return orgID, p.Int64, nil
}

func (s *Service) teamOrg(ctx context.Context, teamID int64) (int64, error) {
	orgID, _, err := s.teamParent(ctx, teamID)
	return orgID, err
}

// --- permission evaluation ---

// Check evaluates whether a user satisfies a requirement.
func (s *Service) Check(ctx context.Context, userID int64, isAdmin bool, req Requirement) bool {
	if isAdmin {
		return true
	}
	key := cacheKey(userID, req)
	if v, ok := s.cache.get(key); ok {
		return v
	}
	granted, err := s.evaluate(ctx, userID, req)
	if err != nil {
		return false
	}
	s.cache.set(userID, key, granted)
	return granted
}

// evaluate computes the union of permissions from org roles, nested teams and
// direct repo grants, then tests membership of the requirement.
func (s *Service) evaluate(ctx context.Context, userID int64, req Requirement) (bool, error) {
	perms, err := s.EffectivePermissions(ctx, userID, req.RepoID)
	if err != nil {
		return false, err
	}
	return perms[Permission{Resource: req.Resource, Action: req.Action}], nil
}

// EffectivePermissions returns the union of a user's permissions for a repo
// (repoID > 0) or org (repoID == 0). System admins should be short-circuited by
// callers.
func (s *Service) EffectivePermissions(ctx context.Context, userID, repoID int64) (map[Permission]bool, error) {
	out := map[Permission]bool{}
	roleIDs := map[int64]bool{}

	// Org roles via org_members and all teams (nested, with parents).
	if repoID > 0 {
		if err := s.collectRepoRoleIDs(ctx, userID, repoID, roleIDs); err != nil {
			return nil, err
		}
	} else {
		if err := s.collectOrgRoleIDs(ctx, userID, roleIDs); err != nil {
			return nil, err
		}
	}
	// Repository-level collaborators use permission ladder, not roles.
	if repoID > 0 {
		if err := s.applyDirectPermission(ctx, userID, repoID, out); err != nil {
			return nil, err
		}
	}

	for rid := range roleIDs {
		perms, err := s.rolePermissions(ctx, rid)
		if err != nil {
			return nil, err
		}
		for _, p := range perms {
			out[p] = true
		}
	}
	return out, nil
}

func (s *Service) collectOrgRoleIDs(ctx context.Context, userID int64, roleIDs map[int64]bool) error {
	// Teams the user belongs to, plus ancestors (inheritance).
	seed, err := s.userTeams(ctx, userID)
	if err != nil {
		return err
	}
	for _, tid := range seed {
		if err := s.collectTeamAncestors(ctx, tid, roleIDs, map[int64]struct{}{}); err != nil {
			return err
		}
	}
	// Direct org membership roles.
	q := s.st.Rebind(`SELECT role_id FROM org_members WHERE user_id = ? AND role_id IS NOT NULL`)
	rows, err := s.st.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var rid int64
		if err := rows.Scan(&rid); err != nil {
			return err
		}
		roleIDs[rid] = true
	}
	return rows.Err()
}

func (s *Service) collectRepoRoleIDs(ctx context.Context, userID, repoID int64, roleIDs map[int64]bool) error {
	seed, err := s.userTeams(ctx, userID)
	if err != nil {
		return err
	}
	for _, tid := range seed {
		if err := s.collectTeamAncestors(ctx, tid, roleIDs, map[int64]struct{}{}); err != nil {
			return err
		}
		// Team-repo grants.
		var roleID sql.NullInt64
		q := s.st.Rebind(`SELECT role_id FROM team_repos WHERE team_id = ? AND repo_id = ?`)
		if err := s.st.DB.QueryRowContext(ctx, q, tid, repoID).Scan(&roleID); err == nil && roleID.Valid {
			roleIDs[roleID.Int64] = true
		}
	}
	return nil
}

// collectTeamAncestors adds a team's role and walks up to its parents.
func (s *Service) collectTeamAncestors(ctx context.Context, teamID int64, roleIDs map[int64]bool, visited map[int64]struct{}) error {
	if _, seen := visited[teamID]; seen {
		return nil
	}
	visited[teamID] = struct{}{}
	var roleID sql.NullInt64
	_, parent, err := s.teamParent(ctx, teamID)
	if err != nil {
		return err
	}
	q := s.st.Rebind(`SELECT role_id FROM teams WHERE id = ?`)
	if err := s.st.DB.QueryRowContext(ctx, q, teamID).Scan(&roleID); err == nil && roleID.Valid {
		roleIDs[roleID.Int64] = true
	}
	if parent != 0 {
		return s.collectTeamAncestors(ctx, parent, roleIDs, visited)
	}
	return nil
}

func (s *Service) userTeams(ctx context.Context, userID int64) ([]int64, error) {
	q := s.st.Rebind(`SELECT team_id FROM team_members WHERE user_id = ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Service) applyDirectPermission(ctx context.Context, userID, repoID int64, out map[Permission]bool) error {
	var perm string
	q := s.st.Rebind(`SELECT permission FROM repo_collaborators WHERE repo_id = ? AND user_id = ?`)
	err := s.st.DB.QueryRowContext(ctx, q, repoID, userID).Scan(&perm)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, p := range ladderPermissions(perm) {
		out[p] = true
	}
	return nil
}

// ladderPermissions maps a repo permission string to its permission set.
func ladderPermissions(perm string) []Permission {
	switch perm {
	case "admin":
		return allPerms()
	case "maintain":
		return union(readPerms(), triagePerms(), writePerms(), maintainPerms())
	case "write":
		return union(readPerms(), triagePerms(), writePerms())
	case "triage":
		return union(readPerms(), triagePerms())
	case "read":
		return readPerms()
	default:
		return nil
	}
}

func (s *Service) rolePermissions(ctx context.Context, roleID int64) ([]Permission, error) {
	q := s.st.Rebind(`SELECT p.resource, p.action FROM role_permissions rp JOIN permissions p ON p.id = rp.permission_id WHERE rp.role_id = ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, roleID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.Resource, &p.Action); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// InvalidateUser drops cached decisions for a user (call on role/member changes).
func (s *Service) InvalidateUser(userID int64) { s.cache.invalidateUser(userID) }

// InvalidateAll clears the whole permission cache.
func (s *Service) InvalidateAll() { s.cache.invalidateAll() }

func cacheKey(userID int64, req Requirement) string {
	return strconv.FormatInt(userID, 10) + ":" + strconv.FormatInt(req.RepoID, 10) + ":" +
		req.Resource + ":" + req.Action
}
