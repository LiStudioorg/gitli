// SPDX-License-Identifier: AGPL-3.0
// Package metrics provides a lightweight in-process counter registry.
package metrics

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Registry is a concurrency-safe collection of named integer counters.
type Registry struct {
	mu        sync.Mutex
	counters  map[string]*int64
	startTime time.Time
}

// New creates an empty Registry.
func New() *Registry {
	return &Registry{
		counters:  make(map[string]*int64),
		startTime: time.Now(),
	}
}

// Inc increments the named counter by one.
func (r *Registry) Inc(name string) { r.Add(name, 1) }

// Add increments the named counter by n.
func (r *Registry) Add(name string, n int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.counters[name]
	if !ok {
		v = new(int64)
		r.counters[name] = v
	}
	*v += n
}

// Snapshot returns a copy of all counter values keyed by name.
func (r *Registry) Snapshot() map[string]int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]int64, len(r.counters))
	for name, v := range r.counters {
		out[name] = *v
	}
	return out
}

// Uptime reports the time elapsed since the Registry was created.
func (r *Registry) Uptime() time.Duration { return time.Since(r.startTime) }

// Prometheus renders the counters in the Prometheus text exposition format.
func (r *Registry) Prometheus() string {
	snap := r.Snapshot()
	names := make([]string, 0, len(snap))
	for name := range snap {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	for _, name := range names {
		metric := promName(name)
		fmt.Fprintf(&b, "# TYPE %s counter\n%s %d\n", metric, metric, snap[name])
	}
	fmt.Fprintf(&b, "# TYPE gitli_uptime_seconds gauge\ngitli_uptime_seconds %f\n", r.Uptime().Seconds())
	return b.String()
}

// promName converts an arbitrary counter name into a valid Prometheus metric
// name, prefixing with gitli_ when needed.
func promName(name string) string {
	var b strings.Builder
	for i, c := range name {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c == '_':
			b.WriteRune(c)
		case c >= '0' && c <= '9':
			if i == 0 {
				b.WriteByte('_')
			}
			b.WriteRune(c)
		default:
			b.WriteByte('_')
		}
	}
	s := b.String()
	if !strings.HasPrefix(s, "gitli_") {
		s = "gitli_" + s
	}
	return s
}
