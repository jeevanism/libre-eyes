package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
)

const tokenBytes = 32

func newToken() (string, error) {
	value := make([]byte, tokenBytes)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func tokenDigest(value string) [sha256.Size]byte {
	return sha256.Sum256([]byte(value))
}

func csrfToken(key []byte, sessionToken string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("libreeyes.csrf.v1\x00"))
	_, _ = mac.Write([]byte(sessionToken))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func verifyDigest(expected []byte, value string) bool {
	actual := tokenDigest(value)
	return subtle.ConstantTimeCompare(expected, actual[:]) == 1
}
