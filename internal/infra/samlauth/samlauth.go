// SPDX-License-Identifier: AGPL-3.0
// Package samlauth implements SAML SP-initiated login using crewjam/saml.
package samlauth

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"

	"github.com/LiStudioorg/gitli/internal/config"
)

// ErrNotConfigured indicates SAML is disabled or misconfigured.
var ErrNotConfigured = errors.New("saml: not configured")

// Service wraps a samlsp middleware and IdP metadata.
type Service struct {
	cfg     config.SAMLConfig
	mw      *samlsp.Middleware
	sp      *saml.ServiceProvider
	idp     *saml.EntityDescriptor
	rootURL *url.URL
	client  *http.Client
}

// New builds a SAML Service. Returns nil when disabled.
func New(ctx context.Context, cfg config.SAMLConfig) (*Service, error) {
	if !cfg.Enabled || cfg.IDPMetadataURL == "" && cfg.IDPMetadataXML == "" {
		return nil, nil
	}
	root, err := url.Parse(cfg.RootURL)
	if err != nil || cfg.RootURL == "" {
		return nil, fmt.Errorf("saml: invalid root_url %q", cfg.RootURL)
	}
	cert, key, err := parseKeyPair(cfg.SPCert, cfg.SPKey)
	if err != nil {
		return nil, err
	}
	idp, err := fetchMetadata(ctx, cfg)
	if err != nil {
		return nil, err
	}
	mw, err := samlsp.New(samlsp.Options{
		EntityID:    root.String(),
		URL:         *root,
		Key:         key,
		Certificate: cert,
		IDPMetadata: idp,
	})
	if err != nil {
		return nil, fmt.Errorf("saml: init sp: %w", err)
	}
	return &Service{cfg: cfg, mw: mw, sp: &mw.ServiceProvider, idp: idp, rootURL: root, client: &http.Client{Timeout: 15 * time.Second}}, nil
}

// Enabled reports whether SAML is configured.
func (s *Service) Enabled() bool { return s != nil }

// Metadata returns the SP metadata XML (for IdP registration).
func (s *Service) Metadata() ([]byte, error) {
	if s == nil {
		return nil, ErrNotConfigured
	}
	return xml.Marshal(s.sp.Metadata())
}

// LoginURL returns the SP-initiated AuthnRequest redirect URL.
func (s *Service) LoginURL() (string, error) {
	if s == nil {
		return "", ErrNotConfigured
	}
	if s.idp == nil || len(s.idp.IDPSSODescriptors) == 0 || len(s.idp.IDPSSODescriptors[0].SingleSignOnServices) == 0 {
		return "", fmt.Errorf("saml: idp has no SSO endpoint")
	}
	ssoURL := s.idp.IDPSSODescriptors[0].SingleSignOnServices[0].Location
	req, err := s.sp.MakeAuthenticationRequest(ssoURL, saml.HTTPRedirectBinding, saml.HTTPPostBinding)
	if err != nil {
		return "", fmt.Errorf("saml: make authn request: %w", err)
	}
	u, err := req.Redirect(ssoURL, s.sp)
	if err != nil {
		return "", fmt.Errorf("saml: build redirect: %w", err)
	}
	return u.String(), nil
}

// ParseResponse validates a SAML response and returns the authenticated identity.
func (s *Service) ParseResponse(r *http.Request) (string, string, error) {
	if s == nil {
		return "", "", ErrNotConfigured
	}
	assertion, err := s.sp.ParseResponse(r, nil)
	if err != nil {
		return "", "", fmt.Errorf("saml: parse response: %w", err)
	}
	name := ""
	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		name = assertion.Subject.NameID.Value
	}
	email := ""
	for _, stmt := range assertion.AttributeStatements {
		for _, attr := range stmt.Attributes {
			switch attr.Name {
			case "email", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress", "urn:oid:0.9.2342.19200300.100.1.3":
				if len(attr.Values) > 0 {
					email = attr.Values[0].Value
				}
			case "displayName", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name":
				if name == "" && len(attr.Values) > 0 {
					name = attr.Values[0].Value
				}
			}
		}
	}
	if name == "" {
		return "", "", fmt.Errorf("saml: missing subject")
	}
	return name, email, nil
}

// RootURL returns the configured SP root URL.
func (s *Service) RootURL() string {
	if s == nil || s.rootURL == nil {
		return ""
	}
	return s.rootURL.String()
}

func parseKeyPair(certPEM, keyPEM string) (*x509.Certificate, *rsa.PrivateKey, error) {
	if certPEM == "" || keyPEM == "" {
		return nil, nil, fmt.Errorf("saml: sp_cert and sp_key are required")
	}
	cb, _ := pem.Decode([]byte(certPEM))
	if cb == nil {
		return nil, nil, fmt.Errorf("saml: invalid sp_cert PEM")
	}
	cert, err := x509.ParseCertificate(cb.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("saml: parse sp_cert: %w", err)
	}
	kb, _ := pem.Decode([]byte(keyPEM))
	if kb == nil {
		return nil, nil, fmt.Errorf("saml: invalid sp_key PEM")
	}
	var key *rsa.PrivateKey
	if k, err := x509.ParsePKCS1PrivateKey(kb.Bytes); err == nil {
		key = k
	} else if k2, err2 := x509.ParsePKCS8PrivateKey(kb.Bytes); err2 == nil {
		rk, ok := k2.(*rsa.PrivateKey)
		if !ok {
			return nil, nil, fmt.Errorf("saml: sp_key is not RSA")
		}
		key = rk
	} else {
		return nil, nil, fmt.Errorf("saml: parse sp_key: %w", err)
	}
	return cert, key, nil
}

func fetchMetadata(ctx context.Context, cfg config.SAMLConfig) (*saml.EntityDescriptor, error) {
	if cfg.IDPMetadataXML != "" {
		return samlsp.ParseMetadata([]byte(cfg.IDPMetadataXML))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.IDPMetadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("saml: metadata request: %w", err)
	}
	client := &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}}}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("saml: fetch metadata: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	return samlsp.ParseMetadata(body)
}
