// SPDX-License-Identifier: AGPL-3.0
package ci

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

func newTestService(t *testing.T, maxConcurrency int) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	ddl := []string{
		`CREATE TABLE ci_pipelines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER NOT NULL,
			commit_sha VARCHAR(64) NOT NULL,
			ref VARCHAR(255) NOT NULL,
			status VARCHAR(32) NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE ci_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pipeline_id INTEGER NOT NULL,
			name VARCHAR(255) NOT NULL,
			stage VARCHAR(255) NOT NULL,
			status VARCHAR(32) NOT NULL,
			runner_id INTEGER,
			script TEXT,
			log_path VARCHAR(512),
			started_at DATETIME,
			finished_at DATETIME,
			created_at DATETIME
		)`,
	}
	for _, q := range ddl {
		if _, err := database.Exec(q); err != nil {
			t.Fatalf("ddl: %v", err)
		}
	}
	return New(store.New(database), nil, maxConcurrency)
}

func TestCreatePipelineAndJobs(t *testing.T) {
	s := newTestService(t, 1)
	ctx := context.Background()

	p, err := s.CreatePipeline(ctx, 42, "abc123", "refs/heads/main")
	if err != nil {
		t.Fatalf("create pipeline: %v", err)
	}
	if p.ID == 0 || p.Status != "pending" {
		t.Fatalf("unexpected pipeline: %+v", p)
	}

	if _, err := s.AddJob(ctx, p.ID, "build", "build", "true"); err != nil {
		t.Fatalf("add job build: %v", err)
	}
	if _, err := s.AddJob(ctx, p.ID, "test", "test", "true"); err != nil {
		t.Fatalf("add job test: %v", err)
	}

	pipes, err := s.ListPipelines(ctx, 42, 10)
	if err != nil {
		t.Fatalf("list pipelines: %v", err)
	}
	if len(pipes) != 1 {
		t.Fatalf("want 1 pipeline, got %d", len(pipes))
	}

	jobs, err := s.ListJobs(ctx, p.ID)
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("want 2 jobs, got %d", len(jobs))
	}
	if jobs[0].Name != "build" || jobs[1].Name != "test" {
		t.Fatalf("unexpected job order: %s, %s", jobs[0].Name, jobs[1].Name)
	}
}

func TestClaimJob(t *testing.T) {
	s := newTestService(t, 1)
	ctx := context.Background()

	p, err := s.CreatePipeline(ctx, 1, "sha", "refs/heads/main")
	if err != nil {
		t.Fatalf("create pipeline: %v", err)
	}
	if _, err := s.AddJob(ctx, p.ID, "build", "build", "script-a"); err != nil {
		t.Fatalf("add job: %v", err)
	}
	if _, err := s.AddJob(ctx, p.ID, "test", "test", "script-b"); err != nil {
		t.Fatalf("add job: %v", err)
	}

	first, err := s.ClaimJob(ctx, 7)
	if err != nil {
		t.Fatalf("claim first: %v", err)
	}
	if first == nil || first.Status != "running" {
		t.Fatalf("expected running job, got %+v", first)
	}
	if first.RunnerID == nil || *first.RunnerID != 7 {
		t.Fatalf("expected runner 7, got %+v", first.RunnerID)
	}
	if first.StartedAt == nil {
		t.Fatalf("expected started_at set")
	}

	second, err := s.ClaimJob(ctx, 7)
	if err != nil {
		t.Fatalf("claim second: %v", err)
	}
	if second == nil {
		t.Fatalf("expected a second pending job")
	}
	if second.ID == first.ID {
		t.Fatalf("expected a different job, both id=%d", first.ID)
	}

	third, err := s.ClaimJob(ctx, 7)
	if err != nil {
		t.Fatalf("claim third: %v", err)
	}
	if third != nil {
		t.Fatalf("expected nil when no pending jobs, got %+v", third)
	}
}

func TestCompleteJob(t *testing.T) {
	s := newTestService(t, 1)
	ctx := context.Background()

	p, err := s.CreatePipeline(ctx, 1, "sha", "refs/heads/main")
	if err != nil {
		t.Fatalf("create pipeline: %v", err)
	}
	j, err := s.AddJob(ctx, p.ID, "build", "build", "script")
	if err != nil {
		t.Fatalf("add job: %v", err)
	}

	if _, err := s.ClaimJob(ctx, 3); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if err := s.CompleteJob(ctx, j.ID, "success", "/logs/job.log"); err != nil {
		t.Fatalf("complete: %v", err)
	}

	jobs, err := s.ListJobs(ctx, p.ID)
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("want 1 job, got %d", len(jobs))
	}
	if jobs[0].Status != "success" || jobs[0].LogPath != "/logs/job.log" {
		t.Fatalf("unexpected job after complete: %+v", jobs[0])
	}
	if jobs[0].FinishedAt == nil {
		t.Fatalf("expected finished_at set")
	}
}

func TestRunWithConcurrencyBound(t *testing.T) {
	s := newTestService(t, 1)
	ctx := context.Background()

	p, err := s.CreatePipeline(ctx, 1, "sha", "refs/heads/main")
	if err != nil {
		t.Fatalf("create pipeline: %v", err)
	}

	const n = 3
	ids := make([]int64, 0, n)
	for i := 0; i < n; i++ {
		j, err := s.AddJob(ctx, p.ID, "job", "build", "script")
		if err != nil {
			t.Fatalf("add job: %v", err)
		}
		ids = append(ids, j.ID)
	}

	var inFlight, maxSeen int64
	var mu sync.Mutex
	var completed int64

	exec := func(context.Context) error {
		mu.Lock()
		inFlight++
		if inFlight > maxSeen {
			maxSeen = inFlight
		}
		mu.Unlock()

		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		inFlight--
		mu.Unlock()
		atomic.AddInt64(&completed, 1)
		return nil
	}

	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			if err := s.RunWith(ctx, id, exec); err != nil {
				t.Errorf("run job %d: %v", id, err)
			}
		}(id)
	}
	wg.Wait()

	if got := atomic.LoadInt64(&completed); got != n {
		t.Fatalf("want %d completed, got %d", n, got)
	}
	if maxSeen != 1 {
		t.Fatalf("expected serialized execution (max in-flight 1), got %d", maxSeen)
	}

	jobs, err := s.ListJobs(ctx, p.ID)
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	for _, j := range jobs {
		if j.Status != "success" {
			t.Fatalf("job %d not success: %s", j.ID, j.Status)
		}
	}
}
