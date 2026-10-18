// SPDX-License-Identifier: AGPL-3.0
// Package pull manages pull requests, reviews and review comments.
package pull

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates a missing pull request.
var ErrNotFound = errors.New("pull: not found")

// Service manages pull requests.
type Service struct {
	st *store.Store
}

// New builds a pull Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// CreateParams describes a pull request creation.
type CreateParams struct {
	RepoID     int64
	AuthorID   int64
	Title      string
	Body       string
	HeadRepoID *int64
	HeadBranch string
	BaseBranch string
}

// Create opens a new pull request.
func (s *Service) Create(ctx context.Context, p CreateParams) (*domain.Pull, error) {
	number, err := s.nextNumber(ctx, p.RepoID)
	if err != nil {
		return nil, err
	}
	q := s.st.Rebind(`INSERT INTO pulls (repo_id, number, title, body, author_id, head_repo_id, head_branch, base_branch) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, p.RepoID, number, p.Title, p.Body, p.AuthorID, p.HeadRepoID, p.HeadBranch, p.BaseBranch)
	if err != nil {
		return nil, fmt.Errorf("pull: insert: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

// ByID fetches a pull request.
func (s *Service) ByID(ctx context.Context, id int64) (*domain.Pull, error) {
	q := s.st.Rebind(`SELECT id, repo_id, number, title, body, author_id, head_repo_id, head_branch, base_branch, state, merged, merged_by, merged_at, created_at, updated_at FROM pulls WHERE id = ?`)
	var p domain.Pull
	err := s.st.DB.QueryRowContext(ctx, q, id).Scan(&p.ID, &p.RepoID, &p.Number, &p.Title, &p.Body, &p.AuthorID, &p.HeadRepoID, &p.HeadBranch, &p.BaseBranch, &p.State, &p.Merged, &p.MergedBy, &p.MergedAt, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &p, err
}

// ByNumber fetches a pull by repo and number.
func (s *Service) ByNumber(ctx context.Context, repoID int64, number int) (*domain.Pull, error) {
	q := s.st.Rebind(`SELECT id FROM pulls WHERE repo_id = ? AND number = ?`)
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

// List returns pulls for a repo.
func (s *Service) List(ctx context.Context, repoID int64, state string, limit, offset int) ([]*domain.Pull, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	base := `SELECT p.id, p.repo_id, p.number, p.title, p.body, p.author_id, p.head_repo_id, p.head_branch, p.base_branch, p.state, p.merged, p.merged_by, p.merged_at, COALESCE(u.username, ''), p.created_at, p.updated_at FROM pulls p LEFT JOIN users u ON u.id = p.author_id WHERE p.repo_id = ?`
	var (
		rows *sql.Rows
		err  error
	)
	if state == "" {
		rows, err = s.st.DB.QueryContext(ctx, s.st.Rebind(base+` ORDER BY p.number DESC LIMIT ? OFFSET ?`), repoID, limit, offset)
	} else {
		rows, err = s.st.DB.QueryContext(ctx, s.st.Rebind(base+` AND p.state = ? ORDER BY p.number DESC LIMIT ? OFFSET ?`), repoID, state, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.Pull
	for rows.Next() {
		var p domain.Pull
		if err := rows.Scan(&p.ID, &p.RepoID, &p.Number, &p.Title, &p.Body, &p.AuthorID, &p.HeadRepoID, &p.HeadBranch, &p.BaseBranch, &p.State, &p.Merged, &p.MergedBy, &p.MergedAt, &p.AuthorName, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

// MarkMerged records a merge.
func (s *Service) MarkMerged(ctx context.Context, id, by int64) error {
	now := time.Now().UTC()
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE pulls SET merged = ?, merged_by = ?, merged_at = ?, state = 'closed', updated_at = ? WHERE id = ?`), true, by, now, now, id)
	return err
}

// SetState updates the pull state.
func (s *Service) SetState(ctx context.Context, id int64, state string) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE pulls SET state = ?, updated_at = ? WHERE id = ?`), state, time.Now().UTC(), id)
	return err
}

// AddReview records a review.
func (s *Service) AddReview(ctx context.Context, pullID, authorID int64, state, body string) (*domain.Review, error) {
	q := s.st.Rebind(`INSERT INTO reviews (pull_id, author_id, state, body) VALUES (?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, pullID, authorID, state, body)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &domain.Review{ID: id, PullID: pullID, AuthorID: authorID, State: state, Body: body}, nil
}

// ListReviews lists reviews for a pull.
func (s *Service) ListReviews(ctx context.Context, pullID int64) ([]*domain.Review, error) {
	q := s.st.Rebind(`SELECT id, pull_id, author_id, state, body, created_at FROM reviews WHERE pull_id = ? ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q, pullID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.Review
	for rows.Next() {
		var r domain.Review
		if err := rows.Scan(&r.ID, &r.PullID, &r.AuthorID, &r.State, &r.Body, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

func (s *Service) nextNumber(ctx context.Context, repoID int64) (int, error) {
	var max sql.NullInt64
	if err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT MAX(number) FROM pulls WHERE repo_id = ?`), repoID).Scan(&max); err != nil {
		return 0, err
	}
	return int(max.Int64) + 1, nil
}
