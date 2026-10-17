// SPDX-License-Identifier: AGPL-3.0
// Package globalsearch implements cross-entity search with a simple query
// language (language:, is:, author:, org:, stars:>N).
package globalsearch

import (
	"context"
	"strconv"
	"strings"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Service performs cross-entity searches.
type Service struct {
	st *store.Store
}

// New builds a global search Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// Query is a parsed search query.
type Query struct {
	Terms    []string
	Language string
	State    string // open | closed
	Author   string
	Org      string
	StarsGt  int
}

// Parse parses a raw query string into structured qualifiers + free terms.
func Parse(raw string) Query {
	var q Query
	for _, tok := range strings.Fields(raw) {
		key, val, ok := strings.Cut(tok, ":")
		if !ok {
			q.Terms = append(q.Terms, tok)
			continue
		}
		switch strings.ToLower(key) {
		case "language":
			q.Language = val
		case "is":
			q.State = val
		case "author":
			q.Author = strings.TrimPrefix(val, "@")
		case "org":
			q.Org = val
		case "stars":
			if strings.HasPrefix(val, ">") {
				if n, err := strconv.Atoi(strings.TrimPrefix(val, ">")); err == nil {
					q.StarsGt = n
				}
			}
		default:
			q.Terms = append(q.Terms, tok)
		}
	}
	return q
}

// RepoResult is a matched repository.
type RepoResult struct {
	ID          int64  `json:"id"`
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Stars       int    `json:"stars_count"`
	Language    string `json:"language"`
}

// IssueResult is a matched issue or pull request.
type IssueResult struct {
	Kind   string `json:"kind"` // issue | pull
	ID     int64  `json:"id"`
	RepoID int64  `json:"repo_id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
}

// UserResult is a matched user or organization.
type UserResult struct {
	Kind     string `json:"kind"` // user | org
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

// Results groups search hits by entity type.
type Results struct {
	Repos  []RepoResult  `json:"repos"`
	Issues []IssueResult `json:"issues"`
	Users  []UserResult  `json:"users"`
}

// Search runs a cross-entity search. Public repos only; issues/pulls require
// their repo to be public.
func (s *Service) Search(ctx context.Context, raw string, limit int) (*Results, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	q := Parse(raw)
	like := termsLike(q.Terms)
	out := &Results{Repos: []RepoResult{}, Issues: []IssueResult{}, Users: []UserResult{}}

	// Repositories (public, not deleted).
	repoWhere := []string{"is_private = ?", "deleted_at IS NULL"}
	repoArgs := []any{false}
	if like != "" {
		repoWhere = append(repoWhere, "(name LIKE ? OR description LIKE ?)")
		repoArgs = append(repoArgs, like, like)
	}
	if q.Language != "" {
		repoWhere = append(repoWhere, "id IN (SELECT repo_id FROM repo_languages WHERE language = ?)")
		repoArgs = append(repoArgs, q.Language)
	}
	if q.StarsGt > 0 {
		repoWhere = append(repoWhere, "stars_count > ?")
		repoArgs = append(repoArgs, q.StarsGt)
	}
	repoArgs = append(repoArgs, limit)
	rq := s.st.Rebind(`SELECT id, owner_id, owner_type, name, description, stars_count FROM repos WHERE ` + strings.Join(repoWhere, " AND ") + ` ORDER BY stars_count DESC, id DESC LIMIT ?`)
	rows, err := s.st.DB.QueryContext(ctx, rq, repoArgs...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var rr RepoResult
		var ownerID int64
		var ownerType string
		if err := rows.Scan(&rr.ID, &ownerID, &ownerType, &rr.Name, &rr.Description, &rr.Stars); err != nil {
			_ = rows.Close()
			return nil, err
		}
		rr.Owner = s.ownerName(ctx, ownerID, ownerType)
		out.Repos = append(out.Repos, rr)
	}
	_ = rows.Close()

	// Issues / pulls on public repos.
	if like != "" {
		out.Issues = s.searchIssues(ctx, like, q, limit)
	}

	// Users and orgs.
	if like != "" {
		out.Users = s.searchUsers(ctx, like, q, limit)
	}
	return out, nil
}

func (s *Service) searchIssues(ctx context.Context, like string, q Query, limit int) []IssueResult {
	var out []IssueResult
	// Issues.
	iw := []string{"i.title LIKE ?", "r.is_private = ?", "r.deleted_at IS NULL"}
	iargs := []any{like, false}
	if q.State == "open" || q.State == "closed" {
		iw = append(iw, "i.state = ?")
		iargs = append(iargs, q.State)
	}
	iargs = append(iargs, limit)
	iq := s.st.Rebind(`SELECT i.id, i.repo_id, i.number, i.title, i.state FROM issues i JOIN repos r ON r.id = i.repo_id WHERE ` + strings.Join(iw, " AND ") + ` ORDER BY i.id DESC LIMIT ?`)
	if rows, err := s.st.DB.QueryContext(ctx, iq, iargs...); err == nil {
		for rows.Next() {
			var ir IssueResult
			ir.Kind = "issue"
			if err := rows.Scan(&ir.ID, &ir.RepoID, &ir.Number, &ir.Title, &ir.State); err == nil {
				out = append(out, ir)
			}
		}
		_ = rows.Close()
	}
	// Pulls.
	pw := []string{"p.title LIKE ?", "r.is_private = ?", "r.deleted_at IS NULL"}
	pargs := []any{like, false}
	if q.State == "open" || q.State == "closed" {
		pw = append(pw, "p.state = ?")
		pargs = append(pargs, q.State)
	}
	pargs = append(pargs, limit)
	pq := s.st.Rebind(`SELECT p.id, p.repo_id, p.number, p.title, p.state FROM pulls p JOIN repos r ON r.id = p.repo_id WHERE ` + strings.Join(pw, " AND ") + ` ORDER BY p.id DESC LIMIT ?`)
	if rows, err := s.st.DB.QueryContext(ctx, pq, pargs...); err == nil {
		for rows.Next() {
			var ir IssueResult
			ir.Kind = "pull"
			if err := rows.Scan(&ir.ID, &ir.RepoID, &ir.Number, &ir.Title, &ir.State); err == nil {
				out = append(out, ir)
			}
		}
		_ = rows.Close()
	}
	return out
}

func (s *Service) searchUsers(ctx context.Context, like string, q Query, limit int) []UserResult {
	var out []UserResult
	rows, err := s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT id, username, full_name FROM users WHERE username LIKE ? OR full_name LIKE ? ORDER BY id LIMIT ?`), like, like, limit)
	if err == nil {
		for rows.Next() {
			var u UserResult
			u.Kind = "user"
			if err := rows.Scan(&u.ID, &u.Name, &u.FullName); err == nil {
				out = append(out, u)
			}
		}
		_ = rows.Close()
	}
	orows, err := s.st.DB.QueryContext(ctx, s.st.Rebind(`SELECT id, name, display_name FROM orgs WHERE name LIKE ? OR display_name LIKE ? ORDER BY id LIMIT ?`), like, like, limit)
	if err == nil {
		for orows.Next() {
			var o UserResult
			o.Kind = "org"
			if err := orows.Scan(&o.ID, &o.Name, &o.FullName); err == nil {
				out = append(out, o)
			}
		}
		_ = orows.Close()
	}
	return out
}

func (s *Service) ownerName(ctx context.Context, ownerID int64, ownerType string) string {
	if ownerType == "org" {
		var name string
		if err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT name FROM orgs WHERE id = ?`), ownerID).Scan(&name); err == nil {
			return name
		}
	}
	var name string
	if err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT username FROM users WHERE id = ?`), ownerID).Scan(&name); err == nil {
		return name
	}
	return ""
}

// termsLike returns a "%term%" pattern from free-text terms (joined by space).
func termsLike(terms []string) string {
	if len(terms) == 0 {
		return ""
	}
	return store.ContainsPattern(strings.Join(terms, " "))
}
