package patientsearchhttp

import (
	"math"
	"sync"
	"time"
)

type boundaryBucket struct {
	tokens     float64
	updatedAt  time.Time
	lastAccess time.Time
}

type boundaryLimiter struct {
	mu         sync.Mutex
	buckets    map[string]boundaryBucket
	capacity   float64
	refillRate float64
	maximum    int
	now        func() time.Time
}

func newBoundaryLimiter(capacity, tokensPerSecond float64, maximum int) *boundaryLimiter {
	return &boundaryLimiter{
		buckets: make(map[string]boundaryBucket), capacity: capacity,
		refillRate: tokensPerSecond, maximum: maximum, now: time.Now,
	}
}

func (l *boundaryLimiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	bucket, ok := l.buckets[key]
	if !ok {
		if len(l.buckets) >= l.maximum {
			l.evictOldest()
		}
		bucket = boundaryBucket{tokens: l.capacity, updatedAt: now}
	}
	if elapsed := now.Sub(bucket.updatedAt).Seconds(); elapsed > 0 {
		bucket.tokens = math.Min(l.capacity, bucket.tokens+elapsed*l.refillRate)
	}
	bucket.updatedAt = now
	bucket.lastAccess = now
	if bucket.tokens >= 1 {
		bucket.tokens--
		l.buckets[key] = bucket
		return true, 0
	}
	l.buckets[key] = bucket
	retry := time.Duration(math.Ceil((1-bucket.tokens)/l.refillRate)) * time.Second
	return false, max(retry, time.Second)
}

func (l *boundaryLimiter) evictOldest() {
	oldestKey := ""
	var oldest time.Time
	for key, bucket := range l.buckets {
		if oldest.IsZero() || bucket.lastAccess.Before(oldest) {
			oldestKey = key
			oldest = bucket.lastAccess
		}
	}
	delete(l.buckets, oldestKey)
}
