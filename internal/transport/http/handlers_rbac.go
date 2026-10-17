// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"errors"
	"net/http"

	"github.com/LiStudioorg/gitli/internal/service/rbac"
)

// orgScoped resolves the {org} URL parameter and proves the given child id
// (team or role) belongs to that org. RequireOrgPermission only authorizes the
// org from the URL, so without this check an admin of org A could pass org B's
// team/role id and act on it.
//
// It writes the error response itself and reports whether the handler should
// continue.
func (s *Server) orgScoped(w http.ResponseWriter, r *http.Request, kind string, id int64) bool {
	o, err := s.deps.Orgs.ByName(r.Context(), urlParam(r, "org"))
	if err != nil {
		writeError(w, http.StatusNotFound, "组织不存在")
		return false
	}
	var owner int64
	switch kind {
	case "team":
		owner, err = s.deps.RBAC.TeamOrgID(r.Context(), id)
	case "role":
		owner, err = s.deps.RBAC.RoleOrgID(r.Context(), id)
	default:
		writeError(w, http.StatusInternalServerError, "未知的资源类型")
		return false
	}
	if err != nil {
		if errors.Is(err, rbac.ErrNotFound) {
			writeError(w, http.StatusNotFound, "资源不存在")
			return false
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return false
	}
	// owner == 0 表示全局内置角色（org_id 为 NULL），不属于任何组织，一律拒绝。
	if owner == 0 || owner != o.ID {
		writeError(w, http.StatusNotFound, "资源不存在")
		return false
	}
	return true
}

// --- org members ---

type orgMemberReq struct {
	Username   string `json:"username"`
	RoleID     *int64 `json:"role_id"`
	Membership string `json:"membership"`
}

func (s *Server) handleAddOrgMember(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	o, err := s.deps.Orgs.ByName(r.Context(), urlParam(r, "org"))
	if err != nil {
		writeError(w, http.StatusNotFound, "组织不存在")
		return
	}
	var req orgMemberReq
	if !decodeJSON(w, r, &req) {
		return
	}
	target, err := s.deps.Users.ByUsername(r.Context(), req.Username)
	if err != nil {
		writeError(w, http.StatusBadRequest, "用户不存在")
		return
	}
	if err := s.deps.RBAC.AddOrgMember(r.Context(), u.ID, o.ID, target.ID, req.RoleID, req.Membership); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
}

func (s *Server) handleRemoveOrgMember(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	o, err := s.deps.Orgs.ByName(r.Context(), urlParam(r, "org"))
	if err != nil {
		writeError(w, http.StatusNotFound, "组织不存在")
		return
	}
	target, err := s.deps.Users.ByUsername(r.Context(), urlParam(r, "username"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "用户不存在")
		return
	}
	if err := s.deps.RBAC.RemoveOrgMember(r.Context(), u.ID, o.ID, target.ID); err != nil {
		writeServiceError(w, err, "成员不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// --- teams ---

type teamReq struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	ParentTeamID *int64 `json:"parent_team_id"`
	RoleID       *int64 `json:"role_id"`
	Permission   string `json:"permission"`
}

func (s *Server) handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	o, err := s.deps.Orgs.ByName(r.Context(), urlParam(r, "org"))
	if err != nil {
		writeError(w, http.StatusNotFound, "组织不存在")
		return
	}
	var req teamReq
	if !decodeJSON(w, r, &req) {
		return
	}
	id, err := s.deps.RBAC.CreateTeam(r.Context(), rbac.CreateTeamParams{
		OrgID: o.ID, Name: req.Name, Description: req.Description,
		ParentTeamID: req.ParentTeamID, RoleID: req.RoleID, Permission: req.Permission,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

type reparentReq struct {
	ParentTeamID *int64 `json:"parent_team_id"`
}

func (s *Server) handleReparentTeam(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	teamID, err := pathInt(r, "team")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的团队 ID")
		return
	}
	if !s.orgScoped(w, r, "team", teamID) {
		return
	}
	var req reparentReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.RBAC.ReparentTeam(r.Context(), teamID, req.ParentTeamID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type teamMemberReq struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

func (s *Server) handleAddTeamMember(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	teamID, err := pathInt(r, "team")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的团队 ID")
		return
	}
	if !s.orgScoped(w, r, "team", teamID) {
		return
	}
	var req teamMemberReq
	if !decodeJSON(w, r, &req) {
		return
	}
	target, err := s.deps.Users.ByUsername(r.Context(), req.Username)
	if err != nil {
		writeError(w, http.StatusBadRequest, "用户不存在")
		return
	}
	if err := s.deps.RBAC.AddTeamMember(r.Context(), u.ID, teamID, target.ID, req.Role); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
}

func (s *Server) handleRemoveTeamMember(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	teamID, err := pathInt(r, "team")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的团队 ID")
		return
	}
	if !s.orgScoped(w, r, "team", teamID) {
		return
	}
	target, err := s.deps.Users.ByUsername(r.Context(), urlParam(r, "username"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "用户不存在")
		return
	}
	if err := s.deps.RBAC.RemoveTeamMember(r.Context(), u.ID, teamID, target.ID); err != nil {
		writeServiceError(w, err, "团队成员不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type teamRepoReq struct {
	RepoID     int64  `json:"repo_id"`
	RoleID     *int64 `json:"role_id"`
	Permission string `json:"permission"`
}

func (s *Server) handleGrantTeamRepo(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	teamID, err := pathInt(r, "team")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的团队 ID")
		return
	}
	if !s.orgScoped(w, r, "team", teamID) {
		return
	}
	var req teamRepoReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.deps.RBAC.GrantTeamRepo(r.Context(), u.ID, teamID, req.RepoID, req.RoleID, req.Permission); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
}

// --- custom roles ---

type roleReq struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Scope       string   `json:"scope"`
	Permissions []string `json:"permissions"`
}

func (s *Server) handleCreateRole(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	o, err := s.deps.Orgs.ByName(r.Context(), urlParam(r, "org"))
	if err != nil {
		writeError(w, http.StatusNotFound, "组织不存在")
		return
	}
	var req roleReq
	if !decodeJSON(w, r, &req) {
		return
	}
	perms, err := parsePermissions(req.Permissions)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := s.deps.RBAC.CreateRole(r.Context(), rbac.CreateRoleParams{
		ActorID: u.ID, OrgID: o.ID, Name: req.Name, Description: req.Description,
		Permissions: perms, Scope: req.Scope,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleDeleteRole(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	roleID, err := pathInt(r, "role")
	if err != nil {
		writeError(w, http.StatusBadRequest, "无效的角色 ID")
		return
	}
	if !s.orgScoped(w, r, "role", roleID) {
		return
	}
	if err := s.deps.RBAC.DeleteRole(r.Context(), u.ID, roleID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func parsePermissions(list []string) ([]rbac.Permission, error) {
	out := make([]rbac.Permission, 0, len(list))
	for _, s := range list {
		p, err := rbac.ParsePermission(s)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// --- collaborators & branch rules (repo-scoped) ---

type rbacCollaboratorReq struct {
	Username   string `json:"username"`
	RoleID     *int64 `json:"role_id"`
	Permission string `json:"permission"`
}

func (s *Server) handleSetCollaborator(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	rp, ok := s.repoFromURL(r.Context(), urlParam(r, "owner"), trimGitSuffix(urlParam(r, "repo")))
	if !ok {
		writeError(w, http.StatusNotFound, "仓库不存在")
		return
	}
	var req rbacCollaboratorReq
	if !decodeJSON(w, r, &req) {
		return
	}
	target, err := s.deps.Users.ByUsername(r.Context(), req.Username)
	if err != nil {
		writeError(w, http.StatusBadRequest, "用户不存在")
		return
	}
	if err := s.deps.RBAC.SetCollaborator(r.Context(), u.ID, rp.ID, target.ID, req.RoleID, req.Permission); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
}

func (s *Server) handleRemoveCollaborator(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	rp, ok := s.repoFromURL(r.Context(), urlParam(r, "owner"), trimGitSuffix(urlParam(r, "repo")))
	if !ok {
		writeError(w, http.StatusNotFound, "仓库不存在")
		return
	}
	target, err := s.deps.Users.ByUsername(r.Context(), urlParam(r, "username"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "用户不存在")
		return
	}
	if err := s.deps.RBAC.RemoveCollaborator(r.Context(), u.ID, rp.ID, target.ID); err != nil {
		writeServiceError(w, err, "协作者不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type branchRuleReq struct {
	Pattern        string `json:"pattern"`
	RequireReviews bool   `json:"require_reviews"`
	RequireSigned  bool   `json:"require_signed"`
	RequireCIJobs  string `json:"require_ci_jobs"`
	PushRoles      string `json:"push_roles"`
	MergeRoles     string `json:"merge_roles"`
}

func (s *Server) handleAddBranchRule(w http.ResponseWriter, r *http.Request) {
	u := requireUser(w, r)
	if u == nil {
		return
	}
	rp, ok := s.repoFromURL(r.Context(), urlParam(r, "owner"), trimGitSuffix(urlParam(r, "repo")))
	if !ok {
		writeError(w, http.StatusNotFound, "仓库不存在")
		return
	}
	var req branchRuleReq
	if !decodeJSON(w, r, &req) {
		return
	}
	id, err := s.deps.RBAC.AddBranchRule(r.Context(), u.ID, rbac.BranchRule{
		RepoID: rp.ID, Pattern: req.Pattern, RequireReviews: req.RequireReviews,
		RequireSigned: req.RequireSigned, RequireCIJobs: req.RequireCIJobs,
		PushRoles: req.PushRoles, MergeRoles: req.MergeRoles,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleListBranchRules(w http.ResponseWriter, r *http.Request) {
	rp, ok := s.repoFromURL(r.Context(), urlParam(r, "owner"), trimGitSuffix(urlParam(r, "repo")))
	if !ok {
		writeError(w, http.StatusNotFound, "仓库不存在")
		return
	}
	rules, err := s.deps.RBAC.ListBranchRules(r.Context(), rp.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rules)
}
