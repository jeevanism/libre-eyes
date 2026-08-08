package episodes

import (
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewUUIDv4ProducesDatabaseCompatibleIdentifier(t *testing.T) {
	identifier, err := newUUIDv4()
	if err != nil {
		t.Fatalf("newUUIDv4() error = %v", err)
	}
	if !validPublicID(identifier) {
		t.Fatalf("newUUIDv4() = %q, want UUID", identifier)
	}
	if identifier[14] != '4' || !strings.ContainsRune("89ab", rune(identifier[19])) {
		t.Fatalf("newUUIDv4() = %q, want RFC 4122 version 4 variant", identifier)
	}
}

func TestDeniedEventForOnlySupportsApprovedPermissions(t *testing.T) {
	for _, permission := range []string{permissionRead, permissionCreate, permissionUpdate, permissionReopen} {
		if _, ok := deniedEventFor(permission); !ok {
			t.Fatalf("deniedEventFor(%q) was not approved", permission)
		}
	}
	if _, ok := deniedEventFor("episode.delete"); ok {
		t.Fatal("deniedEventFor accepted deferred permission")
	}
}

func TestValidPublicID(t *testing.T) {
	if !validPublicID("11111111-1111-4111-8111-111111111111") {
		t.Fatal("validPublicID rejected valid UUID")
	}
	for _, value := range []string{"", "not-a-uuid", "11111111-1111-4111-8111-11111111111z"} {
		if validPublicID(value) {
			t.Fatalf("validPublicID(%q) = true", value)
		}
	}
}

func TestCursorStoreBindsPrincipalAndExpiresWithItsClock(t *testing.T) {
	store := newCursorStore()
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	entry := cursorEntry{
		boundary: pageBoundary{internalID: 42}, userID: 1, sessionID: 2,
		institutionID: 3, siteID: 4, firmID: 5, contextVersion: 6,
		patientID: sha256.Sum256([]byte("11111111-1111-4111-8111-111111111111")), limit: 25,
	}
	token, err := store.put(entry)
	if err != nil {
		t.Fatalf("put() error = %v", err)
	}
	boundary, err := store.get(token, entry)
	if err != nil || boundary.internalID != 42 {
		t.Fatalf("get() = %#v, %v", boundary, err)
	}
	wrongPrincipal := entry
	wrongPrincipal.firmID++
	if _, err := store.get(token, wrongPrincipal); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("get(wrong principal) error = %v, want invalid request", err)
	}
	now = now.Add(cursorLifetime)
	if _, err := store.get(token, entry); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("get(expired) error = %v, want invalid request", err)
	}
}

func TestCursorStoreBindsEventListToEpisode(t *testing.T) {
	store := newCursorStore()
	entry := cursorEntry{
		boundary: pageBoundary{internalID: 42}, userID: 1, sessionID: 2,
		institutionID: 3, siteID: 4, firmID: 5, contextVersion: 6,
		episodeID: sha256.Sum256([]byte("11111111-1111-4111-8111-111111111111")), limit: 25,
	}
	token, err := store.put(entry)
	if err != nil {
		t.Fatalf("put() error = %v", err)
	}
	wrongEpisode := entry
	wrongEpisode.episodeID = sha256.Sum256([]byte("22222222-2222-4222-8222-222222222222"))
	if _, err := store.get(token, wrongEpisode); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("get(wrong episode) error = %v, want invalid request", err)
	}
}
