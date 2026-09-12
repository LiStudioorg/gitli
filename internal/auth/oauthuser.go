package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/bcrypt"

	"gitli/internal/db"
)

// RegisterOAuthUser 直接插入 OAuth 用户（绕过 users.Service 的密码长度/邮箱校验由调用方保证基本格式）。
// 放在 auth 包以避免 users ↔ auth 循环依赖。
func RegisterOAuthUser(ctx context.Context, q db.Querier, username, email, password string) (db.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return db.User{}, fmt.Errorf("hash password: %w", err)
	}
	now := time.Now().UTC().Unix()
	return q.CreateUser(ctx, db.CreateUserParams{
		Username:     NormalizeUsername(username),
		Email:        email,
		PasswordHash: string(hash),
		IsAdmin:      0,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
}

func newJSONDecoder(r io.Reader) *json.Decoder {
	d := json.NewDecoder(r)
	return d
}
