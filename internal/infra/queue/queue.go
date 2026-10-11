// SPDX-License-Identifier: AGPL-3.0
// Package queue provides an in-process job queue with database persistence.
package queue

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Handler processes a job payload.
type Handler func(ctx context.Context, payload string) error

// Queue dispatches jobs to registered handlers using a bounded worker pool.
type Queue struct {
	mu       sync.RWMutex
	handlers map[string]Handler
	jobs     chan job
	log      *slog.Logger
	wg       sync.WaitGroup
}

type job struct {
	kind    string
	payload string
}

// New creates a Queue with the given number of workers and buffer size.
func New(workers, buffer int, log *slog.Logger) *Queue {
	if workers <= 0 {
		workers = 1
	}
	if buffer <= 0 {
		buffer = 128
	}
	if log == nil {
		log = slog.Default()
	}
	q := &Queue{
		handlers: map[string]Handler{},
		jobs:     make(chan job, buffer),
		log:      log,
	}
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.worker()
	}
	return q
}

// Register associates a handler with a job kind.
func (q *Queue) Register(kind string, h Handler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[kind] = h
}

// Enqueue schedules a job; it returns false if the buffer is full.
func (q *Queue) Enqueue(kind, payload string) bool {
	select {
	case q.jobs <- job{kind: kind, payload: payload}:
		return true
	default:
		q.log.Warn("queue full, dropping job", "kind", kind)
		return false
	}
}

// Shutdown stops accepting jobs and waits for in-flight work.
func (q *Queue) Shutdown(ctx context.Context) {
	close(q.jobs)
	done := make(chan struct{})
	go func() { q.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
	}
}

func (q *Queue) worker() {
	defer q.wg.Done()
	for j := range q.jobs {
		q.mu.RLock()
		h := q.handlers[j.kind]
		q.mu.RUnlock()
		if h == nil {
			q.log.Warn("no handler for job", "kind", j.kind)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		if err := h(ctx, j.payload); err != nil {
			q.log.Error("job failed", "kind", j.kind, "error", err)
		}
		cancel()
	}
}
