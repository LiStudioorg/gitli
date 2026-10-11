// SPDX-License-Identifier: AGPL-3.0
// Package domain contains pure domain models with no external dependencies.
package domain

import "time"

// User is an account on the platform.
type User struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	FullName     string
	Bio          string
	AvatarURL    string
	Website      string
	Location     string
	IsAdmin      bool
	IsActive     bool
	TOTPEnabled  bool
	TOTPSecret   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// SSHKey is a public key registered to a user.
type SSHKey struct {
	ID          int64
	UserID      int64
	Title       string
	Fingerprint string
	PublicKey   string
	Verified    bool
	LastUsedAt  *time.Time
	CreatedAt   time.Time
}

// TokenKind distinguishes classic and fine-grained tokens.
type TokenKind string

const (
	TokenClassic     TokenKind = "classic"
	TokenFineGrained TokenKind = "fine_grained"
)

// AccessToken is a personal access token.
type AccessToken struct {
	ID         int64
	UserID     int64
	Name       string
	TokenHash  string
	Kind       TokenKind
	Scopes     []string
	RepoGrants []TokenRepoGrant
	LastUsedAt *time.Time
	ExpiresAt  *time.Time
	CreatedAt  time.Time
}

// TokenRepoGrant binds a fine-grained token to a repo with a permission.
type TokenRepoGrant struct {
	RepoID     int64
	Permission string
}

// Session is a server-side login session.
type Session struct {
	ID        int64
	UserID    int64
	TokenHash string
	IP        string
	UserAgent string
	ExpiresAt *time.Time
	CreatedAt time.Time
}
