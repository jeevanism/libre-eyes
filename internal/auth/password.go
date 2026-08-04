// Package auth implements authentication, sessions, authorization, and user context.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      = 19 * 1024
	argonIterations  = 2
	argonParallelism = 1
	argonSaltLength  = 16
	argonKeyLength   = 32
)

var errInvalidHash = errors.New("invalid password hash")

// PasswordManager creates and verifies the approved Argon2id password format.
type PasswordManager struct{}

// Hash creates a PHC-formatted Argon2id hash using cryptographic randomness.
func (PasswordManager) Hash(password string) (string, error) {
	if password == "" {
		return "", errors.New("password must not be empty")
	}
	salt := make([]byte, argonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	digest := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory,
		argonIterations,
		argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(digest),
	), nil
}

// Verify compares a password with a supported PHC-formatted hash.
func (PasswordManager) Verify(encoded, password string) (bool, error) {
	params, salt, expected, err := parseArgon2id(encoded)
	if err != nil {
		return false, err
	}
	actual := argon2.IDKey([]byte(password), salt, params.iterations, params.memory, params.parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

type argonParameters struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

func parseArgon2id(encoded string) (argonParameters, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		return argonParameters{}, nil, nil, errInvalidHash
	}
	parameters := strings.Split(parts[3], ",")
	if len(parameters) != 3 {
		return argonParameters{}, nil, nil, errInvalidHash
	}
	memory, err := parseParameter(parameters[0], "m")
	if err != nil || memory < argonMemory || memory > 1024*1024 {
		return argonParameters{}, nil, nil, errInvalidHash
	}
	iterations, err := parseParameter(parameters[1], "t")
	if err != nil || iterations < argonIterations || iterations > 20 {
		return argonParameters{}, nil, nil, errInvalidHash
	}
	parallelism, err := parseParameter(parameters[2], "p")
	if err != nil || parallelism < 1 || parallelism > 16 {
		return argonParameters{}, nil, nil, errInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < argonSaltLength || len(salt) > 64 {
		return argonParameters{}, nil, nil, errInvalidHash
	}
	digest, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(digest) < argonKeyLength || len(digest) > 64 {
		return argonParameters{}, nil, nil, errInvalidHash
	}

	return argonParameters{
		memory:      uint32(memory),
		iterations:  uint32(iterations),
		parallelism: uint8(parallelism),
	}, salt, digest, nil
}

func parseParameter(value, name string) (uint64, error) {
	prefix := name + "="
	if !strings.HasPrefix(value, prefix) {
		return 0, errInvalidHash
	}
	parsed, err := strconv.ParseUint(strings.TrimPrefix(value, prefix), 10, 32)
	if err != nil {
		return 0, errInvalidHash
	}
	return parsed, nil
}
