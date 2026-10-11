// SPDX-License-Identifier: AGPL-3.0
// Package auth implements registration, login, sessions and TOTP two-factor auth.
package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/LiStudioorg/gitli/internal/domain"
	infraCrypto "github.com/LiStudioorg/gitli/internal/infra/crypto"
	"github.com/LiStudioorg/gitli/internal/infra/password"
	"github.com/LiStudioorg/gitli/internal/infra/token"
	"github.com/LiStudioorg/gitli/internal/infra/weakpass"
	"github.com/LiStudioorg/gitli/internal/service/store"
	"github.com/LiStudioorg/gitli/internal/service/user"
)

// Errors returned by the auth service.
var (
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	ErrLockedOut          = errors.New("auth: too many attempts")
	ErrTOTPRequired       = errors.New("auth: totp required")
	ErrInvalidTOTP        = errors.New("auth: invalid totp code")
	ErrUserExists         = errors.New("auth: user already exists")
	ErrWeakPassword       = errors.New("auth: password too short")
)

// Service coordinates authentication.
type Service struct {
	st       *store.Store
	users    *user.Service
	hasher   *password.Hasher
	cipher   *infraCrypto.Cipher
	log      *slog.Logger
	limiter  *loginLimiter
	resolver TokenResolver
	weak     *weakpass.Checker
}

// Options configures the auth service.
type Options struct {
	Users             *user.Service
	Cipher            *infraCrypto.Cipher
	Log               *slog.Logger
	MaxAttempt        int
	Lockout           time.Duration
	MinPasswordLength int
}

// New builds the auth Service.
func New(st *store.Store, opts Options) *Service {
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	if opts.MaxAttempt <= 0 {
		opts.MaxAttempt = 5
	}
	if opts.Lockout <= 0 {
		opts.Lockout = 15 * time.Minute
	}
	return &Service{
		st:      st,
		users:   opts.Users,
		hasher:  password.NewHasher(password.DefaultParams(), ""),
		cipher:  opts.Cipher,
		log:     opts.Log,
		limiter: newLoginLimiter(opts.MaxAttempt, opts.Lockout),
		weak:    weakpass.New(opts.MinPasswordLength),
	}
}

// SetHIBP attaches a HaveIBeenPwned checker to the password policy.
func (s *Service) SetHIBP(h *weakpass.HIBP) {
	s.weak.WithHIBP(h)
}

// SetPepper configures the password pepper.
func (s *Service) SetPepper(p string) {
	s.hasher = password.NewHasher(password.DefaultParams(), p)
}

// RegisterParams describes a registration request.
type RegisterParams struct {
	Username string
	Email    string
	Password string
}

// Register creates a new account.
func (s *Service) Register(ctx context.Context, p RegisterParams) (*domain.User, error) {
	if len(p.Password) < 8 {
		return nil, ErrWeakPassword
	}
	exists, err := s.users.Exists(ctx, p.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserExists
	}
	hash, err := s.hasher.Hash(p.Password)
	if err != nil {
		return nil, err
	}
	firstUser, err := s.isFirstUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.users.Create(ctx, user.CreateParams{
		Username:     p.Username,
		Email:        p.Email,
		PasswordHash: hash,
		IsAdmin:      firstUser,
	})
}

// LoginParams describes a login request.
type LoginParams struct {
	Username string
	Password string
	TOTPCode string
	IP       string
	Agent    string
}

// LoginResult holds the outcome of a successful login.
type LoginResult struct {
	User     *domain.User
	Session  *domain.Session
	RawToken string
}

// Login authenticates a user and creates a session. When TOTP is enabled and no
// code is supplied, ErrTOTPRequired is returned.
func (s *Service) Login(ctx context.Context, p LoginParams) (*LoginResult, error) {
	key := strings.ToLower(p.Username) + "|" + p.IP
	if !s.limiter.allow(key) {
		return nil, ErrLockedOut
	}
	u, err := s.users.ByUsername(ctx, p.Username)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			s.limiter.fail(key)
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if err := s.hasher.Verify(p.Password, u.PasswordHash); err != nil {
		s.limiter.fail(key)
		return nil, ErrInvalidCredentials
	}
	if u.TOTPEnabled {
		if p.TOTPCode == "" {
			return nil, ErrTOTPRequired
		}
		if err := s.verifyTOTP(ctx, u, p.TOTPCode); err != nil {
			s.limiter.fail(key)
			return nil, err
		}
	}
	s.limiter.reset(key)

	sess, raw, err := s.createSession(ctx, u.ID, p.IP, p.Agent)
	if err != nil {
		return nil, err
	}
	return &LoginResult{User: u, Session: sess, RawToken: raw}, nil
}

// Authenticate resolves a session token to a user.
func (s *Service) Authenticate(ctx context.Context, rawToken string) (*domain.User, *domain.Session, error) {
	hash := token.Hash(rawToken)
	q := s.st.Rebind(`SELECT id, user_id, token_hash, ip, user_agent, expires_at, created_at FROM sessions WHERE token_hash = ?`)
	var sess domain.Session
	err := s.st.DB.QueryRowContext(ctx, q, hash).Scan(&sess.ID, &sess.UserID, &sess.TokenHash, &sess.IP, &sess.UserAgent, &sess.ExpiresAt, &sess.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, nil, err
	}
	if sess.ExpiresAt != nil && sess.ExpiresAt.Before(time.Now()) {
		_ = s.Logout(ctx, rawToken)
		return nil, nil, ErrInvalidCredentials
	}
	u, err := s.users.ByID(ctx, sess.UserID)
	if err != nil {
		return nil, nil, err
	}
	return u, &sess, nil
}

