// SPDX-License-Identifier: AGPL-3.0
// Package repo manages repositories, collaborators and protections.
package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/infra/fsrepo"
	"github.com/LiStudioorg/gitli/internal/infra/gitcmd"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates a missing repository.
var ErrNotFound = errors.New("repo: not found")

// ErrExists indicates a repository name conflict.
var ErrExists = errors.New("repo: already exists")

// Service manages repositories.
type Service struct {
	st         *store.Store
	fs         *fsrepo.Layout
	git        *gitcmd.Runner
	gitliBin   string
	configPath string
}

// New builds a repo Service.
func New(st *store.Store, layout *fsrepo.Layout, runner *gitcmd.Runner) *Service {
	return &Service{st: st, fs: layout, git: runner, gitliBin: "gitli"}
}

// SetGitliBinary sets the absolute path to the gitli executable used in hooks.
func (s *Service) SetGitliBinary(path string) {
	if path != "" {
		s.gitliBin = path
	}
}

// SetConfigPath sets the absolute config path baked into installed hooks.
func (s *Service) SetConfigPath(path string) {
	s.configPath = path
}

// CreateParams describes repository creation.
type CreateParams struct {
	OwnerID       int64
	OwnerType     domain.OwnerType
	OrgID         *int64
	Name          string
	Description   string
	DefaultBranch string
	IsPrivate     bool
}

const repoColumns = `id, owner_id, owner_type, org_id, name, description, default_branch, is_private, is_fork, fork_id, is_mirror, stars_count, forks_count, size_kb, archived, created_at, updated_at, deleted_at`

