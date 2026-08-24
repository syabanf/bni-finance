package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
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
			ctx := context.WithValue(r.Context(), userKey, claims.AsAuthUser())
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

func bearerToken(r *http.Request) string {
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	const prefix = "Bearer "
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}
