// SPDX-License-Identifier: AGPL-3.0
// Package ldapauth authenticates users against an LDAP directory.
package ldapauth

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"

	"github.com/LiStudioorg/gitli/internal/config"
)

// ErrInvalidCredentials indicates a failed LDAP bind.
var ErrInvalidCredentials = errors.New("ldap: invalid credentials")

// User is the minimal profile retrieved from LDAP.
type User struct {
	Username string
	Email    string
	FullName string
	DN       string
}

// Service performs LDAP authentication.
type Service struct {
	cfg config.LDAPConfig
}

// New builds an LDAP Service from configuration. Returns nil when disabled.
func New(cfg config.LDAPConfig) *Service {
	if !cfg.Enabled || cfg.Host == "" {
		return nil
	}
	return &Service{cfg: cfg}
}

// Enabled reports whether LDAP is configured.
func (s *Service) Enabled() bool { return s != nil }

func (s *Service) connect() (*ldap.Conn, error) {
	addr := net.JoinHostPort(s.cfg.Host, fmt.Sprintf("%d", s.cfg.Port))
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	if s.cfg.UseTLS {
		tlsCfg := &tls.Config{ServerName: s.cfg.Host, InsecureSkipVerify: s.cfg.SkipVerify} //nolint:gosec // opt-in
		return ldap.DialURL("ldaps://"+addr, ldap.DialWithTLSConfig(tlsCfg), ldap.DialWithDialer(dialer))
	}
	return ldap.DialURL("ldap://"+addr, ldap.DialWithDialer(dialer))
}

// Authenticate verifies a username/password pair and returns the user profile.
func (s *Service) Authenticate(ctx context.Context, username, password string) (*User, error) {
	if s == nil {
		return nil, ErrInvalidCredentials
	}
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	conn, err := s.connect()
	if err != nil {
		return nil, fmt.Errorf("ldap: connect: %w", err)
	}
	defer func() { _ = conn.Close() }()
	conn.SetTimeout(10 * time.Second)

	// Bind with the service account to search for the user entry.
	if s.cfg.BindDN != "" {
		if err := conn.Bind(s.cfg.BindDN, s.cfg.BindPassword); err != nil {
			return nil, fmt.Errorf("ldap: service bind: %w", err)
		}
	}

	filter := s.cfg.UserFilter
	if filter == "" {
		filter = "(uid=%s)"
	}
	if !strings.Contains(filter, "%s") {
		return nil, fmt.Errorf("ldap: user_filter must contain %%s")
	}
	safeUser := ldap.EscapeFilter(username)
	filter = strings.ReplaceAll(filter, "%s", safeUser)

	searchReq := ldap.NewSearchRequest(
		s.cfg.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 2, 10, false,
		filter,
		[]string{"uid", "cn", "mail", "displayName", "sAMAccountName"},
		nil,
	)
	res, err := conn.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("ldap: search: %w", err)
	}
	if len(res.Entries) == 0 {
		return nil, ErrInvalidCredentials
	}
	entry := res.Entries[0]

	// Re-bind as the user to verify the password.
	if err := conn.Bind(entry.DN, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	u := &User{DN: entry.DN}
	u.Username = firstNonEmpty(entry.GetAttributeValue("uid"), entry.GetAttributeValue("sAMAccountName"), username)
	u.Email = entry.GetAttributeValue("mail")
	u.FullName = firstNonEmpty(entry.GetAttributeValue("displayName"), entry.GetAttributeValue("cn"), u.Username)
	return u, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
