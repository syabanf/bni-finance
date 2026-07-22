package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/domain"
)

// A JWT is three base64url segments joined by dots, the third being an HMAC of
// the first two. That is small enough to implement directly, which keeps the
// dependency list at one driver — and leaves no room for the algorithm-
// confusion bugs that come from a library accepting whatever `alg` a token
// claims. Only HS256 is ever accepted here.

var (
	ErrTokenInvalid = errors.New("token tidak valid")
	ErrTokenExpired = errors.New("token sudah kedaluwarsa")
)

type header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// Claims is the token payload. Field names follow the JWT registered-claim
// conventions so the token stays inspectable with ordinary tooling.
type Claims struct {
	Subject  string          `json:"sub"`
	Email    string          `json:"email"`
	Name     string          `json:"name"`
	Role     domain.UserRole `json:"role"`
	IssuedAt int64           `json:"iat"`
	Expires  int64           `json:"exp"`
}

func (c Claims) AsAuthUser() domain.AuthUser {
	return domain.AuthUser{ID: c.Subject, Name: c.Name, Email: c.Email, Role: c.Role}
}

// Signer issues and verifies tokens with one shared secret.
type Signer struct {
	secret []byte
	ttl    time.Duration
}

// MinSecretLength keeps a throwaway secret from protecting a real deployment.
const MinSecretLength = 32

func NewSigner(secret string, ttl time.Duration) (*Signer, error) {
	if len(secret) < MinSecretLength {
		return nil, fmt.Errorf("JWT_SECRET minimal %d karakter", MinSecretLength)
	}
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	return &Signer{secret: []byte(secret), ttl: ttl}, nil
}

func (s *Signer) TTL() time.Duration { return s.ttl }

// Sign issues a token for a user, valid for the configured TTL.
func (s *Signer) Sign(u domain.User, now time.Time) (string, time.Time, error) {
	expires := now.Add(s.ttl)
	claims := Claims{
		Subject:  u.ID,
		Email:    u.Email,
		Name:     u.Name,
		Role:     u.Role,
		IssuedAt: now.Unix(),
		Expires:  expires.Unix(),
	}

	headerJSON, err := json.Marshal(header{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("encode header: %w", err)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("encode claims: %w", err)
	}

	signingInput := encodeSegment(headerJSON) + "." + encodeSegment(claimsJSON)
	return signingInput + "." + encodeSegment(s.mac(signingInput)), expires, nil
}

// Verify checks the signature and expiry, returning the claims on success.
func (s *Signer) Verify(token string, now time.Time) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrTokenInvalid
	}

	signingInput := parts[0] + "." + parts[1]
	gotMAC, err := decodeSegment(parts[2])
	if err != nil {
		return nil, ErrTokenInvalid
	}
	// Compare before parsing anything: never act on unverified claims.
	if subtle.ConstantTimeCompare(gotMAC, s.mac(signingInput)) != 1 {
		return nil, ErrTokenInvalid
	}

	headerJSON, err := decodeSegment(parts[0])
	if err != nil {
		return nil, ErrTokenInvalid
	}
	var h header
	if err := json.Unmarshal(headerJSON, &h); err != nil || h.Alg != "HS256" {
		return nil, ErrTokenInvalid
	}

	claimsJSON, err := decodeSegment(parts[1])
	if err != nil {
		return nil, ErrTokenInvalid
	}
	var c Claims
	if err := json.Unmarshal(claimsJSON, &c); err != nil {
		return nil, ErrTokenInvalid
	}
	if c.Subject == "" || !c.Role.Valid() {
		return nil, ErrTokenInvalid
	}
	if c.Expires <= now.Unix() {
		return nil, ErrTokenExpired
	}
	return &c, nil
}

func (s *Signer) mac(input string) []byte {
	m := hmac.New(sha256.New, s.secret)
	m.Write([]byte(input))
	return m.Sum(nil)
}

func encodeSegment(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func decodeSegment(s string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(s) }
