// SPDX-License-Identifier: AGPL-3.0
package middleware

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter performs a fixed-window per-key rate limit in memory.
type RateLimiter struct {
	mu       sync.Mutex
	hits     map[string]*rateWindow
	limit    int
	window   time.Duration
	lastGCed time.Time
}

type rateWindow struct {
	count int
	start time.Time
}

// NewRateLimiter creates a limiter allowing limit events per window.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit <= 0 {
		limit = 5
	}
	if window <= 0 {
		window = 15 * time.Minute
	}
	return &RateLimiter{hits: map[string]*rateWindow{}, limit: limit, window: window, lastGCed: time.Now()}
}

// Allow reports whether key is within its quota, incrementing the counter.
func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gcLocked()
	now := time.Now()
	w, ok := r.hits[key]
	if !ok || now.Sub(w.start) >= r.window {
		r.hits[key] = &rateWindow{count: 1, start: now}
		return true
	}
	if w.count >= r.limit {
		return false
	}
	w.count++
	return true
}

// Reset clears the counter for a key (e.g. after successful login).
func (r *RateLimiter) Reset(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.hits, key)
}

func (r *RateLimiter) gcLocked() {
	now := time.Now()
	if now.Sub(r.lastGCed) < r.window {
		return
	}
	for k, w := range r.hits {
		if now.Sub(w.start) >= r.window {
			delete(r.hits, k)
		}
	}
	r.lastGCed = now
}

// Middleware applies the limiter using clientKey to derive the bucket.
func (r *RateLimiter) Middleware(clientKey func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if r.Allow(clientKey(req)) {
				next.ServeHTTP(w, req)
				return
			}
			http.Error(w, "too many requests", http.StatusTooManyRequests)
		})
	}
}
