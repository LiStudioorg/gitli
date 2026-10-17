// SPDX-License-Identifier: AGPL-3.0
package http

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/LiStudioorg/gitli/internal/domain"
)

// gitAuthTTL bounds how long a successful Basic-auth check is reused. Git
// clients resend credentials on every smart-HTTP request, and Argon2id costs
// 64 MiB per verification, so caching keeps parallel fetches from exhausting
// memory while still refreshing against password changes within a minute.
const gitAuthTTL = 60 * time.Second

const gitAuthMaxEntries = 4096

type gitAuthEntry struct {
	user    *domain.User
	expires time.Time
}

type gitAuthCache struct {
	mu      sync.Mutex
	entries map[string]gitAuthEntry
}

func newGitAuthCache() *gitAuthCache {
	return &gitAuthCache{entries: make(map[string]gitAuthEntry)}
}

func gitAuthKey(username, password string) string {
	sum := sha256.Sum256([]byte(username + "\x00" + password))
	return hex.EncodeToString(sum[:])
}

func (c *gitAuthCache) get(username, password string) (*domain.User, bool) {
	key := gitAuthKey(username, password)
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expires) {
		delete(c.entries, key)
		return nil, false
	}
	return entry.user, true
}

func (c *gitAuthCache) put(username, password string, user *domain.User) {
	key := gitAuthKey(username, password)
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= gitAuthMaxEntries {
		now := time.Now()
		for k, v := range c.entries {
			if now.After(v.expires) {
				delete(c.entries, k)
			}
		}
		if len(c.entries) >= gitAuthMaxEntries {
			c.entries = make(map[string]gitAuthEntry)
		}
	}
	c.entries[key] = gitAuthEntry{user: user, expires: time.Now().Add(gitAuthTTL)}
}

// invalidate drops every cached credential, used after password changes.
func (c *gitAuthCache) invalidate() {
	c.mu.Lock()
	c.entries = make(map[string]gitAuthEntry)
	c.mu.Unlock()
}
