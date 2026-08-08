package episodes

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
	cursorLifetime       = 15 * time.Minute
	maximumCursorEntries = 16384
)

type pageBoundary struct {
	startedAt  *time.Time
	occurredAt *time.Time
	internalID int64
}

type cursorEntry struct {
	boundary       pageBoundary
	userID         int64
	sessionID      int64
	institutionID  int64
	siteID         int64
	firmID         int64
	contextVersion int64
	patientID      [32]byte
	episodeID      [32]byte
	limit          int
	expiresAt      time.Time
}

type cursorStore struct {
	mu      sync.Mutex
	entries map[[32]byte]cursorEntry
	now     func() time.Time
}

// newCursorStore creates a process-local store. Production horizontal scaling
// requires an approved shared or stateless cursor design before use.
func newCursorStore() *cursorStore {
	return &cursorStore{entries: make(map[[32]byte]cursorEntry), now: time.Now}
}

func (s *cursorStore) put(entry cursorEntry) (string, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", errors.New("generate episode cursor")
	}
	token := base64.RawURLEncoding.EncodeToString(random)
	digest := sha256.Sum256([]byte(token))

	s.mu.Lock()
	defer s.mu.Unlock()
	s.prune(s.now().UTC())
	if len(s.entries) >= maximumCursorEntries {
		s.evictOldest()
	}
	entry.boundary = cloneBoundary(entry.boundary)
	entry.expiresAt = s.now().UTC().Add(cursorLifetime)
	s.entries[digest] = entry
	return token, nil
}

func (s *cursorStore) get(token string, expected cursorEntry) (pageBoundary, error) {
	if token == "" || len(token) > 512 {
		return pageBoundary{}, ErrInvalidRequest
	}
	digest := sha256.Sum256([]byte(token))
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()
	s.prune(now)
	entry, ok := s.entries[digest]
	if !ok || !now.Before(entry.expiresAt) || !cursorMatches(entry, expected) {
		return pageBoundary{}, ErrInvalidRequest
	}
	return cloneBoundary(entry.boundary), nil
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
			oldestDigest, oldest = digest, entry.expiresAt
		}
	}
	delete(s.entries, oldestDigest)
}

func cursorMatches(actual, expected cursorEntry) bool {
	return actual.userID == expected.userID &&
		actual.sessionID == expected.sessionID &&
		actual.institutionID == expected.institutionID &&
		actual.siteID == expected.siteID &&
		actual.firmID == expected.firmID &&
		actual.contextVersion == expected.contextVersion &&
		actual.limit == expected.limit &&
		subtle.ConstantTimeCompare(actual.patientID[:], expected.patientID[:]) == 1 &&
		subtle.ConstantTimeCompare(actual.episodeID[:], expected.episodeID[:]) == 1
}

func cloneBoundary(boundary pageBoundary) pageBoundary {
	if boundary.startedAt != nil {
		value := *boundary.startedAt
		boundary.startedAt = &value
	}
	if boundary.occurredAt != nil {
		value := *boundary.occurredAt
		boundary.occurredAt = &value
	}
	return boundary
}
