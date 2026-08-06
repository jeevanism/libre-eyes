package patientsearch

import (
	"math"
	"sync"
	"time"
)

const (
	principalBurst           = 10.0
	principalTokensPerSecond = 1.0
	maximumPrincipalBuckets  = 16384
)

type principalBucket struct {
	tokens     float64
	updatedAt  time.Time
	lastAccess time.Time
}

type principalLimiter struct {
	mu      sync.Mutex
	buckets map[int64]principalBucket
	now     func() time.Time
}

func newPrincipalLimiter() *principalLimiter {
	return &principalLimiter{buckets: make(map[int64]principalBucket), now: time.Now}
}

func (l *principalLimiter) allow(userID int64) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now().UTC()
	bucket, ok := l.buckets[userID]
	if !ok {
		if len(l.buckets) >= maximumPrincipalBuckets {
			l.evictOldest()
		}
		bucket = principalBucket{tokens: principalBurst, updatedAt: now}
	}
	elapsed := now.Sub(bucket.updatedAt).Seconds()
	if elapsed > 0 {
		bucket.tokens = math.Min(principalBurst, bucket.tokens+elapsed*principalTokensPerSecond)
	}
	bucket.updatedAt = now
	bucket.lastAccess = now

	if bucket.tokens >= 1 {
		bucket.tokens--
		l.buckets[userID] = bucket
		return true, 0
	}
	l.buckets[userID] = bucket
	retry := time.Duration(math.Ceil((1-bucket.tokens)/principalTokensPerSecond)) * time.Second
	if retry < time.Second {
		retry = time.Second
	}
	return false, retry
}

func (l *principalLimiter) evictOldest() {
	var oldestUserID int64
	var oldest time.Time
	for userID, bucket := range l.buckets {
		if oldest.IsZero() || bucket.lastAccess.Before(oldest) {
			oldestUserID = userID
			oldest = bucket.lastAccess
		}
	}
	delete(l.buckets, oldestUserID)
}
