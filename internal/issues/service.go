package issues

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gitli/internal/db"
)

var (
	ErrNotFound     = errors.New("issue not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrForbidden    = errors.New("forbidden")
)

// Service Issue 领域逻辑（Issue 与 PR 共用 issues 表，is_pull 区分）。
type Service struct {
	q db.Querier
}

func NewService(q db.Querier) *Service {
	return &Service{q: q}
}

// Create 创建 issue 或 PR（isPull=1 时 number 与 issue 共享序列）。
func (s *Service) Create(ctx context.Context, repoID, authorID int64, title, body string, isPull int64, assigneeID sql.NullInt64) (db.Issue, error) {
	if title == "" {
		return db.Issue{}, fmt.Errorf("%w: title required", ErrInvalidInput)
	}
	number, err := s.q.NextIssueNumber(ctx, repoID)
	if err != nil {
		return db.Issue{}, fmt.Errorf("next issue number: %w", err)
	}
	now := time.Now().UTC().Unix()
	issue, err := s.q.CreateIssue(ctx, db.CreateIssueParams{
		RepoID:     repoID,
		Number:     number,
		AuthorID:   authorID,
		Title:      title,
		Body:       body,
		IsPull:     isPull,
		AssigneeID: assigneeID,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err != nil {
		return db.Issue{}, fmt.Errorf("create issue: %w", err)
	}
	return issue, nil
}

func (s *Service) Get(ctx context.Context, repoID, number int64) (db.Issue, error) {
	issue, err := s.q.GetIssue(ctx, db.GetIssueParams{RepoID: repoID, Number: number})
	if errors.Is(err, sql.ErrNoRows) {
		return db.Issue{}, ErrNotFound
	}
	return issue, err
}

func (s *Service) GetByID(ctx context.Context, id int64) (db.Issue, error) {
	issue, err := s.q.GetIssueByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return db.Issue{}, ErrNotFound
	}
	return issue, err
}

// List 按 state 过滤：open / closed / all。
func (s *Service) List(ctx context.Context, repoID int64, isPull int64, state string) ([]db.Issue, error) {
	switch state {
	case "closed":
		return s.q.ListIssuesByState(ctx, db.ListIssuesByStateParams{RepoID: repoID, IsPull: isPull, Closed: 1})
	case "all":
		return s.q.ListIssues(ctx, db.ListIssuesParams{RepoID: repoID, IsPull: isPull})
	default: // open
		return s.q.ListIssuesByState(ctx, db.ListIssuesByStateParams{RepoID: repoID, IsPull: isPull, Closed: 0})
	}
}

func (s *Service) Close(ctx context.Context, id int64) error {
	return s.q.UpdateIssueState(ctx, db.UpdateIssueStateParams{Closed: 1, UpdatedAt: time.Now().UTC().Unix(), ID: id})
}

func (s *Service) Reopen(ctx context.Context, id int64) error {
	return s.q.UpdateIssueState(ctx, db.UpdateIssueStateParams{Closed: 0, UpdatedAt: time.Now().UTC().Unix(), ID: id})
}

func (s *Service) Comment(ctx context.Context, issueID, authorID int64, body string) (db.IssueComment, error) {
	if body == "" {
		return db.IssueComment{}, fmt.Errorf("%w: body required", ErrInvalidInput)
	}
	c, err := s.q.CreateComment(ctx, db.CreateCommentParams{
		IssueID:   issueID,
		AuthorID:  authorID,
		Body:      body,
		CreatedAt: time.Now().UTC().Unix(),
	})
	if err != nil {
		return db.IssueComment{}, fmt.Errorf("create comment: %w", err)
	}
	return c, nil
}

func (s *Service) ListComments(ctx context.Context, issueID int64) ([]db.IssueComment, error) {
	return s.q.ListComments(ctx, issueID)
}

func (s *Service) AddLabel(ctx context.Context, issueID, labelID int64) error {
	return s.q.AddLabel(ctx, db.AddLabelParams{IssueID: issueID, LabelID: labelID})
}

func (s *Service) RemoveLabel(ctx context.Context, issueID, labelID int64) error {
	return s.q.RemoveLabel(ctx, db.RemoveLabelParams{IssueID: issueID, LabelID: labelID})
}

func (s *Service) ListLabels(ctx context.Context, issueID int64) ([]db.Label, error) {
	return s.q.ListIssueLabels(ctx, issueID)
}

// Assign 指派。assigneeID 为 0 表示取消指派。
func (s *Service) Assign(ctx context.Context, issueID, assigneeID int64) error {
	var nid sql.NullInt64
	if assigneeID != 0 {
		nid = sql.NullInt64{Int64: assigneeID, Valid: true}
	}
	return s.q.UpdateIssueAssignee(ctx, db.UpdateIssueAssigneeParams{AssigneeID: nid, UpdatedAt: time.Now().UTC().Unix(), ID: issueID})
}

// CreateLabel 创建仓库标签。
func (s *Service) CreateLabel(ctx context.Context, repoID int64, name, color string) (db.Label, error) {
	if name == "" {
		return db.Label{}, fmt.Errorf("%w: label name required", ErrInvalidInput)
	}
	if color == "" {
		color = "#58a6ff"
	}
	return s.q.CreateLabel(ctx, db.CreateLabelParams{RepoID: repoID, Name: name, Color: color})
}

func (s *Service) ListRepoLabels(ctx context.Context, repoID int64) ([]db.Label, error) {
	return s.q.ListLabels(ctx, repoID)
}

func (s *Service) GetLabelByName(ctx context.Context, repoID int64, name string) (db.Label, error) {
	l, err := s.q.GetLabelByName(ctx, db.GetLabelByNameParams{RepoID: repoID, Name: name})
	if errors.Is(err, sql.ErrNoRows) {
		return db.Label{}, ErrNotFound
	}
	return l, err
}
