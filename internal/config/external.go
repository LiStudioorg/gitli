// SPDX-License-Identifier: AGPL-3.0
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// setStrSlice parses a comma-separated environment variable into a slice.
func setStrSlice(dst *[]string, key string) {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) > 0 {
		*dst = out
	}
}

// migrateLegacyServer converts the deprecated single-address fields into the
// new listen slices, logging a warning so operators update their config.
func migrateLegacyServer(cfg *Config) {
	s := &cfg.Server
	if s.HTTP != "" {
		slog.Warn("config: [server] http is deprecated, use http_listen array")
		s.HTTPListen = []string{s.HTTP}
		s.HTTP = ""
	}
	if s.ExternalURL != "" {
		slog.Warn("config: [server] external_url is deprecated, use http_external_url")
		if s.HTTPExternalURL == "" {
			s.HTTPExternalURL = s.ExternalURL
		}
		s.ExternalURL = ""
	}
	if s.SSH != "" {
		slog.Warn("config: [server] ssh is deprecated, use ssh_listen array")
		s.SSHListen = []string{s.SSH}
		s.SSH = ""
	}
}

// ExternalURLs holds resolved externally visible addresses.
type ExternalURLs struct {
	HTTPBase string // e.g. https://git.example.com (no trailing slash)
	SSHHost  string // e.g. git.example.com
	SSHPort  int    // e.g. 2222, or 22 when omitted
}

// ResolveExternalURLs derives the externally visible HTTP and SSH addresses
// from the configured external URLs, falling back to the first listen address
// (only meaningful when there is no reverse proxy / NAT).
func ResolveExternalURLs(cfg *Config) ExternalURLs {
	s := &cfg.Server
	out := ExternalURLs{SSHPort: 22}

	// HTTP base.
	if s.HTTPExternalURL != "" {
		out.HTTPBase = strings.TrimRight(s.HTTPExternalURL, "/")
	} else if len(s.HTTPListen) > 0 {
		out.HTTPBase = "http://" + rewriteWildcard(s.HTTPListen[0])
	} else {
		out.HTTPBase = "http://localhost:8080"
	}
	out.HTTPBase = strings.TrimRight(out.HTTPBase, "/")

	// SSH external: explicit wins, else derive from HTTP host + ssh listen port.
	if s.SSHExternal != "" {
		host, port := splitHostPort(s.SSHExternal, 22)
		out.SSHHost = host
		out.SSHPort = port
		return out
	}
	out.SSHHost = hostFromURL(out.HTTPBase)
	if len(s.SSHListen) > 0 {
		_, port := splitHostPort(s.SSHListen[0], 22)
		out.SSHPort = port
	}
	return out
}

// SSHCloneURL builds the ssh:// clone URL for owner/repo.
func (e ExternalURLs) SSHCloneURL(owner, repo string) string {
	host := e.SSHHost
	if host == "" {
		host = "localhost"
	}
	if e.SSHPort == 22 || e.SSHPort == 0 {
		return fmt.Sprintf("git@%s:%s/%s.git", host, owner, repo)
	}
	return fmt.Sprintf("ssh://git@%s:%d/%s/%s.git", host, e.SSHPort, owner, repo)
}

// HTTPSCloneURL builds the https clone URL for owner/repo.
func (e ExternalURLs) HTTPSCloneURL(owner, repo string) string {
	return fmt.Sprintf("%s/%s/%s.git", e.HTTPBase, owner, repo)
}

// rewriteWildcard replaces bind-all wildcard hosts with localhost.
func rewriteWildcard(addr string) string {
	host, port := splitHostPort(addr, 0)
	switch host {
	case "0.0.0.0", "::", "":
		host = "localhost"
	}
	if port == 0 {
		return host
	}
	if strings.Contains(host, ":") {
		return "[" + host + "]:" + strconv.Itoa(port)
	}
	return host + ":" + strconv.Itoa(port)
}

// splitHostPort parses "host:port" or "[v6]:port" or "host", returning the
// host and port (def when absent). A leading ":port" yields an empty host.
func splitHostPort(addr string, def int) (string, int) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", def
	}
	if strings.HasPrefix(addr, "[") {
		if i := strings.Index(addr, "]"); i >= 0 {
			host := addr[1:i]
			rest := addr[i+1:]
			if strings.HasPrefix(rest, ":") {
				if p, err := strconv.Atoi(rest[1:]); err == nil {
					return host, p
				}
			}
			return host, def
		}
	}
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return addr, def
	}
	host := addr[:i]
	if p, err := strconv.Atoi(addr[i+1:]); err == nil {
		return host, p
	}
	return addr, def
}

// hostFromURL extracts the hostname from a URL-like string.
func hostFromURL(raw string) string {
	x := raw
	if i := strings.Index(x, "://"); i >= 0 {
		x = x[i+3:]
	}
	if i := strings.IndexAny(x, "/?"); i >= 0 {
		x = x[:i]
	}
	host, _ := splitHostPort(x, 22)
	return host
}
