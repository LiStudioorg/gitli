// SPDX-License-Identifier: AGPL-3.0
package rbac

import (
	"sync"
	"time"
)

// cacheEntry holds a cached decision with an expiry.
type cacheEntry struct {
	value   bool
	expires time.Time
}

// permCache is a bounded, TTL-guarded cache keyed by permission strings.
// It also tracks keys per user so decisions can be invalidated precisely.
type permCache struct {
	mu      sync.Mutex
	max     int
	ttl     time.Duration
	entries map[string]cacheEntry
	byUser  map[int64]map[string]struct{}
}

func newPermCache(max int, ttl time.Duration) *permCache {
	if max <= 0 {
		max = 10000
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &permCache{
		max:     max,
		ttl:     ttl,
		entries: make(map[string]cacheEntry),
		byUser:  make(map[int64]map[string]struct{}),
	}
}

// get returns a cached decision if present and fresh.
func (c *permCache) get(key string) (bool, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok {
		return false, false
	}
	if time.Now().After(e.expires) {
		c.delLocked(key)
		return false, false
	}
	return e.value, true
}

// set stores a decision, evicting the oldest entries when over capacity.
func (c *permCache) set(userID int64, key string, value bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.entries[key]; !exists && len(c.entries) >= c.max {
		c.evictLocked()
	}
	c.entries[key] = cacheEntry{value: value, expires: time.Now().Add(c.ttl)}
	keys, ok := c.byUser[userID]
	if !ok {
		keys = make(map[string]struct{})
		c.byUser[userID] = keys
	}
	keys[key] = struct{}{}
}

// invalidateUser drops all cached decisions for a user.
func (c *permCache) invalidateUser(userID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.byUser[userID] {
		delete(c.entries, key)
	}
	delete(c.byUser, userID)
}

// invalidateAll clears the whole cache.
func (c *permCache) invalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]cacheEntry)
	c.byUser = make(map[int64]map[string]struct{})
}

func (c *permCache) delLocked(key string) {
	delete(c.entries, key)
	for u, keys := range c.byUser {
		delete(keys, key)
		if len(keys) == 0 {
			delete(c.byUser, u)
		}
	}
}

func (c *permCache) evictLocked() {
	// Drop expired first, then remove an arbitrary slice to bound work.
	now := time.Now()
	for k, e := range c.entries {
		if now.After(e.expires) {
			c.delLocked(k)
		}
	}
	for k := range c.entries {
		if len(c.entries) < c.max {
			break
		}
		c.delLocked(k)
	}
}
