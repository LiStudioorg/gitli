package auth

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"gitli/internal/db"
)

// PATScope 令牌作用域。
type PATScope struct {
	// Kind: "all" 或 "repo"
	Kind string
	// RepoID 当 Kind=="repo" 时有效
	RepoID int64
	// Write 仓库级令牌的写权限（read 为 false 时仅可读）
	Write bool
}

// ParseScope 解析 pats.scope 字段："all" 或 "repo:<id>:<read|write>"。
func ParseScope(s string) PATScope {
	if s == "" || s == "all" {
		return PATScope{Kind: "all"}
	}
	parts := strings.Split(s, ":")
	if len(parts) == 3 && parts[0] == "repo" {
		var id int64
		for _, c := range parts[1] {
			if c < '0' || c > '9' {
				return PATScope{Kind: "all"}
			}
			id = id*10 + int64(c-'0')
		}
		return PATScope{Kind: "repo", RepoID: id, Write: parts[2] == "write"}
	}
	return PATScope{Kind: "all"}
}

// PATCanAccessRepo 判断某 PAT（已解析 scope）能否访问 repo（read 或 write）。
// 全局 PAT：交给 CanAccess 做用户级判断。
// 仓库级 PAT：直接按 scope 判定，不受用户其它权限影响（最小权限）。
func PATCanAccessRepo(ctx context.Context, q db.Querier, pat db.Pat, user db.User, repo db.Repo, write bool) bool {
	sc := ParseScope(pat.Scope)
	if sc.Kind == "all" {
		return CanAccess(ctx, q, &user, repo, write)
	}
	if sc.RepoID != repo.ID {
		return false
	}
	if write {
		return sc.Write
	}
	return true
}

// AuthenticateGitPAT Git 认证专用：按 token 找 PAT + 用户。
// 返回 nil, nil 表示 token 无效。
func AuthenticateGitPAT(ctx context.Context, q db.Querier, token string) (*db.Pat, *db.User) {
	pat, err := q.GetPATByHash(ctx, hashToken(token))
	if err != nil {
		return nil, nil
	}
	if pat.ExpiresAt.Valid && pat.ExpiresAt.Int64 < time.Now().UTC().Unix() {
		return nil, nil // 已过期
	}
	user, err := q.GetUserByID(ctx, pat.UserID)
	if err != nil {
		return nil, nil
	}
	_ = q.TouchPAT(ctx, db.TouchPATParams{
		LastUsedAt: sql.NullInt64{Int64: time.Now().UTC().Unix(), Valid: true},
		ID:         pat.ID,
	})
	return &pat, &user
}
