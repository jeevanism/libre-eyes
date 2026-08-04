package auth

import "testing"

func TestTokenAndCSRFDerivation(t *testing.T) {
	token, err := newToken()
	if err != nil {
		t.Fatalf("newToken() error = %v", err)
	}
	if len(token) < 32 {
		t.Fatalf("token length = %d, want at least 32", len(token))
	}

	key := []byte("01234567890123456789012345678901")
	csrf := csrfToken(key, token)
	digest := tokenDigest(csrf)
	if !verifyDigest(digest[:], csrf) {
		t.Fatal("verifyDigest() = false, want true")
	}
	if verifyDigest(digest[:], csrf+"x") {
		t.Fatal("verifyDigest() modified token = true, want false")
	}
}
