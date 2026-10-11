// SPDX-License-Identifier: AGPL-3.0
package rbac

import (
	"testing"
	"time"
)

func TestParsePermission(t *testing.T) {
	p, err := ParsePermission("pulls:merge")
	if err != nil {
		t.Fatal(err)
	}
	if p.Resource != "pulls" || p.Action != "merge" {
		t.Fatalf("got %+v", p)
	}
	if _, err := ParsePermission("bad"); err == nil {
		t.Fatal("expected error for malformed permission")
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
