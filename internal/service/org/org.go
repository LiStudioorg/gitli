// SPDX-License-Identifier: AGPL-3.0
// Package org manages organizations and teams.
package org

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates a missing org or team.
var ErrNotFound = errors.New("org: not found")

// Service manages organizations and teams.
type Service struct {
	st *store.Store
}

// New builds an org Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Create creates an organization owned by ownerID.
func (s *Service) Create(ctx context.Context, name, displayName string, ownerID int64) (*domain.Org, error) {
	q := s.st.Rebind(`INSERT INTO orgs (name, display_name, owner_id) VALUES (?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, name, displayName, ownerID)
	if err != nil {
		return nil, fmt.Errorf("org: insert: %w", err)
	}
	id, _ := res.LastInsertId()
	return &domain.Org{ID: id, Name: name, DisplayName: displayName, OwnerID: ownerID, CreatedAt: time.Now()}, nil
}

// ByName fetches an organization by name.
func (s *Service) ByName(ctx context.Context, name string) (*domain.Org, error) {
	q := s.st.Rebind(`SELECT id, name, display_name, description, avatar_url, owner_id, created_at, updated_at FROM orgs WHERE name = ?`)
	var o domain.Org
	err := s.st.DB.QueryRowContext(ctx, q, name).Scan(&o.ID, &o.Name, &o.DisplayName, &o.Description, &o.AvatarURL, &o.OwnerID, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &o, err
}

// ByID fetches an organization by id.
func (s *Service) ByID(ctx context.Context, id int64) (*domain.Org, error) {
	q := s.st.Rebind(`SELECT id, name, display_name, description, avatar_url, owner_id, created_at, updated_at FROM orgs WHERE id = ?`)
	var o domain.Org
	err := s.st.DB.QueryRowContext(ctx, q, id).Scan(&o.ID, &o.Name, &o.DisplayName, &o.Description, &o.AvatarURL, &o.OwnerID, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &o, err
}

// CreateTeam creates a team within an org.
func (s *Service) CreateTeam(ctx context.Context, orgID int64, name string, permission domain.Perm) (*domain.Team, error) {
	q := s.st.Rebind(`INSERT INTO teams (org_id, name, permission) VALUES (?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, orgID, name, permission.String())
	if err != nil {
		return nil, fmt.Errorf("org: insert team: %w", err)
	}
	id, _ := res.LastInsertId()
	return &domain.Team{ID: id, OrgID: orgID, Name: name, Permission: permission}, nil
}

// AddTeamMember adds a user to a team.
func (s *Service) AddTeamMember(ctx context.Context, teamID, userID int64, role string) error {
	if role == "" {
		role = "member"
	}
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`), teamID, userID, role)
	return err
}

// TeamPermission returns the effective permission a user has in an org via teams.
func (s *Service) TeamPermission(ctx context.Context, orgID, userID int64) (domain.Perm, error) {
	q := s.st.Rebind(`SELECT t.permission FROM teams t JOIN team_members m ON m.team_id = t.id WHERE t.org_id = ? AND m.user_id = ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, orgID, userID)
	if err != nil {
		return domain.PermNone, err
	}
	defer func() { _ = rows.Close() }()
	best := domain.PermNone
	for rows.Next() {
		var perm string
		if err := rows.Scan(&perm); err != nil {
			return domain.PermNone, err
		}
		if p := domain.ParsePerm(perm); p > best {
			best = p
		}
	}
	return best, rows.Err()
}

// IsMember reports whether a user belongs to any team in the org.
func (s *Service) IsMember(ctx context.Context, orgID, userID int64) (bool, error) {
	perm, err := s.TeamPermission(ctx, orgID, userID)
	if err != nil {
		return false, err
	}
	return perm > domain.PermNone, nil
}
