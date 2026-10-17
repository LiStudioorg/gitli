// SPDX-License-Identifier: AGPL-3.0
package rbac

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// AuditEntry records a permission change.
type AuditEntry struct {
	ActorID    *int64
	TargetType string
	TargetID   *int64
	Action     string
	OldValue   string
	NewValue   string
}

// recordAudit writes a permission change to permission_audit.
func (s *Service) recordAudit(ctx context.Context, e AuditEntry) error {
	q := s.st.Rebind(`INSERT INTO permission_audit (actor_id, target_type, target_id, action, old_value, new_value) VALUES (?, ?, ?, ?, ?, ?)`)
	_, err := s.st.DB.ExecContext(ctx, q, e.ActorID, e.TargetType, e.TargetID, e.Action, e.OldValue, e.NewValue)
	return err
}

// --- custom roles ---

// CreateRoleParams describes an org custom role.
type CreateRoleParams struct {
	ActorID     int64
	OrgID       int64
	Name        string
	Description string
	Permissions []Permission
	Scope       string
}

// CreateRole creates an org-scoped custom role.
func (s *Service) CreateRole(ctx context.Context, p CreateRoleParams) (int64, error) {
	if p.Scope == "" {
		p.Scope = "org"
	}
	tx, err := s.st.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, s.st.Rebind(`INSERT INTO roles (org_id, name, description, is_builtin, scope) VALUES (?, ?, ?, 0, ?)`),
		p.OrgID, p.Name, p.Description, p.Scope)
	if err != nil {
		return 0, fmt.Errorf("rbac: create role: %w", err)
	}
	roleID, _ := res.LastInsertId()
	for _, perm := range p.Permissions {
		var pid int64
		if err := tx.QueryRowContext(ctx, s.st.Rebind(`SELECT id FROM permissions WHERE resource = ? AND action = ?`), perm.Resource, perm.Action).Scan(&pid); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, s.st.Rebind(`INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)`), roleID, pid); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	tid := roleID
	_ = s.recordAudit(ctx, AuditEntry{ActorID: &p.ActorID, TargetType: "role", TargetID: &tid, Action: "create", NewValue: p.Name})
	s.cache.invalidateAll()
	return roleID, nil
}

// SetRolePermissions replaces a role's permission set.
func (s *Service) SetRolePermissions(ctx context.Context, actorID, roleID int64, perms []Permission) error {
	builtin, err := s.isBuiltinRole(ctx, roleID)
	if err != nil {
		return err
	}
	if builtin {
		return ErrBuiltinRole
	}
	tx, err := s.st.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, s.st.Rebind(`DELETE FROM role_permissions WHERE role_id = ?`), roleID); err != nil {
		return err
	}
	for _, perm := range perms {
		var pid int64
		if err := tx.QueryRowContext(ctx, s.st.Rebind(`SELECT id FROM permissions WHERE resource = ? AND action = ?`), perm.Resource, perm.Action).Scan(&pid); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return err
		}
		if _, err := tx.ExecContext(ctx, s.st.Rebind(`INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)`), roleID, pid); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	_ = s.recordAudit(ctx, AuditEntry{ActorID: &actorID, TargetType: "role", TargetID: &roleID, Action: "set_permissions", NewValue: permsString(perms)})
	s.cache.invalidateAll()
	return nil
}

// DeleteRole deletes a custom role.
func (s *Service) DeleteRole(ctx context.Context, actorID, roleID int64) error {
	builtin, err := s.isBuiltinRole(ctx, roleID)
	if err != nil {
		return err
	}
	if builtin {
		return ErrBuiltinRole
	}
	if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM role_permissions WHERE role_id = ?`), roleID); err != nil {
		return err
	}
	if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM roles WHERE id = ?`), roleID); err != nil {
		return err
	}
	_ = s.recordAudit(ctx, AuditEntry{ActorID: &actorID, TargetType: "role", TargetID: &roleID, Action: "delete"})
	s.cache.invalidateAll()
	return nil
}

