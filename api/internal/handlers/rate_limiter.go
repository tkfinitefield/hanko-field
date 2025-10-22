package handlers

import (
	"strings"
	"sync"
	"time"
)

type rateLimiter interface {
	Allow(key string) bool
}

type simpleRateLimiter struct {
	limit  int
	window time.Duration
	clock  func() time.Time
	mu     sync.Mutex
	store  map[string]rateEntry
}

type rateEntry struct {
	count int
	reset time.Time
}

func newSimpleRateLimiter(limit int, window time.Duration, clock func() time.Time) rateLimiter {
	if limit <= 0 || window <= 0 {
		return nil
	}
	if clock == nil {
		clock = time.Now
	}
	return &simpleRateLimiter{
		limit:  limit,
		window: window,
		clock:  clock,
		store:  make(map[string]rateEntry),
	}
}

func (l *simpleRateLimiter) Allow(key string) bool {
	if l == nil {
		return true
	}
	key = strings.TrimSpace(key)
	if key == "" {
		key = "anonymous"
	}
	now := l.clock()
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.store[key]
	if !ok || now.After(entry.reset) {
		l.store[key] = rateEntry{count: 1, reset: now.Add(l.window)}
		l.pruneExpiredLocked(now)
		return true
	}

	if entry.count >= l.limit {
		return false
	}
	entry.count++
	l.store[key] = entry
	return true
}

func (l *simpleRateLimiter) pruneExpiredLocked(now time.Time) {
	if len(l.store) == 0 {
		return
	}
	for key, entry := range l.store {
		if now.After(entry.reset) {
			delete(l.store, key)
		}
	}
}
