// SPDX-License-Identifier: AGPL-3.0
package rbac

import (
	"testing"
	"time"
)

func TestParsePermission(t *testing.T) {
	okCases := []struct {
		in       string
		resource string
		action   string
	}{
		{"pulls:merge", "pulls", "merge"},
		{"repo:read", "repo", "read"},
		{"org:admin", "org", "admin"},
		{"a:b:c", "a", "b:c"},
		{":action", "", "action"},
		{"resource:", "resource", ""},
	}
	for _, c := range okCases {
		p, err := ParsePermission(c.in)
		if err != nil {
			t.Fatalf("ParsePermission(%q): %v", c.in, err)
		}
		if p.Resource != c.resource || p.Action != c.action {
			t.Fatalf("ParsePermission(%q)=%+v want %s:%s", c.in, p, c.resource, c.action)
		}
	}
	for _, in := range []string{"plain", "", "nocolon"} {
		if _, err := ParsePermission(in); err == nil {
			t.Fatalf("ParsePermission(%q): expected error", in)
		}
	}
}

func TestParsePermissionRoundTrip(t *testing.T) {
	for _, want := range BuiltinPermissions {
		got, err := ParsePermission(want.String())
		if err != nil {
			t.Fatalf("%s: %v", want, err)
		}
		if got != want {
			t.Fatalf("round trip %s => %+v", want, got)
		}
	}
}

func TestRankOf(t *testing.T) {
	cases := []struct {
		name string
		want int
	}{
		{"read", 0},
		{"triage", 1},
		{"write", 2},
		{"maintain", 3},
		{"admin", 4},
		{"none", -1},
		{"", -1},
		{"unknown", -1},
	}
	for _, c := range cases {
		if got := RankOf(c.name); got != c.want {
			t.Fatalf("RankOf(%q)=%d want %d", c.name, got, c.want)
		}
	}
}

func TestMatchPattern(t *testing.T) {
	cases := []struct {
		pat, name string
		want      bool
	}{
		{"main", "main", true},
		{"release/*", "release/1.0", true},
		{"release/*", "main", false},
		{"*", "anything", true},
		{"v?.0", "v1.0", true},
		{"v?.0", "v10.0", false},
	}
	for _, c := range cases {
		if got := MatchPattern(c.pat, c.name); got != c.want {
			t.Fatalf("MatchPattern(%q,%q)=%v want %v", c.pat, c.name, got, c.want)
		}
	}
}

func TestLadderPermissionsMonotonic(t *testing.T) {
	read := map[Permission]bool{}
	for _, p := range ladderPermissions("read") {
		read[p] = true
	}
	write := map[Permission]bool{}
	for _, p := range ladderPermissions("write") {
		write[p] = true
	}
	for p := range read {
		if !write[p] {
			t.Fatalf("write should include read permission %s", p.String())
		}
	}
	if !write[Permission{"contents", "write"}] {
		t.Fatal("write should include contents:write")
	}
	if read[Permission{"contents", "write"}] {
		t.Fatal("read must not include contents:write")
	}
}

func TestCacheExpiryAndInvalidation(t *testing.T) {
	c := newPermCache(10, 50*time.Millisecond)
	c.set(1, "1:0:repo:read", true)
	if v, ok := c.get("1:0:repo:read"); !ok || !v {
		t.Fatal("expected cache hit")
	}
	time.Sleep(70 * time.Millisecond)
	if _, ok := c.get("1:0:repo:read"); ok {
		t.Fatal("expected cache entry to expire")
	}
	c.set(2, "2:0:repo:write", true)
	c.invalidateUser(2)
	if _, ok := c.get("2:0:repo:write"); ok {
		t.Fatal("expected user invalidation to remove entry")
	}
}

func TestCacheCapacityEviction(t *testing.T) {
	c := newPermCache(3, time.Minute)
	for i := int64(1); i <= 5; i++ {
		c.set(i, cacheKey(i, Requirement{Resource: "repo", Action: "read"}), true)
	}
	c.mu.Lock()
	n := len(c.entries)
	c.mu.Unlock()
	if n > 3 {
		t.Fatalf("cache exceeded capacity: %d", n)
	}
}
