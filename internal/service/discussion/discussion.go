// SPDX-License-Identifier: AGPL-3.0
// Package discussion implements repository discussions with categories and
// Q&A answers.
package discussion

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates a missing discussion/comment/category.
var ErrNotFound = errors.New("discussion: not found")

// Category is a discussion category.
type Category struct {
	ID           int64  `json:"id"`
	RepoID       int64  `json:"repo_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	IsAnswerable bool   `json:"is_answerable"`
}

// Discussion is a discussion thread.
type Discussion struct {
	ID         int64     `json:"id"`
	RepoID     int64     `json:"repo_id"`
	CategoryID *int64    `json:"category_id"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	AuthorID   int64     `json:"author_id"`
	IsAnswered bool      `json:"is_answered"`
	CreatedAt  time.Time `json:"created_at"`
}

// Comment is a reply in a discussion.
type Comment struct {
	ID           int64     `json:"id"`
	DiscussionID int64     `json:"discussion_id"`
	AuthorID     int64     `json:"author_id"`
	Body         string    `json:"body"`
	IsAnswer     bool      `json:"is_answer"`
	CreatedAt    time.Time `json:"created_at"`
}

// DefaultCategories are seeded for each repository on first use.
var DefaultCategories = []Category{
	{Name: "Announcements", Description: "公告与更新", IsAnswerable: false},
	{Name: "General", Description: "综合讨论", IsAnswerable: false},
	{Name: "Ideas", Description: "想法与建议", IsAnswerable: false},
	{Name: "Q&A", Description: "问答", IsAnswerable: true},
}

// Service manages discussions.
type Service struct {
	st *store.Store
}

// New builds a discussion Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// EnsureCategories seeds the default categories for a repo if none exist.
func (s *Service) EnsureCategories(ctx context.Context, repoID int64) error {
	var n int
	if err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT COUNT(*) FROM discussion_categories WHERE repo_id = ?`), repoID).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	ins := s.st.Rebind(`INSERT INTO discussion_categories (repo_id, name, description, is_answerable) VALUES (?, ?, ?, ?)`)
	for _, c := range DefaultCategories {
		if _, err := s.st.DB.ExecContext(ctx, ins, repoID, c.Name, c.Description, c.IsAnswerable); err != nil {
			return err
		}
	}
	return nil
}

// ListCategories returns the categories for a repo.
func (s *Service) ListCategories(ctx context.Context, repoID int64) ([]*Category, error) {
	rows, err := s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT id, repo_id, name, description, is_answerable FROM discussion_categories WHERE repo_id = ? ORDER BY id`), repoID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.RepoID, &c.Name, &c.Description, &c.IsAnswerable); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// Create opens a discussion.
func (s *Service) Create(ctx context.Context, repoID, authorID int64, categoryID *int64, title, body string) (*Discussion, error) {
	q := s.st.Rebind(`INSERT INTO discussions (repo_id, category_id, title, body, author_id) VALUES (?, ?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, repoID, categoryID, title, body, authorID)
	if err != nil {
		return nil, fmt.Errorf("discussion: insert: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

// ByID loads a discussion.
func (s *Service) ByID(ctx context.Context, id int64) (*Discussion, error) {
	q := s.st.Rebind(`SELECT id, repo_id, category_id, title, body, author_id, is_answered, created_at FROM discussions WHERE id = ?`)
	var d Discussion
	var cat sql.NullInt64
	err := s.st.DB.QueryRowContext(ctx, q, id).Scan(&d.ID, &d.RepoID, &cat, &d.Title, &d.Body, &d.AuthorID, &d.IsAnswered, &d.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if cat.Valid {
		d.CategoryID = &cat.Int64
	}
	return &d, nil
}

// List returns discussions for a repo, optionally filtered by category.
func (s *Service) List(ctx context.Context, repoID int64, categoryID *int64, limit int) ([]*Discussion, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	var rows *sql.Rows
	var err error
	if categoryID != nil {
		rows, err = s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT id, repo_id, category_id, title, body, author_id, is_answered, created_at FROM discussions WHERE repo_id = ? AND category_id = ? ORDER BY id DESC LIMIT ?`), repoID, *categoryID, limit)
	} else {
		rows, err = s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT id, repo_id, category_id, title, body, author_id, is_answered, created_at FROM discussions WHERE repo_id = ? ORDER BY id DESC LIMIT ?`), repoID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*Discussion
	for rows.Next() {
		var d Discussion
		var cat sql.NullInt64
		if err := rows.Scan(&d.ID, &d.RepoID, &cat, &d.Title, &d.Body, &d.AuthorID, &d.IsAnswered, &d.CreatedAt); err != nil {
			return nil, err
		}
		if cat.Valid {
			d.CategoryID = &cat.Int64
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}

// AddComment posts a reply.
func (s *Service) AddComment(ctx context.Context, discussionID, authorID int64, body string) (*Comment, error) {
	q := s.st.Rebind(`INSERT INTO discussion_comments (discussion_id, author_id, body) VALUES (?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, discussionID, authorID, body)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Comment{ID: id, DiscussionID: discussionID, AuthorID: authorID, Body: body, CreatedAt: time.Now().UTC()}, nil
}

// ListComments returns the replies for a discussion.
func (s *Service) ListComments(ctx context.Context, discussionID int64) ([]*Comment, error) {
	rows, err := s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT id, discussion_id, author_id, body, is_answer, created_at FROM discussion_comments WHERE discussion_id = ? ORDER BY id`), discussionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.DiscussionID, &c.AuthorID, &c.Body, &c.IsAnswer, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// MarkAnswer marks a comment as the accepted answer (Q&A only) and flags the
// discussion as answered.
func (s *Service) MarkAnswer(ctx context.Context, discussionID, commentID int64) error {
	d, err := s.ByID(ctx, discussionID)
	if err != nil {
		return err
	}
	answerable := false
	if d.CategoryID != nil {
		var a bool
		_ = s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT is_answerable FROM discussion_categories WHERE id = ?`), *d.CategoryID).Scan(&a)
		answerable = a
	}
	if !answerable {
		return fmt.Errorf("discussion: only Q&A discussions can be marked answered")
	}
	if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE discussion_comments SET is_answer = ? WHERE discussion_id = ?`), false, discussionID); err != nil {
		return err
	}
	if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE discussion_comments SET is_answer = ? WHERE id = ? AND discussion_id = ?`), true, commentID, discussionID); err != nil {
		return err
	}
	_, err = s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE discussions SET is_answered = ?, updated_at = ? WHERE id = ?`), true, time.Now().UTC(), discussionID)
	return err
}

// Delete removes a discussion owned by the author.
func (s *Service) Delete(ctx context.Context, discussionID, authorID int64) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM discussions WHERE id = ? AND author_id = ?`), discussionID, authorID)
	return err
}
