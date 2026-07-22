package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/domain"
)

const secret = "rahasia-uji-yang-panjangnya-lebih-dari-32-karakter"

func testUser() domain.User {
	return domain.User{
		ID:    "1a2b3c4d-0000-0000-0000-000000000001",
		Email: "admin@example.com",
		Name:  "Admin Uji",
		Role:  domain.RoleAdmin,
	}
}

func TestSignerRejectsWeakSecret(t *testing.T) {
	if _, err := NewSigner("pendek", time.Hour); err == nil {
		t.Error("secret pendek harus ditolak")
	}
	if _, err := NewSigner(secret, time.Hour); err != nil {
		t.Errorf("secret yang cukup panjang ditolak: %v", err)
	}
}

func TestSignThenVerify(t *testing.T) {
	s, _ := NewSigner(secret, time.Hour)
	now := time.Unix(1_800_000_000, 0)

	token, expires, err := s.Sign(testUser(), now)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if !expires.Equal(now.Add(time.Hour)) {
		t.Errorf("kedaluwarsa harusnya +1 jam, dapat %v", expires)
	}

	claims, err := s.Verify(token, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.Subject != testUser().ID || claims.Role != domain.RoleAdmin {
		t.Errorf("claims tidak sesuai: %+v", claims)
	}
	if claims.AsAuthUser().Email != "admin@example.com" {
		t.Errorf("email hilang dari claims: %+v", claims)
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	s, _ := NewSigner(secret, time.Hour)
	now := time.Unix(1_800_000_000, 0)

	token, _, _ := s.Sign(testUser(), now)
	if _, err := s.Verify(token, now.Add(2*time.Hour)); err != ErrTokenExpired {
		t.Errorf("token kedaluwarsa harus ditolak dengan ErrTokenExpired, dapat %v", err)
	}
}

func TestVerifyRejectsTampering(t *testing.T) {
	s, _ := NewSigner(secret, time.Hour)
	now := time.Now()
	token, _, _ := s.Sign(testUser(), now)
	parts := strings.Split(token, ".")

	// Rewriting the payload to claim admin — the classic attack — must fail on
	// the signature, which covers header and payload together.
	forged := Claims{
		Subject: "penyusup", Email: "jahat@example.com", Name: "Jahat",
		Role: domain.RoleAdmin, IssuedAt: now.Unix(), Expires: now.Add(time.Hour).Unix(),
	}
	raw, _ := json.Marshal(forged)
	swapped := parts[0] + "." + base64.RawURLEncoding.EncodeToString(raw) + "." + parts[2]
	if _, err := s.Verify(swapped, now); err != ErrTokenInvalid {
		t.Error("payload yang ditukar harus ditolak")
	}

	// A token signed with a different key must not verify here.
	other, _ := NewSigner("rahasia-lain-yang-juga-lebih-dari-32-karakter", time.Hour)
	foreign, _, _ := other.Sign(testUser(), now)
	if _, err := s.Verify(foreign, now); err != ErrTokenInvalid {
		t.Error("token dari secret lain harus ditolak")
	}

	for _, bad := range []string{"", "a.b", "a.b.c.d", "bukan-token", parts[0] + "." + parts[1] + ".xxx"} {
		if _, err := s.Verify(bad, now); err == nil {
			t.Errorf("token cacat %q diterima", bad)
		}
	}
}

// A JWT library that honours the token's own `alg` can be tricked into
// accepting alg=none. This implementation only ever uses HS256, and this test
// pins that down.
func TestVerifyRejectsAlgNone(t *testing.T) {
	s, _ := NewSigner(secret, time.Hour)
	now := time.Now()

	head, _ := json.Marshal(map[string]string{"alg": "none", "typ": "JWT"})
	body, _ := json.Marshal(Claims{
		Subject: "penyusup", Role: domain.RoleAdmin, Expires: now.Add(time.Hour).Unix(),
	})
	unsigned := base64.RawURLEncoding.EncodeToString(head) + "." +
		base64.RawURLEncoding.EncodeToString(body) + "."

	if _, err := s.Verify(unsigned, now); err == nil {
		t.Fatal("token alg=none diterima")
	}
}

func TestVerifyRejectsUnknownRole(t *testing.T) {
	s, _ := NewSigner(secret, time.Hour)
	now := time.Now()

	u := testUser()
	u.Role = "superadmin" // bukan peran yang dikenal
	token, _, _ := s.Sign(u, now)

	if _, err := s.Verify(token, now); err != ErrTokenInvalid {
		t.Error("peran tak dikenal harus ditolak meski tanda tangannya sah")
	}
}
