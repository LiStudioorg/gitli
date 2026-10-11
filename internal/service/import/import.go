// SPDX-License-Identifier: AGPL-3.0
// Package importer imports repositories from external git hosts.
package importer

import (
	"context"
	"fmt"

	"github.com/LiStudioorg/gitli/internal/infra/gitcmd"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Supported import sources.
const (
	SourceGitHub  = "github"
	SourceGitLab  = "gitlab"
	SourceGitea   = "gitea"
	SourceGeneric = "generic"
)

// Import job statuses.
const (
	StatusPending = "pending"
	StatusDone    = "done"
	StatusFailed  = "failed"
)

// Job describes a repository import operation.
type Job struct {
	ID        int64
	RepoID    int64
	Source    string
	SourceURL string
	Status    string
	Message   string
}

// Service manages repository import jobs.
type Service struct {
	st     *store.Store
	runner *gitcmd.Runner
}

// New builds an import Service.
func New(st *store.Store, runner *gitcmd.Runner) *Service {
	return &Service{st: st, runner: runner}
}

var validSources = map[string]bool{
	SourceGitHub:  true,
	SourceGitLab:  true,
	SourceGitea:   true,
	SourceGeneric: true,
}

// Create validates and inserts a new import job in the pending state.
func (s *Service) Create(ctx context.Context, repoID int64, source, sourceURL string) (*Job, error) {
	if !validSources[source] {
		return nil, fmt.Errorf("invalid import source: %q", source)
	}
	if sourceURL == "" {
		return nil, fmt.Errorf("source url is required")
	}
	q := s.st.Rebind(`INSERT INTO repo_imports (repo_id, source, source_url, status, message) VALUES (?, ?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, repoID, source, sourceURL, StatusPending, "")
	if err != nil {
		return nil, fmt.Errorf("insert import: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("import id: %w", err)
	}
	return &Job{
		ID:        id,
		RepoID:    repoID,
		Source:    source,
		SourceURL: sourceURL,
		Status:    StatusPending,
	}, nil
}

// ByID returns a single import job.
func (s *Service) ByID(ctx context.Context, id int64) (*Job, error) {
	q := s.st.Rebind(`SELECT id, repo_id, source, source_url, status, message FROM repo_imports WHERE id = ?`)
	var j Job
	err := s.st.DB.QueryRowContext(ctx, q, id).Scan(&j.ID, &j.RepoID, &j.Source, &j.SourceURL, &j.Status, &j.Message)
	if err != nil {
		return nil, fmt.Errorf("load import: %w", err)
	}
	return &j, nil
}

// ListByRepo returns all import jobs for a repository.
func (s *Service) ListByRepo(ctx context.Context, repoID int64) ([]*Job, error) {
	q := s.st.Rebind(`SELECT id, repo_id, source, source_url, status, message FROM repo_imports WHERE repo_id = ? ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID)
	if err != nil {
		return nil, fmt.Errorf("list imports: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []*Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.RepoID, &j.Source, &j.SourceURL, &j.Status, &j.Message); err != nil {
			return nil, fmt.Errorf("scan import: %w", err)
		}
		out = append(out, &j)
	}
	return out, rows.Err()
}

// Run clones the remote source into localPath and records the result.
func (s *Service) Run(ctx context.Context, jobID int64, localPath string) error {
	var j Job
	q := s.st.Rebind(`SELECT id, repo_id, source, source_url, status, message FROM repo_imports WHERE id = ?`)
	if err := s.st.DB.QueryRowContext(ctx, q, jobID).Scan(&j.ID, &j.RepoID, &j.Source, &j.SourceURL, &j.Status, &j.Message); err != nil {
		return fmt.Errorf("load import: %w", err)
	}

	if err := s.runner.CloneBare(ctx, j.SourceURL, localPath); err != nil {
		msg := err.Error()
		if uerr := s.setStatus(ctx, jobID, StatusFailed, msg); uerr != nil {
			return fmt.Errorf("clone: %w; record failure: %v", err, uerr)
		}
		return fmt.Errorf("clone import: %w", err)
	}

	if err := s.setStatus(ctx, jobID, StatusDone, ""); err != nil {
		return err
	}
	return nil
}

func (s *Service) setStatus(ctx context.Context, jobID int64, status, message string) error {
	q := s.st.Rebind(`UPDATE repo_imports SET status = ?, message = ? WHERE id = ?`)
	if _, err := s.st.DB.ExecContext(ctx, q, status, message, jobID); err != nil {
		return fmt.Errorf("update import status: %w", err)
	}
	return nil
}
