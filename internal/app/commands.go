// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/ci"
)

// RepoOwnerSlug returns the filesystem slug (user or org name) for a repo.
func (a *App) RepoOwnerSlug(ctx context.Context, r *domain.Repo) string {
	if r.OwnerType == domain.OwnerOrg && r.OrgID != nil {
		if o, err := a.Deps.Orgs.ByID(ctx, *r.OrgID); err == nil {
			return o.Name
		}
	}
	if u, err := a.Deps.Users.ByID(ctx, r.OwnerID); err == nil {
		return u.Username
	}
	return fmt.Sprintf("owner-%d", r.OwnerID)
}

// RunCIJob executes a claimed CI job's script on the host. The script is run by
// the system shell as a file so that no external input is interpolated into a
// command line. Output is streamed to a per-job log file.
func (a *App) RunCIJob(ctx context.Context, job *ci.Job) error {
	logDir := filepath.Join(a.Cfg.Storage.Packages, "..", "ci-logs")
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		return err
	}
	logPath := filepath.Join(logDir, fmt.Sprintf("job-%d.log", job.ID))
	f, err := os.Create(logPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	scriptPath := filepath.Join(logDir, fmt.Sprintf("job-%d.sh", job.ID))
	if err := os.WriteFile(scriptPath, []byte(job.Script), 0o700); err != nil {
		return err
	}
	defer func() { _ = os.Remove(scriptPath) }()

	timeout := time.Duration(a.Cfg.CI.RunnerTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = time.Hour
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, "/bin/sh", scriptPath)
	cmd.Dir = a.Cfg.Storage.Repos
	cmd.Stdout = f
	cmd.Stderr = f
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "LANG=C.UTF-8"}
	return cmd.Run()
}

// PromoteAdmin grants administrator rights to a user.
func (a *App) PromoteAdmin(ctx context.Context, userID int64) error {
	q := a.Deps.DB.Rebind(`UPDATE users SET is_admin = ?, updated_at = ? WHERE id = ?`)
	_, err := a.Deps.DB.ExecContext(ctx, q, true, time.Now().UTC(), userID)
	return err
}
