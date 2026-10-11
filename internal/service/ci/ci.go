// SPDX-License-Identifier: AGPL-3.0
// Package ci implements a lightweight, host-executed CI service with bounded
// concurrency and runner polling.
package ci

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Pipeline represents a single CI run for a commit.
type Pipeline struct {
	ID        int64
	RepoID    int64
	CommitSHA string
	Ref       string
	Status    string
}

// Job is a single unit of work inside a pipeline, claimed by a runner.
type Job struct {
	ID         int64
	PipelineID int64
	Name       string
	Stage      string
	Status     string
	RunnerID   *int64
	Script     string
	LogPath    string
	StartedAt  *time.Time
	FinishedAt *time.Time
}

// Service manages pipelines, jobs and job execution.
type Service struct {
	st  *store.Store
	log *slog.Logger
	sem chan struct{}
}

// New builds a CI Service. maxConcurrency bounds how many jobs may run at once;
// values below 1 fall back to 1.
func New(st *store.Store, log *slog.Logger, maxConcurrency int) *Service {
	if maxConcurrency < 1 {
		maxConcurrency = 1
	}
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		st:  st,
		log: log,
		sem: make(chan struct{}, maxConcurrency),
	}
}

// CreatePipeline inserts a new pending pipeline.
func (s *Service) CreatePipeline(ctx context.Context, repoID int64, commitSHA, ref string) (*Pipeline, error) {
	p := &Pipeline{RepoID: repoID, CommitSHA: commitSHA, Ref: ref, Status: "pending"}
	q := s.st.Rebind(`INSERT INTO ci_pipelines (repo_id, commit_sha, ref, status) VALUES (?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, p.RepoID, p.CommitSHA, p.Ref, p.Status)
	if err != nil {
		return nil, fmt.Errorf("insert pipeline: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("pipeline id: %w", err)
	}
	p.ID = id
	return p, nil
}

// AddJob inserts a pending job attached to a pipeline.
func (s *Service) AddJob(ctx context.Context, pipelineID int64, name, stage, script string) (*Job, error) {
	j := &Job{PipelineID: pipelineID, Name: name, Stage: stage, Script: script, Status: "pending"}
	q := s.st.Rebind(`INSERT INTO ci_jobs (pipeline_id, name, stage, status, script) VALUES (?, ?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, j.PipelineID, j.Name, j.Stage, j.Status, j.Script)
	if err != nil {
		return nil, fmt.Errorf("insert job: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("job id: %w", err)
	}
	j.ID = id
	return j, nil
}

// ListPipelines returns the most recent pipelines for a repository.
func (s *Service) ListPipelines(ctx context.Context, repoID int64, limit int) ([]*Pipeline, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := s.st.Rebind(`SELECT id, repo_id, commit_sha, ref, status FROM ci_pipelines WHERE repo_id = ? ORDER BY id DESC LIMIT ?`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("list pipelines: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []*Pipeline
	for rows.Next() {
		p := &Pipeline{}
		if err := rows.Scan(&p.ID, &p.RepoID, &p.CommitSHA, &p.Ref, &p.Status); err != nil {
			return nil, fmt.Errorf("scan pipeline: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListJobs returns all jobs belonging to a pipeline in insertion order.
func (s *Service) ListJobs(ctx context.Context, pipelineID int64) ([]*Job, error) {
	q := s.st.Rebind(`SELECT id, pipeline_id, name, stage, status, runner_id, script, log_path, started_at, finished_at FROM ci_jobs WHERE pipeline_id = ? ORDER BY id ASC`)
	rows, err := s.st.DB.QueryContext(ctx, q, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []*Job
	for rows.Next() {
		j := &Job{}
		if err := scanJob(rows, j); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// scanner is satisfied by *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanJob(sc scanner, j *Job) error {
	var (
		runnerID   sql.NullInt64
		logPath    sql.NullString
		startedAt  sql.NullTime
		finishedAt sql.NullTime
	)
	if err := sc.Scan(&j.ID, &j.PipelineID, &j.Name, &j.Stage, &j.Status, &runnerID, &j.Script, &logPath, &startedAt, &finishedAt); err != nil {
		return err
	}
	if runnerID.Valid {
		j.RunnerID = &runnerID.Int64
	}
	if logPath.Valid {
		j.LogPath = logPath.String
	}
	if startedAt.Valid {
		j.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		j.FinishedAt = &finishedAt.Time
	}
	return nil
}

// ClaimJob atomically selects one pending job, marks it running for runnerID and
// returns it. It returns (nil, nil) when no pending job is available.
func (s *Service) ClaimJob(ctx context.Context, runnerID int64) (*Job, error) {
	tx, err := s.st.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin claim: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	sel := s.st.Rebind(`SELECT id, pipeline_id, name, stage, status, runner_id, script, log_path, started_at, finished_at FROM ci_jobs WHERE status = ? ORDER BY id ASC LIMIT 1`)
	j := &Job{}
	err = scanJob(tx.QueryRowContext(ctx, sel, "pending"), j)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select pending job: %w", err)
	}

	now := time.Now().UTC()
	upd := s.st.Rebind(`UPDATE ci_jobs SET status = ?, runner_id = ?, started_at = ? WHERE id = ? AND status = ?`)
	res, err := tx.ExecContext(ctx, upd, "running", runnerID, now, j.ID, "pending")
	if err != nil {
		return nil, fmt.Errorf("mark job running: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("claim rows: %w", err)
	}
	if affected == 0 {
		return nil, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit claim: %w", err)
	}

	j.Status = "running"
	j.RunnerID = &runnerID
	j.StartedAt = &now
	return j, nil
}

// CompleteJob marks a job finished with the supplied terminal status and log path.
func (s *Service) CompleteJob(ctx context.Context, jobID int64, status, logPath string) error {
	now := time.Now().UTC()
	q := s.st.Rebind(`UPDATE ci_jobs SET status = ?, log_path = ?, finished_at = ? WHERE id = ?`)
	if _, err := s.st.DB.ExecContext(ctx, q, status, logPath, now, jobID); err != nil {
		return fmt.Errorf("complete job: %w", err)
	}
	return nil
}

// RunWith executes exec under the concurrency semaphore, marking the job running
// before invocation and finished (success/failed) afterwards.
func (s *Service) RunWith(ctx context.Context, jobID int64, exec func(ctx context.Context) error) error {
	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	case <-ctx.Done():
		return ctx.Err()
	}

	now := time.Now().UTC()
	start := s.st.Rebind(`UPDATE ci_jobs SET status = ?, started_at = ? WHERE id = ?`)
	if _, err := s.st.DB.ExecContext(ctx, start, "running", now, jobID); err != nil {
		return fmt.Errorf("start job %d: %w", jobID, err)
	}

	runErr := exec(ctx)

	status := "success"
	if runErr != nil {
		status = "failed"
	}
	finish := time.Now().UTC()
	done := s.st.Rebind(`UPDATE ci_jobs SET status = ?, finished_at = ? WHERE id = ?`)
	if _, err := s.st.DB.ExecContext(ctx, done, status, finish, jobID); err != nil {
		return fmt.Errorf("finish job %d: %w", jobID, err)
	}
	if runErr != nil {
		s.log.Warn("ci job failed", "job_id", jobID, "error", runErr)
	}
	return runErr
}
