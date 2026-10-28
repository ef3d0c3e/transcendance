package main

import (
	"sync"
	"time"
)

// Fixed window rate limit per client address
type limiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu   sync.Mutex
	hits map[string]*window
}

type window struct {
	start time.Time
	count int
}

func newLimiter(limit int, period time.Duration, now func() time.Time) *limiter {
	return &limiter{limit: limit, window: period, now: now, hits: map[string]*window{}}
}

func (l *limiter) allow(key string) bool {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	// Forget finished windows before the map grows large
	if len(l.hits) > 10000 {
		for k, w := range l.hits {
			if now.Sub(w.start) >= l.window {
				delete(l.hits, k)
			}
		}
	}
	w, ok := l.hits[key]
	if !ok || now.Sub(w.start) >= l.window {
		l.hits[key] = &window{start: now, count: 1}
		return true
	}
	if w.count >= l.limit {
		return false
	}
	w.count++
	return true
}
