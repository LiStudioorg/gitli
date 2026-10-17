// SPDX-License-Identifier: AGPL-3.0
// Package explore serves trending repositories and topic aggregation.
package explore

import (
	"context"
	"strings"
	"time"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Service powers the explore page.
type Service struct {
	st *store.Store
}

// New builds an explore Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// RepoCard is a lightweight repo summary for listing.
type RepoCard struct {
	ID          int64  `json:"id"`
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Stars       int    `json:"stars_count"`
	Forks       int    `json:"forks_count"`
	Topic       string `json:"topic,omitempty"`
}

// RecomputeTrending aggregates a trending score for every public repo and
// writes the cache table. score = stars*10 + forks*5 (+ recent push bonus).
func (s *Service) RecomputeTrending(ctx context.Context) (int, error) {
	if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM trending_cache`)); err != nil {
		return 0, err
	}
	q := s.st.Rebind(`INSERT INTO trending_cache (repo_id, score, computed_at)
		SELECT id, (stars_count * 10 + forks_count * 5), ? FROM repos WHERE is_private = ? AND deleted_at IS NULL AND archived = ?`)
	if _, err := s.st.DB.ExecContext(ctx, q, time.Now().UTC(), false, false); err != nil {
		return 0, err
	}
	var n int
	if err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT COUNT(*) FROM trending_cache`)).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// Trending returns top repos by cached score, optionally filtered by language.
func (s *Service) Trending(ctx context.Context, language string, limit int) ([]RepoCard, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	where := []string{"r.is_private = ?", "r.deleted_at IS NULL", "r.archived = ?"}
	args := []any{false, false}
	if language != "" {
		where = append(where, "r.id IN (SELECT repo_id FROM repo_languages WHERE language = ?)")
		args = append(args, language)
	}
	args = append(args, limit)
	q := s.st.Rebind(`SELECT r.id, r.owner_id, r.owner_type, r.name, r.description, r.stars_count, r.forks_count
		FROM trending_cache t JOIN repos r ON r.id = t.repo_id
		WHERE ` + strings.Join(where, " AND ") + ` ORDER BY t.score DESC LIMIT ?`)
	return s.queryCards(ctx, q, args...)
}

// ByTopic returns public repos carrying a topic (with optional language filter).
func (s *Service) ByTopic(ctx context.Context, topic, language string, limit int) ([]RepoCard, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	where := []string{"r.is_private = ?", "r.deleted_at IS NULL", "rt.topic = ?"}
	args := []any{false, topic}
	if language != "" {
		where = append(where, "r.id IN (SELECT repo_id FROM repo_languages WHERE language = ?)")
		args = append(args, language)
	}
	args = append(args, limit)
	q := s.st.Rebind(`SELECT r.id, r.owner_id, r.owner_type, r.name, r.description, r.stars_count, r.forks_count
		FROM repo_topics rt JOIN repos r ON r.id = rt.repo_id
		WHERE ` + strings.Join(where, " AND ") + ` ORDER BY r.stars_count DESC, r.id DESC LIMIT ?`)
	return s.queryCards(ctx, q, args...)
}

// TopTopics returns the most used topics with repo counts.
func (s *Service) TopTopics(ctx context.Context, limit int) ([]TopicCount, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	q := s.st.Rebind(`SELECT rt.topic, COUNT(*) AS c FROM repo_topics rt JOIN repos r ON r.id = rt.repo_id
		WHERE r.is_private = ? AND r.deleted_at IS NULL
		GROUP BY rt.topic ORDER BY c DESC, rt.topic LIMIT ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, false, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []TopicCount
	for rows.Next() {
		var t TopicCount
		if err := rows.Scan(&t.Topic, &t.Count); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// TopicCount is a topic with its repository count.
type TopicCount struct {
	Topic string `json:"topic"`
	Count int    `json:"count"`
}

func (s *Service) queryCards(ctx context.Context, query string, args ...any) ([]RepoCard, error) {
	rows, err := s.st.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []RepoCard{}
	for rows.Next() {
		var c RepoCard
		var ownerID int64
		var ownerType string
		if err := rows.Scan(&c.ID, &ownerID, &ownerType, &c.Name, &c.Description, &c.Stars, &c.Forks); err != nil {
			return nil, err
		}
		c.Owner = s.ownerName(ctx, ownerID, ownerType)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Service) ownerName(ctx context.Context, ownerID int64, ownerType string) string {
	if ownerType == "org" {
		var name string
		if err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT name FROM orgs WHERE id = ?`), ownerID).Scan(&name); err == nil {
			return name
		}
	}
	var name string
	if err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT username FROM users WHERE id = ?`), ownerID).Scan(&name); err == nil {
		return name
	}
	return ""
}
