// SPDX-License-Identifier: AGPL-3.0
// Package template parses issue and pull request templates, including YAML
// front matter, from .gitli/ and .github/ directories.
package template

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// IssueTemplate is a parsed issue template.
type IssueTemplate struct {
	Name      string   `json:"name"`
	About     string   `json:"about"`
	Title     string   `json:"title"`
	Labels    []string `json:"labels"`
	Assignees []string `json:"assignees"`
	Body      string   `json:"body"`
	Path      string   `json:"path"`
}

// frontMatter holds the optional YAML header of a template.
type frontMatter struct {
	Name      string     `yaml:"name"`
	About     string     `yaml:"about"`
	Title     string     `yaml:"title"`
	Labels    labelsType `yaml:"labels"`
	Assignees labelsType `yaml:"assignees"`
}

// labelsType accepts either a comma-separated scalar or a YAML list.
type labelsType []string

// UnmarshalYAML supports both "a, b" and ["a", "b"] forms.
func (l *labelsType) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		parts := strings.Split(value.Value, ",")
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				*l = append(*l, p)
			}
		}
		return nil
	}
	var list []string
	if err := value.Decode(&list); err != nil {
		return err
	}
	*l = list
	return nil
}

// ParseIssue parses a template body with optional YAML front matter.
func ParseIssue(path string, data []byte) (*IssueTemplate, error) {
	fm, body := splitFrontMatter(string(data))
	tpl := &IssueTemplate{Path: path, Body: body}
	if fm != "" {
		var m frontMatter
		if err := yaml.Unmarshal([]byte(fm), &m); err != nil {
			return nil, fmt.Errorf("template: parse front matter %s: %w", path, err)
		}
		tpl.Name = m.Name
		tpl.About = m.About
		tpl.Title = m.Title
		tpl.Labels = m.Labels
		tpl.Assignees = m.Assignees
	}
	if tpl.Name == "" {
		tpl.Name = baseName(path)
	}
	return tpl, nil
}

// ParsePullRequest returns the body of a PR template (front matter stripped).
func ParsePullRequest(data []byte) string {
	_, body := splitFrontMatter(string(data))
	return body
}

// splitFrontMatter splits an optional leading "---\n...\n---" block from body.
func splitFrontMatter(s string) (front, body string) {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if !strings.HasPrefix(s, "---\n") {
		return "", s
	}
	rest := s[len("---\n"):]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", s
	}
	front = rest[:idx]
	body = rest[idx+len("\n---"):]
	body = strings.TrimPrefix(body, "\n")
	return front, body
}

func baseName(path string) string {
	path = strings.TrimSuffix(path, ".md")
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[i+1:]
	}
	return path
}
