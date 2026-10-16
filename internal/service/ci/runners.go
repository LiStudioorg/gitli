// SPDX-License-Identifier: AGPL-3.0
package ci

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// RunnerReg is a registered CI runner as stored by the CI service.
type RunnerReg struct {
	ID       int64
	Name     string
	Tags     string
	Status   string
	LastSeen *time.Time
}

func hashRunnerToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// RegisterRunner creates a runner and returns its id and one-time plaintext
// token. Only the token hash is persisted.
func (s *Service) RegisterRunner(ctx context.Context, name, tags string) (int64, string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return 0, "", fmt.Errorf("runner token: %w", err)
	}
	token := "glcr_" + hex.EncodeToString(buf)
	q := s.st.Rebind(`INSERT INTO ci_runners (name, token_hash, tags, status) VALUES (?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, name, hashRunnerToken(token), tags, "offline")
	if err != nil {
		return 0, "", fmt.Errorf("insert runner: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, "", fmt.Errorf("runner id: %w", err)
	}
	return id, token, nil
}

// AuthenticateRunner reports whether token matches the stored hash for id.
func (s *Service) AuthenticateRunner(ctx context.Context, id int64, token string) (bool, error) {
	q := s.st.Rebind(`SELECT token_hash FROM ci_runners WHERE id = ?`)
	var stored string
	err := s.st.DB.QueryRowContext(ctx, q, id).Scan(&stored)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("auth runner: %w", err)
	}
	got := hashRunnerToken(token)
	return subtle.ConstantTimeCompare([]byte(got), []byte(stored)) == 1, nil
}

// ListRunners returns all registered runners in insertion order.
func (s *Service) ListRunners(ctx context.Context) ([]RunnerReg, error) {
	q := s.st.Rebind(`SELECT id, name, tags, status, last_seen_at FROM ci_runners ORDER BY id ASC`)
	rows, err := s.st.DB.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list runners: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []RunnerReg
	for rows.Next() {
		var r RunnerReg
		var last sql.NullTime
		if err := rows.Scan(&r.ID, &r.Name, &r.Tags, &r.Status, &last); err != nil {
			return nil, fmt.Errorf("scan runner: %w", err)
		}
		if last.Valid {
			r.LastSeen = &last.Time
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SetRunnerStatus updates a runner's status.
func (s *Service) SetRunnerStatus(ctx context.Context, id int64, status string) error {
	q := s.st.Rebind(`UPDATE ci_runners SET status = ? WHERE id = ?`)
	if _, err := s.st.DB.ExecContext(ctx, q, status, id); err != nil {
		return fmt.Errorf("set runner status: %w", err)
	}
	return nil
}

// DeleteRunner removes a runner by id.
func (s *Service) DeleteRunner(ctx context.Context, id int64) error {
	q := s.st.Rebind(`DELETE FROM ci_runners WHERE id = ?`)
	if _, err := s.st.DB.ExecContext(ctx, q, id); err != nil {
		return fmt.Errorf("delete runner: %w", err)
	}
	return nil
}
