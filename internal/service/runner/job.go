// SPDX-License-Identifier: AGPL-3.0
package runner

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Job is a scheduled workflow job.
type Job struct {
	ID              int64
	RunnerID        string
	RepoID          int64
	CommitSHA       string
	Ref             string
	WorkflowPath    string
	WorkflowContent string
	Status          string
	StartedAt       *time.Time
	FinishedAt      *time.Time
	ExitCode        *int
	CreatedAt       time.Time
}

// CreateJobParams describes a job to enqueue.
type CreateJobParams struct {
	RepoID          int64
	CommitSHA       string
	Ref             string
	WorkflowPath    string
	WorkflowContent string
}

// CreateJob enqueues a pending job.
func (s *Service) CreateJob(ctx context.Context, p CreateJobParams) (*Job, error) {
	q := s.st.Rebind(`INSERT INTO runner_jobs (repo_id, commit_sha, ref, workflow_path, workflow_content, status) VALUES (?, ?, ?, ?, ?, 'pending')`)
	res, err := s.st.DB.ExecContext(ctx, q, p.RepoID, p.CommitSHA, p.Ref, p.WorkflowPath, p.WorkflowContent)
	if err != nil {
		return nil, fmt.Errorf("runner: insert job: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.JobByID(ctx, id)
}

const jobColumns = `id, runner_id, repo_id, commit_sha, ref, workflow_path, workflow_content, status, started_at, finished_at, exit_code, created_at`

// JobByID loads a job.
func (s *Service) JobByID(ctx context.Context, id int64) (*Job, error) {
	row := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT `+jobColumns+` FROM runner_jobs WHERE id = ?`), id)
	j, err := scanJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return j, err
}

// ClaimJob atomically claims the oldest pending job for a runner. Returns
// ErrNoJob when none is available.
func (s *Service) ClaimJob(ctx context.Context, runnerID string) (*Job, error) {
	tx, err := s.st.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var id int64
	q := s.st.Rebind(`SELECT id FROM runner_jobs WHERE status = 'pending' ORDER BY id LIMIT 1`)
	if err := tx.QueryRowContext(ctx, q).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoJob
		}
		return nil, err
	}
	now := time.Now().UTC()
	uq := s.st.Rebind(`UPDATE runner_jobs SET status = 'running', runner_id = ?, started_at = ? WHERE id = ? AND status = 'pending'`)
	res, err := tx.ExecContext(ctx, uq, runnerID, now, id)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNoJob
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.JobByID(ctx, id)
}

// CompleteJob records a job's final status. status defaults to failed.
func (s *Service) CompleteJob(ctx context.Context, id int64, status string, exitCode int) error {
	switch status {
	case "success", "failed", "cancelled":
	default:
		status = "failed"
	}
	now := time.Now().UTC()
	q := s.st.Rebind(`UPDATE runner_jobs SET status = ?, exit_code = ?, finished_at = ? WHERE id = ?`)
	_, err := s.st.DB.ExecContext(ctx, q, status, exitCode, now, id)
	return err
}

// CancelJob marks a pending or running job as cancelled.
func (s *Service) CancelJob(ctx context.Context, id int64) error {
	now := time.Now().UTC()
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE runner_jobs SET status = 'cancelled', finished_at = ? WHERE id = ? AND status IN ('pending','running')`), now, id)
	return err
}

// ListJobsByRepo returns recent jobs for a repository.
func (s *Service) ListJobsByRepo(ctx context.Context, repoID int64, limit int) ([]*Job, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT `+jobColumns+` FROM runner_jobs WHERE repo_id = ? ORDER BY id DESC LIMIT ?`), repoID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanJobs(rows)
}

// ListJobsByRunner returns recent jobs claimed by a runner.
func (s *Service) ListJobsByRunner(ctx context.Context, runnerID string, limit int) ([]*Job, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT `+jobColumns+` FROM runner_jobs WHERE runner_id = ? ORDER BY id DESC LIMIT ?`), runnerID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanJobs(rows)
}

func scanJobs(rows *sql.Rows) ([]*Job, error) {
	var out []*Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func scanJob(sc scanner) (*Job, error) {
	var j Job
	var runnerID sql.NullString
	var started, finished sql.NullTime
	var exit sql.NullInt64
	err := sc.Scan(&j.ID, &runnerID, &j.RepoID, &j.CommitSHA, &j.Ref, &j.WorkflowPath, &j.WorkflowContent, &j.Status, &started, &finished, &exit, &j.CreatedAt)
	if err != nil {
		return nil, err
	}
	if runnerID.Valid {
		j.RunnerID = runnerID.String
	}
	if started.Valid {
		t := started.Time
		j.StartedAt = &t
	}
	if finished.Valid {
		t := finished.Time
		j.FinishedAt = &t
	}
	if exit.Valid {
		n := int(exit.Int64)
		j.ExitCode = &n
	}
	return &j, nil
}

// AppendLog stores a log chunk for a job, truncating beyond the configured cap.
func (s *Service) AppendLog(ctx context.Context, jobID int64, chunk string) error {
	if chunk == "" {
		return nil
	}
	current, err := s.LogSize(ctx, jobID)
	if err != nil {
		return err
	}
	if current >= s.logMax {
		return nil
	}
	if remaining := s.logMax - current; int64(len(chunk)) > remaining {
		chunk = chunk[:remaining]
	}
	_, err = s.st.DB.ExecContext(ctx, s.st.Rebind(`INSERT INTO runner_job_logs (job_id, chunk) VALUES (?, ?)`), jobID, chunk)
	return err
}

// LogSize returns the current stored log size for a job in bytes.
func (s *Service) LogSize(ctx context.Context, jobID int64) (int64, error) {
	var n sql.NullInt64
	err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT COALESCE(SUM(LENGTH(chunk)),0) FROM runner_job_logs WHERE job_id = ?`), jobID).Scan(&n)
	return n.Int64, err
}

// Logs returns the stored log chunks in order after the given log row id.
func (s *Service) Logs(ctx context.Context, jobID, afterID int64, limit int) ([]LogChunk, error) {
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	rows, err := s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT id, chunk, created_at FROM runner_job_logs WHERE job_id = ? AND id > ? ORDER BY id LIMIT ?`), jobID, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []LogChunk
	for rows.Next() {
		var c LogChunk
		if err := rows.Scan(&c.ID, &c.Chunk, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// LogChunk is a single stored log fragment.
type LogChunk struct {
	ID        int64
	Chunk     string
	CreatedAt time.Time
}
