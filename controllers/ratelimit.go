package controllers

import (
	"sync"
	"time"
)

// Keys tracked before full buckets are forgotten, so the map cannot grow without bound
const attemptLimiterMaxKeys = 10000

// AttemptLimiter is a token bucket per key (e.g. a client address): each failed attempt takes a token, and
// tokens come back one every `refill`, up to `burst`. Successful attempts are free.
type AttemptLimiter struct {
	burst  float64
	refill time.Duration
	now    func() time.Time

	mu      sync.Mutex
	buckets map[string]*attemptBucket
}

type attemptBucket struct {
	tokens float64
	last   time.Time
}

func NewAttemptLimiter(burst int, refill time.Duration) *AttemptLimiter {
	return &AttemptLimiter{
		burst:   float64(burst),
		refill:  refill,
		now:     time.Now,
		buckets: make(map[string]*attemptBucket),
	}
}

// Refill a bucket up to the current time, the caller holds the lock
func (l *AttemptLimiter) bucket(key string, now time.Time) *attemptBucket {
	b, ok := l.buckets[key]
	if !ok {
		return &attemptBucket{tokens: l.burst, last: now}
	}
	b.tokens = min(l.burst, b.tokens+float64(now.Sub(b.last))/float64(l.refill))
	b.last = now
	return b
}

// Blocked reports whether the key has no attempt left, and how long until it gets one
func (l *AttemptLimiter) Blocked(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.bucket(key, l.now())
	if b.tokens >= 1 {
		return false, 0
	}
	return true, time.Duration((1 - b.tokens) * float64(l.refill))
}

func (l *AttemptLimiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if len(l.buckets) >= attemptLimiterMaxKeys {
		for k := range l.buckets {
			if l.bucket(k, now).tokens >= l.burst {
				delete(l.buckets, k)
			}
		}
	}
	b := l.bucket(key, now)
	b.tokens = max(0, b.tokens-1)
	l.buckets[key] = b
}
