package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"gitli/internal/db"
)

const (
	sessionTTL  = 14 * 24 * time.Hour
	patPrefix   = "gitli_"
	patTTLBytes = 24 // 24 random bytes -> 32 base64 chars
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrSessionExpired     = errors.New("session expired")
)

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return hex.EncodeToString(b)
}

// HashPassword 用 bcrypt 哈希密码。
func HashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(h), nil
}

// CheckPassword 校验 bcrypt 密码。
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// Login 校验用户名密码，成功则创建 session 并返回 session cookie 值与 CSRF token。
func Login(ctx context.Context, q db.Querier, username, password string) (sessionID, csrfToken string, err error) {
	user, err := q.GetUserByUsername(ctx, username)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}
	if !CheckPassword(user.PasswordHash, password) {
		return "", "", ErrInvalidCredentials
	}
	sessionID = randomHex(32)
	csrfToken = randomHex(32)
	now := time.Now().UTC()
	if err := q.CreateSession(ctx, db.CreateSessionParams{
		ID:        hashToken(sessionID),
		UserID:    user.ID,
		CsrfToken: csrfToken,
		ExpiresAt: now.Add(sessionTTL).Unix(),
		CreatedAt: now.Unix(),
	}); err != nil {
		return "", "", fmt.Errorf("create session: %w", err)
	}
	return sessionID, csrfToken, nil
}

// Logout 删除 session。
func Logout(ctx context.Context, q db.Querier, sessionID string) error {
	return q.DeleteSession(ctx, hashToken(sessionID))
}

// SessionUser 校验 session 并返回对应用户。
func SessionUser(ctx context.Context, q db.Querier, sessionID string) (db.User, error) {
	sess, err := q.GetSession(ctx, hashToken(sessionID))
	if err != nil {
		return db.User{}, ErrSessionExpired
	}
	if sess.ExpiresAt < time.Now().UTC().Unix() {
		_ = q.DeleteSession(ctx, sess.ID)
		return db.User{}, ErrSessionExpired
	}
	user, err := q.GetUserByID(ctx, sess.UserID)
	if err != nil {
		return db.User{}, fmt.Errorf("get session user: %w", err)
	}
	return user, nil
}

// SessionUserWithCSRF 同 SessionUser，额外返回 CSRF token。
func SessionUserWithCSRF(ctx context.Context, q db.Querier, sessionID string) (db.User, string, error) {
	sess, err := q.GetSession(ctx, hashToken(sessionID))
	if err != nil {
		return db.User{}, "", ErrSessionExpired
	}
	if sess.ExpiresAt < time.Now().UTC().Unix() {
		_ = q.DeleteSession(ctx, sess.ID)
		return db.User{}, "", ErrSessionExpired
	}
	user, err := q.GetUserByID(ctx, sess.UserID)
	if err != nil {
		return db.User{}, "", fmt.Errorf("get session user: %w", err)
	}
	return user, sess.CsrfToken, nil
}

// NewPAT 生成新 Personal Access Token，返回明文（仅此一次）并存储哈希。
// 目前 PAT 直接作为 session 等价物处理；M2 起加入 pats 表。
func NewPAT() string {
	return patPrefix + randomHex(patTTLBytes)
}

// CheckConstantTime 常量时间比较（供 token 校验辅助）。
func CheckConstantTime(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// NormalizeUsername 规范化用户名。
func NormalizeUsername(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}
