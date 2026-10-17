// SPDX-License-Identifier: AGPL-3.0
package samlauth

import (
	"context"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
)

func TestDisabledReturnsNil(t *testing.T) {
	s, err := New(context.Background(), config.SAMLConfig{Enabled: false})
	if err != nil || s != nil {
		t.Fatalf("disabled saml = %v %v", s, err)
	}
}

func TestRequiresCertAndKey(t *testing.T) {
	_, err := New(context.Background(), config.SAMLConfig{
		Enabled: true, IDPMetadataXML: "<xml/>", RootURL: "https://git.example.com",
	})
	if err == nil {
		t.Fatal("expected error when sp_cert/sp_key missing")
	}
}

func TestInvalidRootURL(t *testing.T) {
	_, err := New(context.Background(), config.SAMLConfig{
		Enabled: true, IDPMetadataXML: "<xml/>", SPCert: "x", SPKey: "y", RootURL: "%zz",
	})
	if err == nil {
		t.Fatal("expected error for invalid root url")
	}
}

func TestNilService(t *testing.T) {
	var s *Service
	if s.Enabled() {
		t.Fatal("nil should be disabled")
	}
	if _, err := s.LoginURL(); err != ErrNotConfigured {
		t.Fatalf("nil LoginURL = %v", err)
	}
}
