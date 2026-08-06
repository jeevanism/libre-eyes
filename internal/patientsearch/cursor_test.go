package patientsearch

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCursorIsOpaqueBoundAndReplayable(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	store := newCursorStore()
	store.now = func() time.Time { return now }
	family := "sensitive-family-name"
	entry := cursorEntry{
		boundary: PageBoundary{FamilyNameNormalized: &family, DateOfBirth: time.Date(1980, 1, 2, 0, 0, 0, 0, time.UTC), PublicID: "018f34f6-45f2-4a57-8ac0-358cea11ec62"},
		userID:   1, sessionID: 2, institutionID: 3, siteID: 4, firmID: 5,
		contextVersion: 6, criteriaDigest: [32]byte{7}, limit: 25,
		orderingVersion: cursorOrderingVersion, normalizationVersion: NameNormalizationVersion,
	}

	token, err := store.put(entry)
	if err != nil {
		t.Fatalf("put() error = %v", err)
	}
	if strings.Contains(token, family) || len(token) > 512 {
		t.Fatalf("cursor %q is not bounded opaque output", token)
	}
	for attempt := 0; attempt < 2; attempt++ {
		boundary, err := store.get(token, entry)
		if err != nil {
			t.Fatalf("get() replay %d error = %v", attempt, err)
		}
		if boundary.FamilyNameNormalized == nil || *boundary.FamilyNameNormalized != family {
			t.Fatalf("boundary = %#v, want copied family", boundary)
		}
	}

	wrongSession := entry
	wrongSession.sessionID++
	if _, err := store.get(token, wrongSession); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("wrong-session get() error = %v, want ErrInvalidRequest", err)
	}
	if _, err := store.get(token+"tampered", entry); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("tampered get() error = %v, want ErrInvalidRequest", err)
	}
	wrongCriteria := entry
	wrongCriteria.criteriaDigest[0]++
	if _, err := store.get(token, wrongCriteria); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("wrong-criteria get() error = %v, want ErrInvalidRequest", err)
	}

	now = now.Add(cursorLifetime)
	if _, err := store.get(token, entry); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expired get() error = %v, want ErrInvalidRequest", err)
	}
}

func TestPrincipalLimiterCombinesOperationsByUser(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	limiter := newPrincipalLimiter()
	limiter.now = func() time.Time { return now }

	for request := 1; request <= int(principalBurst); request++ {
		if allowed, _ := limiter.allow(17); !allowed {
			t.Fatalf("request %d denied before burst was exhausted", request)
		}
	}
	if allowed, retry := limiter.allow(17); allowed || retry != time.Second {
		t.Fatalf("exhausted request = (%v, %s), want (false, 1s)", allowed, retry)
	}
	if allowed, _ := limiter.allow(18); !allowed {
		t.Fatal("a different principal inherited another user's limit")
	}
	now = now.Add(time.Second)
	if allowed, _ := limiter.allow(17); !allowed {
		t.Fatal("one token was not replenished after one second")
	}
}

func TestFormatIdentifierPreservesLegacySpacingSemantics(t *testing.T) {
	rule := "xxx xxx xxxx"
	value, err := formatIdentifier(PrimaryIdentifierRecord{
		TypeID: 4, Label: "Hospital number", OriginalValue: "1234567890",
		DisplayPrefix: "[", DisplaySuffix: "]", SpacingRule: &rule,
	})
	if err != nil {
		t.Fatalf("formatIdentifier() error = %v", err)
	}
	if value != "[123 456 7890]" {
		t.Fatalf("value = %q, want %q", value, "[123 456 7890]")
	}

	short, err := formatIdentifier(PrimaryIdentifierRecord{
		TypeID: 4, Label: "Hospital number", OriginalValue: "123", SpacingRule: &rule,
	})
	if err != nil {
		t.Fatalf("short formatIdentifier() error = %v", err)
	}
	if short != "123" {
		t.Fatalf("short value = %q, want %q", short, "123")
	}
}

func TestFutureDateUsesLocalCalendarDay(t *testing.T) {
	location := time.FixedZone("synthetic-east", 12*60*60)
	now := time.Date(2026, 8, 6, 0, 30, 0, 0, location)
	if isFutureDate(time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC), now) {
		t.Fatal("the current local calendar date was classified as future")
	}
	if !isFutureDate(time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC), now) {
		t.Fatal("the next local calendar date was not classified as future")
	}
}

func TestParseDateOfBirthRejectsImpossibleAndNonISOValues(t *testing.T) {
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	tests := []string{"2023-02-30", "2023-13-01", "2023-01-32", "30/02/2023", ""}
	for _, value := range tests {
		if _, err := parseDateOfBirth(value, now); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("parseDateOfBirth(%q) error = %v, want ErrInvalidRequest", value, err)
		}
	}
	if date, err := parseDateOfBirth("2000-02-29", now); err != nil || date.Format(dateLayout) != "2000-02-29" {
		t.Fatalf("valid leap date = %s, %v", date, err)
	}
}
