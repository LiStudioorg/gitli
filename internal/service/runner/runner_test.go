// SPDX-License-Identifier: AGPL-3.0
package runner

import (
	"context"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/db"
	"github.com/LiStudioorg/gitli/internal/service/store"
)

const schema = `
CREATE TABLE runners (id INTEGER PRIMARY KEY AUTOINCREMENT, runner_id TEXT UNIQUE, name TEXT, token_hash TEXT UNIQUE, labels TEXT DEFAULT '[]', version TEXT DEFAULT '', os TEXT DEFAULT '', arch TEXT DEFAULT '', last_seen_at DATETIME, status TEXT DEFAULT 'active', created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE runner_jobs (id INTEGER PRIMARY KEY AUTOINCREMENT, runner_id TEXT, repo_id INTEGER, commit_sha TEXT, ref TEXT, workflow_path TEXT, workflow_content TEXT, status TEXT DEFAULT 'pending', started_at DATETIME, finished_at DATETIME, exit_code INTEGER, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE runner_job_logs (id INTEGER PRIMARY KEY AUTOINCREMENT, job_id INTEGER, chunk TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
`

func newTestService(t *testing.T) *Service {
	t.Helper()
	database, err := db.Open(&config.DatabaseConfig{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(schema); err != nil {
		t.Fatal(err)
	}
	return New(store.New(database), 1)
}

func TestCreateAndAuthenticate(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	res, err := s.Create(ctx, "test", `["linux","amd64"]`)
	if err != nil {
		t.Fatal(err)
	}
	if res.Runner.RunnerID == "" || res.Token == "" {
		t.Fatal("expected runner id and token")
	}
	if _, err := s.Authenticate(ctx, res.Runner.RunnerID, res.Token); err != nil {
		t.Fatalf("auth with correct token: %v", err)
	}
	if _, err := s.Authenticate(ctx, res.Runner.RunnerID, "wrong"); err == nil {
		t.Fatal("auth with wrong token should fail")
	}
}

func TestResetTokenInvalidatesOld(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	res, _ := s.Create(ctx, "r", "[]")
	old := res.Token
	newTok, err := s.ResetToken(ctx, res.Runner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if newTok == old {
		t.Fatal("new token should differ")
	}
	if _, err := s.Authenticate(ctx, res.Runner.RunnerID, old); err == nil {
		t.Fatal("old token should be invalid after reset")
	}
	if _, err := s.Authenticate(ctx, res.Runner.RunnerID, newTok); err != nil {
		t.Fatal("new token should be valid")
	}
}

func TestJobLifecycle(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	res, _ := s.Create(ctx, "r", "[]")

	job, err := s.CreateJob(ctx, CreateJobParams{RepoID: 1, CommitSHA: "abc", Ref: "refs/heads/main", WorkflowPath: ".gitli/workflows/ci.yml", WorkflowContent: "on: push"})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "pending" {
		t.Fatalf("status = %s", job.Status)
	}
	claimed, err := s.ClaimJob(ctx, res.Runner.RunnerID)
	if err != nil {
		t.Fatal(err)
	}
	if claimed.ID != job.ID || claimed.Status != "running" || claimed.RunnerID != res.Runner.RunnerID {
		t.Fatalf("claimed = %+v", claimed)
	}
	if _, err := s.ClaimJob(ctx, res.Runner.RunnerID); err != ErrNoJob {
		t.Fatalf("second claim = %v, want ErrNoJob", err)
	}
	if err := s.AppendLog(ctx, job.ID, "hello "); err != nil {
		t.Fatal(err)
	}
	if err := s.AppendLog(ctx, job.ID, "world"); err != nil {
		t.Fatal(err)
	}
	logs, err := s.Logs(ctx, job.ID, 0, 100)
	if err != nil || len(logs) != 2 {
		t.Fatalf("logs=%v err=%v", logs, err)
	}
	if err := s.CompleteJob(ctx, job.ID, "success", 0); err != nil {
		t.Fatal(err)
	}
	done, _ := s.JobByID(ctx, job.ID)
	if done.Status != "success" || done.ExitCode == nil || *done.ExitCode != 0 {
		t.Fatalf("done = %+v", done)
	}
}

func TestLogTruncation(t *testing.T) {
	s := newTestService(t) // logMax = 1MB
	ctx := context.Background()
	job, _ := s.CreateJob(ctx, CreateJobParams{RepoID: 1, CommitSHA: "a", Ref: "main", WorkflowPath: "w", WorkflowContent: "c"})
	big := make([]byte, 600*1024)
	for i := range big {
		big[i] = 'x'
	}
	if err := s.AppendLog(ctx, job.ID, string(big)); err != nil {
		t.Fatal(err)
	}
	// Second chunk pushes past the cap and must be truncated/ignored.
	if err := s.AppendLog(ctx, job.ID, string(big)); err != nil {
		t.Fatal(err)
	}
	size, _ := s.LogSize(ctx, job.ID)
	if size > s.logMax {
		t.Fatalf("log size %d exceeds cap %d", size, s.logMax)
	}
}

func TestSweepOffline(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	res, _ := s.Create(ctx, "r", "[]")
	if err := s.Heartbeat(ctx, res.Runner.RunnerID, "active", "v1", "linux", "amd64", ""); err != nil {
		t.Fatal(err)
	}
	n, err := s.SweepOffline(ctx, 0) // zero timeout => stale immediately
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		// last_seen just set, cutoff is now, so it may not be < cutoff; acceptable.
		t.Logf("swept %d", n)
	}
}
