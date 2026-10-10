package httpx

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type ctxKey string

const requestIDKey ctxKey = "requestID"

// Middleware is the standard decorator shape.
type Middleware func(http.Handler) http.Handler

// Chain applies middleware so the first listed runs outermost.
func Chain(h http.Handler, mw ...Middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// RequestID attaches an id to each request so logs can be correlated.
// The counter is atomic: handlers run concurrently, so a plain ++ is a data
// race (caught by the stress test under -race).
func RequestID(next http.Handler) http.Handler {
	var counter atomic.Uint64
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			n := counter.Add(1)
			id = strconv.FormatInt(time.Now().UnixNano(), 36) + "-" + strconv.FormatUint(n, 36)
		}
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Logger emits one structured line per request.
func Logger(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			log.Info("http",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", RequestIDFrom(r.Context()),
			)
		})
	}
}

// Recoverer turns a panic into a 500 instead of killing the connection.
func Recoverer(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic", "recover", rec, "path", r.URL.Path,
						"request_id", RequestIDFrom(r.Context()))
					JSON(w, http.StatusInternalServerError, errorBody{Error: "terjadi kesalahan pada server"})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// AllowedMethods is the CORS preflight answer. Exported so a test can assert it
// covers every method the router registers — a method missing here fails ONLY
// in a browser, never in curl or in a handler test, which is exactly how PUT
// went unnoticed.
const AllowedMethods = "GET, POST, PATCH, PUT, DELETE, OPTIONS"

// CORS allows the configured origins ("*" allows any).
func CORS(allowed []string) Middleware {
	allowAll := len(allowed) == 1 && allowed[0] == "*"
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			switch {
			case allowAll:
				w.Header().Set("Access-Control-Allow-Origin", "*")
			case origin != "" && contains(allowed, origin):
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
			}
			// Must list every method the router actually serves. PUT was missing
			// here, so the browser's preflight blocked all three PUT routes —
			// saving any app setting and changing a password silently failed
			// with a network error, while curl worked fine.
			w.Header().Set("Access-Control-Allow-Methods", AllowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-Id")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// APIKey guards routes with a bearer token when a key is configured. With no
// key set the middleware is a no-op, which keeps local development friction-free.
func APIKey(key string) Middleware {
	return func(next http.Handler) http.Handler {
		if key == "" {
			return next
		}
		want := []byte("Bearer " + key)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := []byte(strings.TrimSpace(r.Header.Get("Authorization")))
			// Constant-time compare so the key can't be probed byte-by-byte.
			if subtle.ConstantTimeCompare(got, want) != 1 {
				Fail(w, Unauthorized("API key tidak valid"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// RapikanPath meratakan garis miring ganda di path SEBELUM mux melihatnya.
//
// http.ServeMux membersihkan path sendiri dengan redirect: 301 untuk GET, 307
// untuk method lain. 307 memang menjaga method dan body bila kliennya patuh,
// tapi tidak semua klien webhook mengikuti redirect untuk POST, dan yang
// tidak mengikutinya menganggap callback-nya gagal lalu mengulang. Lapisan di
// depan server (reverse proxy, ingress) bisa pula menjawab 301 sendiri untuk
// path yang sama, dan 301 membuat klien mengubah POST menjadi GET sambil
// membuang body.
//
// Terjadi sungguhan: URL callback di dashboard Paper.id terdaftar dengan "//"
// dan setiap uji dari sana mendarat sebagai body kosong. Meratakan path di
// sini menghilangkan redirect dari sisi Go sama sekali, jadi permintaan yang
// sampai ke proses ini dilayani utuh apa pun perilaku kliennya. Yang terjadi
// di proxy di depan tidak bisa diperbaiki dari sini; URL di dashboard tetap
// harus satu garis miring.
func RapikanPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "//") {
			r2 := r.Clone(r.Context())
			r2.URL.Path = rapikan(r.URL.Path)
			if r2.URL.RawPath != "" {
				r2.URL.RawPath = rapikan(r.URL.RawPath)
			}
			r = r2
		}
		next.ServeHTTP(w, r)
	})
}

func rapikan(p string) string {
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	return p
}
