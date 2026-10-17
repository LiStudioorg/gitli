// SPDX-License-Identifier: AGPL-3.0
// Package comment manages comments on issues and pull requests.
package comment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates a missing comment.
var ErrNotFound = errors.New("comment: not found")

// Service manages comments.
type Service struct {
	st *store.Store
}

// New builds a comment Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// CreateParams describes a comment.
type CreateParams struct {
	RepoID   int64
	IssueID  *int64
	PullID   *int64
	AuthorID int64
	Body     string
}

// Create inserts a comment.
func (s *Service) Create(ctx context.Context, p CreateParams) (*domain.Comment, error) {
	q := s.st.Rebind(`INSERT INTO comments (repo_id, issue_id, pull_id, author_id, body) VALUES (?, ?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, p.RepoID, p.IssueID, p.PullID, p.AuthorID, p.Body)
	if err != nil {
		return nil, fmt.Errorf("comment: insert: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

// ByID fetches a comment.
func (s *Service) ByID(ctx context.Context, id int64) (*domain.Comment, error) {
	q := s.st.Rebind(`SELECT id, repo_id, issue_id, pull_id, author_id, body, created_at, updated_at FROM comments WHERE id = ?`)
	var c domain.Comment
	err := s.st.DB.QueryRowContext(ctx, q, id).Scan(&c.ID, &c.RepoID, &c.IssueID, &c.PullID, &c.AuthorID, &c.Body, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &c, err
}

// ListForIssue lists comments on an issue.
func (s *Service) ListForIssue(ctx context.Context, issueID int64) ([]*domain.Comment, error) {
	return s.list(ctx, `issue_id`, issueID)
}

// ListForPull lists comments on a pull request.
func (s *Service) ListForPull(ctx context.Context, pullID int64) ([]*domain.Comment, error) {
	return s.list(ctx, `pull_id`, pullID)
}

func (s *Service) list(ctx context.Context, column string, id int64) ([]*domain.Comment, error) {
	q := s.st.Rebind(`SELECT id, repo_id, issue_id, pull_id, author_id, body, created_at, updated_at FROM comments WHERE ` + column + ` = ? ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.Comment
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(&c.ID, &c.RepoID, &c.IssueID, &c.PullID, &c.AuthorID, &c.Body, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// Delete removes a comment authored by userID.
func (s *Service) Delete(ctx context.Context, id, authorID int64) error {
	return s.st.ExecAffectedOr(ctx, ErrNotFound, s.st.Rebind(`DELETE FROM comments WHERE id = ? AND author_id = ?`), id, authorID)
}
