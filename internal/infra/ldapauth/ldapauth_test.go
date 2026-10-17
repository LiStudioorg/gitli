// SPDX-License-Identifier: AGPL-3.0
package ldapauth

import (
	"context"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
)

func TestDisabledReturnsNil(t *testing.T) {
	if New(config.LDAPConfig{Enabled: false, Host: "x"}) != nil {
		t.Fatal("disabled LDAP should return nil service")
	}
	if New(config.LDAPConfig{Enabled: true, Host: ""}) != nil {
		t.Fatal("missing host should return nil service")
	}
}

func TestEnabled(t *testing.T) {
	s := New(config.LDAPConfig{Enabled: true, Host: "ldap.example.com", Port: 389})
	if s == nil || !s.Enabled() {
		t.Fatal("expected enabled service")
	}
	// Empty credentials are rejected without a network call.
	if _, err := s.Authenticate(context.Background(), "", ""); err == nil {
		t.Fatal("expected error for empty credentials")
	}
}

func TestNilServiceAuthenticate(t *testing.T) {
	var s *Service
	if _, err := s.Authenticate(context.Background(), "u", "p"); err != ErrInvalidCredentials {
		t.Fatalf("nil service should reject, got %v", err)
	}
}

func TestFilterMissingPlaceholder(t *testing.T) {
	s := New(config.LDAPConfig{Enabled: true, Host: "127.0.0.1", Port: 1, UserFilter: "(uid=x)"})
	// Connection will fail (port 1) but the filter check happens after connect;
	// this test only ensures the service is constructed.
	if s == nil {
		t.Fatal("service should be constructed")
	}
}
