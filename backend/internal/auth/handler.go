package auth

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

// timeNow is a package-level seam so tests can freeze the clock.
var timeNow = time.Now

type Handler struct {
	svc      *Service
	pembatas *Pembatas
	// baseURL adalah alamat aplikasi web, dipakai merakit tautan reset.
	//
	// Dari konfigurasi server, BUKAN dari header Host permintaan. Host bisa
	// dipalsukan siapa pun yang menembak origin langsung — dan tautan reset
	// yang dirakit dari host palsu akan mengirim token yang sah ke domain
	// penyerang, lewat email yang tampak resmi karena memang dari kita.
	baseURL string
}

// NewHandler membuat handler auth.
//
// baseURL adalah parameter, bukan pemanggilan berantai seperti
// NewHandler(svc).PakaiBaseURL(u), dan itu bukan sekadar selera: penjaga
// dokumentasi membaca router.go dengan go/ast untuk menentukan rute mana yang
// publik, dan rantai tambahan membuatnya kehilangan jejak — seluruh rute auth
// lalu dilaporkan "berada di balik autentikasi" padahal tidak. Ia gagal
// berisik, jadi arahnya aman; tapi bentuk panggilan yang lebih datar membuatnya
// tidak perlu gagal sama sekali.
func NewHandler(svc *Service, baseURL string) *Handler {
	return &Handler{svc: svc, pembatas: NewPembatas(), baseURL: baseURL}
}

// RegisterPublic wires the routes that must work WITHOUT a token — login being
// the obvious one, since you have no token until it succeeds.
func (h *Handler) RegisterPublic(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/auth/forgot-password", h.lupaKataSandi)
	mux.HandleFunc("POST /api/v1/auth/reset-password", h.resetKataSandi)
	mux.HandleFunc("GET /api/v1/auth/quick-login", h.quickLoginAccounts)
	mux.HandleFunc("POST /api/v1/auth/quick-login", h.quickLogin)
}

// quickLoginAccounts lists the allow-listed demo accounts. Public because the
// sign-in page must render before anyone has a token — and it exposes only
// name/email/role, never a credential. Returns 404 when the feature is off, so
// a production deployment gives no hint that it exists.
func (h *Handler) quickLoginAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.svc.QuickLoginAccounts(r.Context())
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"data": accounts})
}

func (h *Handler) quickLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	result, err := h.svc.QuickLogin(r.Context(), in.Email)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

// RegisterProtected wires everything that requires a signed-in caller.
func (h *Handler) RegisterProtected(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/auth/me", h.me)
	mux.HandleFunc("PATCH /api/v1/auth/me", h.updateProfile)
	mux.HandleFunc("PUT /api/v1/auth/password", h.changePassword)

	// User administration — admin only.
	mux.HandleFunc("GET /api/v1/users", RequireAdmin(h.listUsers))
	mux.HandleFunc("POST /api/v1/users", RequireAdmin(h.createUser))
	mux.HandleFunc("PATCH /api/v1/users/{id}/role", RequireAdmin(h.setRole))
	mux.HandleFunc("PUT /api/v1/users/{id}/password", RequireAdmin(h.resetPassword))
	mux.HandleFunc("DELETE /api/v1/users/{id}", RequireAdmin(h.deleteUser))
}

// lupaKataSandi mengirim tautan reset ke email yang terdaftar.
//
// Jawabannya SELALU 200 dengan pesan yang sama, ada atau tidak ada akunnya.
// Membedakannya mengubah formulir ini jadi alat pemeriksa keanggotaan: siapa
// pun bisa mencoba daftar alamat dan mengetahui mana yang punya akun di sini.
func (h *Handler) lupaKataSandi(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}

	// Dibatasi per email, memakai pembatas yang sama dengan login.
	//
	// Tanpa ini, formulir ini adalah tombol kirim-email gratis: seseorang bisa
	// menembaknya berulang kali dan membanjiri kotak masuk orang lain, memakai
	// kuota SMTP kita, sampai alamat pengirimnya ditandai spam oleh penyedia
	// email penerima. Kerusakannya menimpa orang yang tidak melakukan apa-apa.
	if sisa, terkunci := h.pembatas.Terkunci(in.Email); terkunci {
		detik := int(sisa.Seconds()) + 1
		w.Header().Set("Retry-After", strconv.Itoa(detik))
		httpx.Fail(w, httpx.NewError(http.StatusTooManyRequests, fmt.Sprintf(
			"terlalu banyak permintaan — coba lagi dalam %d menit", (detik+59)/60), nil))
		return
	}
	h.pembatas.Gagal(in.Email)

	if err := h.svc.MintaResetKataSandi(r.Context(), in.Email, h.baseURL); err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{
		"message": "Kalau email itu terdaftar, tautan reset sudah dikirim ke sana.",
	})
}

