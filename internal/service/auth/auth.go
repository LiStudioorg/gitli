// SPDX-License-Identifier: AGPL-3.0
// Package auth implements registration, login, sessions and TOTP two-factor auth.
package auth

import (
	"context"
	"crypto/rand"
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
	ErrWeakPassword       = errors.New("密码强度不足")
	ErrAlreadySetUp       = errors.New("auth: instance already initialized")
	ErrSessionNotFound    = errors.New("auth: session not found")
	ErrAccountInactive    = errors.New("auth: account is disabled")
)

// Service coordinates authentication.
type Service struct {
	st        *store.Store
	users     *user.Service
	hasher    *password.Hasher
	cipher    *infraCrypto.Cipher
	log       *slog.Logger
	limiter   *loginLimiter
	resolver  TokenResolver
	weak      *weakpass.Checker
	bindIP    bool
	bindAgent bool
	ldap      LDAPAuthenticator
}

// Options configures the auth service.
type Options struct {
	Users             *user.Service
	Cipher            *infraCrypto.Cipher
	Log               *slog.Logger
	MaxAttempt        int
	Lockout           time.Duration
	MinPasswordLength int
	BindIP            bool
	BindAgent         bool
	LDAP              LDAPAuthenticator
}

// LDAPAuthenticator verifies credentials against an external directory.
type LDAPAuthenticator interface {
	Authenticate(ctx context.Context, username, password string) (string, string, string, error)
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
		st:        st,
		users:     opts.Users,
		hasher:    password.NewHasher(password.DefaultParams(), ""),
		cipher:    opts.Cipher,
		log:       opts.Log,
		limiter:   newLoginLimiter(opts.MaxAttempt, opts.Lockout),
		weak:      weakpass.New(opts.MinPasswordLength),
		bindIP:    opts.BindIP,
		bindAgent: opts.BindAgent,
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
// minRegisterPasswordLength is stricter than the global minimum because
// self-service registration is the easiest account to create and brute force.
const minRegisterPasswordLength = 12

func (s *Service) Register(ctx context.Context, p RegisterParams) (*domain.User, error) {
	if len(p.Password) < minRegisterPasswordLength {
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
	viaLDAP := false
	u, err := s.users.ByUsername(ctx, p.Username)
	if err != nil {
		if !errors.Is(err, user.ErrNotFound) {
			return nil, err
		}
		lu, lerr := s.loginLDAP(ctx, p.Username, p.Password)
		if lerr != nil {
			s.limiter.fail(key)
			return nil, ErrInvalidCredentials
		}
		u = lu
		viaLDAP = true
	}
	// H-04: a deactivated account must not be able to log in.
	if !u.IsActive {
		s.limiter.fail(key)
		return nil, ErrAccountInactive
	}
	if !viaLDAP {
		if verr := s.hasher.Verify(p.Password, u.PasswordHash); verr != nil {
			// Local password failed: fall back to LDAP when configured.
			lu, lerr := s.loginLDAP(ctx, p.Username, p.Password)
			if lerr != nil {
				s.limiter.fail(key)
				return nil, ErrInvalidCredentials
			}
			u = lu
		}
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

// VerifyPassword checks a username/password pair without creating a session.
// Used by the git HTTP Basic auth flow.
func (s *Service) VerifyPassword(ctx context.Context, username, password string) (*domain.User, error) {
	// H-05: Git HTTP Basic auth hits this on every request, so it needs the same
	// brute-force protection as the web login form.
	key := strings.ToLower(username) + "|git"
	if !s.limiter.allow(key) {
		return nil, ErrLockedOut
	}
	u, err := s.users.ByUsername(ctx, username)
	if err != nil {
		s.limiter.fail(key)
		return nil, ErrInvalidCredentials
	}
	// H-04: deactivated accounts cannot authenticate over Git either.
	if !u.IsActive {
		s.limiter.fail(key)
		return nil, ErrAccountInactive
	}
	if err := s.hasher.Verify(password, u.PasswordHash); err != nil {
		s.limiter.fail(key)
		return nil, ErrInvalidCredentials
	}
	s.limiter.success(key)
	return u, nil
}

// Authenticate resolves a session token to a user without binding checks.
func (s *Service) Authenticate(ctx context.Context, rawToken string) (*domain.User, *domain.Session, error) {
	return s.AuthenticateRequest(ctx, rawToken, "", "")
}

// AuthenticateRequest resolves a session token and, when configured, verifies
// that the client IP and User-Agent still match the session binding.
func (s *Service) AuthenticateRequest(ctx context.Context, rawToken, ip, agent string) (*domain.User, *domain.Session, error) {
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
	if s.bindIP && ip != "" && sess.IP != "" && sess.IP != ip {
		return nil, nil, ErrInvalidCredentials
	}
	if s.bindAgent && agent != "" && sess.UserAgent != "" && sess.UserAgent != agent {
		return nil, nil, ErrInvalidCredentials
	}
	_, _ = s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE sessions SET last_seen_at = ? WHERE id = ?`), time.Now().UTC(), sess.ID)
	u, err := s.users.ByID(ctx, sess.UserID)
	if err != nil {
		return nil, nil, err
	}
	if !u.IsActive {
		_ = s.Logout(ctx, rawToken)
		return nil, nil, ErrAccountInactive
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

// CreateSession creates a session for an already-authenticated user and returns
// the raw session token (used by external login flows such as OIDC).
func (s *Service) CreateSession(ctx context.Context, u *domain.User, ip, agent string) (string, error) {
	_, raw, err := s.createSession(ctx, u.ID, ip, agent)
	return raw, err
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

// SessionInfo describes a device session for the settings UI (no token value).
type SessionInfo struct {
	ID         int64      `json:"id"`
	IP         string     `json:"ip"`
	UserAgent  string     `json:"user_agent"`
	CreatedAt  time.Time  `json:"created_at"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	Current    bool       `json:"current"`
}

// ListSessions returns a user's active sessions. rawCurrent is the raw session
// token of the caller, used to mark the current session.
func (s *Service) ListSessions(ctx context.Context, userID int64, rawCurrent string) ([]SessionInfo, error) {
	currentHash := ""
	if rawCurrent != "" {
		currentHash = token.Hash(rawCurrent)
	}
	q := s.st.Rebind(`SELECT id, token_hash, ip, user_agent, expires_at, created_at, last_seen_at FROM sessions WHERE user_id = ? ORDER BY id DESC`)
	rows, err := s.st.DB.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []SessionInfo
	for rows.Next() {
		var si SessionInfo
		var hash string
		var exp, lastSeen sql.NullTime
		if err := rows.Scan(&si.ID, &hash, &si.IP, &si.UserAgent, &exp, &si.CreatedAt, &lastSeen); err != nil {
			return nil, err
		}
		if exp.Valid {
			t := exp.Time
			si.ExpiresAt = &t
		}
		if lastSeen.Valid {
			t := lastSeen.Time
			si.LastSeenAt = &t
		}
		si.Current = currentHash != "" && hash == currentHash
		out = append(out, si)
	}
	return out, rows.Err()
}

// RevokeSession deletes one session owned by the user.
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID int64) error {
	return s.st.ExecAffectedOr(ctx, ErrSessionNotFound, s.st.Rebind(`DELETE FROM sessions WHERE id = ? AND user_id = ?`), sessionID, userID)
}

// RevokeOtherSessions deletes all sessions for a user except the current one.
func (s *Service) RevokeOtherSessions(ctx context.Context, userID int64, rawCurrent string) error {
	hash := token.Hash(rawCurrent)
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM sessions WHERE user_id = ? AND token_hash != ?`), userID, hash)
	return err
}

// loginLDAP verifies credentials against LDAP and provision/links the local
// user. Returns ErrInvalidCredentials when LDAP is disabled or rejects the
// credentials.
func (s *Service) loginLDAP(ctx context.Context, username, password string) (*domain.User, error) {
	if s.ldap == nil {
		return nil, ErrInvalidCredentials
	}
	ldapUser, email, fullName, err := s.ldap.Authenticate(ctx, username, password)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	// Link by username, then by email.
	if u, err := s.users.ByUsername(ctx, username); err == nil {
		return u, nil
	}
	if email != "" {
		if u, err := s.users.ByEmail(ctx, email); err == nil {
			return u, nil
		}
	}
	name := ldapUser
	if name == "" {
		name = username
	}
	if email == "" {
		email = name + "@ldap.local"
	}
	u, err := s.users.Create(ctx, user.CreateParams{Username: name, Email: email})
	if err != nil {
		// Race or name conflict: try the requested username.
		if u2, e2 := s.users.ByUsername(ctx, username); e2 == nil {
			return u2, nil
		}
		return nil, err
	}
	if fullName != "" {
		u.FullName = fullName
		_ = s.users.UpdateProfile(ctx, u)
	}
	return u, nil
}

// ResetPassword sets a new random password for the user and returns it once.
func (s *Service) ResetPassword(ctx context.Context, u *domain.User) (string, error) {
	key, err := randomBytes(12)
	if err != nil {
		return "", err
	}
	pw := fmt.Sprintf("%x", key)
	hash, err := s.hasher.Hash(pw)
	if err != nil {
		return "", err
	}
	if err := s.users.UpdatePassword(ctx, u.ID, hash); err != nil {
		return "", err
	}
	return pw, nil
}

func randomBytes(n int) ([]byte, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

// IsInitialized reports whether at least one user account exists. An empty
// database means the instance still needs its first administrator.
func (s *Service) IsInitialized(ctx context.Context) (bool, error) {
	first, err := s.isFirstUser(ctx)
	if err != nil {
		return false, err
	}
	return !first, nil
}

// SetupParams describes the first-run administrator creation request.
type SetupParams struct {
	Username string
	Email    string
	Password string
}

// Setup creates the initial administrator account. It fails with ErrAlreadySetUp
// when any user already exists so the endpoint cannot be reused.
func (s *Service) Setup(ctx context.Context, p SetupParams) (*domain.User, error) {
	if len(p.Password) < 8 {
		return nil, ErrWeakPassword
	}
	first, err := s.isFirstUser(ctx)
	if err != nil {
		return nil, err
	}
	if !first {
		return nil, ErrAlreadySetUp
	}
	hash, err := s.hasher.Hash(p.Password)
	if err != nil {
		return nil, err
	}
	return s.users.Create(ctx, user.CreateParams{
		Username:     p.Username,
		Email:        p.Email,
		PasswordHash: hash,
		IsAdmin:      true,
	})
}

// RevokeAllSessions deletes every session for a user. Used after a password
// change, 2FA reset or deactivation so stolen cookies stop working (M-02).
func (s *Service) RevokeAllSessions(ctx context.Context, userID int64) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM sessions WHERE user_id = ?`), userID)
	return err
}
