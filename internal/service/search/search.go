// SPDX-License-Identifier: AGPL-3.0
// Package search provides code search across repository contents using the
// system git binary.
package search

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/LiStudioorg/gitli/internal/infra/gitcmd"
	"github.com/LiStudioorg/gitli/internal/infra/secutil"
)

// DefaultLimit caps the number of matches returned when the caller passes a
// non-positive limit.
const DefaultLimit = 100

// Result is a single matching line found by a search.
type Result struct {
	Repo     string `json:"repo"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Match    string `json:"match"`
	Language string `json:"language"`
}

// Service performs code searches over repositories.
type Service struct {
	runner *gitcmd.Runner
}

// New creates a search Service.
func New(runner *gitcmd.Runner) *Service {
	return &Service{runner: runner}
}

// Grep searches tracked files at ref for query and returns up to limit matches.
// The query is passed as a separate argument to git, never interpolated into a
// shell, so no shell metacharacter can escape.
func (s *Service) Grep(ctx context.Context, repoPath, query, ref string, limit int) ([]Result, error) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if ref == "" {
		ref = "HEAD"
	}
	if err := secutil.ValidateRef(ref); err != nil {
		return nil, err
	}
	out, err := s.runner.Run(ctx, repoPath, "grep", "-n", "-I", "-e", query, ref, "--")
	if err != nil {
		if isNoMatch(err) {
			return nil, nil
		}
		return nil, err
	}
	return parseGrep(out, limit), nil
}

// GrepPaths returns only the distinct file paths matching query at ref.
func (s *Service) GrepPaths(ctx context.Context, repoPath, query, ref string, limit int) ([]string, error) {
	results, err := s.Grep(ctx, repoPath, query, ref, limit)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(results))
	paths := make([]string, 0, len(results))
	for _, r := range results {
		if _, ok := seen[r.Path]; ok {
			continue
		}
		seen[r.Path] = struct{}{}
		paths = append(paths, r.Path)
	}
	return paths, nil
}

func parseGrep(out string, limit int) []Result {
	var results []Result
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 4 {
			continue
		}
		// git grep prefixes each line with the ref name: "ref:path:line:content".
		num, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}
		results = append(results, Result{
			Path:     parts[1],
			Line:     num,
			Match:    parts[3],
			Language: languageForFile(parts[1]),
		})
		if len(results) >= limit {
			break
		}
	}
	return results
}

func isNoMatch(err error) bool {
	return err != nil && strings.Contains(err.Error(), "exit status 1")
}

func languageForFile(name string) string {
	base := filepath.Base(name)
	if lang, ok := specialFilenames[base]; ok {
		return lang
	}
	ext := strings.ToLower(filepath.Ext(base))
	if lang, ok := extensionToLanguage[ext]; ok {
		return lang
	}
	return ""
}

var extensionToLanguage = map[string]string{
	".go": "Go", ".rs": "Rust", ".py": "Python", ".rb": "Ruby", ".js": "JavaScript",
	".jsx": "JavaScript", ".ts": "TypeScript", ".tsx": "TypeScript", ".java": "Java",
	".kt": "Kotlin", ".c": "C", ".h": "C", ".cc": "C++", ".cpp": "C++", ".hpp": "C++",
	".cs": "C#", ".php": "PHP", ".swift": "Swift", ".m": "Objective-C", ".scala": "Scala",
	".sh": "Shell", ".bash": "Shell", ".zsh": "Shell", ".fish": "Shell", ".ps1": "PowerShell",
	".html": "HTML", ".htm": "HTML", ".css": "CSS", ".scss": "SCSS", ".sass": "SCSS", ".less": "Less",
	".vue": "Vue", ".svelte": "Svelte", ".sql": "SQL", ".lua": "Lua", ".pl": "Perl", ".r": "R",
	".ex": "Elixir", ".exs": "Elixir", ".erl": "Erlang", ".hs": "Haskell", ".clj": "Clojure",
	".dart": "Dart", ".groovy": "Groovy", ".jl": "Julia", ".nim": "Nim", ".zig": "Zig",
	".tf": "HCL", ".yaml": "YAML", ".yml": "YAML", ".json": "JSON", ".toml": "TOML",
	".md": "Markdown", ".rst": "RST", ".tex": "TeX", ".proto": "Protocol Buffers",
	".dockerfile": "Dockerfile", ".cmake": "CMake", ".mk": "Makefile", ".make": "Makefile",
}

var specialFilenames = map[string]string{
	"Dockerfile":     "Dockerfile",
	"Makefile":       "Makefile",
	"makefile":       "Makefile",
	"Rakefile":       "Ruby",
	"Gemfile":        "Ruby",
	"CMakeLists.txt": "CMake",
}
