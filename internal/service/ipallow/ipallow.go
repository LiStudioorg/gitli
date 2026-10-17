// SPDX-License-Identifier: AGPL-3.0
// Package ipallow manages per-organization IP allowlists.
package ipallow

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/LiStudioorg/gitli/internal/service/store"
)

// ErrNotFound indicates a missing rule.
var ErrNotFound = errors.New("ipallow: not found")

// Rule is an allowlist entry.
type Rule struct {
	ID          int64  `json:"id"`
	OrgID       int64  `json:"org_id"`
	CIDR        string `json:"cidr"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"created_at"`
}

// Service manages IP allowlists.
type Service struct {
	st *store.Store
}

// New builds an ipallow Service.
func New(st *store.Store) *Service {
	return &Service{st: st}
}

// ValidateCIDR checks that s is a valid IPv4/IPv6 CIDR or single IP.
func ValidateCIDR(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("ipallow: empty cidr")
	}
	if strings.Contains(s, "/") {
		if _, _, err := net.ParseCIDR(s); err != nil {
			return fmt.Errorf("ipallow: invalid cidr %q", s)
		}
		return nil
	}
	if net.ParseIP(s) == nil {
		return fmt.Errorf("ipallow: invalid ip %q", s)
	}
	return nil
}

// netOf converts a CIDR or single IP into a *net.IPNet.
func netOf(s string) *net.IPNet {
	if strings.Contains(s, "/") {
		_, n, err := net.ParseCIDR(s)
		if err == nil {
			return n
		}
		return nil
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return nil
	}
	bits := 32
	if ip.To4() == nil {
		bits = 128
	}
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}
}

// Add creates an allowlist rule.
func (s *Service) Add(ctx context.Context, orgID int64, cidr, description string, enabled bool) (*Rule, error) {
	if err := ValidateCIDR(cidr); err != nil {
		return nil, err
	}
	q := s.st.Rebind(`INSERT INTO ip_allowlist (org_id, cidr, description, enabled) VALUES (?, ?, ?, ?)`)
	res, err := s.st.DB.ExecContext(ctx, q, orgID, strings.TrimSpace(cidr), description, enabled)
	if err != nil {
		return nil, fmt.Errorf("ipallow: insert: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.ByID(ctx, id)
}

// ByID loads a rule.
func (s *Service) ByID(ctx context.Context, id int64) (*Rule, error) {
	q := s.st.Rebind(`SELECT id, org_id, cidr, description, enabled, created_at FROM ip_allowlist WHERE id = ?`)
	var r Rule
	err := s.st.DB.QueryRowContext(ctx, q, id).Scan(&r.ID, &r.OrgID, &r.CIDR, &r.Description, &r.Enabled, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// List returns the rules for an org.
func (s *Service) List(ctx context.Context, orgID int64) ([]*Rule, error) {
	q := s.st.Rebind(`SELECT id, org_id, cidr, description, enabled, created_at FROM ip_allowlist WHERE org_id = ? ORDER BY id`)
	rows, err := s.st.DB.QueryContext(ctx, q, orgID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*Rule
	for rows.Next() {
		var r Rule
		if err := rows.Scan(&r.ID, &r.OrgID, &r.CIDR, &r.Description, &r.Enabled, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

// Delete removes a rule.
func (s *Service) Delete(ctx context.Context, orgID, id int64) error {
	return s.st.ExecAffectedOr(ctx, ErrNotFound, s.st.Rebind(`DELETE FROM ip_allowlist WHERE id = ? AND org_id = ?`), id, orgID)
}

// SetEnabled toggles enforcement for an org.
func (s *Service) SetEnabled(ctx context.Context, orgID int64, enabled bool) error {
	_, err := s.st.DB.ExecContext(ctx, s.st.Rebind(`UPDATE orgs SET ip_allowlist_enabled = ? WHERE id = ?`), enabled, orgID)
	return err
}

// Enabled reports whether enforcement is on for an org.
func (s *Service) Enabled(ctx context.Context, orgID int64) (bool, error) {
	var enabled bool
	err := s.st.DB.QueryRowContext(ctx, s.st.Rebind(`SELECT ip_allowlist_enabled FROM orgs WHERE id = ?`), orgID).Scan(&enabled)
	return enabled, err
}

// Check reports whether clientIP is permitted for an org. When enforcement is
// disabled the check always succeeds.
func (s *Service) Check(ctx context.Context, orgID int64, clientIP string) (bool, error) {
	enabled, err := s.Enabled(ctx, orgID)
	if err != nil || !enabled {
		return true, nil
	}
	rules, err := s.List(ctx, orgID)
	if err != nil {
		return false, err
	}
	ip := net.ParseIP(strings.TrimSpace(clientIP))
	if ip == nil {
		return false, nil
	}
	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		if n := netOf(r.CIDR); n != nil && n.Contains(ip) {
			return true, nil
		}
	}
	return false, nil
}
