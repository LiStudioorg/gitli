// SPDX-License-Identifier: AGPL-3.0
// Package codeowners parses CODEOWNERS files and resolves file-path owners.
package codeowners

import (
	"bufio"
	"fmt"
	"strings"
)

// Rule is a single CODEOWNERS line: a path pattern and its owners.
type Rule struct {
	Pattern string
	Owners  []string
}

// parseLine parses one CODEOWNERS line, returning ok=false for blanks/comments.
func parseLine(line string) (Rule, bool) {
	// Strip trailing comments (a '#' not preceded by whitespace is literal).
	if i := strings.IndexByte(line, '#'); i >= 0 {
		line = line[:i]
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return Rule{}, false
	}
	pattern := fields[0]
	owners := make([]string, 0, len(fields)-1)
	for _, o := range fields[1:] {
		o = strings.TrimPrefix(o, "@")
		if o != "" {
			owners = append(owners, o)
		}
	}
	if len(owners) == 0 {
		return Rule{}, false
	}
	return Rule{Pattern: pattern, Owners: owners}, true
}

// Parse decodes a CODEOWNERS file body into ordered rules.
func Parse(data []byte) ([]Rule, error) {
	var rules []Rule
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		if r, ok := parseLine(sc.Text()); ok {
			rules = append(rules, r)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("codeowners: read: %w", err)
	}
	return rules, nil
}

// OwnersFor returns the owners that apply to a file path. Later rules take
// precedence, matching GitHub's "last matching pattern wins" semantics.
func OwnersFor(rules []Rule, path string) []string {
	var owners []string
	for _, r := range rules {
		if matchPattern(r.Pattern, path) {
			owners = r.Owners
		}
	}
	return owners
}

// matchPattern implements CODEOWNERS pattern matching with '*', '?', '**' and
// leading '/' (root-anchored) plus trailing '/' (directory) semantics.
func matchPattern(pattern, path string) bool {
	anchored := strings.HasPrefix(pattern, "/")
	pattern = strings.TrimPrefix(pattern, "/")
	dirOnly := strings.HasSuffix(pattern, "/")
	pattern = strings.TrimSuffix(pattern, "/")

	if dirOnly {
		// A directory pattern matches files beneath it (at any depth when not
		// anchored, from the root when anchored).
		if strings.HasPrefix(path, pattern+"/") {
			return true
		}
		if !anchored {
			segs := strings.Split(path, "/")
			for i := range segs {
				if strings.HasPrefix(strings.Join(segs[i:], "/"), pattern+"/") {
					return true
				}
			}
		}
		return false
	}

	if !strings.Contains(pattern, "/") && !anchored {
		// A bare name matches at any depth.
		return matchSegments(pattern, path) || matchSegments(pattern, baseName(path))
	}
	if anchored {
		return matchSegments(pattern, path)
	}
	// Non-anchored with slash: match against the full path and any suffix.
	segs := strings.Split(path, "/")
	for i := range segs {
		if matchSegments(pattern, strings.Join(segs[i:], "/")) {
			return true
		}
	}
	return false
}

func baseName(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[i+1:]
	}
	return path
}

// matchSegments matches a glob pattern (with '*' and '?') against a string,
// treating '**' as matching across path separators.
func matchSegments(pattern, s string) bool {
	return globMatch(pattern, s)
}

func globMatch(pattern, s string) bool {
	px, sx := 0, 0
	nextPx, nextSx := -1, 0
	for sx < len(s) {
		if px < len(pattern) && (pattern[px] == '?' || pattern[px] == s[sx]) {
			px++
			sx++
		} else if px < len(pattern) && pattern[px] == '*' {
			nextPx = px
			nextSx = sx + 1
			px++
		} else if nextPx != -1 {
			px = nextPx + 1
			sx = nextSx
			nextSx++
		} else {
			return false
		}
	}
	for px < len(pattern) && pattern[px] == '*' {
		px++
	}
	return px == len(pattern)
}
