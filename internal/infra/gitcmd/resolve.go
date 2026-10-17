// SPDX-License-Identifier: AGPL-3.0
package gitcmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Options configures git binary resolution.
type Options struct {
	Bin            string
	TimeoutSeconds int
	CheckVersion   bool
	MinVersion     string
	Log            *slog.Logger
}

var (
	gitPath cacheStore
	verRe   = regexp.MustCompile(`(\d+)\.(\d+)(?:\.(\d+))?`)
)

type cacheStore struct {
	path string
	ok   bool
}

// ResolveGitPath resolves and caches the git binary path. It returns an error
// when the binary cannot be found or is not executable.
func ResolveGitPath(bin string) (string, error) {
	if gitPath.ok {
		return gitPath.path, nil
	}
	resolved, err := resolveOnce(bin)
	if err != nil {
		return "", err
	}
	gitPath = cacheStore{path: resolved, ok: true}
	return resolved, nil
}

// termuxGitCandidates lists the well-known Git locations shipped by Termux.
// Android's seccomp policy rejects faccessat2, which the Go runtime uses inside
// os/exec.LookPath, so we never call LookPath on Android.
var termuxGitCandidates = []string{
	"/data/data/com.termux/files/usr/bin/git",
	"/data/data/com.termux/files/usr/local/bin/git",
}

func resolveOnce(bin string) (string, error) {
	if bin == "" {
		if runtime.GOOS == "android" || isTermux() {
			if p, ok := findTermuxGit(); ok {
				return p, nil
			}
			return "", fmt.Errorf("gitcmd: 未能自动定位 git。Android/Termux 下请在 config.toml 中显式设置 [git] bin，例如：%s（先执行 pkg install git）", termuxGitCandidates[0])
		}
		p, err := exec.LookPath("git")
		if err != nil {
			return "", fmt.Errorf("gitcmd: git not found in PATH: %w", err)
		}
		return p, nil
	}
	info, err := os.Stat(bin)
	if err != nil {
		return "", fmt.Errorf("gitcmd: stat %s: %w", bin, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("gitcmd: %s is a directory", bin)
	}
	if info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("gitcmd: %s is not executable", bin)
	}
	abs, err := filepath.Abs(bin)
	if err != nil {
		return bin, nil
	}
	return abs, nil
}

// isTermux reports whether we are running inside Termux regardless of GOOS,
// which covers binaries cross-compiled for linux and executed on Android.
func isTermux() bool {
	if _, err := os.Stat("/data/data/com.termux"); err == nil {
		return true
	}
	if p := os.Getenv("PREFIX"); strings.HasPrefix(p, "/data/data/com.termux/") {
		return true
	}
	return false
}

func findTermuxGit() (string, bool) {
	for _, p := range termuxGitCandidates {
		if info, err := os.Stat(p); err == nil && !info.IsDir() && info.Mode().Perm()&0o111 != 0 {
			return p, true
		}
	}
	return "", false
}

// Configure resolves the git binary, checks its version and returns a Runner.
// It exits the process when check_version is enabled and the version is too low.
func Configure(opts Options) (*Runner, error) {
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	path, err := ResolveGitPath(opts.Bin)
	if err != nil {
		return nil, err
	}
	timeout := DefaultTimeout
	if opts.TimeoutSeconds > 0 {
		timeout = time.Duration(opts.TimeoutSeconds) * time.Second
	}
	r := &Runner{bin: path, env: buildSafeEnv(), timeout: timeout}
	if opts.MinVersion != "" {
		ver, err := r.version(context.Background())
		if err != nil {
			log.Warn("could not determine git version", "error", err)
		} else if lessThan(ver, opts.MinVersion) {
			msg := fmt.Sprintf("git version %s is below required %s", ver, opts.MinVersion)
			if opts.CheckVersion {
				return nil, fmt.Errorf("gitcmd: %s", msg)
			}
			log.Warn(msg)
		} else {
			log.Info("git binary ready", "path", path, "version", ver, "timeout", timeout)
		}
	} else {
		log.Info("git binary ready", "path", path, "timeout", timeout)
	}
	return r, nil
}

func (r *Runner) version(ctx context.Context) (string, error) {
	out, err := r.run(ctx, "", "--version")
	if err != nil {
		return "", err
	}
	m := verRe.FindStringSubmatch(out)
	if m == nil {
		return "", fmt.Errorf("gitcmd: unparseable version %q", strings.TrimSpace(out))
	}
	return m[0], nil
}

func lessThan(got, want string) bool {
	g := parseVer(got)
	w := parseVer(want)
	for i := 0; i < 3; i++ {
		if g[i] != w[i] {
			return g[i] < w[i]
		}
	}
	return false
}

func parseVer(v string) [3]int {
	m := verRe.FindStringSubmatch(v)
	var out [3]int
	if m == nil {
		return out
	}
	for i := 0; i < 3 && i+1 < len(m); i++ {
		n, _ := strconv.Atoi(m[i+1])
		out[i] = n
	}
	return out
}
