// SPDX-License-Identifier: AGPL-3.0
// Package topic manages repository topics (labels used for discovery).
package topic

import (
	"context"
	"fmt"
	"strings"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// MaxLength bounds a topic name.
const MaxLength = 50

// Service manages repository topics.
type Service struct {
	st *store.Store
}

// New builds a topic Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Add validates and stores a topic, ignoring duplicates.
func (s *Service) Add(ctx context.Context, repoID int64, name string) error {
	if repoID <= 0 {
		return fmt.Errorf("topic: invalid repo id")
	}
	name, err := normalize(name)
	if err != nil {
		return err
	}
	var n int
	q := s.st.Rebind(`SELECT COUNT(*) FROM repo_topics WHERE repo_id = ? AND topic = ?`)
	if err := s.st.DB.QueryRowContext(ctx, q, repoID, name).Scan(&n); err != nil {
		return fmt.Errorf("topic: lookup: %w", err)
	}
	if n > 0 {
		return nil
	}
	insert := s.st.Rebind(`INSERT INTO repo_topics (repo_id, topic) VALUES (?, ?)`)
	if _, err := s.st.DB.ExecContext(ctx, insert, repoID, name); err != nil {
		return fmt.Errorf("topic: insert: %w", err)
	}
	return nil
}

// Remove deletes a topic from a repository.
func (s *Service) Remove(ctx context.Context, repoID int64, name string) error {
	if repoID <= 0 {
		return fmt.Errorf("topic: invalid repo id")
	}
	name, err := normalize(name)
	if err != nil {
		return err
	}
	q := s.st.Rebind(`DELETE FROM repo_topics WHERE repo_id = ? AND topic = ?`)
	if _, err := s.st.DB.ExecContext(ctx, q, repoID, name); err != nil {
		return fmt.Errorf("topic: delete: %w", err)
	}
	return nil
}

// List returns the topics of a repository in insertion order.
func (s *Service) List(ctx context.Context, repoID int64) ([]string, error) {
	q := s.st.Rebind(`SELECT topic FROM repo_topics WHERE repo_id = ? ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID)
	if err != nil {
		return nil, fmt.Errorf("topic: list: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// normalize lowercases and trims a topic, rejecting invalid names.
func normalize(name string) (string, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if err := Validate(name); err != nil {
		return "", err
	}
	return name, nil
}

// Validate reports whether name is a legal topic: lowercase alphanumeric plus
// '-' and '.', at most MaxLength characters, and not starting with '-'.
func Validate(name string) error {
	if name == "" {
		return fmt.Errorf("topic: name is required")
	}
	if len(name) > MaxLength {
		return fmt.Errorf("topic: name too long (max %d)", MaxLength)
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("topic: name must not start with '-'")
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '.':
		default:
			return fmt.Errorf("topic: invalid character %q", r)
		}
	}
	return nil
}