func (s *Service) isBuiltinRole(ctx context.Context, roleID int64) (bool, error) {
	var builtin bool
	err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT is_builtin FROM roles WHERE id = ?`), roleID).Scan(&builtin)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrNotFound
	}
	return builtin, err
}

func permsString(perms []Permission) string {
	parts := make([]string, 0, len(perms))
	for _, p := range perms {
		parts = append(parts, p.String())
	}
	return strings.Join(parts, ",")
}

// --- org & team membership ---

// AddOrgMember adds or updates an org member with an optional role.
func (s *Service) AddOrgMember(ctx context.Context, actorID, orgID, userID int64, roleID *int64, membership string) error {
	if membership == "" {
		membership = "member"
	}
	q := s.st.Rebind(`INSERT INTO org_members (org_id, user_id, role_id, membership) VALUES (?, ?, ?, ?)`)
	if _, err := s.st.DB.ExecContext(ctx, q, orgID, userID, roleID, membership); err != nil {
		uq := s.st.Rebind(`UPDATE org_members SET role_id = ?, membership = ?, updated_at = ? WHERE org_id = ? AND user_id = ?`)
		if _, err2 := s.st.DB.ExecContext(ctx, uq, roleID, membership, time.Now().UTC(), orgID, userID); err2 != nil {
			return fmt.Errorf("rbac: add org member: %w", err)
		}
	}
	_ = s.recordAudit(ctx, AuditEntry{ActorID: &actorID, TargetType: "org_member", TargetID: &userID, Action: "add", NewValue: membership})
	s.cache.invalidateUser(userID)
	return nil
}

// RemoveOrgMember removes an org member.
func (s *Service) RemoveOrgMember(ctx context.Context, actorID, orgID, userID int64) error {
	if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM org_members WHERE org_id = ? AND user_id = ?`), orgID, userID); err != nil {
		return err
	}
	_ = s.recordAudit(ctx, AuditEntry{ActorID: &actorID, TargetType: "org_member", TargetID: &userID, Action: "remove"})
	s.cache.invalidateUser(userID)
	return nil
}

// AddTeamMember adds a user to a team.
func (s *Service) AddTeamMember(ctx context.Context, actorID, teamID, userID int64, role string) error {
	if role == "" {
		role = "member"
	}
	q := s.st.Rebind(`INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`)
	if _, err := s.st.DB.ExecContext(ctx, q, teamID, userID, role); err != nil {
		return fmt.Errorf("rbac: add team member: %w", err)
	}
	_ = s.recordAudit(ctx, AuditEntry{ActorID: &actorID, TargetType: "team_member", TargetID: &userID, Action: "add", NewValue: role})
	s.cache.invalidateUser(userID)
	return nil
}

// RemoveTeamMember removes a user from a team.
func (s *Service) RemoveTeamMember(ctx context.Context, actorID, teamID, userID int64) error {
	if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM team_members WHERE team_id = ? AND user_id = ?`), teamID, userID); err != nil {
		return err
	}
	_ = s.recordAudit(ctx, AuditEntry{ActorID: &actorID, TargetType: "team_member", TargetID: &userID, Action: "remove"})
	s.cache.invalidateUser(userID)
	return nil
}

// GrantTeamRepo grants a team access to a repo.
func (s *Service) GrantTeamRepo(ctx context.Context, actorID, teamID, repoID int64, roleID *int64, permission string) error {
	if permission == "" {
		permission = "read"
	}
	q := s.st.Rebind(`INSERT INTO team_repos (team_id, repo_id, role_id, permission) VALUES (?, ?, ?, ?)`)
	if _, err := s.st.DB.ExecContext(ctx, q, teamID, repoID, roleID, permission); err != nil {
		uq := s.st.Rebind(`UPDATE team_repos SET role_id = ?, permission = ? WHERE team_id = ? AND repo_id = ?`)
		if _, err2 := s.st.DB.ExecContext(ctx, uq, roleID, permission, teamID, repoID); err2 != nil {
			return fmt.Errorf("rbac: grant team repo: %w", err)
		}
	}
	_ = s.recordAudit(ctx, AuditEntry{ActorID: &actorID, TargetType: "team_repo", TargetID: &repoID, Action: "grant", NewValue: permission})
	s.cache.invalidateAll()
	return nil
}

// SetCollaborator grants a user direct repo access.
func (s *Service) SetCollaborator(ctx context.Context, actorID, repoID, userID int64, roleID *int64, permission string) error {
	if permission == "" {
		permission = "read"
	}
	q := s.st.Rebind(`INSERT INTO repo_collaborators (repo_id, user_id, role_id, permission) VALUES (?, ?, ?, ?)`)
	if _, err := s.st.DB.ExecContext(ctx, q, repoID, userID, roleID, permission); err != nil {
		uq := s.st.Rebind(`UPDATE repo_collaborators SET role_id = ?, permission = ?, updated_at = ? WHERE repo_id = ? AND user_id = ?`)
		if _, err2 := s.st.DB.ExecContext(ctx, uq, roleID, permission, time.Now().UTC(), repoID, userID); err2 != nil {
			return fmt.Errorf("rbac: set collaborator: %w", err)
		}
	}
	_ = s.recordAudit(ctx, AuditEntry{ActorID: &actorID, TargetType: "repo_collaborator", TargetID: &userID, Action: "set", NewValue: permission})
	s.cache.invalidateUser(userID)
	return nil
}

// RemoveCollaborator removes a direct repo grant.
func (s *Service) RemoveCollaborator(ctx context.Context, actorID, repoID, userID int64) error {
	if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM repo_collaborators WHERE repo_id = ? AND user_id = ?`), repoID, userID); err != nil {
		return err
	}
	_ = s.recordAudit(ctx, AuditEntry{ActorID: &actorID, TargetType: "repo_collaborator", TargetID: &userID, Action: "remove"})
	s.cache.invalidateUser(userID)
	return nil
}