// Logout deletes a session.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	hash := token.Hash(rawToken)
	q := s.st.Rebind(`DELETE FROM sessions WHERE token_hash = ?`)
	_, err := s.st.DB.ExecContext(ctx, q, hash)
	return err
}

// BeginTOTP generates a new TOTP secret (encrypted at rest) for the user.
func (s *Service) BeginTOTP(ctx context.Context, u *domain.User) (secret string, otpauth string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "gitli",
		AccountName: u.Username,
	})
	if err != nil {
		return "", "", fmt.Errorf("auth: generate totp: %w", err)
	}
	enc, err := s.encryptSecret(key.Secret())
	if err != nil {
		return "", "", err
	}
	if err := s.users.UpdateTOTP(ctx, u.ID, enc, false); err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

// EnableTOTP verifies a code and enables TOTP, returning recovery codes.
func (s *Service) EnableTOTP(ctx context.Context, u *domain.User, code string) ([]string, error) {
	secret, err := s.decryptSecret(u.TOTPSecret)
	if err != nil {
		return nil, err
	}
	if !totp.Validate(code, secret) {
		return nil, ErrInvalidTOTP
	}
	recodes, hashes, err := generateRecoveryCodes(10)
	if err != nil {
		return nil, err
	}
	if err := s.storeRecoveryCodes(ctx, u.ID, hashes); err != nil {
		return nil, err
	}
	if err := s.users.UpdateTOTP(ctx, u.ID, u.TOTPSecret, true); err != nil {
		return nil, err
	}
	return recodes, nil
}

// DisableTOTP turns off TOTP for a user.
func (s *Service) DisableTOTP(ctx context.Context, u *domain.User, code string) error {
	if u.TOTPEnabled {
		if err := s.verifyTOTP(ctx, u, code); err != nil {
			return err
		}
	}
	return s.users.UpdateTOTP(ctx, u.ID, "", false)
}

func (s *Service) verifyTOTP(ctx context.Context, u *domain.User, code string) error {
	secret, err := s.decryptSecret(u.TOTPSecret)
	if err != nil {
		return err
	}
	if totp.Validate(code, secret) {
		return s.checkAndMarkCode(ctx, u.ID, code)
	}
	ok, err := s.verifyRecoveryCode(ctx, u.ID, code)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return ErrInvalidTOTP
}

func (s *Service) checkAndMarkCode(ctx context.Context, userID int64, code string) error {
	codeHash := hashString(code)
	var existing int
	err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT 1 FROM totp_used_codes WHERE user_id = ? AND code_hash = ?`), userID, codeHash).Scan(&existing)
	if err == nil {
		return ErrInvalidTOTP
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = s.st.DB.ExecContext(ctx, s.st.Rebind(`INSERT INTO totp_used_codes (user_id, code_hash) VALUES (?, ?)`), userID, codeHash)
	return err
}

func (s *Service) createSession(ctx context.Context, userID int64, ip, agent string) (*domain.Session, string, error) {
	tok, err := token.New("gls")
	if err != nil {
		return nil, "", err
	}
	expires := time.Now().Add(30 * 24 * time.Hour)
	q := s.st.Rebind(`INSERT INTO sessions (user_id, token_hash, ip, user_agent, expires_at) VALUES (?, ?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, userID, tok.Hash, ip, agent, expires)
	if err != nil {
		return nil, "", err
	}
	id, _ := res.LastInsertId()
	return &domain.Session{ID: id, UserID: userID, IP: ip, UserAgent: agent, ExpiresAt: &expires}, tok.Plain, nil
}

// UpdatePassword validates and stores a new password hash for the user.
func (s *Service) UpdatePassword(ctx context.Context, u *domain.User, newPassword string) error {
	if err := s.validatePassword(newPassword); err != nil {
		return err
	}
	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return err
	}
	return s.users.UpdatePassword(ctx, u.ID, hash)
}

func (s *Service) validatePassword(pw string) error {
	if s.weak != nil {
		return s.weak.Validate(pw)
	}
	if len(pw) < 12 {
		return ErrWeakPassword
	}
	return nil
}

func (s *Service) isFirstUser(ctx context.Context) (bool, error) {
	var count int
	if err := s.st.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return false, err
	}
	return count == 0, nil
}

func (s *Service) encryptSecret(plain string) (string, error) {
	if s.cipher == nil {
		return plain, nil
	}
	sealed, err := s.cipher.Encrypt([]byte(plain))
	if err != nil {
		return "", err
	}
	return "enc:" + hex.EncodeToString(sealed), nil
}

func (s *Service) decryptSecret(stored string) (string, error) {
	if !strings.HasPrefix(stored, "enc:") {
		return stored, nil
	}
	if s.cipher == nil {
		return "", errors.New("auth: no cipher configured for encrypted secret")
	}
	raw, err := hex.DecodeString(strings.TrimPrefix(stored, "enc:"))
	if err != nil {
		return "", err
	}
	plain, err := s.cipher.Decrypt(raw)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
