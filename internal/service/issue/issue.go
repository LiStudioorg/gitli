// SPDX-License-Identifier: AGPL-3.0
// Package issue manages issues, labels, milestones and comments.
package issue

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates a missing issue.
var ErrNotFound = errors.New("issue: not found")

// Service manages issues.
type Service struct {
	st *store.Store
}

// New builds an issue Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Create opens a new issue, allocating the next number per repo.
func (s *Service) Create(ctx context.Context, repoID, authorID int64, title, body string) (*domain.Issue, error) {
	number, err := s.nextNumber(ctx, repoID)
	if err != nil {
		return nil, err
	}
	q := s.st.Rebind(`INSERT INTO issues (repo_id, number, title, body, author_id) VALUES (?, ?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, repoID, number, title, body, authorID)
	if err != nil {
		return nil, fmt.Errorf("issue: insert: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

// ByID fetches an issue.
func (s *Service) ByID(ctx context.Context, id int64) (*domain.Issue, error) {
	q := s.st.Rebind(`SELECT id, repo_id, number, title, body, author_id, state, milestone_id, closed_at, created_at, updated_at FROM issues WHERE id = ?`)
	var i domain.Issue
	err := s.st.DB.QueryRowContext(ctx, q, id).Scan(&i.ID, &i.RepoID, &i.Number, &i.Title, &i.Body, &i.AuthorID, &i.State, &i.MilestoneID, &i.ClosedAt, &i.CreatedAt, &i.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &i, err
}

// ByNumber fetches an issue by repo and number.
func (s *Service) ByNumber(ctx context.Context, repoID int64, number int) (*domain.Issue, error) {
	q := s.st.Rebind(`SELECT id FROM issues WHERE repo_id = ? AND number = ?`)
	var id int64
	err := s.st.DB.QueryRowContext(ctx, q, repoID, number).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

// List returns issues for a repo, optionally filtered by state.
func (s *Service) List(ctx context.Context, repoID int64, state string, limit, offset int) ([]*domain.Issue, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	var (
		rows *sql.Rows
		err  error
	)
	if state == "" {
		rows, err = s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT id, repo_id, number, title, body, author_id, state, milestone_id, closed_at, created_at, updated_at FROM issues WHERE repo_id = ? ORDER BY number DESC LIMIT ? OFFSET ?`), repoID, limit, offset)
	} else {
		rows, err = s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT id, repo_id, number, title, body, author_id, state, milestone_id, closed_at, created_at, updated_at FROM issues WHERE repo_id = ? AND state = ? ORDER BY number DESC LIMIT ? OFFSET ?`), repoID, state, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Issue
	for rows.Next() {
		var i domain.Issue
		if err := rows.Scan(&i.ID, &i.RepoID, &i.Number, &i.Title, &i.Body, &i.AuthorID, &i.State, &i.MilestoneID, &i.ClosedAt, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &i)
	}
	return out, rows.Err()
}

// SetState updates the open/closed state.
func (s *Service) SetState(ctx context.Context, id int64, state string) error {
	var closedAt any
	if state == "closed" {
		closedAt = time.Now().UTC()
	}
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE issues SET state = ?, closed_at = ?, updated_at = ? WHERE id = ?`), state, closedAt, time.Now().UTC(), id)
	return err
}

// AddLabel attaches a label to an issue.
func (s *Service) AddLabel(ctx context.Context, issueID, labelID int64) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`INSERT INTO issue_labels (issue_id, label_id) VALUES (?, ?)`), issueID, labelID)
	return err
}

// Labels lists labels for an issue.
func (s *Service) Labels(ctx context.Context, issueID int64) ([]*domain.Label, error) {
	q := s.st.Rebind(`SELECT l.id, l.repo_id, l.name, l.color, l.description, l.created_at FROM labels l JOIN issue_labels il ON il.label_id = l.id WHERE il.issue_id = ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Label
	for rows.Next() {
		var l domain.Label
		if err := rows.Scan(&l.ID, &l.RepoID, &l.Name, &l.Color, &l.Description, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &l)
	}
	return out, rows.Err()
}

func (s *Service) nextNumber(ctx context.Context, repoID int64) (int, error) {
	var max sql.NullInt64
	if err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT MAX(number) FROM issues WHERE repo_id = ?`), repoID).Scan(&max); err != nil {
		return 0, err
	}
	return int(max.Int64) + 1, nil
}
