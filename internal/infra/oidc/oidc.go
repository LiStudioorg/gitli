// SPDX-License-Identifier: AGPL-3.0
// Package oidc implements a minimal OpenID Connect authorization-code client.
package oidc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	discoveryTimeout = 10 * time.Second
	defaultScope     = "openid profile email"
	verifierBytes    = 32
	stateBytes       = 32
)

// ErrDisabled indicates the provider is not enabled.
var ErrDisabled = errors.New("oidc: provider disabled")

// Config holds the settings for an OIDC provider connection.
type Config struct {
	Enabled      bool
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// Discovery is the subset of the OIDC discovery document used by the client.
type Discovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JwksURI               string `json:"jwks_uri"`
	Issuer                string `json:"issuer"`
}

// Provider is a minimal OIDC authorization-code client.
type Provider struct {
	cfg       Config
	discovery *Discovery
	http      *http.Client
}

// TokenResponse is the parsed response of a token exchange.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// UserInfo is the subset of the userinfo endpoint response used by the client.
type UserInfo struct {
	Subject           string `json:"sub"`
	Email             string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
	Name              string `json:"name"`
	Picture           string `json:"picture"`
}

// New builds a Provider. When cfg.Enabled is false it returns a disabled
// provider without performing any network access.
func New(ctx context.Context, cfg Config) (*Provider, error) {
	if !cfg.Enabled {
		return &Provider{cfg: cfg}, nil
	}

	p := &Provider{cfg: cfg, http: &http.Client{Timeout: discoveryTimeout}}

	doc, err := p.fetchDiscovery(ctx)
	if err != nil {
		return nil, err
	}
	if doc.Issuer != cfg.Issuer {
		return nil, fmt.Errorf("oidc: issuer mismatch: discovery %q, config %q", doc.Issuer, cfg.Issuer)
	}
	p.discovery = doc
	return p, nil
}

func (p *Provider) fetchDiscovery(ctx context.Context) (*Discovery, error) {
	endpoint := strings.TrimSuffix(p.cfg.Issuer, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("oidc: discovery request: %w", err)
	}
	resp, err := p.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oidc: discovery fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oidc: discovery status %d", resp.StatusCode)
	}
	var doc Discovery
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("oidc: decode discovery: %w", err)
	}
	return &doc, nil
}

// Enabled reports whether the provider is enabled and configured.
func (p *Provider) Enabled() bool {
	return p != nil && p.cfg.Enabled
}

// AuthCodeURL builds the authorization request URL for the code flow.
func (p *Provider) AuthCodeURL(state, nonce, codeChallenge string) (string, error) {
	if !p.Enabled() || p.discovery == nil {
		return "", ErrDisabled
	}
	scopes := p.cfg.Scopes
	if len(scopes) == 0 {
		scopes = strings.Fields(defaultScope)
	}
	q := url.Values{}
	q.Set("client_id", p.cfg.ClientID)
	q.Set("redirect_uri", p.cfg.RedirectURL)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(scopes, " "))
	q.Set("state", state)
	q.Set("nonce", nonce)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	return p.discovery.AuthorizationEndpoint + "?" + q.Encode(), nil
}

// Exchange swaps an authorization code for tokens.
func (p *Provider) Exchange(ctx context.Context, code, codeVerifier string) (*TokenResponse, error) {
	if !p.Enabled() || p.discovery == nil {
		return nil, ErrDisabled
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", p.cfg.RedirectURL)
	form.Set("client_id", p.cfg.ClientID)
	form.Set("client_secret", p.cfg.ClientSecret)
	form.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.discovery.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("oidc: token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oidc: token exchange: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oidc: token status %d", resp.StatusCode)
	}
	var tr TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("oidc: decode token: %w", err)
	}
	return &tr, nil
}

// UserInfo fetches the user profile associated with an access token.
func (p *Provider) UserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	if !p.Enabled() || p.discovery == nil {
		return nil, ErrDisabled
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.discovery.UserinfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("oidc: userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := p.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oidc: userinfo fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oidc: userinfo status %d", resp.StatusCode)
	}
	var ui UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&ui); err != nil {
		return nil, fmt.Errorf("oidc: decode userinfo: %w", err)
	}
	return &ui, nil
}

// GenerateVerifier returns a new PKCE code verifier (43-128 chars, base64url).
func GenerateVerifier() (string, error) {
	return randomToken(verifierBytes)
}

// Challenge returns the S256 PKCE code challenge for a verifier.
func Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// RandomState returns a new random OAuth2 state value.
func RandomState() (string, error) {
	return randomToken(stateBytes)
}

func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", fmt.Errorf("oidc: random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
