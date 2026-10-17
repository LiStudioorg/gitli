// SPDX-License-Identifier: AGPL-3.0
package domain

import "time"

// OwnerType distinguishes user-owned and org-owned repos.
type OwnerType string

const (
	OwnerUser OwnerType = "user"
	OwnerOrg  OwnerType = "org"
)

// Perm represents a permission level; higher values are stronger.
type Perm int

const (
	PermNone Perm = iota
	PermRead
	PermWrite
	PermAdmin
)

// ParsePerm maps a string to a Perm.
func ParsePerm(s string) Perm {
	switch s {
	case "read":
		return PermRead
	case "write":
		return PermWrite
	case "admin":
		return PermAdmin
	default:
		return PermNone
	}
}

// String renders a Perm back to its canonical string.
func (p Perm) String() string {
	switch p {
	case PermRead:
		return "read"
	case PermWrite:
		return "write"
	case PermAdmin:
		return "admin"
	default:
		return "none"
	}
}

// Repo is a Git repository.
type Repo struct {
	ID            int64
	OwnerID       int64
	OwnerType     OwnerType
	OrgID         *int64
	Name          string
	Description   string
	DefaultBranch string
	IsPrivate     bool
	IsFork        bool
	ForkID        *int64
	IsMirror      bool
	StarsCount    int
	ForksCount    int
	SizeKB        int64
	Archived      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Collaborator grants a user explicit access to a repo.
type Collaborator struct {
	ID         int64
	RepoID     int64
	UserID     int64
	Permission Perm
	CreatedAt  time.Time
}

// BranchProtection protects branches matching a pattern.
type BranchProtection struct {
	ID            int64
	RepoID        int64
	Pattern       string
	RequireReview bool
	RequireSigned bool
	CreatedAt     time.Time
}

// TagProtection protects tags matching a pattern.
type TagProtection struct {
	ID        int64
	RepoID    int64
	Pattern   string
	CreatedAt time.Time
}

// Org is an organization.
type Org struct {
	ID          int64
	Name        string
	DisplayName string
	Description string
	AvatarURL   string
	OwnerID     int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Team groups members within an org.
type Team struct {
	ID          int64
	OrgID       int64
	Name        string
	Description string
	Permission  Perm
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TeamMember is a membership in a team.
type TeamMember struct {
	ID        int64
	TeamID    int64
	UserID    int64
	Role      string
	CreatedAt time.Time
}

// DeployKey is a repository-scoped SSH key used for automated access.
type DeployKey struct {
	ID          int64
	RepoID      int64
	Title       string
	Fingerprint string
	PublicKey   string
	ReadOnly    bool
	CreatedAt   time.Time
	LastUsedAt  *time.Time
}
