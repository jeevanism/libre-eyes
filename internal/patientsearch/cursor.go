package patientsearch

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

const (
	cursorLifetime        = 15 * time.Minute
	cursorPruneInterval   = time.Minute
	maximumCursorEntries  = 16384
	cursorOrderingVersion = int16(1)
)

type cursorEntry struct {
	boundary             PageBoundary
	userID               int64
	sessionID            int64
	institutionID        int64
	siteID               int64
	firmID               int64
	contextVersion       int64
	criteriaDigest       [32]byte
	limit                int
	orderingVersion      int16
	normalizationVersion int16
	expiresAt            time.Time
}

type cursorStore struct {
	mu        sync.Mutex
	entries   map[[32]byte]cursorEntry
	now       func() time.Time
	nextPrune time.Time
}

func newCursorStore() *cursorStore {
	return &cursorStore{entries: make(map[[32]byte]cursorEntry), now: time.Now}
}

func (s *cursorStore) put(entry cursorEntry) (string, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", errors.New("generate patient search cursor")
	}
	token := base64.RawURLEncoding.EncodeToString(random)
	digest := sha256.Sum256([]byte(token))

	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()
	s.pruneIfDue(now)
	if len(s.entries) >= maximumCursorEntries {
		s.evictOldest()
	}
	entry.boundary = cloneBoundary(entry.boundary)
	entry.expiresAt = now.Add(cursorLifetime)
	s.entries[digest] = entry
	return token, nil
}

func (s *cursorStore) get(token string, expected cursorEntry) (PageBoundary, error) {
	if token == "" || len(token) > 512 {
		return PageBoundary{}, ErrInvalidRequest
	}
	digest := sha256.Sum256([]byte(token))

	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()
	s.pruneIfDue(now)
	entry, ok := s.entries[digest]
	if !ok {
		return PageBoundary{}, ErrInvalidRequest
	}
	if !now.Before(entry.expiresAt) {
		delete(s.entries, digest)
		return PageBoundary{}, ErrInvalidRequest
	}
	if !cursorMatches(entry, expected) {
		return PageBoundary{}, ErrInvalidRequest
	}
	return cloneBoundary(entry.boundary), nil
}

func (s *cursorStore) pruneIfDue(now time.Time) {
	if !s.nextPrune.IsZero() && now.Before(s.nextPrune) {
		return
	}
	s.prune(now)
	s.nextPrune = now.Add(cursorPruneInterval)
}

func cursorMatches(actual, expected cursorEntry) bool {
	return actual.userID == expected.userID &&
		actual.sessionID == expected.sessionID &&
		actual.institutionID == expected.institutionID &&
		actual.siteID == expected.siteID &&
		actual.firmID == expected.firmID &&
		actual.contextVersion == expected.contextVersion &&
		actual.limit == expected.limit &&
		actual.orderingVersion == expected.orderingVersion &&
		actual.normalizationVersion == expected.normalizationVersion &&
		subtle.ConstantTimeCompare(actual.criteriaDigest[:], expected.criteriaDigest[:]) == 1
}

func (s *cursorStore) prune(now time.Time) {
	for digest, entry := range s.entries {
		if !now.Before(entry.expiresAt) {
			delete(s.entries, digest)
		}
	}
}

func (s *cursorStore) evictOldest() {
	var oldestDigest [32]byte
	var oldest time.Time
	for digest, entry := range s.entries {
		if oldest.IsZero() || entry.expiresAt.Before(oldest) {
			oldestDigest = digest
			oldest = entry.expiresAt
		}
	}
	delete(s.entries, oldestDigest)
}

func cloneBoundary(boundary PageBoundary) PageBoundary {
	boundary.FamilyNameNormalized = cloneString(boundary.FamilyNameNormalized)
	boundary.GivenNameNormalized = cloneString(boundary.GivenNameNormalized)
	return boundary
}
