// SPDX-License-Identifier: AGPL-3.0
// Package mirror manages repository mirroring between gitli and remote hosts.
package mirror

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/LiStudioorg/gitli/internal/infra/gitcmd"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Direction identifies the mirror data flow.
const (
	DirectionPull = "pull"
	DirectionPush = "push"
)

// Mirror describes a configured repository mirror.
type Mirror struct {
	ID              int64
	RepoID          int64
	RemoteURL       string
	Direction       string
	IntervalMinutes int
	LastSyncedAt    *time.Time
}

// Service manages repository mirrors.
type Service struct {
	st     *store.Store
	runner *gitcmd.Runner
}

// New builds a mirror Service.
func New(st *store.Store, runner *gitcmd.Runner) *Service {
	return &Service{st: st, runner: runner}
}

// Create validates and inserts a new mirror.
func (s *Service) Create(ctx context.Context, repoID int64, remoteURL, direction string, intervalMinutes int) (*Mirror, error) {
	if direction != DirectionPull && direction != DirectionPush {
		return nil, fmt.Errorf("invalid mirror direction: %q", direction)
	}
	if remoteURL == "" {
		return nil, fmt.Errorf("remote url is required")
	}
	if err := validateRemoteURL(remoteURL); err != nil {
		return nil, err
	}
	if intervalMinutes <= 0 {
		intervalMinutes = 60
	}
	q := s.st.Rebind(`INSERT INTO repo_mirrors (repo_id, remote_url, direction, interval_minutes) VALUES (?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, repoID, remoteURL, direction, intervalMinutes)
	if err != nil {
		return nil, fmt.Errorf("insert mirror: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("mirror id: %w", err)
	}
	return &Mirror{
		ID:              id,
		RepoID:          repoID,
		RemoteURL:       remoteURL,
		Direction:       direction,
		IntervalMinutes: intervalMinutes,
	}, nil
}

// ListByRepo returns all mirrors configured for a repository.
func (s *Service) ListByRepo(ctx context.Context, repoID int64) ([]*Mirror, error) {
	q := s.st.Rebind(`SELECT id, repo_id, remote_url, direction, interval_minutes, last_synced_at FROM repo_mirrors WHERE repo_id = ? ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID)
	if err != nil {
		return nil, fmt.Errorf("list mirrors: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []*Mirror
	for rows.Next() {
		var m Mirror
		var last sql.NullTime
		if err := rows.Scan(&m.ID, &m.RepoID, &m.RemoteURL, &m.Direction, &m.IntervalMinutes, &last); err != nil {
			return nil, fmt.Errorf("scan mirror: %w", err)
		}
		if last.Valid {
			t := last.Time
			m.LastSyncedAt = &t
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

// Sync performs a mirror sync against localPath using the configured direction.
func (s *Service) Sync(ctx context.Context, mirrorID int64, localPath string) error {
	q := s.st.Rebind(`SELECT id, repo_id, remote_url, direction, interval_minutes, last_synced_at FROM repo_mirrors WHERE id = ?`)
	var m Mirror
	var last sql.NullTime
	err := s.st.DB.QueryRowContext(ctx, q, mirrorID).Scan(&m.ID, &m.RepoID, &m.RemoteURL, &m.Direction, &m.IntervalMinutes, &last)
	if err != nil {
		return fmt.Errorf("load mirror: %w", err)
	}
	if last.Valid {
		t := last.Time
		m.LastSyncedAt = &t
	}

	switch m.Direction {
	case DirectionPull:
		if _, err := s.runner.Run(ctx, localPath, "fetch", "--prune", m.RemoteURL, "+refs/heads/*:refs/heads/*"); err != nil {
			return fmt.Errorf("mirror pull: %w", err)
		}
	case DirectionPush:
		if _, err := s.runner.Run(ctx, localPath, "push", m.RemoteURL, "--mirror"); err != nil {
			return fmt.Errorf("mirror push: %w", err)
		}
	default:
		return fmt.Errorf("invalid mirror direction: %q", m.Direction)
	}

	uq := s.st.Rebind(`UPDATE repo_mirrors SET last_synced_at = ? WHERE id = ?`)
	if _, err := s.st.DB.ExecContext(ctx, uq, time.Now().UTC(), m.ID); err != nil {
		return fmt.Errorf("update last_synced_at: %w", err)
	}
	return nil
}

// validateRemoteURL rejects remote URLs with unsupported schemes or no host.
// Private hosts are permitted (internal mirrors are a legitimate use case).
func validateRemoteURL(raw string) error {
	if strings.HasPrefix(raw, "git@") || strings.HasPrefix(raw, "ssh://") {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid remote url: %w", err)
	}
	switch u.Scheme {
	case "http", "https", "git", "ssh":
	default:
		return fmt.Errorf("unsupported remote url scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("remote url missing host")
	}
	return nil
}
