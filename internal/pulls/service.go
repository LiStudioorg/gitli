package pulls

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gitli/internal/db"
	"gitli/internal/git"
	"gitli/internal/issues"
)

var (
	ErrNotFound          = errors.New("pull not found")
	ErrInvalidInput      = errors.New("invalid input")
	ErrConflict          = errors.New("merge conflict")
	ErrRebaseUnsupported = errors.New("rebase unsupported")
	ErrAlreadyMerged     = errors.New("pull already merged")
	ErrSameBranch        = errors.New("base and head must differ")
)

// Service PR 领域逻辑。
type Service struct {
	q       db.Querier
	issues  *issues.Service
	reposFn func(ctx context.Context, ownerName, name string) (db.Repo, string, error) // 返回 repo 与磁盘路径
}

// NewService reposFn 由 web 层注入（用于解析分支所在仓库磁盘路径）。
func NewService(q db.Querier, issuesSvc *issues.Service, reposFn func(ctx context.Context, ownerName, name string) (db.Repo, string, error)) *Service {
	return &Service{q: q, issues: issuesSvc, reposFn: reposFn}
}

// Create 创建 PR：校验分支存在、base != head，复用 issues 表（is_pull=1）。
func (s *Service) Create(ctx context.Context, repo db.Repo, repoPath, base, head, title, body string, authorID int64) (db.Issue, error) {
	if base == "" || head == "" {
		return db.Issue{}, fmt.Errorf("%w: base/head required", ErrInvalidInput)
	}
	if base == head {
		return db.Issue{}, ErrSameBranch
	}
	if _, err := git.ResolveRef(ctx, repoPath, base); err != nil {
		return db.Issue{}, fmt.Errorf("%w: base branch %q not found", ErrInvalidInput, base)
	}
	if _, err := git.ResolveRef(ctx, repoPath, head); err != nil {
		return db.Issue{}, fmt.Errorf("%w: head branch %q not found", ErrInvalidInput, head)
	}
	// 已有同分支对且未关闭的 PR 则拒绝
	if _, err := s.q.GetOpenPullByBranches(ctx, db.GetOpenPullByBranchesParams{
		RepoID: repo.ID, HeadBranch: head, BaseBranch: base,
	}); err == nil {
		return db.Issue{}, fmt.Errorf("%w: open pull for %s..%s exists", ErrInvalidInput, base, head)
	}
	issue, err := s.issues.Create(ctx, repo.ID, authorID, title, body, 1, sql.NullInt64{})
	if err != nil {
		return db.Issue{}, err
	}
	if err := s.q.CreatePull(ctx, db.CreatePullParams{
		IssueID:    issue.ID,
		RepoID:     repo.ID,
		HeadRepoID: repo.ID,
		HeadBranch: head,
		BaseBranch: base,
	}); err != nil {
		return db.Issue{}, fmt.Errorf("create pull: %w", err)
	}
	return issue, nil
}

type PullDetail struct {
	Issue db.Issue
	Pull  db.Pull
}

func (s *Service) Get(ctx context.Context, repoID, number int64) (PullDetail, error) {
	issue, err := s.issues.Get(ctx, repoID, number)
	if err != nil {
		return PullDetail{}, err
	}
	if issue.IsPull != 1 {
		return PullDetail{}, ErrNotFound
	}
	pull, err := s.q.GetPullByIssue(ctx, issue.ID)
	if err != nil {
		return PullDetail{}, fmt.Errorf("get pull: %w", err)
	}
	return PullDetail{Issue: issue, Pull: pull}, nil
}

func (s *Service) List(ctx context.Context, repoID int64, state string) ([]db.ListPullsWithIssueRow, error) {
	var rows []db.ListPullsWithIssueRow
	all, err := s.q.ListPullsWithIssue(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("list pulls: %w", err)
	}
	for _, r := range all {
		switch state {
		case "closed":
			if r.Closed == 1 {
				rows = append(rows, r)
			}
		case "all":
			rows = append(rows, r)
		default:
			if r.Closed == 0 {
				rows = append(rows, r)
			}
		}
	}
	return rows, nil
}

// Merge 合并 PR。method: merge | squash | rebase。
func (s *Service) Merge(ctx context.Context, repo db.Repo, repoPath string, detail PullDetail, method, msg string) (string, error) {
	if detail.Pull.Merged == 1 || detail.Issue.Closed == 1 {
		return "", ErrAlreadyMerged
	}
	if msg == "" {
		msg = fmt.Sprintf("Merge pull request #%d from %s", detail.Issue.Number, detail.Pull.HeadBranch)
	}
	var commit string
	var err error
	switch method {
	case "squash":
		commit, err = git.MergeBranch(ctx, repoPath, detail.Pull.BaseBranch, detail.Pull.HeadBranch, msg, true)
	case "rebase":
		_, err = git.RebaseBranch(ctx, repoPath, detail.Pull.BaseBranch, detail.Pull.HeadBranch)
		commit = detail.Pull.HeadBranch
	default: // merge
		commit, err = git.MergeBranch(ctx, repoPath, detail.Pull.BaseBranch, detail.Pull.HeadBranch, msg, false)
	}
	if err != nil {
		return "", err
	}
	now := time.Now().UTC().Unix()
	if err := s.q.MergePull(ctx, db.MergePullParams{MergeCommit: sql.NullString{String: commit, Valid: true}, IssueID: detail.Issue.ID}); err != nil {
		return "", fmt.Errorf("merge pull: %w", err)
	}
	if err := s.q.UpdateIssueState(ctx, db.UpdateIssueStateParams{Closed: 1, UpdatedAt: now, ID: detail.Issue.ID}); err != nil {
		return "", fmt.Errorf("close issue: %w", err)
	}
	return commit, nil
}

// Diff 输出 base...head diff 行。
func (s *Service) Diff(ctx context.Context, repoPath, base, head string) ([]git.DiffLine, error) {
	text, err := git.DiffRange(ctx, repoPath, base, head)
	if err != nil {
		return nil, err
	}
	return git.ParseDiffText(string(text)), nil
}
