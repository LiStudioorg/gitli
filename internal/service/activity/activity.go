// SPDX-License-Identifier: AGPL-3.0
// Package activity records and serves the public activity timeline.
package activity

import (
	"context"
	"database/sql"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Service manages activities.
type Service struct {
	st *store.Store
}

// New builds an activity Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Record inserts an activity entry.
func (s *Service) Record(ctx context.Context, a *domain.Activity) error {
	q := s.st.Rebind(`INSERT INTO activities (actor_id, verb, object_type, object_id, repo_id, repo_name, summary) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	_, err := s.st.DB.ExecContext(ctx, q, a.ActorID, a.Verb, a.ObjectType, a.ObjectID, a.RepoID, a.RepoName, a.Summary)
	return err
}

// ListByActor returns a user's recent activity.
func (s *Service) ListByActor(ctx context.Context, actorID int64, limit int) ([]*domain.Activity, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	q := s.st.Rebind(`SELECT ` + activityColumns + ` FROM activities WHERE actor_id = ? ORDER BY id DESC LIMIT ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, actorID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scan(rows)
}

// ListByRepo returns a repository's recent activity.
func (s *Service) ListByRepo(ctx context.Context, repoID int64, limit int) ([]*domain.Activity, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	q := s.st.Rebind(`SELECT ` + activityColumns + ` FROM activities WHERE repo_id = ? ORDER BY id DESC LIMIT ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scan(rows)
}

// ListPublic returns recent public activities across all repositories.
func (s *Service) ListPublic(ctx context.Context, limit int) ([]*domain.Activity, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	q := s.st.Rebind(`SELECT ` + activityColumns + ` FROM activities WHERE repo_id IS NOT NULL ORDER BY id DESC LIMIT ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scan(rows)
}

const activityColumns = `id, actor_id, verb, object_type, object_id, repo_id, repo_name, summary, created_at`

func scan(rows *sql.Rows) ([]*domain.Activity, error) {
	var out []*domain.Activity
	for rows.Next() {
		var a domain.Activity
		if err := rows.Scan(&a.ID, &a.ActorID, &a.Verb, &a.ObjectType, &a.ObjectID, &a.RepoID, &a.RepoName, &a.Summary, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}
