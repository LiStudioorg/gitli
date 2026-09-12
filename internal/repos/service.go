package repos

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gitli/internal/db"
	"gitli/internal/git"
)

var (
	ErrRepoNotFound = errors.New("repo not found")
	ErrNameTaken    = errors.New("repo name already taken")
	ErrInvalidInput = errors.New("invalid input")
	ErrForbidden    = errors.New("forbidden")
)

// Visibility 三级可见性。
type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityOrg     Visibility = "org"
	VisibilityPrivate Visibility = "private"
)

func (v Visibility) Valid() bool {
	return v == VisibilityPublic || v == VisibilityOrg || v == VisibilityPrivate
}

type Service struct {
	q        db.Querier
	reposDir string
}

func NewService(q db.Querier, reposDir string) *Service {
	return &Service{q: q, reposDir: reposDir}
}

// Create 创建仓库并初始化裸仓库目录。
func (s *Service) Create(ctx context.Context, owner db.User, name, description, visibility string) (db.Repo, error) {
	if err := git.ValidateRepoName(name); err != nil {
		return db.Repo{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	v := Visibility(visibility)
	if visibility == "" {
		v = VisibilityPublic
	}
	if !v.Valid() {
		return db.Repo{}, fmt.Errorf("%w: bad visibility", ErrInvalidInput)
	}
	now := time.Now().UTC().Unix()
	repo, err := s.q.CreateRepo(ctx, db.CreateRepoParams{
		OwnerID:     owner.ID,
		Name:        name,
		Description: description,
		Visibility:  string(v),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		if isUnique(err) {
			return db.Repo{}, ErrNameTaken
		}
		return db.Repo{}, fmt.Errorf("create repo: %w", err)
	}
	path := git.RepoPath(s.reposDir, owner.Username, name)
	if err := git.InitRepo(ctx, path); err != nil {
		// 回滚 DB 记录，避免脏数据
		_ = s.q.DeleteRepo(ctx, repo.ID)
		return db.Repo{}, fmt.Errorf("init repo: %w", err)
	}
	return repo, nil
}

// GetByUsernameAndName 查仓库（附带 owner 用户名已经隐含在参数里）。
func (s *Service) Get(ctx context.Context, ownerName, name string) (db.Repo, db.User, error) {
	repo, err := s.q.GetRepoByOwnerAndName(ctx, db.GetRepoByOwnerAndNameParams{
		Username: ownerName,
		Name:     name,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return db.Repo{}, db.User{}, ErrRepoNotFound
	}
	if err != nil {
		return db.Repo{}, db.User{}, fmt.Errorf("get repo: %w", err)
	}
	owner, err := s.q.GetUserByID(ctx, repo.OwnerID)
	if err != nil {
		return db.Repo{}, db.User{}, fmt.Errorf("get owner: %w", err)
	}
	return repo, owner, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (db.Repo, error) {
	r, err := s.q.GetRepoByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return db.Repo{}, ErrRepoNotFound
	}
	return r, err
}

func (s *Service) ListByOwner(ctx context.Context, ownerName string) ([]db.Repo, error) {
	return s.q.ListReposByOwner(ctx, ownerName)
}

func (s *Service) ListPublic(ctx context.Context) ([]db.Repo, error) {
	return s.q.ListPublicRepos(ctx)
}

// Delete 删除 DB 记录与磁盘目录。
func (s *Service) Delete(ctx context.Context, repo db.Repo, owner db.User) error {
	if err := s.q.DeleteRepo(ctx, repo.ID); err != nil {
		return fmt.Errorf("delete repo: %w", err)
	}
	return git.DeleteRepo(git.RepoPath(s.reposDir, owner.Username, repo.Name))
}

// RootDir 返回所有仓库的公共父目录（GIT_PROJECT_ROOT 用）。
func (s *Service) RootDir() string { return s.reposDir }

// DiskPath 返回裸仓库路径。
func (s *Service) DiskPath(ownerName, repoName string) string {
	return git.RepoPath(s.reposDir, ownerName, repoName)
}

func isUnique(err error) bool {
	return err != nil && (contains(err.Error(), "UNIQUE constraint failed") || contains(err.Error(), "constraint failed: UNIQUE"))
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
