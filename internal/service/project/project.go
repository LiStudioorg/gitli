// SPDX-License-Identifier: AGPL-3.0
// Package project implements kanban project boards.
package project

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates a missing project/column/card.
var ErrNotFound = errors.New("project: not found")

// Project is a board.
type Project struct {
	ID          int64  `json:"id"`
	OwnerID     int64  `json:"owner_id"`
	RepoID      *int64 `json:"repo_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Column is a board column.
type Column struct {
	ID        int64   `json:"id"`
	ProjectID int64   `json:"project_id"`
	Name      string  `json:"name"`
	Position  int     `json:"position"`
	Cards     []*Card `json:"cards,omitempty"`
}

// Card is a board card linked to an issue or pull request.
type Card struct {
	ID       int64  `json:"id"`
	ColumnID int64  `json:"column_id"`
	IssueID  *int64 `json:"issue_id"`
	PullID   *int64 `json:"pull_id"`
	Title    string `json:"title"`
	Position int    `json:"position"`
}

// DefaultColumns are created with every new board.
var DefaultColumns = []string{"To Do", "In Progress", "Done"}

// Service manages project boards.
type Service struct {
	st *store.Store
}

// New builds a project Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Create makes a board with the default columns.
func (s *Service) Create(ctx context.Context, ownerID int64, repoID *int64, name, description string) (*Project, error) {
	if name == "" {
		return nil, fmt.Errorf("project: name required")
	}
	tx, err := s.st.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, s.st.Rebind(`INSERT INTO projects (owner_id, repo_id, name, description) VALUES (?, ?, ?, ?)`), ownerID, repoID, name, description)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	for i, cname := range DefaultColumns {
		if _, err := tx.ExecContext(ctx, s.st.Rebind(`INSERT INTO project_columns (project_id, name, position) VALUES (?, ?, ?)`), id, cname, i); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

// ByID loads a project.
func (s *Service) ByID(ctx context.Context, id int64) (*Project, error) {
	var p Project
	var repo sql.NullInt64
	err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT id, owner_id, repo_id, name, description FROM projects WHERE id = ?`), id).
		Scan(&p.ID, &p.OwnerID, &repo, &p.Name, &p.Description)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if repo.Valid {
		p.RepoID = &repo.Int64
	}
	return &p, nil
}

// ListByRepo lists boards for a repo.
func (s *Service) ListByRepo(ctx context.Context, repoID int64) ([]*Project, error) {
	rows, err := s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT id FROM projects WHERE repo_id = ? ORDER BY id DESC`), repoID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*Project
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if p, err := s.ByID(ctx, id); err == nil {
			out = append(out, p)
		}
	}
	return out, rows.Err()
}

// Board returns a project with its columns and cards.
func (s *Service) Board(ctx context.Context, projectID int64) (*Project, []*Column, error) {
	p, err := s.ByID(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}
	rows, err := s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT id, project_id, name, position FROM project_columns WHERE project_id = ? ORDER BY position, id`), projectID)
	if err != nil {
		return nil, nil, err
	}
	var cols []*Column
	for rows.Next() {
		var c Column
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Name, &c.Position); err != nil {
			_ = rows.Close()
			return nil, nil, err
		}
		cols = append(cols, &c)
	}
	_ = rows.Close()
	for _, c := range cols {
		cards, err := s.listCards(ctx, c.ID)
		if err != nil {
			return nil, nil, err
		}
		c.Cards = cards
	}
	return p, cols, nil
}

func (s *Service) listCards(ctx context.Context, columnID int64) ([]*Card, error) {
	rows, err := s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT id, column_id, issue_id, pull_id, title, position FROM project_cards WHERE column_id = ? ORDER BY position, id`), columnID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []*Card{}
	for rows.Next() {
		var c Card
		var issue, pull sql.NullInt64
		if err := rows.Scan(&c.ID, &c.ColumnID, &issue, &pull, &c.Title, &c.Position); err != nil {
			return nil, err
		}
		if issue.Valid {
			c.IssueID = &issue.Int64
		}
		if pull.Valid {
			c.PullID = &pull.Int64
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// AddCard appends a card to a column.
func (s *Service) AddCard(ctx context.Context, columnID int64, issueID, pullID *int64, title string) (*Card, error) {
	var pos int
	_ = s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT COALESCE(MAX(position)+1,0) FROM project_cards WHERE column_id = ?`), columnID).Scan(&pos)
	res, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`INSERT INTO project_cards (column_id, issue_id, pull_id, title, position) VALUES (?, ?, ?, ?, ?)`), columnID, issueID, pullID, title, pos)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Card{ID: id, ColumnID: columnID, IssueID: issueID, PullID: pullID, Title: title, Position: pos}, nil
}

// MoveCard moves a card to another column/position.
func (s *Service) MoveCard(ctx context.Context, cardID, columnID int64, position int) error {
	res, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE project_cards SET column_id = ?, position = ? WHERE id = ?`), columnID, position, cardID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteCard removes a card.
func (s *Service) DeleteCard(ctx context.Context, cardID int64) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM project_cards WHERE id = ?`), cardID)
	return err
}

// AddColumn appends a column.
func (s *Service) AddColumn(ctx context.Context, projectID int64, name string) (*Column, error) {
	var pos int
	_ = s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT COALESCE(MAX(position)+1,0) FROM project_columns WHERE project_id = ?`), projectID).Scan(&pos)
	res, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`INSERT INTO project_columns (project_id, name, position) VALUES (?, ?, ?)`), projectID, name, pos)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Column{ID: id, ProjectID: projectID, Name: name, Position: pos, Cards: []*Card{}}, nil
}

// Delete removes a project and its columns/cards.
func (s *Service) Delete(ctx context.Context, projectID int64) error {
	if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM project_cards WHERE column_id IN (SELECT id FROM project_columns WHERE project_id = ?)`), projectID); err != nil {
		return err
	}
	if _, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM project_columns WHERE project_id = ?`), projectID); err != nil {
		return err
	}
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM projects WHERE id = ?`), projectID)
	return err
}
