// SPDX-License-Identifier: AGPL-3.0
package domain

import "time"

// Issue is a repository issue.
type Issue struct {
	ID          int64
	RepoID      int64
	Number      int
	Title       string
	Body        string
	AuthorID    int64
	State       string
	MilestoneID *int64
	ClosedAt    *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Pull is a pull request.
type Pull struct {
	ID         int64
	RepoID     int64
	Number     int
	Title      string
	Body       string
	AuthorID   int64
	HeadRepoID *int64
	HeadBranch string
	BaseBranch string
	State      string
	Merged     bool
	MergedBy   *int64
	MergedAt   *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Comment attaches to an issue or pull request.
type Comment struct {
	ID        int64
	RepoID    int64
	IssueID   *int64
	PullID    *int64
	AuthorID  int64
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Label is a repository label.
type Label struct {
	ID          int64
	RepoID      int64
	Name        string
	Color       string
	Description string
	CreatedAt   time.Time
}

// Milestone groups issues and pulls.
type Milestone struct {
	ID          int64
	RepoID      int64
	Title       string
	Description string
	DueAt       *time.Time
	State       string
	CreatedAt   time.Time
}

// Review is a review on a pull request.
type Review struct {
	ID        int64
	PullID    int64
	AuthorID  int64
	State     string
	Body      string
	CreatedAt time.Time
}

// ReviewComment is an inline review comment.
type ReviewComment struct {
	ID        int64
	ReviewID  int64
	Path      string
	Line      int
	Body      string
	CreatedAt time.Time
}
