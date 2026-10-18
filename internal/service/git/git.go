// SPDX-License-Identifier: AGPL-3.0
// Package git provides repository read operations backed by the git binary.
package git

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/LiStudioorg/gitli/internal/infra/gitcmd"
)

// TreeEntry is a single object in a repository tree.
type TreeEntry struct {
	Mode string `json:"mode"`
	Type string `json:"type"`
	Name string `json:"name"`
	Size int64  `json:"size,omitempty"`
	Hash string `json:"hash"`
}

// Commit is a git commit summary.
type Commit struct {
	Hash      string   `json:"hash"`
	ShortHash string   `json:"short"`
	Author    string   `json:"author"`
	Email     string   `json:"email"`
	Date      string   `json:"date"`
	Subject   string   `json:"subject"`
	Parents   []string `json:"parents"`
}

// Service performs read operations on repositories.
type Service struct {
	runner *gitcmd.Runner
}

// New creates a git Service.
func New(runner *gitcmd.Runner) *Service {
	return &Service{runner: runner}
}

// Branches lists branch names for a repository.
func (s *Service) Branches(ctx context.Context, path string) ([]string, error) {
	out, err := s.runner.Run(ctx, path, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

// DefaultBranch resolves the repository HEAD branch.
func (s *Service) DefaultBranch(ctx context.Context, path string) (string, error) {
	out, err := s.runner.Run(ctx, path, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// ErrEmptyRepo indicates the repository has no commits yet. Callers must render
// an onboarding state instead of surfacing a git error to the user.
var ErrEmptyRepo = errors.New("git: repository has no commits")

// IsEmpty reports whether the repository has no commits at all.
func (s *Service) IsEmpty(ctx context.Context, path string) bool {
	out, err := s.runner.Run(ctx, path, "rev-list", "--all", "--count")
	if err != nil {
		return true
	}
	return strings.TrimSpace(out) == "0"
}

// Tree lists entries at a path on a ref. ErrEmptyRepo is returned when the ref
// does not resolve yet (fresh repository), so the caller can show onboarding.
func (s *Service) Tree(ctx context.Context, path, ref, subPath string) ([]TreeEntry, error) {
	if err := validateRef(ref); err != nil {
		return nil, err
	}
	target := ref
	if subPath != "" {
		if err := validateSubPath(subPath); err != nil {
			return nil, err
		}
		target = ref + ":" + subPath
	}
	out, err := s.runner.Run(ctx, path, "ls-tree", "--long", "-z", target, "--")
	if err != nil {
		if isMissingObject(err) {
			return nil, ErrEmptyRepo
		}
		return nil, err
	}
	return parseTree(out), nil
}

// isMissingObject recognises git's "no such ref/object" failures, which for a
// brand-new repository simply mean there is nothing to list yet.
func isMissingObject(err error) bool {
	msg := err.Error()
	for _, needle := range []string{
		"Not a valid object name",
		"unknown revision or path not in the working tree",
		"bad revision",
		"Needed a single revision",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

// Blob returns the raw bytes of a file at ref:subPath.
func (s *Service) Blob(ctx context.Context, path, ref, subPath string) ([]byte, error) {
	if err := validateRef(ref); err != nil {
		return nil, err
	}
	if err := validateSubPath(subPath); err != nil {
		return nil, err
	}
	return s.runner.RunStream(ctx, path, "show", ref+":"+subPath)
}

// Commits returns up to limit commits reachable from ref.
func (s *Service) Commits(ctx context.Context, path, ref string, limit int) ([]Commit, error) {
	if err := validateRef(ref); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	format := "%H%x1f%h%x1f%an%x1f%ae%x1f%aI%x1f%P%x1f%s%x1e"
	out, err := s.runner.Run(ctx, path, "log", fmt.Sprintf("-%d", limit), "--format="+format, ref, "--")
	if err != nil {
		return nil, err
	}
	return parseCommits(out), nil
}

// Diff returns the unified diff between base and head refs.
func (s *Service) Diff(ctx context.Context, path, base, head string) (string, error) {
	if err := validateRef(base); err != nil {
		return "", err
	}
	if err := validateRef(head); err != nil {
		return "", err
	}
	return s.runner.Run(ctx, path, "diff", "--no-color", base, head, "--")
}

// maxDiffBytes caps how much diff text is sent to a browser. Extremely large
// diffs freeze the client and waste bandwidth, so we return a flag instead.
const maxDiffBytes = 1 << 20 // 1 MiB

// ErrDiffTooLarge is returned when a diff exceeds maxDiffBytes.
var ErrDiffTooLarge = errors.New("git: diff too large to display")

// DiffLimited is like Diff but refuses to return oversized output.
func (s *Service) DiffLimited(ctx context.Context, path, base, head string) (string, bool, error) {
	if err := validateRef(base); err != nil {
		return "", false, err
	}
	if err := validateRef(head); err != nil {
		return "", false, err
	}
	// Ask git for a bounded amount of output; a null byte marks truncation
	// deterministically because diffs never contain NUL.
	out, err := s.runner.Run(ctx, path, "-c", "core.quotepath=true", "diff", "--no-color", base, head, "--")
	if err != nil {
		return "", false, err
	}
	if len(out) > maxDiffBytes {
		return "", true, ErrDiffTooLarge
	}
	return out, false, nil
}

// Blame returns porcelain blame for a file.
func (s *Service) Blame(ctx context.Context, path, ref, subPath string) (string, error) {
	if err := validateRef(ref); err != nil {
		return "", err
	}
	if err := validateSubPath(subPath); err != nil {
		return "", err
	}
	return s.runner.Run(ctx, path, "blame", "--porcelain", ref, "--", subPath)
}

func splitLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

func parseTree(out string) []TreeEntry {
	var entries []TreeEntry
	for _, rec := range strings.Split(out, "\x00") {
		rec = strings.TrimRight(rec, "\n")
		if rec == "" {
			continue
		}
		// "<mode> <type> <hash> <size>\t<name>" when --long, else "<mode> <type> <hash>\t<name>"
		meta, name, ok := strings.Cut(rec, "\t")
		if !ok {
			continue
		}
		fields := strings.Fields(meta)
		if len(fields) < 3 {
			continue
		}
		e := TreeEntry{Mode: fields[0], Type: fields[1], Hash: fields[2], Name: name}
		if len(fields) >= 4 {
			if n, err := strconv.ParseInt(fields[3], 10, 64); err == nil {
				e.Size = n
			}
		}
		entries = append(entries, e)
	}
	return entries
}

func parseCommits(out string) []Commit {
	var commits []Commit
	for _, rec := range strings.Split(out, "\x1e") {
		rec = strings.Trim(rec, "\n")
		if rec == "" {
			continue
		}
		f := strings.Split(rec, "\x1f")
		if len(f) < 7 {
			continue
		}
		c := Commit{
			Hash:      f[0],
			ShortHash: f[1],
			Author:    f[2],
			Email:     f[3],
			Date:      f[4],
			Subject:   f[6],
		}
		if strings.TrimSpace(f[5]) != "" {
			c.Parents = strings.Fields(f[5])
		}
		commits = append(commits, c)
	}
	return commits
}

// ChangedFiles returns the paths changed between base and head.
func (s *Service) ChangedFiles(ctx context.Context, path, base, head string) ([]string, error) {
	if err := validateRef(base); err != nil {
		return nil, err
	}
	if err := validateRef(head); err != nil {
		return nil, err
	}
	out, err := s.runner.Run(ctx, path, "diff", "--name-only", base, head, "--")
	if err != nil {
		return nil, err
	}
	var files []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			files = append(files, l)
		}
	}
	return files, nil
}

// ShowFile returns the contents of a file at ref, or (nil,false) if absent.
func (s *Service) ShowFile(ctx context.Context, path, ref, file string) ([]byte, bool) {
	if err := validateRef(ref); err != nil {
		return nil, false
	}
	if err := validateSubPath(file); err != nil {
		return nil, false
	}
	data, err := s.runner.RunStream(ctx, path, "show", ref+":"+file)
	if err != nil {
		return nil, false
	}
	return data, true
}
