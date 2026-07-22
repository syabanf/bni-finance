package api_test

import (
	"testing"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/auth"
	"github.com/syabanf/bni-finance/backend/internal/domain"
)

// Every /api route now sits behind a bearer token, so the test servers need a
// signer and a couple of ready-made tokens.

const testSecret = "rahasia-uji-yang-panjangnya-lebih-dari-32-karakter"

func testSigner(t *testing.T) *auth.Signer {
	t.Helper()
	s, err := auth.NewSigner(testSecret, time.Hour)
	if err != nil {
		t.Fatalf("buat signer: %v", err)
	}
	return s
}

func tokenFor(t *testing.T, signer *auth.Signer, role domain.UserRole) string {
	t.Helper()
	user := domain.User{
		ID:    "11111111-1111-1111-1111-111111111111",
		Email: string(role) + "@example.com",
		Name:  "Penguji " + string(role),
		Role:  role,
	}
	if role == domain.RoleUser {
		user.ID = "22222222-2222-2222-2222-222222222222"
	}
	token, _, err := signer.Sign(user, time.Now())
	if err != nil {
		t.Fatalf("tanda tangani token: %v", err)
	}
	return token
}
