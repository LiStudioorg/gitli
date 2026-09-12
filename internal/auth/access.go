package auth

import (
	"context"
	"database/sql"
	"errors"

	"gitli/internal/db"
	"gitli/internal/repos"
)

var ErrDenied = errors.New("access denied")

// CanAccess 唯一的仓库可见性判断入口。所有路由/handler 必须复用本函数，禁止散落判断。
//
// 规则：
//   - public：任何人（含匿名）可读；写仅 owner/协作者/组织 owner 角色
//   - org：组织内 owner 角色可写、成员可读写、非成员等同 private
//   - private：owner、协作者可读写
func CanAccess(ctx context.Context, q db.Querier, user *db.User, repo db.Repo, write bool) bool {
	if user != nil && user.ID == repo.OwnerID {
		return true
	}
	if user != nil && user.IsAdmin == 1 {
		return true
	}
	if write {
		if user == nil {
			return false
		}
		// 协作者
		if isCollaborator(ctx, q, repo.ID, user.ID) {
			return true
		}
		// 组织仓库：owner 角色成员可写
		if isOrgRepo(ctx, q, repo.OwnerID) {
			role := orgRole(ctx, q, repo.OwnerID, user.ID)
			return role == "owner"
		}
		return false
	}
	// 读权限
	switch repos.Visibility(repo.Visibility) {
	case repos.VisibilityPublic:
		return true
	case repos.VisibilityOrg:
		if user == nil {
			return false
		}
		// 组织仓库：成员可读，非成员等同 private（协作者仍可读）
		if isOrgRepo(ctx, q, repo.OwnerID) {
			if isOrgMemberByOrgID(ctx, q, repo.OwnerID, user.ID) {
				return true
			}
			return isCollaborator(ctx, q, repo.ID, user.ID)
		}
		return isCollaborator(ctx, q, repo.ID, user.ID)
	case repos.VisibilityPrivate:
		if user == nil {
			return false
		}
		// 组织仓库的 private：成员可读（组织内共享）
		if isOrgRepo(ctx, q, repo.OwnerID) && isOrgMemberByOrgID(ctx, q, repo.OwnerID, user.ID) {
			return true
		}
		return isCollaborator(ctx, q, repo.ID, user.ID)
	}
	return false
}

// isOrgRepo 判断 ownerID 是否为一个组织（orgs 表 id 与 repos.owner_id 关联）。
func isOrgRepo(ctx context.Context, q db.Querier, ownerID int64) bool {
	n, err := q.CountOrgByID(ctx, ownerID)
	return err == nil && n > 0
}

func isOrgMemberByOrgID(ctx context.Context, q db.Querier, orgID, userID int64) bool {
	n, err := q.IsOrgMemberByOrgID(ctx, db.IsOrgMemberByOrgIDParams{OrgID: orgID, UserID: userID})
	return err == nil && n > 0
}

func orgRole(ctx context.Context, q db.Querier, orgID, userID int64) string {
	m, err := q.GetOrgMember(ctx, db.GetOrgMemberParams{OrgID: orgID, UserID: userID})
	if err != nil {
		return ""
	}
	return m.Role
}

func isCollaborator(ctx context.Context, q db.Querier, repoID, userID int64) bool {
	n, err := q.IsCollaborator(ctx, db.IsCollaboratorParams{RepoID: repoID, UserID: userID})
	if err != nil {
		return false
	}
	return n > 0
}

// BasicAuthUser 从 Basic Auth 解析用户（PAT 优先，密码兜底校验 bcrypt）。
func BasicAuthUser(ctx context.Context, q db.Querier, r req) (*db.User, bool) {
	username, token, ok := r.BasicAuth()
	if !ok || username == "" || token == "" {
		return nil, false
	}
	user, err := q.GetUserByUsername(ctx, NormalizeUsername(username))
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, false
		}
		return nil, false
	}
	// 先试 PAT（git 推拉主路径）
	if _, err := AuthenticatePAT(ctx, q, token); err == nil {
		u := user
		return &u, true
	}
	// 再试密码（兼容 Web 用户直接 clone）
	if CheckPassword(user.PasswordHash, token) {
		return &user, true
	}
	return nil, false
}

// req 抽象 http.Request 的 BasicAuth，避免本包直接依赖 net/http 之外的东西。
type req interface {
	BasicAuth() (string, string, bool)
}