// --- branch protection ---

// BranchRule describes a branch protection rule.
type BranchRule struct {
	ID             int64
	RepoID         int64
	Pattern        string
	RequireReviews bool
	RequireSigned  bool
	RequireCIJobs  string
	PushRoles      string
	MergeRoles     string
}

// AddBranchRule creates a branch protection rule.
func (s *Service) AddBranchRule(ctx context.Context, actorID int64, r BranchRule) (int64, error) {
	q := s.st.Rebind(`INSERT INTO branch_protection_rules (repo_id, pattern, require_reviews, require_signed, require_ci_jobs, push_roles, merge_roles) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, r.RepoID, r.Pattern, r.RequireReviews, r.RequireSigned, r.RequireCIJobs, r.PushRoles, r.MergeRoles)
	if err != nil {
		return 0, fmt.Errorf("rbac: add branch rule: %w", err)
	}
	id, _ := res.LastInsertId()
	rid := id
	_ = s.recordAudit(ctx, AuditEntry{ActorID: &actorID, TargetType: "branch_rule", TargetID: &rid, Action: "add", NewValue: r.Pattern})
	return id, nil
}

// ListBranchRules returns branch protection rules for a repo.
func (s *Service) ListBranchRules(ctx context.Context, repoID int64) ([]BranchRule, error) {
	q := s.st.Rebind(`SELECT id, repo_id, pattern, require_reviews, require_signed, require_ci_jobs, push_roles, merge_roles FROM branch_protection_rules WHERE repo_id = ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []BranchRule
	for rows.Next() {
		var r BranchRule
		if err := rows.Scan(&r.ID, &r.RepoID, &r.Pattern, &r.RequireReviews, &r.RequireSigned, &r.RequireCIJobs, &r.PushRoles, &r.MergeRoles); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// IsOrgOwner reports whether the user is the owner of the org.
func (s *Service) IsOrgOwner(ctx context.Context, orgID, userID int64) (bool, error) {
	var owner int64
	err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT owner_id FROM orgs WHERE id = ?`), orgID).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrNotFound
	}
	if err != nil {
		return false, err
	}
	return owner == userID, nil
}

// TeamOrgID returns the org that owns the team. Handlers use it to prove the
// {team} path parameter really belongs to the {org} they were authorized for —
// without it, an org admin could pass another org's team id and act on it.
func (s *Service) TeamOrgID(ctx context.Context, teamID int64) (int64, error) {
	var orgID int64
	err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT org_id FROM teams WHERE id = ?`), teamID).Scan(&orgID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	return orgID, nil
}

// RoleOrgID returns the org that owns the role. Built-in/global roles have a
// NULL org_id, reported as 0 so callers can treat them as not org-scoped.
func (s *Service) RoleOrgID(ctx context.Context, roleID int64) (int64, error) {
	var orgID sql.NullInt64
	err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT org_id FROM roles WHERE id = ?`), roleID).Scan(&orgID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	if !orgID.Valid {
		return 0, nil
	}
	return orgID.Int64, nil
}
