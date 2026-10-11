// SPDX-License-Identifier: AGPL-3.0
package git

import "testing"

func TestValidateRefRejectsInjection(t *testing.T) {
	bad := []string{"", "-", "--output=/tmp/x", "-x", "a\nb", "a\x00b"}
	for _, r := range bad {
		if err := validateRef(r); err == nil {
			t.Fatalf("validateRef(%q) should fail", r)
		}
	}
	good := []string{"main", "HEAD", "refs/heads/main", "v1.0.0", "abc123"}
	for _, r := range good {
		if err := validateRef(r); err != nil {
			t.Fatalf("validateRef(%q) should pass: %v", r, err)
		}
	}
}

func TestValidateSubPath(t *testing.T) {
	for _, p := range []string{"", "-x", "../etc/passwd", "a/../b", "a\x00b"} {
		if err := validateSubPath(p); err == nil {
			t.Fatalf("validateSubPath(%q) should fail", p)
		}
	}
	for _, p := range []string{"README.md", "internal/app/app.go", "a/b/c.txt"} {
		if err := validateSubPath(p); err != nil {
			t.Fatalf("validateSubPath(%q) should pass: %v", p, err)
		}
	}
}
