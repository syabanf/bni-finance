package auth

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

// Quick login hands out a session with no password, so its boundaries are the
// whole feature. These tests pin them.

type fakeUsers struct {
	byEmail map[string]domain.User
}

// Returns bare httpx.ErrNotFound, matching what the real Repository does —
// the callers use errors.Is against that sentinel.
func (f *fakeUsers) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := f.byEmail[strings.ToLower(email)]
	if !ok {
		return nil, httpx.ErrNotFound
	}
	return &u, nil
}

func (f *fakeUsers) GetByID(context.Context, string) (*domain.User, error) {
	return nil, httpx.ErrNotFound
}
func (f *fakeUsers) List(context.Context) ([]domain.User, error) { return nil, nil }
func (f *fakeUsers) Create(context.Context, string, string, string, domain.UserRole) (*domain.User, error) {
	return nil, nil
}
func (f *fakeUsers) UpdateName(context.Context, string, string) (*domain.User, error) {
	return nil, nil
}
func (f *fakeUsers) UpdateRole(context.Context, string, domain.UserRole) (*domain.User, error) {
	return nil, nil
}
func (f *fakeUsers) UpdatePasswordHash(context.Context, string, string) error { return nil }
func (f *fakeUsers) Delete(context.Context, string) error                     { return nil }
func (f *fakeUsers) CountAdmins(context.Context) (int, error)                 { return 1, nil }

func seededUsers() *fakeUsers {
	return &fakeUsers{byEmail: map[string]domain.User{
		"admin@contoh.local": {ID: "u1", Email: "admin@contoh.local", Name: "Admin", Role: domain.RoleAdmin},
		"staf@contoh.local":  {ID: "u2", Email: "staf@contoh.local", Name: "Staf", Role: domain.RoleUser},
		"bos@contoh.local":   {ID: "u3", Email: "bos@contoh.local", Name: "Bos", Role: domain.RoleAdmin},
	}}
}

func quickService(t *testing.T, allow ...string) *Service {
	t.Helper()
	signer, err := NewSigner("rahasia-uji-yang-cukup-panjang-sekali", time.Hour)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	return NewService(seededUsers(), signer, allow...)
}

// Off by default. Anything else would mean a deployment that never set the
// variable still exposes passwordless sign-in.
func TestQuickLoginDisabledByDefault(t *testing.T) {
	svc := quickService(t)

	if svc.QuickLoginEnabled() {
		t.Fatal("quick login harus mati bila tidak dikonfigurasi")
	}
	if _, err := svc.QuickLoginAccounts(context.Background()); statusOf(err) != 404 {
		t.Errorf("daftar akun harus 404 saat mati, dapat %v", err)
	}
	// 404, not 403: a production deployment should give no hint the route exists.
	_, err := svc.QuickLogin(context.Background(), "admin@contoh.local")
	if statusOf(err) != 404 {
		t.Errorf("quick login harus 404 saat mati, dapat %v", err)
	}
}

// The allow-list is the entire guard. An account left off it must stay
// password-protected even while the feature is on.
func TestQuickLoginRefusesAccountsOutsideAllowList(t *testing.T) {
	svc := quickService(t, "admin@contoh.local")

	if _, err := svc.QuickLogin(context.Background(), "bos@contoh.local"); statusOf(err) != 403 {
		t.Fatalf("akun di luar daftar harus 403, dapat %v", err)
	}
}

func TestQuickLoginIssuesTokenForAllowedAccount(t *testing.T) {
	svc := quickService(t, "admin@contoh.local", "staf@contoh.local")

	res, err := svc.QuickLogin(context.Background(), "staf@contoh.local")
	if err != nil {
		t.Fatalf("QuickLogin: %v", err)
	}
	if res.Token == "" || res.User.ID == "" {
		t.Fatalf("hasil tidak lengkap: %+v", res)
	}
	if res.User.Email != "staf@contoh.local" || res.User.Role != domain.RoleUser {
		t.Errorf("pengguna salah: %+v", res.User)
	}
	// The token must be a real one the rest of the API accepts, not a marker.
	claims, err := svc.signer.Verify(res.Token, time.Now())
	if err != nil {
		t.Fatalf("token hasil quick login tidak valid: %v", err)
	}
	if claims.Role != domain.RoleUser {
		t.Errorf("peran di token salah: %s", claims.Role)
	}
}

// Case and stray whitespace in the env var must not silently disable an entry —
// that would fail open into "nothing works" or, worse, mismatch on comparison.
func TestQuickLoginNormalisesConfiguredEmails(t *testing.T) {
	svc := quickService(t, "  ADMIN@Contoh.Local  ", "")

	if len(svc.quickLogin) != 1 {
		t.Fatalf("entri kosong harus dibuang: %#v", svc.quickLogin)
	}
	if _, err := svc.QuickLogin(context.Background(), "Admin@Contoh.LOCAL"); err != nil {
		t.Errorf("pencocokan email harus abaikan huruf besar/kecil: %v", err)
	}
}

// The listing feeds a public page, so it must expose no credential material.
func TestQuickLoginAccountsExposeNoSecrets(t *testing.T) {
	svc := quickService(t, "admin@contoh.local", "staf@contoh.local")

	accounts, err := svc.QuickLoginAccounts(context.Background())
	if err != nil {
		t.Fatalf("QuickLoginAccounts: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("harus 2 akun, dapat %d", len(accounts))
	}
	// domain.AuthUser has no password field at all; assert the type stays that
	// way by checking the JSON the handler would send.
	raw := mustJSON(t, accounts)
	for _, forbidden := range []string{"password", "hash", "Hash"} {
		if strings.Contains(raw, forbidden) {
			t.Errorf("daftar akun membocorkan %q: %s", forbidden, raw)
		}
	}
}

// A configured account that was never seeded is a misconfiguration. It must not
// break the sign-in page — the other buttons still work.
func TestQuickLoginSkipsMissingAccountsInListing(t *testing.T) {
	svc := quickService(t, "admin@contoh.local", "tidak-ada@contoh.local")

	accounts, err := svc.QuickLoginAccounts(context.Background())
	if err != nil {
		t.Fatalf("akun yang belum ada tidak boleh menggagalkan daftar: %v", err)
	}
	if len(accounts) != 1 || accounts[0].Email != "admin@contoh.local" {
		t.Errorf("daftar salah: %+v", accounts)
	}
	// But signing in as it must still fail — the listing skipping it is not
	// permission to mint a token for an account that doesn't exist.
	if _, err := svc.QuickLogin(context.Background(), "tidak-ada@contoh.local"); err == nil {
		t.Error("akun yang belum ada tidak boleh bisa masuk")
	}
}

func statusOf(err error) int {
	var he *httpx.Error
	if !errors.As(err, &he) {
		return 0
	}
	return he.Status
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(raw)
}
