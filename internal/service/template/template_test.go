// SPDX-License-Identifier: AGPL-3.0
package template

import (
	"reflect"
	"testing"
)

func TestParseIssueFrontMatter(t *testing.T) {
	body := `---
name: Bug report
about: Create a report to help us improve
title: "[Bug] "
labels: bug, triage
assignees: alice,bob
---
**Describe the bug**
`
	tpl, err := ParseIssue(".github/ISSUE_TEMPLATE/bug.md", []byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if tpl.Name != "Bug report" || tpl.About == "" || tpl.Title == "" {
		t.Fatalf("meta parsed wrong: %+v", tpl)
	}
	if !reflect.DeepEqual(tpl.Labels, []string{"bug", "triage"}) {
		t.Fatalf("labels = %v", tpl.Labels)
	}
	if !reflect.DeepEqual(tpl.Assignees, []string{"alice", "bob"}) {
		t.Fatalf("assignees = %v", tpl.Assignees)
	}
	if tpl.Body == "" {
		t.Fatal("expected body content")
	}
}

func TestParseIssueListLabels(t *testing.T) {
	body := "---\nname: X\nlabels:\n  - a\n  - b\n---\nbody"
	tpl, err := ParseIssue("x.md", []byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tpl.Labels, []string{"a", "b"}) {
		t.Fatalf("labels = %v", tpl.Labels)
	}
}

func TestParsePullRequest(t *testing.T) {
	body := "## What changed\n- fix\n"
	if got := ParsePullRequest([]byte(body)); got != body {
		t.Fatalf("pr body = %q", got)
	}
}

func TestParseNoFrontMatter(t *testing.T) {
	tpl, err := ParseIssue("docs/t.md", []byte("just a body"))
	if err != nil {
		t.Fatal(err)
	}
	if tpl.Name != "t" || tpl.Body != "just a body" {
		t.Fatalf("tpl = %+v", tpl)
	}
}
