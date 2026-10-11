// SPDX-License-Identifier: AGPL-3.0
// Package secutil provides shared security helpers: strict name validation,
// path containment and filesystem permission enforcement.
package secutil

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// nameRe is the strict allowed pattern for owners and repositories.
	nameRe = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9._-]{0,98}[A-Za-z0-9])?$`)
)

// ValidateName rejects names containing traversal or path separators and
// enforces a conservative character set.
func ValidateName(kind, name string) error {
	if name == "" {
		return fmt.Errorf("%s: name required", kind)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("%s: invalid name %q", kind, name)
	}
	if strings.ContainsAny(name, "/\\~") || strings.Contains(name, "..") {
		return fmt.Errorf("%s: illegal characters in %q", kind, name)
	}
	if !nameRe.MatchString(name) {
		return fmt.Errorf("%s: name %q does not match allowed pattern", kind, name)
	}
	return nil
}

// ValidateRef rejects git revision arguments that could be interpreted as
// options (leading '-') or contain control characters. This prevents argument
// injection such as `--output=/path` reaching git as a positional revision.
func ValidateRef(ref string) error {
	if ref == "" {
		return fmt.Errorf("git: empty ref")
	}
	if strings.HasPrefix(ref, "-") {
		return fmt.Errorf("git: ref may not start with '-'")
	}
	if strings.ContainsAny(ref, "\x00\n\r") {
		return fmt.Errorf("git: ref contains control characters")
	}
	return nil
}

// ValidateSubPath rejects option-like or traversal path arguments passed to git.
func ValidateSubPath(p string) error {
	if p == "" {
		return fmt.Errorf("git: empty path")
	}
	if strings.HasPrefix(p, "-") {
		return fmt.Errorf("git: path may not start with '-'")
	}
	if strings.Contains(p, "..") || strings.ContainsAny(p, "\x00\n\r") {
		return fmt.Errorf("git: path contains forbidden characters")
	}
	return nil
}

// ValidateOutboundURL rejects non-http(s) schemes and, when blockPrivate is
// set, hosts that resolve to private, loopback or link-local addresses.
// It is used to mitigate SSRF for webhooks, mirrors and imports.
func ValidateOutboundURL(raw string, blockPrivate bool) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("secutil: invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("secutil: unsupported scheme %q", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("secutil: missing host")
	}
	if !blockPrivate {
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		// Unresolvable hosts are allowed; the connection attempt will fail.
		return nil
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() || ip.IsUnspecified() {
			return fmt.Errorf("secutil: refusing private address %s", ip)
		}
	}
	return nil
}

// SafeJoin joins base and elem, returning an error if the result escapes base.
func SafeJoin(base string, elem ...string) (string, error) {
	joined := filepath.Join(append([]string{base}, elem...)...)
	return EnsureWithin(base, joined)
}

// EnsureWithin verifies that target stays within base after cleaning.
func EnsureWithin(base, target string) (string, error) {
	cleanBase := filepath.Clean(base)
	cleanTarget := filepath.Clean(target)
	if cleanTarget == cleanBase {
		return cleanTarget, nil
	}
	if !strings.HasPrefix(cleanTarget, cleanBase+string(os.PathSeparator)) {
		return "", fmt.Errorf("secutil: path %q escapes base %q", target, base)
	}
	return cleanTarget, nil
}

// CheckPerms fails if path's permission bits are looser than want.
func CheckPerms(path string, want os.FileMode) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	got := info.Mode().Perm()
	if got&^want != 0 {
		return fmt.Errorf("secutil: %s has mode %04o, expected no bits outside %04o", path, got, want)
	}
	return nil
}

// EnforceDirPerms verifies permissions for a set of directories.
func EnforceDirPerms(checks map[string]os.FileMode) error {
	for p, want := range checks {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if err := CheckPerms(p, want); err != nil {
			return err
		}
	}
	return nil
}
