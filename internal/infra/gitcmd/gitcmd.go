// SPDX-License-Identifier: AGPL-3.0
// Package gitcmd runs the system git binary via exec.CommandContext with
// a timeout and a strict environment whitelist.
package gitcmd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DefaultTimeout bounds any single git invocation.
const DefaultTimeout = 5 * time.Minute

// safeEnvKeys are the only host environment variables passed through to git.
// Dangerous variables (GIT_SSH_COMMAND, GIT_EXTERNAL_DIFF, GIT_CONFIG_*, ...)
// are deliberately excluded so untrusted input can never inject a hook.
var safeEnvKeys = []string{
	"PATH",
	"HOME",
	"TMPDIR",
	"LANG",
	"LC_ALL",
	"TZ",
}

// Runner executes git commands against repositories on disk.
type Runner struct {
	bin     string
	env     []string
	timeout time.Duration
}

// New creates a Runner using the system git binary.
func New() *Runner {
	return &Runner{bin: "git", env: buildSafeEnv(), timeout: DefaultTimeout}
}

// WithTimeout returns a copy of the runner with a custom timeout.
func (r *Runner) WithTimeout(d time.Duration) *Runner {
	return &Runner{bin: r.bin, env: r.env, timeout: d}
}

// WithEnv returns a copy of the runner with additional whitelisted env vars.
// Keys are added on top of the safe base; dangerous keys are still rejected.
func (r *Runner) WithEnv(extra map[string]string) *Runner {
	env := append([]string{}, r.env...)
	for k, v := range extra {
		if isDangerousKey(k) {
			continue
		}
		env = append(env, k+"="+v)
	}
	return &Runner{bin: r.bin, env: env, timeout: r.timeout}
}

// Path returns the resolved git binary path.
func (r *Runner) Path() string { return r.bin }

// Env returns the whitelisted environment for git subprocesses.
func (r *Runner) Env() []string {
	base := []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_PROTOCOL=version=2",
	}
	return append(base, r.env...)
}

// Init creates a bare repository at path with the given default branch.
func (r *Runner) Init(ctx context.Context, path, defaultBranch string) error {
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	_, err := r.run(ctx, "", "init", "--bare", "--initial-branch="+defaultBranch, path)
	return err
}

// CloneBare mirrors a remote into a local bare repository.
func (r *Runner) CloneBare(ctx context.Context, url, dst string) error {
	_, err := r.run(ctx, "", "clone", "--bare", url, dst)
	return err
}

// Run executes an arbitrary git command in dir and returns stdout.
func (r *Runner) Run(ctx context.Context, dir string, args ...string) (string, error) {
	return r.run(ctx, dir, args...)
}

// RunStream executes git and returns raw stdout bytes.
func (r *Runner) RunStream(ctx context.Context, dir string, args ...string) ([]byte, error) {
	return r.runStream(ctx, dir, args...)
}

func (r *Runner) run(ctx context.Context, dir string, args ...string) (string, error) {
	out, err := r.runStream(ctx, dir, args...)
	return string(out), err
}

func (r *Runner) runStream(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	cmd := r.command(cctx, dir, args...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if cctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("git %s: timed out after %s", strings.Join(args, " "), r.timeout)
		}
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, errBuf.String())
	}
	return out.Bytes(), nil
}

func (r *Runner) command(ctx context.Context, dir string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, r.bin, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	env := append([]string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_PROTOCOL=version=2",
	}, r.env...)
	cmd.Env = env
	return cmd
}

func buildSafeEnv() []string {
	src := envSource()
	out := make([]string, 0, len(safeEnvKeys))
	for _, k := range safeEnvKeys {
		if v, ok := src(k); ok {
			out = append(out, k+"="+v)
		}
	}
	return out
}

func isDangerousKey(k string) bool {
	u := strings.ToUpper(k)
	switch {
	case u == "GIT_SSH_COMMAND", u == "GIT_SSH", u == "GIT_EXTERNAL_DIFF",
		u == "GIT_CONFIG", u == "GIT_CONFIG_GLOBAL", u == "GIT_CONFIG_SYSTEM",
		u == "GIT_DIR", u == "GIT_WORK_TREE", u == "GIT_ALTERNATE_OBJECT_DIRECTORIES",
		u == "GIT_OBJECT_DIRECTORY", u == "GIT_INDEX_FILE":
		return true
	case strings.HasPrefix(u, "GIT_CONFIG_KEY_"), strings.HasPrefix(u, "GIT_CONFIG_VALUE_"):
		return true
	default:
		return false
	}
}
