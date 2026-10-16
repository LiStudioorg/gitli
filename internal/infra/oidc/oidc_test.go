// SPDX-License-Identifier: AGPL-3.0
package oidc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newTestProvider(t *testing.T) (*Provider, *httptest.Server) {
	t.Helper()

	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		doc := Discovery{
			AuthorizationEndpoint: srv.URL + "/authorize",
			TokenEndpoint:         srv.URL + "/token",
			UserinfoEndpoint:      srv.URL + "/userinfo",
			JwksURI:               srv.URL + "/jwks",
			Issuer:                srv.URL,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doc)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.Form.Get("grant_type") != "authorization_code" {
			http.Error(w, "bad grant", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "access-123",
			IDToken:     "id-456",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer access-123" {
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(UserInfo{
			Subject:           "sub-1",
			Email:             "u@example.com",
			PreferredUsername: "user",
			Name:              "User",
			Picture:           "https://example.com/avatar.png",
		})
	})

	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	p, err := New(context.Background(), Config{
		Enabled:      true,
		Issuer:       srv.URL,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "https://app.example.com/callback",
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return p, srv
}

func TestRoundTrip(t *testing.T) {
	p, _ := newTestProvider(t)

	verifier, err := GenerateVerifier()
	if err != nil {
		t.Fatalf("GenerateVerifier() error: %v", err)
	}
	state, err := RandomState()
	if err != nil {
		t.Fatalf("RandomState() error: %v", err)
	}

	rawURL, err := p.AuthCodeURL(state, "nonce-1", Challenge(verifier))
	if err != nil {
		t.Fatalf("AuthCodeURL() error: %v", err)
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}
	q := u.Query()
	checks := map[string]string{
		"client_id":             "client-id",
		"redirect_uri":          "https://app.example.com/callback",
		"response_type":         "code",
		"state":                 state,
		"nonce":                 "nonce-1",
		"code_challenge_method": "S256",
	}
	for k, want := range checks {
		if got := q.Get(k); got != want {
			t.Errorf("auth url %s = %q, want %q", k, got, want)
		}
	}
	if q.Get("code_challenge") == "" {
		t.Error("auth url missing code_challenge")
	}
	if !strings.Contains(q.Get("scope"), "openid") {
		t.Errorf("auth url scope = %q, want contains openid", q.Get("scope"))
	}

	tr, err := p.Exchange(context.Background(), "auth-code", verifier)
	if err != nil {
		t.Fatalf("Exchange() error: %v", err)
	}
	if tr.AccessToken != "access-123" || tr.TokenType != "Bearer" || tr.ExpiresIn != 3600 {
		t.Errorf("Exchange() = %+v", tr)
	}

	ui, err := p.UserInfo(context.Background(), tr.AccessToken)
	if err != nil {
		t.Fatalf("UserInfo() error: %v", err)
	}
	if ui.Subject != "sub-1" || ui.Email != "u@example.com" || ui.PreferredUsername != "user" {
		t.Errorf("UserInfo() = %+v", ui)
	}
}

func TestIssuerMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(Discovery{Issuer: "https://evil.example.com"})
	}))
	defer srv.Close()

	_, err := New(context.Background(), Config{Enabled: true, Issuer: srv.URL})
	if err == nil {
		t.Fatal("New() = nil error, want issuer mismatch")
	}
}

func TestDisabledProvider(t *testing.T) {
	p, err := New(context.Background(), Config{})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if p.Enabled() {
		t.Fatal("Enabled() = true, want false")
	}
	if _, err := p.AuthCodeURL("s", "n", "c"); err == nil {
		t.Fatal("AuthCodeURL() = nil error, want error")
	}
	if _, err := p.Exchange(context.Background(), "c", "v"); err == nil {
		t.Fatal("Exchange() = nil error, want error")
	}
	if _, err := p.UserInfo(context.Background(), "t"); err == nil {
		t.Fatal("UserInfo() = nil error, want error")
	}
}

func TestPKCEFormat(t *testing.T) {
	verifier, err := GenerateVerifier()
	if err != nil {
		t.Fatalf("GenerateVerifier() error: %v", err)
	}
	if len(verifier) < 43 || len(verifier) > 128 {
		t.Errorf("verifier length = %d, want [43,128]", len(verifier))
	}
	if _, err := base64.RawURLEncoding.DecodeString(verifier); err != nil {
		t.Errorf("verifier not base64url: %v", err)
	}

	challenge := Challenge(verifier)
	if len(challenge) != 43 {
		t.Errorf("challenge length = %d, want 43", len(challenge))
	}
	if _, err := base64.RawURLEncoding.DecodeString(challenge); err != nil {
		t.Errorf("challenge not base64url: %v", err)
	}
}

func TestRandomStateUnique(t *testing.T) {
	a, err := RandomState()
	if err != nil {
		t.Fatalf("RandomState() error: %v", err)
	}
	b, err := RandomState()
	if err != nil {
		t.Fatalf("RandomState() error: %v", err)
	}
	if a == b {
		t.Error("RandomState() returned identical values")
	}
	if a == "" {
		t.Error("RandomState() returned empty value")
	}
}
