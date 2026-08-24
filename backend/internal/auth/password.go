// Package auth holds the whole authentication surface: local accounts,
// password hashing, and signed tokens — all on the standard library.
package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// Hash format: pbkdf2_sha256$<iterations>$<salt-b64>$<key-b64>
//
// PBKDF2-HMAC-SHA256 is in the standard library as of Go 1.24, so local
// accounts need no third-party crypto dependency. The iteration count is stored
// per-hash, which means it can be raised later without invalidating existing
// passwords.
const (
	hashPrefix     = "pbkdf2_sha256"
	defaultIters   = 600_000 // OWASP guidance for PBKDF2-HMAC-SHA256
	saltLen        = 16
	keyLen         = 32
	hashFieldCount = 4
)

// HashPassword derives a storable hash with a fresh random salt.
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("buat salt: %w", err)
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, defaultIters, keyLen)
	if err != nil {
		return "", fmt.Errorf("turunkan kunci: %w", err)
	}
	return fmt.Sprintf("%s$%d$%s$%s",
		hashPrefix, defaultIters,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword reports whether password matches the encoded hash. A malformed
// hash is a mismatch, never a panic — a corrupt row must not lock out the app
// or leak its shape through a distinct error.
func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != hashFieldCount || parts[0] != hashPrefix {
		return false
	}
	iters, err := strconv.Atoi(parts[1])
	if err != nil || iters <= 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, iters, len(want))
	if err != nil {
		return false
	}
	// Constant-time so a wrong password can't be narrowed down by timing.
	return subtle.ConstantTimeCompare(got, want) == 1
}