// Create creates a repo record and initializes its bare Git directory.
func (s *Service) Create(ctx context.Context, p CreateParams) (*domain.Repo, error) {
	if p.DefaultBranch == "" {
		p.DefaultBranch = "main"
	}
	if p.OwnerType == "" {
		p.OwnerType = domain.OwnerUser
	}
	if err := validateRepoName(p.Name); err != nil {
		return nil, err
	}
	existing, err := s.ByOwnerName(ctx, p.OwnerID, p.Name)
	if err == nil && existing != nil {
		return nil, ErrExists
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	qt := s.st.Rebind(`INSERT INTO repos (owner_id, owner_type, org_id, name, description, default_branch, is_private) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, qt, p.OwnerID, string(p.OwnerType), p.OrgID, p.Name, p.Description, p.DefaultBranch, p.IsPrivate)
	if err != nil {
		return nil, fmt.Errorf("repo: insert: %w", err)
	}
	id, _ := res.LastInsertId()

	owner := s.ownerSlug(ctx, p.OwnerID, p.OwnerType, p.OrgID)
	path, err := s.fs.Ensure(owner, p.Name)
	if err != nil {
		return nil, err
	}
	if err := s.git.Init(ctx, path, p.DefaultBranch); err != nil {
		return nil, err
	}
	if err := s.fs.InstallHooks(path, owner, p.Name, s.gitliBin, s.configPath); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

// ByID fetches a repo by primary key.
func (s *Service) ByID(ctx context.Context, id int64) (*domain.Repo, error) {
	q := s.st.Rebind(`SELECT ` + repoColumns + ` FROM repos WHERE id = ?`)
	return s.scanOne(ctx, q, id)
}

// ByOwnerName fetches a repo by owner id and name.
func (s *Service) ByOwnerName(ctx context.Context, ownerID int64, name string) (*domain.Repo, error) {
	q := s.st.Rebind(`SELECT ` + repoColumns + ` FROM repos WHERE owner_id = ? AND name = ?`)
	return s.scanOne(ctx, q, ownerID, name)
}

// ListByOwner lists repos for an owner.
func (s *Service) ListByOwner(ctx context.Context, ownerID int64, limit, offset int) ([]*domain.Repo, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	q := s.st.Rebind(`SELECT ` + repoColumns + ` FROM repos WHERE owner_id = ? ORDER BY updated_at DESC LIMIT ? OFFSET ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, ownerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return s.scanMany(rows)
}

// ListPublic lists non-private, non-archived repos ordered by recency.
func (s *Service) ListPublic(ctx context.Context, limit, offset int) ([]*domain.Repo, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	if offset < 0 {
		offset = 0
	}
	q := s.st.Rebind(`SELECT ` + repoColumns + ` FROM repos WHERE is_private = 0 AND archived = 0 ORDER BY updated_at DESC LIMIT ? OFFSET ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return s.scanMany(rows)
}

// Search lists public repos whose name matches the query, ordered by recency.
func (s *Service) Search(ctx context.Context, q string, limit, offset int) ([]*domain.Repo, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	if offset < 0 {
		offset = 0
	}
	pattern := "%" + q + "%"
	query := s.st.Rebind(`SELECT ` + repoColumns + ` FROM repos WHERE is_private = 0 AND archived = 0 AND name LIKE ? ORDER BY updated_at DESC LIMIT ? OFFSET ?`)
	rows, err := s.st.DB.QueryContext(ctx, query, pattern, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return s.scanMany(rows)
}

// Transfer moves a repo to a new owner, relocating on disk.
func (s *Service) Transfer(ctx context.Context, id, newOwnerID int64, newOwnerType domain.OwnerType, newOrgID *int64) error {
	r, err := s.ByID(ctx, id)
	if err != nil {
		return err
	}
	oldOwner := s.ownerSlug(ctx, r.OwnerID, r.OwnerType, r.OrgID)
	newOwner := s.ownerSlug(ctx, newOwnerID, newOwnerType, newOrgID)
	q := s.st.Rebind(`UPDATE repos SET owner_id = ?, owner_type = ?, org_id = ?, updated_at = ? WHERE id = ?`)
	if _, err := s.st.DB.ExecContext(ctx, q, newOwnerID, string(newOwnerType), newOrgID, time.Now().UTC(), id); err != nil {
		return err
	}
	return s.fs.Move(oldOwner, r.Name, newOwner, r.Name)
}

// AddCollaborator grants a user access.
func (s *Service) AddCollaborator(ctx context.Context, repoID, userID int64, perm domain.Perm) error {
	q := s.st.Rebind(`INSERT INTO collaborators (repo_id, user_id, permission) VALUES (?, ?, ?)`)
	_, err := s.st.DB.ExecContext(ctx, q, repoID, userID, perm.String())
	return err
}

// CollaboratorPermission returns explicit collaborator permission.
func (s *Service) CollaboratorPermission(ctx context.Context, repoID, userID int64) (domain.Perm, error) {
	q := s.st.Rebind(`SELECT permission FROM collaborators WHERE repo_id = ? AND user_id = ?`)
	var perm string
	err := s.st.DB.QueryRowContext(ctx, q, repoID, userID).Scan(&perm)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PermNone, nil
	}
	if err != nil {
		return domain.PermNone, err
	}
	return domain.ParsePerm(perm), nil
}

// AddBranchProtection protects branches matching pattern.
func (s *Service) AddBranchProtection(ctx context.Context, repoID int64, pattern string, requireReview, requireSigned, requireCodeowners bool) error {
	q := s.st.Rebind(`INSERT INTO branch_protections (repo_id, pattern, require_reviews, require_signed, require_codeowners_review) VALUES (?, ?, ?, ?, ?)`)
	_, err := s.st.DB.ExecContext(ctx, q, repoID, pattern, requireReview, requireSigned, requireCodeowners)
	return err
}

// ListBranchProtections lists protections for a repo.
func (s *Service) ListBranchProtections(ctx context.Context, repoID int64) ([]*domain.BranchProtection, error) {
	q := s.st.Rebind(`SELECT id, repo_id, pattern, require_reviews, require_signed, require_codeowners_review, created_at FROM branch_protections WHERE repo_id = ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.BranchProtection
	for rows.Next() {
		var b domain.BranchProtection
		if err := rows.Scan(&b.ID, &b.RepoID, &b.Pattern, &b.RequireReview, &b.RequireSigned, &b.RequireCodeowners, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &b)
	}
	return out, rows.Err()
}

// AddTagProtection protects tags matching pattern.
func (s *Service) AddTagProtection(ctx context.Context, repoID int64, pattern string) error {
	q := s.st.Rebind(`INSERT INTO tag_protections (repo_id, pattern) VALUES (?, ?)`)
	_, err := s.st.DB.ExecContext(ctx, q, repoID, pattern)
	return err
}

// ListTagProtectionPatterns returns the protected tag patterns for a repo.
func (s *Service) ListTagProtectionPatterns(ctx context.Context, repoID int64) ([]string, error) {
	q := s.st.Rebind(`SELECT pattern FROM tag_protections WHERE repo_id = ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Service) ownerSlug(ctx context.Context, ownerID int64, ownerType domain.OwnerType, orgID *int64) string {
	if ownerType == domain.OwnerOrg && orgID != nil {
		var name string
		if err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT name FROM orgs WHERE id = ?`), *orgID).Scan(&name); err == nil {
			return name
		}
	}
	var name string
	if err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT username FROM users WHERE id = ?`), ownerID).Scan(&name); err == nil {
		return name
	}
	return fmt.Sprintf("owner-%d", ownerID)
}

func (s *Service) scanOne(ctx context.Context, query string, args ...any) (*domain.Repo, error) {
	row := s.st.DB.QueryRowContext(ctx, query, args...)
	r, err := scanRepo(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return r, err
}

func (s *Service) scanMany(rows *sql.Rows) ([]*domain.Repo, error) {
	var out []*domain.Repo
	for rows.Next() {
		r, err := scanRepo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanRepo(sc scanner) (*domain.Repo, error) {
	var r domain.Repo
	var ownerType string
	var deleted sql.NullTime
	err := sc.Scan(&r.ID, &r.OwnerID, &ownerType, &r.OrgID, &r.Name, &r.Description, &r.DefaultBranch,
		&r.IsPrivate, &r.IsFork, &r.ForkID, &r.IsMirror, &r.StarsCount, &r.ForksCount,
		&r.SizeKB, &r.Archived, &r.CreatedAt, &r.UpdatedAt, &deleted)
	if err != nil {
		return nil, err
	}
	if deleted.Valid {
		t := deleted.Time
		r.DeletedAt = &t
	}
	r.OwnerType = domain.OwnerType(ownerType)
	return &r, nil
}

func validateRepoName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("repo: name required")
	}
	if strings.ContainsAny(name, " /\\") || name == "." || name == ".." {
		return fmt.Errorf("repo: invalid name %q", name)
	}
	return nil
}
