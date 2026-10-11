// SPDX-License-Identifier: AGPL-3.0
package git

import "testing"

func TestRefUpdateBranchName(t *testing.T) {
	cases := []struct {
		ref  string
		want string
	}{
		{"refs/heads/main", "main"},
		{"refs/heads/feature/x", "feature/x"},
		{"refs/tags/v1.0", ""},
		{"HEAD", ""},
		{"refs/heads/", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := (RefUpdate{Ref: c.ref}).BranchName(); got != c.want {
			t.Fatalf("BranchName(%q)=%q want %q", c.ref, got, c.want)
		}
	}
}

func TestRefUpdateTagName(t *testing.T) {
	cases := []struct {
		ref  string
		want string
	}{
		{"refs/tags/v1.0", "v1.0"},
		{"refs/tags/release/1", "release/1"},
		{"refs/heads/main", ""},
		{"HEAD", ""},
		{"refs/tags/", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := (RefUpdate{Ref: c.ref}).TagName(); got != c.want {
			t.Fatalf("TagName(%q)=%q want %q", c.ref, got, c.want)
		}
	}
}

func TestRefUpdateIsDelete(t *testing.T) {
	const zero = "0000000000000000000000000000000000000000"
	cases := []struct {
		new  string
		want bool
	}{
		{zero, true},
		{"dev", false},
		{"ab12", false},
		{"", true},
		{"0000", true},
	}
	for _, c := range cases {
		if got := (RefUpdate{New: c.new}).IsDelete(); got != c.want {
			t.Fatalf("IsDelete(new=%q)=%v want %v", c.new, got, c.want)
		}
	}
}

func TestGlobMatch(t *testing.T) {
	cases := []struct {
		pattern string
		name    string
		want    bool
	}{
		{"main", "main", true},
		{"main", "mainx", false},
		{"*", "", true},
		{"*", "anything", true},
		{"**", "a/b/c", true},
		{"release/*", "release/1.0", true},
		{"release/*", "release/1.0/rc1", true},
		{"release/*", "main", false},
		{"v?.0", "v1.0", true},
		{"v?.0", "v10.0", false},
		{"a*c", "abc", true},
		{"a*c", "ac", true},
		{"a*c", "abdc", true},
		{"a*c", "abd", false},
		{"*main", "main", true},
		{"*main", "feature/main", true},
		{"main*", "main", true},
		{"main*", "maintenance", true},
		{"", "x", false},
		{"", "", true},
		{"?at", "cat", true},
		{"?at", "at", false},
	}
	for _, c := range cases {
		if got := globMatch(c.pattern, c.name); got != c.want {
			t.Fatalf("globMatch(%q,%q)=%v want %v", c.pattern, c.name, got, c.want)
		}
		if got := MatchPattern(c.pattern, c.name); got != c.want {
			t.Fatalf("MatchPattern(%q,%q)=%v want %v", c.pattern, c.name, got, c.want)
		}
	}
}
