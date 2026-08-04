package authhttp

import (
	"sync"
	"time"
)

type fixedWindowEntry struct {
	startedAt time.Time
	count     int
}

// fixedWindowLimiter provides bounded, process-local protection for anonymous
// endpoints. Credential lockout remains authoritative in PostgreSQL.
type fixedWindowLimiter struct {
	mu         sync.Mutex
	entries    map[string]fixedWindowEntry
	maxEntries int
}

func newFixedWindowLimiter(maxEntries int) *fixedWindowLimiter {
	return &fixedWindowLimiter{
		entries:    make(map[string]fixedWindowEntry),
		maxEntries: maxEntries,
	}
}

func (l *fixedWindowLimiter) allow(key string, limit int, window time.Duration, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, exists := l.entries[key]
	if exists && now.Sub(entry.startedAt) >= window {
		delete(l.entries, key)
		exists = false
	}
	if !exists {
		l.makeRoom(now, window)
		l.entries[key] = fixedWindowEntry{startedAt: now, count: 1}
		return true, 0
	}
	if entry.count >= limit {
		return false, max(window-now.Sub(entry.startedAt), time.Second)
	}
	entry.count++
	l.entries[key] = entry
	return true, 0
}

func (l *fixedWindowLimiter) makeRoom(now time.Time, window time.Duration) {
	if len(l.entries) < l.maxEntries {
		return
	}
	var oldestKey string
	var oldestStart time.Time
	for key, entry := range l.entries {
		if now.Sub(entry.startedAt) >= window {
			delete(l.entries, key)
			continue
		}
		if oldestKey == "" || entry.startedAt.Before(oldestStart) {
			oldestKey = key
			oldestStart = entry.startedAt
		}
	}
	if len(l.entries) >= l.maxEntries && oldestKey != "" {
		delete(l.entries, oldestKey)
	}
}
