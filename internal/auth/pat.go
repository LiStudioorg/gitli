package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"gitli/internal/db"
)

var ErrPATNotFound = errors.New("PAT not found")

// AuthenticatePAT 用 PAT 明文查库校验，返回所属用户。
func AuthenticatePAT(ctx context.Context, q db.Querier, token string) (db.User, error) {
	row, err := q.GetPATByHash(ctx, hashToken(token))
	if err != nil {
		return db.User{}, ErrPATNotFound
	}
	user, err := q.GetUserByID(ctx, row.UserID)
	if err != nil {
		return db.User{}, ErrPATNotFound
	}
	_ = q.TouchPAT(ctx, db.TouchPATParams{
		LastUsedAt: sql.NullInt64{Int64: time.Now().UTC().Unix(), Valid: true},
		ID:         row.ID,
	})
	return user, nil
}


// NewToken 生成 PAT 明文（调用方负责 CreatePAT 存哈希）。
func NewToken() string { return NewPAT() }

// HashTokenForStore 生成入库用的哈希。
func HashTokenForStore(token string) string { return hashToken(token) }
