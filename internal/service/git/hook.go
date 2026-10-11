// SPDX-License-Identifier: AGPL-3.0
package git

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

// RefUpdate describes a single ref update reported by a pre-receive hook.
type RefUpdate struct {
	Old string
	New string
	Ref string
}

// BranchName returns the branch name if the update targets a branch, else "".
func (u RefUpdate) BranchName() string {
	const p = "refs/heads/"
	if strings.HasPrefix(u.Ref, p) {
		return strings.TrimPrefix(u.Ref, p)
	}
	return ""
}

// TagName returns the tag name if the update targets a tag, else "".
func (u RefUpdate) TagName() string {
	const p = "refs/tags/"
	if strings.HasPrefix(u.Ref, p) {
		return strings.TrimPrefix(u.Ref, p)
	}
	return ""
}

// IsDelete reports whether the update deletes the ref.
func (u RefUpdate) IsDelete() bool {
	return strings.Trim(u.New, "0") == ""
}

// ParsePreReceive reads pre-receive stdin lines of the form "<old> <new> <ref>".
func ParsePreReceive(r io.Reader) ([]RefUpdate, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	var updates []RefUpdate
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, fmt.Errorf("git: malformed pre-receive line %q", line)
		}
		updates = append(updates, RefUpdate{Old: fields[0], New: fields[1], Ref: fields[2]})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return updates, nil
}

// ProtectionRule is the minimal protection info needed for enforcement.
type ProtectionRule struct {
	Pattern       string
	RequireReview bool
	RequireSigned bool
}

// ProtectionChecker resolves protection rules for a repository.
type ProtectionChecker interface {
	BranchProtections(ctx context.Context, repoID int64) ([]ProtectionRule, error)
	TagProtections(ctx context.Context, repoID int64) ([]string, error)
}

// MatchPattern performs simple glob matching supporting '*' and '?'.
func MatchPattern(pattern, name string) bool {
	return globMatch(pattern, name)
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

// EnforceProtections checks ref updates against branch and tag protections.
// isAdmin bypasses all rules; forceDeleteBlocks deletions of protected refs.
func EnforceProtections(ctx context.Context, checker ProtectionChecker, repoID int64, updates []RefUpdate, isAdmin bool) error {
	if isAdmin {
		return nil
	}
	branchRules, err := checker.BranchProtections(ctx, repoID)
	if err != nil {
		return err
	}
	tagRules, err := checker.TagProtections(ctx, repoID)
	if err != nil {
		return err
	}
	for _, u := range updates {
		if name := u.BranchName(); name != "" {
			for _, rule := range branchRules {
				if MatchPattern(rule.Pattern, name) {
					if u.IsDelete() {
						return fmt.Errorf("分支 %q 受保护，禁止删除", name)
					}
					if rule.RequireSigned {
						return fmt.Errorf("分支 %q 受保护，要求签名提交", name)
					}
					return fmt.Errorf("分支 %q 受保护，禁止直接推送", name)
				}
			}
		}
		if name := u.TagName(); name != "" {
			for _, pat := range tagRules {
				if MatchPattern(pat, name) {
					if u.IsDelete() {
						return fmt.Errorf("标签 %q 受保护，禁止删除", name)
					}
					return fmt.Errorf("标签 %q 受保护，禁止覆盖", name)
				}
			}
		}
	}
	return nil
}
