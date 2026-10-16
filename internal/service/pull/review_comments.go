// SPDX-License-Identifier: AGPL-3.0
package pull

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ReviewComment is a line comment attached to a review.
type ReviewComment struct {
	ID        int64
	ReviewID  int64
	Path      string
	Line      int
	Body      string
	CreatedAt time.Time
}

// AddReviewComment inserts a comment on a review.
func (s *Service) AddReviewComment(ctx context.Context, reviewID int64, path string, line int, body string) (*ReviewComment, error) {
	q := s.st.Rebind(`INSERT INTO review_comments (review_id, path, line, body) VALUES (?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, reviewID, path, line, body)
	if err != nil {
		return nil, fmt.Errorf("pull: insert review comment: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.reviewCommentByID(ctx, id)
}

// ListReviewComments lists comments for a review in insertion order.
func (s *Service) ListReviewComments(ctx context.Context, reviewID int64) ([]ReviewComment, error) {
	q := s.st.Rebind(`SELECT id, review_id, path, line, body, created_at FROM review_comments WHERE review_id = ? ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q, reviewID)
	if err != nil {
		return nil, fmt.Errorf("pull: list review comments: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []ReviewComment
	for rows.Next() {
		var c ReviewComment
		if err := rows.Scan(&c.ID, &c.ReviewID, &c.Path, &c.Line, &c.Body, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Service) reviewCommentByID(ctx context.Context, id int64) (*ReviewComment, error) {
	q := s.st.Rebind(`SELECT id, review_id, path, line, body, created_at FROM review_comments WHERE id = ?`)
	var c ReviewComment
	err := s.st.DB.QueryRowContext(ctx, q, id).Scan(&c.ID, &c.ReviewID, &c.Path, &c.Line, &c.Body, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("pull: query review comment: %w", err)
	}
	return &c, nil
}
