package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"time"

	"gitli/internal/auth"
	"gitli/internal/db"
)

var (
	ErrUsernameTaken  = errors.New("username already taken")
	ErrEmailTaken     = errors.New("email already taken")
	ErrInvalidInput   = errors.New("invalid input")
	ErrUserNotFound   = errors.New("user not found")
	usernameRe        = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,38}$`)
	emailRe           = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

// Service 提供用户领域逻辑。
type Service struct {
	q db.Querier
}

func NewService(q db.Querier) *Service {
	return &Service{q: q}
}

// Register 创建新用户（首个用户自动成为管理员）。
func (s *Service) Register(ctx context.Context, username, email, password string) (db.User, error) {
	username = auth.NormalizeUsername(username)
	email = normalizeEmail(email)
	if !usernameRe.MatchString(username) {
		return db.User{}, fmt.Errorf("%w: username must match %s", ErrInvalidInput, usernameRe)
	}
	if !emailRe.MatchString(email) {
		return db.User{}, fmt.Errorf("%w: invalid email", ErrInvalidInput)
	}
	if len(password) < 8 {
		return db.User{}, fmt.Errorf("%w: password must be at least 8 characters", ErrInvalidInput)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return db.User{}, err
	}

	isAdmin := int64(0)
	count, err := s.q.CountUsers(ctx)
	if err != nil {
		return db.User{}, fmt.Errorf("count users: %w", err)
	}
	if count == 0 {
		isAdmin = 1
	}

	now := time.Now().UTC().Unix()
	user, err := s.q.CreateUser(ctx, db.CreateUserParams{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		IsAdmin:      isAdmin,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		if isUniqueViolation(err, "username") || isUniqueViolation(err, "users.username") {
			return db.User{}, ErrUsernameTaken
		}
		if isUniqueViolation(err, "email") || isUniqueViolation(err, "users.email") {
			return db.User{}, ErrEmailTaken
		}
		// modernc.org/sqlite 错误文本里带列名，兜底区分
		if isUniqueViolation(err, "") {
			if stringsContains(err.Error(), "users.username") {
				return db.User{}, ErrUsernameTaken
			}
			if stringsContains(err.Error(), "users.email") {
				return db.User{}, ErrEmailTaken
			}
			return db.User{}, ErrUsernameTaken
		}
		return db.User{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (s *Service) GetByUsername(ctx context.Context, username string) (db.User, error) {
	u, err := s.q.GetUserByUsername(ctx, auth.NormalizeUsername(username))
	if errors.Is(err, sql.ErrNoRows) {
		return db.User{}, ErrUserNotFound
	}
	return u, err
}

func (s *Service) GetByID(ctx context.Context, id int64) (db.User, error) {
	u, err := s.q.GetUserByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return db.User{}, ErrUserNotFound
	}
	return u, err
}

func (s *Service) Search(ctx context.Context, query string) ([]db.User, error) {
	return s.q.SearchUsers(ctx, sql.NullString{String: query, Valid: query != ""})
}

func normalizeEmail(s string) string {
	out := ""
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			r += 32
		}
		out += string(r)
	}
	return out
}

func stringsContains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// isUniqueViolation 判断 sqlite 错误是否为唯一约束冲突（modernc.org/sqlite 返回 SQLITE_CONSTRAINT_UNIQUE 文本）。
func isUniqueViolation(err error, _ string) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return stringsContains(msg, "UNIQUE constraint failed") || stringsContains(msg, "constraint failed: UNIQUE")
}
