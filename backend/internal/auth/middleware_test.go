package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/scope"
)

// Lingkup yang dipasang middleware mengikuti chapter di token, untuk peran
// apa pun selain admin. User berchapter hanya melihat chapter itu; ST tanpa
// chapter melihat seluruh chapter.
func TestRequireAuthMemasangLingkupDariToken(t *testing.T) {
	signer, err := NewSigner("rahasia-uji-yang-cukup-panjang-untuk-hs256-0123456789", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	ch := "ch-a"
	kasus := []struct {
		nama     string
		user     domain.User
		terbatas bool
	}{
		{"user berchapter", domain.User{ID: "u1", Role: domain.RoleUser, ChapterID: &ch}, true},
		{"st nasional", domain.User{ID: "u2", Role: domain.RoleST}, false},
		{"mc berchapter", domain.User{ID: "u3", Role: domain.RoleMC, ChapterID: &ch}, true},
		{"admin", domain.User{ID: "u4", Role: domain.RoleAdmin}, false},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			token, _, err := signer.Sign(k.user, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			var lim scope.Limit
			h := RequireAuth(signer)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				lim = scope.Chapter(r.Context())
			}))
			req := httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			h.ServeHTTP(httptest.NewRecorder(), req)

			if lim.Buntu {
				t.Fatal("lingkup tidak terpasang sama sekali")
			}
			if lim.Terbatas != k.terbatas || (k.terbatas && lim.ChapterID != ch) {
				t.Errorf("lingkup = %+v, mau terbatas=%v", lim, k.terbatas)
			}
		})
	}
}
