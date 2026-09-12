package orgs

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
	ErrOrgNotFound  = errors.New("org not found")
	ErrNameTaken    = errors.New("org name already taken")
	ErrInvalidInput = errors.New("invalid input")
)

// Service 组织领域逻辑。
type Service struct {
	q db.Querier
}

func NewService(q db.Querier) *Service {
	return &Service{q: q}
}

// CreateOrg 创建组织（名字校验复用仓库名规则）。
func (s *Service) CreateOrg(ctx context.Context, name, description string) (db.Org, error) {
	if err := git.ValidateRepoName(name); err != nil {
		return db.Org{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	org, err := s.q.CreateOrg(ctx, db.CreateOrgParams{
		Name:        name,
		Description: description,
		CreatedAt:   time.Now().UTC().Unix(),
	})
	if err != nil {
		if isUnique(err) {
			return db.Org{}, ErrNameTaken
		}
		return db.Org{}, fmt.Errorf("create org: %w", err)
	}
	return org, nil
}

func (s *Service) GetByName(ctx context.Context, name string) (db.Org, error) {
	org, err := s.q.GetOrgByName(ctx, name)
	if errors.Is(err, sql.ErrNoRows) {
		return db.Org{}, ErrOrgNotFound
	}
	return org, err
}

func (s *Service) GetByID(ctx context.Context, id int64) (db.Org, error) {
	org, err := s.q.GetOrgByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return db.Org{}, ErrOrgNotFound
	}
	return org, err
}

// AddMember 添加成员；第一个成员自动成为 owner 角色。
func (s *Service) AddMember(ctx context.Context, orgID, userID int64, role string) error {
	n, err := s.q.CountOrgMembers(ctx, orgID)
	if err != nil {
		return fmt.Errorf("count org members: %w", err)
	}
	if n == 0 {
		role = "owner"
	}
	if role != "owner" && role != "member" {
		role = "member"
	}
	return s.q.AddOrgMember(ctx, db.AddOrgMemberParams{
		OrgID:     orgID,
		UserID:    userID,
		Role:      role,
		CreatedAt: time.Now().UTC().Unix(),
	})
}

func (s *Service) IsMember(ctx context.Context, orgID, userID int64) bool {
	n, err := s.q.IsOrgMemberByOrgID(ctx, db.IsOrgMemberByOrgIDParams{OrgID: orgID, UserID: userID})
	return err == nil && n > 0
}

// UserRole 返回用户在组织内的角色；非成员返回空串。
func (s *Service) UserRole(ctx context.Context, orgID, userID int64) string {
	m, err := s.q.GetOrgMember(ctx, db.GetOrgMemberParams{OrgID: orgID, UserID: userID})
	if err != nil {
		return ""
	}
	return m.Role
}

func (s *Service) ListMembers(ctx context.Context, orgID int64) ([]db.ListOrgMembersWithUsersRow, error) {
	return s.q.ListOrgMembersWithUsers(ctx, orgID)
}

func (s *Service) ListRepos(ctx context.Context, name string) ([]db.Repo, error) {
	return s.q.ListOrgRepos(ctx, name)
}

func isUnique(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "UNIQUE constraint failed") || contains(msg, "constraint failed: UNIQUE")
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