// resetKataSandi menukar token yang sah dengan kata sandi baru.
func (h *Handler) resetKataSandi(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	if err := h.svc.ResetKataSandi(r.Context(), in.Token, in.Password); err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{
		"message": "Kata sandi berhasil diubah. Silakan masuk dengan kata sandi baru.",
	})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var in domain.LoginInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}

	// Diperiksa SEBELUM kata sandinya diverifikasi. Memverifikasi lebih dulu
	// berarti setiap percobaan yang ditolak tetap membayar biaya PBKDF2 600.000
	// iterasi — pembatas yang justru menjadikan penebakan sebagai cara
	// menghabiskan CPU server.
	if sisa, terkunci := h.pembatas.Terkunci(in.Email); terkunci {
		detik := int(sisa.Seconds()) + 1
		w.Header().Set("Retry-After", strconv.Itoa(detik))
		// Dibulatkan KE ATAS, bukan ditambah satu. Versi pertama memakai
		// detik/60+1, sehingga tepat 900 detik dilaporkan sebagai "16 menit" —
		// menyuruh orang menunggu semenit lebih lama daripada perlu, tiap kali.
		httpx.Fail(w, httpx.NewError(http.StatusTooManyRequests, fmt.Sprintf(
			"terlalu banyak percobaan masuk — coba lagi dalam %d menit", (detik+59)/60), nil))
		return
	}

	result, err := h.svc.Login(r.Context(), in)
	if err != nil {
		// Hanya kredensial yang salah yang dihitung. Galat lain — basis data
		// mati, permintaan cacat — bukan penebakan, dan menghitungnya membuat
		// gangguan sesaat pada server ikut mengunci orang yang tidak melakukan
		// apa-apa.
		if httpx.StatusOf(err) == http.StatusUnauthorized {
			h.pembatas.Gagal(in.Email)
		}
		httpx.Fail(w, err)
		return
	}
	h.pembatas.Berhasil(in.Email)
	httpx.JSON(w, http.StatusOK, result)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserFrom(r.Context())
	// Re-read rather than echoing the token: a role changed after the token was
	// issued should be visible on the next request, not at the next sign-in.
	user, err := h.svc.Me(r.Context(), caller.ID)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (h *Handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserFrom(r.Context())
	var in domain.UpdateProfileInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	user, err := h.svc.UpdateProfile(r.Context(), caller.ID, in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserFrom(r.Context())
	var in domain.UpdatePasswordInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	if err := h.svc.ChangeOwnPassword(r.Context(), caller.ID, in); err != nil {
		httpx.Fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.svc.List(r.Context())
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.List(users, len(users), len(users), 0))
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var in domain.CreateUserInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	user, err := h.svc.Create(r.Context(), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, user)
}

func (h *Handler) setRole(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Role domain.UserRole `json:"role"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	user, err := h.svc.SetRole(r.Context(), r.PathValue("id"), in.Role)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserFrom(r.Context())
	target := r.PathValue("id")
	if target == caller.ID {
		httpx.Fail(w, httpx.BadRequest("untuk mengubah kata sandi sendiri gunakan PUT /auth/password"))
		return
	}
	var in domain.UpdatePasswordInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	if err := h.svc.ResetPassword(r.Context(), target, in); err != nil {
		httpx.Fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	caller, _ := UserFrom(r.Context())
	if err := h.svc.Delete(r.Context(), r.PathValue("id"), caller.ID); err != nil {
		httpx.Fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
