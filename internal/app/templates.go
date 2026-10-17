// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"strings"

	"github.com/LiStudioorg/gitli/internal/service/template"
)

var issueTemplateDirs = []string{".gitli/ISSUE_TEMPLATE", ".github/ISSUE_TEMPLATE"}
var prTemplatePaths = []string{
	".gitli/PULL_REQUEST_TEMPLATE.md", ".github/PULL_REQUEST_TEMPLATE.md", "PULL_REQUEST_TEMPLATE.md",
}

// ListIssueTemplates returns the repository's issue templates at ref.
func (a *App) ListIssueTemplates(ctx context.Context, repoPath, ref string) []*template.IssueTemplate {
	var out []*template.IssueTemplate
	for _, dir := range issueTemplateDirs {
		names, err := a.Deps.GitCmd.Run(ctx, repoPath, "ls-tree", "-r", "--name-only", ref, "--", dir)
		if err != nil {
			continue
		}
		for _, name := range strings.Split(names, "\n") {
			name = strings.TrimSpace(name)
			if name == "" || (!strings.HasSuffix(name, ".md") && !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml")) {
				continue
			}
			data, ok := a.Deps.Git.ShowFile(ctx, repoPath, ref, name)
			if !ok {
				continue
			}
			if tpl, err := template.ParseIssue(name, data); err == nil {
				out = append(out, tpl)
			}
		}
	}
	return out
}

// PullRequestTemplate returns the repository's PR template body, if any.
func (a *App) PullRequestTemplate(ctx context.Context, repoPath, ref string) string {
	for _, p := range prTemplatePaths {
		if data, ok := a.Deps.Git.ShowFile(ctx, repoPath, ref, p); ok {
			return template.ParsePullRequest(data)
		}
	}
	return ""
}
