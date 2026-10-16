// SPDX-License-Identifier: AGPL-3.0
// Package runner implements the standalone runner subsystem: runner
// registration (id + hashed token), job scheduling and log storage.
package runner

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Errors returned by the runner service.
var (
	ErrNotFound     = errors.New("runner: not found")
	ErrUnauthorized = errors.New("runner: unauthorized")
	ErrNoJob        = errors.New("runner: no job available")
)

// Service manages runners, jobs and logs.
type Service struct {
	st     *store.Store
	logMax int64 // per-job log cap in bytes
}

// New builds the runner Service. logMaxMB bounds each job's stored logs.
func New(st *store.Store, logMaxMB int) *Service {
	if logMaxMB <= 0 {
		logMaxMB = 10
	}
	return &Service{st: st, logMax: int64(logMaxMB) << 20}
}

// Runner is a registered runner.
type Runner struct {
	ID         int64
	RunnerID   string
	Name       string
	TokenHash  string
	Labels     string
	Version    string
	OS         string
	Arch       string
	LastSeenAt *time.Time
	Status     string
	CreatedAt  time.Time
}

// NewRunnerResult carries the one-time plaintext token.
type NewRunnerResult struct {
	Runner *Runner
	Token  string
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// generateToken creates a random plaintext token.
func generateToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "glr_" + hex.EncodeToString(buf), nil
}

// generateRunnerID creates a readable runner id like "runner-a1b2c3".
func generateRunnerID() (string, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "runner-" + hex.EncodeToString(buf)[:6], nil
}

// Create provisions a new runner and returns its id and one-time token.
func (s *Service) Create(ctx context.Context, name, labels string) (*NewRunnerResult, error) {
	if name == "" {
		name = "runner"
	}
	if labels == "" {
		labels = "[]"
	}
	runnerID, err := generateRunnerID()
	if err != nil {
		return nil, err
	}
	token, err := generateToken()
	if err != nil {
		return nil, err
	}
	q := s.st.Rebind(`INSERT INTO runners (runner_id, name, token_hash, labels, status) VALUES (?, ?, ?, ?, 'active')`)
	res, err := s.st.DB.ExecContext(ctx, q, runnerID, name, hashToken(token), labels)
	if err != nil {
		return nil, fmt.Errorf("runner: insert: %w", err)
	}
	id, _ := res.LastInsertId()
	r, err := s.ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &NewRunnerResult{Runner: r, Token: token}, nil
}

// ResetToken issues a new token for a runner, invalidating the old one.
func (s *Service) ResetToken(ctx context.Context, id int64) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}
	res, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE runners SET token_hash = ? WHERE id = ?`), hashToken(token), id)
	if err != nil {
		return "", err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return "", ErrNotFound
	}
	return token, nil
}

// Update changes a runner's name, labels or status.
func (s *Service) Update(ctx context.Context, id int64, name, labels, status string) error {
	sets := []string{}
	args := []any{}
	if name != "" {
		sets = append(sets, "name = ?")
		args = append(args, name)
	}
	if labels != "" {
		sets = append(sets, "labels = ?")
		args = append(args, labels)
	}
	if status != "" {
		if status != "active" && status != "offline" && status != "disabled" {
			return fmt.Errorf("runner: invalid status %q", status)
		}
		sets = append(sets, "status = ?")
		args = append(args, status)
	}
	if len(sets) == 0 {
		return nil
	}
	args = append(args, id)
	q := s.st.Rebind(`UPDATE runners SET ` + strings.Join(sets, ", ") + ` WHERE id = ?`)
	_, err := s.st.DB.ExecContext(ctx, q, args...)
	return err
}

// Delete removes a runner.
func (s *Service) Delete(ctx context.Context, id int64) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`DELETE FROM runners WHERE id = ?`), id)
	return err
}

const runnerColumns = `id, runner_id, name, token_hash, labels, version, os, arch, last_seen_at, status, created_at`

// ByID loads a runner by primary key.
func (s *Service) ByID(ctx context.Context, id int64) (*Runner, error) {
	return s.scanOne(ctx, s.st.Rebind(`SELECT `+runnerColumns+` FROM runners WHERE id = ?`), id)
}

// ByRunnerID loads a runner by its public id.
func (s *Service) ByRunnerID(ctx context.Context, runnerID string) (*Runner, error) {
	return s.scanOne(ctx, s.st.Rebind(`SELECT `+runnerColumns+` FROM runners WHERE runner_id = ?`), runnerID)
}

// List returns all runners.
func (s *Service) List(ctx context.Context) ([]*Runner, error) {
	rows, err := s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT `+runnerColumns+` FROM runners ORDER BY id DESC`))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*Runner
	for rows.Next() {
		r, err := scanRunner(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Authenticate validates a (runner_id, token) pair.
func (s *Service) Authenticate(ctx context.Context, runnerID, token string) (*Runner, error) {
	r, err := s.ByRunnerID(ctx, runnerID)
	if err != nil {
		return nil, ErrUnauthorized
	}
	if r.Status == "disabled" {
		return nil, ErrUnauthorized
	}
	if subtleEqual(hashToken(token), r.TokenHash) {
		return r, nil
	}
	return nil, ErrUnauthorized
}

// Heartbeat records a runner heartbeat and metadata.
func (s *Service) Heartbeat(ctx context.Context, runnerID, status, version, osName, arch, labels string) error {
	if status == "" {
		status = "active"
	}
	q := s.st.Rebind(`UPDATE runners SET last_seen_at = ?, status = ?, version = ?, os = ?, arch = ?, labels = COALESCE(NULLIF(?, ''), labels) WHERE runner_id = ?`)
	_, err := s.st.DB.ExecContext(ctx, q, time.Now().UTC(), status, version, osName, arch, labels, runnerID)
	return err
}

// SweepOffline marks runners offline when their heartbeat is stale.
func (s *Service) SweepOffline(ctx context.Context, timeout time.Duration) (int64, error) {
	cutoff := time.Now().Add(-timeout)
	q := s.st.Rebind(`UPDATE runners SET status = 'offline' WHERE status = 'active' AND (last_seen_at IS NULL OR last_seen_at < ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, cutoff)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (s *Service) scanOne(ctx context.Context, query string, args ...any) (*Runner, error) {
	row := s.st.DB.QueryRowContext(ctx, query, args...)
	r, err := scanRunner(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return r, err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanRunner(sc scanner) (*Runner, error) {
	var r Runner
	var last sql.NullTime
	err := sc.Scan(&r.ID, &r.RunnerID, &r.Name, &r.TokenHash, &r.Labels, &r.Version, &r.OS, &r.Arch, &last, &r.Status, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	if last.Valid {
		t := last.Time
		r.LastSeenAt = &t
	}
	return &r, nil
}

func subtleEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
