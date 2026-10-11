// SPDX-License-Identifier: AGPL-3.0
package config

import "testing"

func TestResolveExternalExplicit(t *testing.T) {
	cfg := Defaults()
	cfg.Server.HTTPExternalURL = "https://git.example.com/"
	cfg.Server.SSHExternal = "git.example.com:2222"
	e := ResolveExternalURLs(cfg)
	if e.HTTPBase != "https://git.example.com" {
		t.Fatalf("http base = %q", e.HTTPBase)
	}
	if got := e.HTTPSCloneURL("a", "b"); got != "https://git.example.com/a/b.git" {
		t.Fatalf("https clone = %q", got)
	}
	if got := e.SSHCloneURL("a", "b"); got != "ssh://git@git.example.com:2222/a/b.git" {
		t.Fatalf("ssh clone = %q", got)
	}
}

func TestSSHPort22OmitsPort(t *testing.T) {
	e := ExternalURLs{HTTPBase: "https://g.ex.com", SSHHost: "g.ex.com", SSHPort: 22}
	if got := e.SSHCloneURL("a", "b"); got != "git@g.ex.com:a/b.git" {
		t.Fatalf("ssh clone = %q", got)
	}
}

func TestResolveExternalInferred(t *testing.T) {
	cfg := Defaults()
	cfg.Server.HTTPListen = []string{"0.0.0.0:8080"}
	cfg.Server.HTTPExternalURL = ""
	cfg.Server.SSHListen = []string{"0.0.0.0:2222"}
	cfg.Server.SSHExternal = ""
	e := ResolveExternalURLs(cfg)
	if e.HTTPBase != "http://localhost:8080" {
		t.Fatalf("http base = %q", e.HTTPBase)
	}
	if e.SSHHost != "localhost" || e.SSHPort != 2222 {
		t.Fatalf("ssh = %s:%d", e.SSHHost, e.SSHPort)
	}
}

func TestResolveExternalWildcardListen(t *testing.T) {
	cfg := Defaults()
	cfg.Server.HTTPExternalURL = ""
	cfg.Server.HTTPListen = []string{"[::]:8080"}
	cfg.Server.SSHExternal = ""
	cfg.Server.SSHListen = nil
	e := ResolveExternalURLs(cfg)
	if e.HTTPBase != "http://localhost:8080" {
		t.Fatalf("http base = %q, want http://localhost:8080", e.HTTPBase)
	}
	if e.SSHHost != "localhost" {
		t.Fatalf("ssh host = %q, want localhost", e.SSHHost)
	}
	if e.SSHPort != 22 {
		t.Fatalf("ssh port = %d, want 22", e.SSHPort)
	}
}

func TestResolveExternalSSHHostDefaultPort(t *testing.T) {
	cfg := Defaults()
	cfg.Server.HTTPExternalURL = "https://git.example.com"
	cfg.Server.SSHExternal = "host"
	e := ResolveExternalURLs(cfg)
	if e.SSHHost != "host" {
		t.Fatalf("ssh host = %q, want host", e.SSHHost)
	}
	if e.SSHPort != 22 {
		t.Fatalf("ssh port = %d, want 22", e.SSHPort)
	}
	if got := e.SSHCloneURL("a", "b"); got != "git@host:a/b.git" {
		t.Fatalf("ssh clone = %q", got)
	}
}

func TestExternalDiffersFromListen(t *testing.T) {
	cfg := Defaults()
	cfg.Server.HTTPListen = []string{"127.0.0.1:8080"}
	cfg.Server.HTTPExternalURL = "https://public.example.com"
	e := ResolveExternalURLs(cfg)
	if e.HTTPBase != "https://public.example.com" {
		t.Fatalf("external should win: %q", e.HTTPBase)
	}
}

func TestSplitHostPort(t *testing.T) {
	cases := []struct {
		in   string
		host string
		port int
	}{
		{"0.0.0.0:8080", "0.0.0.0", 8080},
		{"[::]:8081", "::", 8081},
		{"git.example.com:2222", "git.example.com", 2222},
		{"git.example.com", "git.example.com", 22},
	}
	for _, c := range cases {
		h, p := splitHostPort(c.in, 22)
		if h != c.host || p != c.port {
			t.Fatalf("%s => %s:%d, want %s:%d", c.in, h, p, c.host, c.port)
		}
	}
}
