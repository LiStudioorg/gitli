// SPDX-License-Identifier: AGPL-3.0
package git

import (
	"context"
	"path/filepath"
	"strings"
)

// Language is a detected language with byte statistics.
type Language struct {
	Language   string  `json:"language"`
	Bytes      int64   `json:"bytes"`
	Percentage float64 `json:"percentage"`
}

// extensionToLanguage maps file extensions to language names.
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

// specialFilenames maps extension-less filenames to languages.
var specialFilenames = map[string]string{
	"Dockerfile":     "Dockerfile",
	"Makefile":       "Makefile",
	"makefile":       "Makefile",
	"Rakefile":       "Ruby",
	"Gemfile":        "Ruby",
	"CMakeLists.txt": "CMake",
}

// DetectLanguages scans tracked files at ref and aggregates byte counts per
// language using the git ls-tree long output.
func (s *Service) DetectLanguages(ctx context.Context, path, ref string) ([]Language, error) {
	if ref == "" {
		ref = "HEAD"
	}
	out, err := s.runner.Run(ctx, path, "ls-tree", "-r", "--long", "-z", ref)
	if err != nil {
		return nil, err
	}
	return aggregateLanguages(parseTreeBlobs(out)), nil
}

type blobEntry struct {
	size int64
	lang string
}

func parseTreeBlobs(out string) []blobEntry {
	var entries []blobEntry
	for _, rec := range strings.Split(out, "\x00") {
		rec = strings.TrimRight(rec, "\n")
		if rec == "" {
			continue
		}
		meta, name, ok := strings.Cut(rec, "\t")
		if !ok {
			continue
		}
		fields := strings.Fields(meta)
		if len(fields) < 4 {
			continue
		}
		var size int64
		for _, c := range fields[3] {
			if c < '0' || c > '9' {
				size = 0
				break
			}
			size = size*10 + int64(c-'0')
		}
		lang := languageForFile(name)
		if lang == "" {
			continue
		}
		entries = append(entries, blobEntry{size: size, lang: lang})
	}
	return entries
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

func aggregateLanguages(entries []blobEntry) []Language {
	totals := map[string]int64{}
	var sum int64
	for _, e := range entries {
		totals[e.lang] += e.size
		sum += e.size
	}
	out := make([]Language, 0, len(totals))
	for lang, bytes := range totals {
		pct := 0.0
		if sum > 0 {
			pct = float64(bytes) / float64(sum) * 100
			pct = float64(int(pct*10+0.5)) / 10
		}
		out = append(out, Language{Language: lang, Bytes: bytes, Percentage: pct})
	}
	// sort by bytes descending (simple insertion to avoid importing sort twice)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Bytes > out[j-1].Bytes; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
