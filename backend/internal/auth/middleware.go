package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
	"github.com/syabanf/bni-finance/backend/internal/scope"
)

type ctxKey string

const userKey ctxKey = "authUser"

// UserFrom returns the signed-in user attached by RequireAuth.
func UserFrom(ctx context.Context) (domain.AuthUser, bool) {
	u, ok := ctx.Value(userKey).(domain.AuthUser)
	return u, ok
}

// RequireAuth rejects anything without a valid bearer token and puts the
// caller's identity in the request context.
//
// This is where authorisation lives, and it has to: the backend connects to
// Postgres as ONE trusted role, so the database sees a single identity and can
// enforce nothing per user. Every check that matters happens here or nowhere.
func RequireAuth(signer *Signer) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				httpx.Fail(w, httpx.Unauthorized("token tidak disertakan"))
				return
			}
			claims, err := signer.Verify(token, timeNow())
			if err != nil {
				if errors.Is(err, ErrTokenExpired) {
					httpx.Fail(w, httpx.Unauthorized("sesi sudah berakhir — masuk kembali"))
					return
				}
				httpx.Fail(w, httpx.Unauthorized("token tidak valid"))
				return
			}
			user := claims.AsAuthUser()
			ctx := context.WithValue(r.Context(), userKey, user)

			// Lingkup chapter dipasang DI SINI, satu kali, untuk seluruh
			// permintaan. Repository membacanya lewat scope.Chapter dan
			// menempelkannya pada query.
			//
			// Dipasang di sini dan bukan di tiap handler karena yang terlewat
			// tidak bisa ketahuan dari membaca handler-nya: query tanpa lingkup
			// tetap berjalan dan tetap mengembalikan data. Yang menyelamatkan
			// justru scope.Chapter yang gagal tertutup — tapi mengandalkan
			// jaring itu berarti setiap kelalaian berakhir sebagai layar kosong
			// yang membingungkan, bukan sebagai kode yang benar.
			if chapterID, terbatas := user.ChapterScope(); terbatas {
				ctx = scope.WithChapter(ctx, chapterID)
			} else {
				ctx = scope.WithoutLimit(ctx)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin wraps a handler so only administrators reach it. Mirrors the
// capability checks the UI already applies, but enforced server-side — the UI
// check is a convenience, this one is the actual boundary.
func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFrom(r.Context())
		if !ok {
			httpx.Fail(w, httpx.Unauthorized("token tidak disertakan"))
			return
		}
		if user.Role != domain.RoleAdmin {
			httpx.Fail(w, httpx.Forbidden("butuh akses admin"))
			return
		}
		next(w, r)
	}
}

// RequireWrite mengizinkan peran yang boleh mengubah data — admin secara
// nasional, ST di dalam chapternya sendiri.
//
// Pembatasan CHAPTER-nya tidak di sini, melainkan di repository lewat
// ChapterScope. Middleware hanya menyatakan "boleh menulis"; yang menyatakan
// "menulis apa" adalah query-nya. Memisahkan keduanya disengaja: penambahan
// endpoint baru tidak bisa lupa membatasi chapter, karena batas itu ikut pada
// query-nya, bukan pada daftar middleware yang harus diingat satu per satu.
func RequireWrite(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFrom(r.Context())
		if !ok {
			httpx.Fail(w, httpx.Unauthorized("token tidak disertakan"))
			return
		}
		if !user.Role.CanWrite() {
			httpx.Fail(w, httpx.Forbidden("peran ini hanya boleh membaca"))
			return
		}
		next(w, r)
	}
}

func bearerToken(r *http.Request) string {
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	const prefix = "Bearer "
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}
