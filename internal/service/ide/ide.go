// SPDX-License-Identifier: AGPL-3.0
// Package ide provides code-navigation primitives for the web IDE: directory
// listing, file reading and declaration ("symbol") search over a git tree.
package ide

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/LiStudioorg/gitli/internal/infra/gitcmd"
)

// DefaultReadLimit caps how many bytes ReadFile will return.
const DefaultReadLimit int64 = 1 << 20

// DefaultSymbolLimit caps how many symbols Symbols will return.
const DefaultSymbolLimit = 100

// Service exposes read-only IDE operations over repositories on disk.
type Service struct {
	runner *gitcmd.Runner
}

// New builds an IDE Service backed by a git runner.
func New(runner *gitcmd.Runner) *Service {
	return &Service{runner: runner}
}

// FileEntry describes one entry inside a directory listing.
type FileEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}

// Symbol is a declaration discovered by Symbols.
type Symbol struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	File string `json:"file"`
	Line int    `json:"line"`
}

// ListDir lists the entries of dir at ref using `git ls-tree`. dir may be
// empty for the repository root.
func (s *Service) ListDir(ctx context.Context, repoPath, ref, dir string) ([]FileEntry, error) {
	if ref == "" {
		ref = "HEAD"
	}
	spec := ref
	if dir != "" {
		spec = ref + ":" + strings.Trim(dir, "/")
	}
	out, err := s.runner.Run(ctx, repoPath, "ls-tree", "-l", "--full-tree", spec)
	if err != nil {
		return nil, fmt.Errorf("ide: ls-tree: %w", err)
	}
	return parseTree(out), nil
}

// parseTree parses `git ls-tree -l` output lines of the form:
//
//	<mode> <type> <object>\t<size>\t<path>
//
// The size column is only meaningful for blobs and is "-" for trees.
func parseTree(out string) []FileEntry {
	entries := make([]FileEntry, 0)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		meta, path, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		fields := strings.Fields(meta)
		if len(fields) < 2 {
			continue
		}
		kind := fields[1]
		typ := "file"
		if kind == "tree" {
			typ = "dir"
		}
		base := path
		if i := strings.LastIndex(path, "/"); i >= 0 {
			base = path[i+1:]
		}
		entries = append(entries, FileEntry{Path: base, Type: typ, Size: entrySize(meta)})
	}
	return entries
}

// entrySize extracts the size field from an ls-tree metadata column. The
// metadata is "<mode> <type> <object> [<size>]"; size is absent for trees.
func entrySize(meta string) int64 {
	fields := strings.Fields(meta)
	if len(fields) < 4 {
		return 0
	}
	n, err := strconv.ParseInt(fields[3], 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// ReadFile returns the contents of path at ref. It refuses files larger than
// maxBytes (default DefaultReadLimit) so the IDE never buffers huge blobs.
func (s *Service) ReadFile(ctx context.Context, repoPath, ref, path string, maxBytes int64) ([]byte, error) {
	if ref == "" {
		ref = "HEAD"
	}
	if maxBytes <= 0 {
		maxBytes = DefaultReadLimit
	}
	spec := ref + ":" + strings.Trim(path, "/")
	raw, err := s.runner.RunStream(ctx, repoPath, "show", spec)
	if err != nil {
		return nil, fmt.Errorf("ide: show: %w", err)
	}
	if int64(len(raw)) > maxBytes {
		return nil, fmt.Errorf("ide: file %q exceeds %d bytes", path, maxBytes)
	}
	return raw, nil
}

// symbolRe matches common declaration keywords followed by an identifier.
var symbolRe = regexp.MustCompile(`(?m)^\s*(?:(?:pub|export|static|async|final|public|private|protected)\s+)*(func|class|interface|type|struct|const|var|def|enum|trait|impl)\s+([A-Za-z_][A-Za-z0-9_]*)`)

// Symbols searches for declaration lines matching query across the tracked
// files at ref. An empty query returns no results.
func (s *Service) Symbols(ctx context.Context, repoPath, ref, query string, limit int) ([]Symbol, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if ref == "" {
		ref = "HEAD"
	}
	if limit <= 0 {
		limit = DefaultSymbolLimit
	}
	pattern := `(func|class|interface|type|struct|const|var|def) +.*` + regexp.QuoteMeta(query)
	out, err := s.runner.Run(ctx, repoPath, "grep", "-n", "-I", "-E", "-e", pattern, ref)
	if err != nil {
		if isNoMatch(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("ide: grep: %w", err)
	}
	return parseSymbols(out, limit), nil
}

// parseSymbols parses `git grep -n` output of the form
// "<ref>:<path>:<line>:<text>".
func parseSymbols(out string, limit int) []Symbol {
	syms := make([]Symbol, 0)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		rest := line
		if i := strings.Index(rest, ":"); i >= 0 {
			rest = rest[i+1:]
		}
		file, rest, ok := strings.Cut(rest, ":")
		if !ok {
			continue
		}
		lineNo, rest, ok := strings.Cut(rest, ":")
		if !ok {
			continue
		}
		n, err := strconv.Atoi(lineNo)
		if err != nil {
			continue
		}
		m := symbolRe.FindStringSubmatch(rest)
		if m == nil {
			continue
		}
		syms = append(syms, Symbol{Name: m[2], Kind: m[1], File: file, Line: n})
		if len(syms) >= limit {
			break
		}
	}
	return syms
}

// isNoMatch reports whether a git error indicates grep found nothing, which
// git signals with exit status 1 rather than an empty result.
func isNoMatch(err error) bool {
	return strings.Contains(err.Error(), "exit status 1")
}
