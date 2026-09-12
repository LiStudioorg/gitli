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
//   - public：任何人（含匿名）可读；写仅 owner/协作者
//   - org：owner、协作者可读写（组织成员判定在 M5 加入）
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
		return isCollaborator(ctx, q, repo.ID, user.ID)
	}
	// 读权限
	switch repos.Visibility(repo.Visibility) {
	case repos.VisibilityPublic:
		return true
	case repos.VisibilityOrg, repos.VisibilityPrivate:
		if user == nil {
			return false
		}
		return isCollaborator(ctx, q, repo.ID, user.ID)
	}
	return false
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
