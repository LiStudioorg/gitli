// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"testing"
	"time"

	"github.com/LiStudioorg/gitli/internal/domain"
)

func TestGitAuthCacheHitAndInvalidate(t *testing.T) {
	c := newGitAuthCache()
	u := &domain.User{ID: 7, Username: "alice"}

	if _, ok := c.get("alice", "secret"); ok {
		t.Fatal("cache must start empty")
	}
	c.put("alice", "secret", u)
	got, ok := c.get("alice", "secret")
	if !ok || got.ID != u.ID {
		t.Fatalf("expected cache hit, got ok=%v user=%v", ok, got)
	}
	if _, ok := c.get("alice", "other"); ok {
		t.Fatal("different password must not hit")
	}

	c.invalidate()
	if _, ok := c.get("alice", "secret"); ok {
		t.Fatal("invalidate must clear entries")
	}
}

func TestGitAuthCacheExpires(t *testing.T) {
	c := newGitAuthCache()
	c.put("bob", "pw", &domain.User{ID: 1})
	c.mu.Lock()
	for k, v := range c.entries {
		v.expires = time.Now().Add(-time.Second)
		c.entries[k] = v
	}
	c.mu.Unlock()
	if _, ok := c.get("bob", "pw"); ok {
		t.Fatal("expired entry must not be served")
	}
}
