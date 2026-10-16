// SPDX-License-Identifier: AGPL-3.0
// Package pathperm manages path-level permission rules for repositories.
package pathperm

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"unicode"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// Rule is a path-level permission grant.
type Rule struct {
	ID         int64
	RepoID     int64
	Pattern    string
	RoleID     *int64
	Permission string
}

// Service manages path permission rules.
type Service struct {
	st *store.Store
}

// New builds a pathperm Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// validPermissions is the closed set of accepted permission values.
var validPermissions = map[string]bool{
	"read":  true,
	"write": true,
	"admin": true,
}

// Create validates and inserts a path permission rule.
func (s *Service) Create(ctx context.Context, repoID int64, pattern, permission string) (*Rule, error) {
	if repoID <= 0 {
		return nil, fmt.Errorf("pathperm: invalid repo id")
	}
	if permission == "" {
		permission = "read"
	}
	if !validPermissions[permission] {
		return nil, fmt.Errorf("pathperm: invalid permission %q", permission)
	}
	if err := validatePattern(pattern); err != nil {
		return nil, err
	}
	q := s.st.Rebind(`INSERT INTO path_permissions (repo_id, path_pattern, permission) VALUES (?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, repoID, pattern, permission)
	if err != nil {
		return nil, fmt.Errorf("pathperm: create: %w", err)
	}
	id, _ := res.LastInsertId()
	return &Rule{ID: id, RepoID: repoID, Pattern: pattern, Permission: permission}, nil
}

// List returns the rules configured for a repository.
func (s *Service) List(ctx context.Context, repoID int64) ([]Rule, error) {
	q := s.st.Rebind(`SELECT id, repo_id, path_pattern, role_id, permission FROM path_permissions WHERE repo_id = ? ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q, repoID)
	if err != nil {
		return nil, fmt.Errorf("pathperm: list: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Rule
	for rows.Next() {
		var r Rule
		var roleID sql.NullInt64
		if err := rows.Scan(&r.ID, &r.RepoID, &r.Pattern, &roleID, &r.Permission); err != nil {
			return nil, err
		}
		if roleID.Valid {
			v := roleID.Int64
			r.RoleID = &v
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Delete removes a rule scoped to a repository.
func (s *Service) Delete(ctx context.Context, repoID, id int64) error {
	q := s.st.Rebind(`DELETE FROM path_permissions WHERE repo_id = ? AND id = ?`)
	res, err := s.st.DB.ExecContext(ctx, q, repoID, id)
	if err != nil {
		return fmt.Errorf("pathperm: delete: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("pathperm: rule not found")
	}
	return nil
}

// validatePattern rejects traversal tokens and control characters.
func validatePattern(pattern string) error {
	if strings.TrimSpace(pattern) == "" {
		return fmt.Errorf("pathperm: pattern is required")
	}
	if strings.Contains(pattern, "..") {
		return fmt.Errorf("pathperm: pattern must not contain %q", "..")
	}
	for _, r := range pattern {
		if r == 0 || unicode.IsControl(r) {
			return fmt.Errorf("pathperm: pattern must not contain control characters")
		}
	}
	return nil
}

// Match returns the permission of the most specific (longest) rule whose
// pattern matches path. When patterns share a length the first one wins.
func Match(rules []Rule, path string) (string, bool) {
	best := ""
	found := false
	for _, r := range rules {
		if !globMatch(r.Pattern, path) {
			continue
		}
		if !found || len(r.Pattern) > len(best) {
			best = r.Pattern
			found = true
		}
	}
	if !found {
		return "", false
	}
	for _, r := range rules {
		if r.Pattern == best {
			return r.Permission, true
		}
	}
	return "", false
}

// globMatch implements a simple glob matcher supporting '*' (any sequence) and
// '?' (any single character). Matching is slash-agnostic.
func globMatch(pattern, name string) bool {
	px, nx := 0, 0
	nextPx, nextNx := -1, 0
	for nx < len(name) {
		if px < len(pattern) {
			switch pattern[px] {
			case '*':
				nextPx = px
				nextNx = nx
				px++
				continue
			case '?':
				px++
				nx++
				continue
			default:
				if pattern[px] == name[nx] {
					px++
					nx++
					continue
				}
			}
		}
		if nextPx >= 0 {
			px = nextPx + 1
			nextNx++
			nx = nextNx
			continue
		}
		return false
	}
	for px < len(pattern) && pattern[px] == '*' {
		px++
	}
	return px == len(pattern)
}
