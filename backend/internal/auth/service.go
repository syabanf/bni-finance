package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/mailer"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type Store interface {
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
	Create(ctx context.Context, email, passwordHash, name string, role domain.UserRole, chapterID *string) (*domain.User, error)
	UpdateName(ctx context.Context, id, name string) (*domain.User, error)
	UpdateRole(ctx context.Context, id string, role domain.UserRole) (*domain.User, error)
	// Guarded: pemeriksaan admin-terakhir dan penulisannya terjadi dalam satu
	// transaksi. Memisahkannya adalah balapan yang pernah menyisakan nol admin.
	UpdateRoleGuarded(ctx context.Context, id string, role domain.UserRole) (*domain.User, error)
	DeleteGuarded(ctx context.Context, id string) error
	UpdatePasswordHash(ctx context.Context, id, hash string) error
	Delete(ctx context.Context, id string) error
	CountAdmins(ctx context.Context) (int, error)

	// --- token reset kata sandi ---
	SimpanTokenReset(ctx context.Context, userID, hash string, berlaku time.Time) error
	// AmbilTokenReset mengembalikan pemilik token yang MASIH BERLAKU dan belum
	// dipakai. Kedaluwarsa dan sudah-dipakai sama-sama dijawab ErrNotFound —
	// membedakannya di respons akan memberi tahu penyerang bahwa tokennya
	// pernah ada.
	AmbilTokenReset(ctx context.Context, hash string, sekarang time.Time) (userID string, err error)
	// PakaiTokenReset menandai token terpakai DAN menulis kata sandi baru dalam
	// satu transaksi. Dipisah, sebuah kegagalan di antaranya bisa meninggalkan
	// token yang sudah dipakai tapi kata sandinya belum berubah — atau token
	// yang masih bisa dipakai ulang setelah kata sandinya berubah.
	PakaiTokenReset(ctx context.Context, hash, userID, passwordHash string, sekarang time.Time) error
	// HapusTokenResetLama membuang token kedaluwarsa; dipanggil berkala.
	HapusTokenResetLama(ctx context.Context, sebelum time.Time) (int64, error)

	// --- OTP login ---
	SimpanOTP(ctx context.Context, userID, hash string, berlaku time.Time) error
	// PakaiOTP memeriksa kode DAN menaikkan hitungan percobaannya dalam satu
	// transaksi. Dipisah, seseorang bisa menembakkan tebakan secara paralel dan
	// seluruhnya lolos pemeriksaan batas sebelum satu pun hitungan tersimpan.
	//
	// Mengembalikan sisaPercobaan agar pemanggilnya bisa memberi tahu — "kode
	// salah" tanpa menyebut sisa percobaan membuat orang mengetik ulang sampai
	// terkunci tanpa peringatan.
	PakaiOTP(ctx context.Context, userID, hash string, sekarang time.Time, maks int) (sisaPercobaan int, err error)
	HapusOTPLama(ctx context.Context, sebelum time.Time) (int64, error)
	// OTPAktif membaca sakelar login_otp_enabled dari app_settings.
	OTPAktif(ctx context.Context) bool
}

var _ Store = (*Repository)(nil)

type Service struct {
	repo   Store
	signer *Signer
	now    func() time.Time

	// quickLogin is the lower-cased allow-list for passwordless sign-in.
	// Empty means the feature is off — see the quick login section below.
	quickLogin []string

	// mail boleh nil: aplikasinya tetap berjalan tanpa pengirim email, hanya fitur yang
	// membutuhkannya yang menjawab 503 dengan pesan yang jelas.
	mail PengirimEmail
}

// PakaiPengirimEmail memasang pengirim email.
//
// Terpisah dari NewService, bukan parameter tambahan: seluruh pemanggil yang
// sudah ada — termasuk belasan tes — tidak perlu diubah hanya untuk menyatakan
// "tidak pakai email".
func (s *Service) PakaiPengirimEmail(m PengirimEmail) *Service {
	s.mail = m
	return s
}

func NewService(repo Store, signer *Signer, quickLogin ...string) *Service {
	allowed := make([]string, 0, len(quickLogin))
	for _, e := range quickLogin {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" {
			allowed = append(allowed, e)
		}
	}
	return &Service{repo: repo, signer: signer, now: time.Now, quickLogin: allowed}
}

// invalidCredentials is deliberately the same message for an unknown email and
// a wrong password — telling them apart would enumerate accounts.
func invalidCredentials() error {
	return httpx.Unauthorized("email atau kata sandi salah")
}

func (s *Service) Login(ctx context.Context, in domain.LoginInput) (*domain.LoginResult, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}

	user, err := s.repo.GetByEmail(ctx, in.Email)
	if err != nil {
		if errors.Is(err, httpx.ErrNotFound) {
			// Still hash once, so a missing account and a wrong password take
			// comparable time and can't be told apart by the clock.
			_ = VerifyPassword(dummyHash, in.Password)
			return nil, invalidCredentials()
		}
		return nil, err
	}
	if !VerifyPassword(user.PasswordHash, in.Password) {
		return nil, invalidCredentials()
	}

	// OTP HANYA BERLAKU BILA EMAIL BENAR-BENAR BISA DIKIRIM.
	//
	// Sakelarnya menyala tapi SMTP mati berarti tidak seorang pun bisa masuk —
	// termasuk admin yang harus mematikan sakelarnya. Lapisan keamanan yang
	// bisa mengunci seluruh orang di luar aplikasinya, tanpa jalan kembali,
	// lebih berbahaya daripada ketiadaannya. Jadi syaratnya dua, bukan satu.
	if s.mail != nil && s.mail.Siap() && s.repo.OTPAktif(ctx) {
		if err := s.kirimOTP(ctx, user); err != nil {
			return nil, err
		}
		// Token TIDAK diterbitkan di sini. Inilah inti OTP: kata sandi yang
		// benar saja belum cukup.
		return &domain.LoginResult{ButuhOTP: true, User: user.AsAuthUser()}, nil
	}

	token, expires, err := s.signer.Sign(*user, s.now())
	if err != nil {
		return nil, err
	}
	return &domain.LoginResult{Token: token, ExpiresAt: expires, User: user.AsAuthUser()}, nil
}

// kirimOTP membangkitkan kode, menyimpan hash-nya, lalu mengirimkannya.
func (s *Service) kirimOTP(ctx context.Context, user *domain.User) error {
	kode, hash, berlaku, err := BuatOTP(time.Now())
	if err != nil {
		return err
	}
	if err := s.repo.SimpanOTP(ctx, user.ID, hash, berlaku); err != nil {
		return err
	}
	menit := int(UmurOTP.Minutes())
	return s.mail.Kirim(ctx, mailer.Pesan{
		Ke:     user.Email,
		Subjek: "Kode masuk " + kode + " — BNI Finance Hub",
		Teks: "Halo " + user.Name + ",\r\n\r\n" +
			"Kode masuk Anda: " + kode + "\r\n\r\n" +
			"Berlaku " + strconv.Itoa(menit) + " menit dan hanya bisa dipakai sekali.\r\n\r\n" +
			"Kalau bukan Anda yang mencoba masuk, abaikan email ini dan " +
			"segera ganti kata sandi Anda — seseorang mengetahuinya.\r\n",
		HTML: `<div style="font:15px/1.6 -apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;color:#1a1a1a">` +
			`<p>Halo ` + htmlAman(user.Name) + `,</p>` +
			`<p>Kode masuk Anda:</p>` +
			`<p style="font:700 32px/1 ui-monospace,SFMono-Regular,Menlo,monospace;` +
			`letter-spacing:8px;margin:20px 0;color:#c8102e">` + htmlAman(kode) + `</p>` +
			`<p style="color:#666;font-size:13px">Berlaku ` + strconv.Itoa(menit) +
			` menit dan hanya bisa dipakai sekali.</p>` +
			`<p style="color:#666;font-size:13px">Kalau bukan Anda yang mencoba masuk, abaikan email ini ` +
			`dan segera ganti kata sandi Anda — seseorang mengetahuinya.</p></div>`,
	})
}

// VerifikasiOTP menukar kode yang benar dengan token.
func (s *Service) VerifikasiOTP(ctx context.Context, email, kode string) (*domain.LoginResult, error) {
	user, err := s.repo.GetByEmail(ctx, strings.TrimSpace(email))
	if err != nil || user == nil {
		// Sama seperti login: email tak dikenal dan kode salah tidak dibedakan.
		return nil, httpx.BadRequest("kode tidak cocok atau sudah kedaluwarsa")
	}

	sisa, err := s.repo.PakaiOTP(ctx, user.ID, HashToken(strings.TrimSpace(kode)), time.Now(), MaksPercobaanOTP)
	if err != nil {
		if sisa > 0 {
			// Sisa percobaan DISEBUTKAN. Tanpa itu orang mengetik ulang sampai
			// terkunci tanpa peringatan, lalu meminta kode baru berulang kali —
			// yang justru menghasilkan lebih banyak email dan lebih banyak kode
			// hidup sekaligus.
			return nil, httpx.BadRequest(fmt.Sprintf(
				"kode tidak cocok — sisa %d percobaan", sisa))
		}
		return nil, httpx.BadRequest("kode tidak cocok atau sudah kedaluwarsa — minta kode baru")
	}

	token, expires, err := s.signer.Sign(*user, s.now())
	if err != nil {
		return nil, err
	}
	return &domain.LoginResult{Token: token, ExpiresAt: expires, User: user.AsAuthUser()}, nil
}

// dummyHash is a real, valid hash of an unusable password. Verifying against it
// costs the same as a genuine check.
var dummyHash = mustHash("kata-sandi-yang-tidak-akan-pernah-cocok")

func mustHash(pw string) string {
	h, err := HashPassword(pw)
	if err != nil {
		// Only reachable if the system entropy source fails at init.
		panic("auth: gagal menyiapkan hash pembanding: " + err.Error())
	}
	return h
}

func (s *Service) Me(ctx context.Context, id string) (*domain.AuthUser, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	out := user.AsAuthUser()
	return &out, nil
}

func (s *Service) UpdateProfile(ctx context.Context, id string, in domain.UpdateProfileInput) (*domain.AuthUser, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	if in.Name == nil {
		return s.Me(ctx, id)
	}
	user, err := s.repo.UpdateName(ctx, id, strings.TrimSpace(*in.Name))
	if err != nil {
		return nil, err
	}
	out := user.AsAuthUser()
	return &out, nil
}

// ChangeOwnPassword requires the current password — otherwise a stolen token
// would be enough to take over the account permanently.
func (s *Service) ChangeOwnPassword(ctx context.Context, id string, in domain.UpdatePasswordInput) error {
	if err := in.Validate(); err != nil {
		return httpx.BadRequest(err.Error())
	}
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !VerifyPassword(user.PasswordHash, in.CurrentPassword) {
		return httpx.Unauthorized("kata sandi saat ini salah")
	}
	hash, err := HashPassword(in.NewPassword)
	if err != nil {
		return err
	}
	return s.repo.UpdatePasswordHash(ctx, id, hash)
}

// ResetPassword is the admin path: no current password, but it cannot be used
// on yourself — that's what ChangeOwnPassword is for.
func (s *Service) ResetPassword(ctx context.Context, targetID string, in domain.UpdatePasswordInput) error {
	if err := in.Validate(); err != nil {
		return httpx.BadRequest(err.Error())
	}
	if _, err := s.repo.GetByID(ctx, targetID); err != nil {
		return err
	}
	hash, err := HashPassword(in.NewPassword)
	if err != nil {
		return err
	}
	return s.repo.UpdatePasswordHash(ctx, targetID, hash)
}

func (s *Service) List(ctx context.Context) ([]domain.AuthUser, error) {
	users, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AuthUser, len(users))
	for i, u := range users {
		out[i] = u.AsAuthUser()
	}
	return out, nil
}

func (s *Service) Create(ctx context.Context, in domain.CreateUserInput) (*domain.AuthUser, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	role := domain.RoleUser
	if in.Role != nil {
		role = *in.Role
	}
	hash, err := HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	user, err := s.repo.Create(ctx, in.Email, hash, strings.TrimSpace(in.Name), role, in.ChapterID)
	if err != nil {
		return nil, err
	}
	out := user.AsAuthUser()
	return &out, nil
}

func (s *Service) SetRole(ctx context.Context, id string, role domain.UserRole) (*domain.AuthUser, error) {
	if !role.Valid() {
		return nil, httpx.BadRequest("role harus 'admin' atau 'user'")
	}
	user, err := s.repo.UpdateRoleGuarded(ctx, id, role)
	if err != nil {
		return nil, lastAdminAsConflict(err)
	}
	out := user.AsAuthUser()
	return &out, nil
}

func (s *Service) Delete(ctx context.Context, id, actorID string) error {
	if id == actorID {
		return httpx.Conflict("tidak bisa menghapus akun sendiri")
	}
	return lastAdminAsConflict(s.repo.DeleteGuarded(ctx, id))
}

// lastAdminAsConflict memetakan sentinel repository ke 409 dengan pesan yang
// bisa ditindak operator.
func lastAdminAsConflict(err error) error {
	if errors.Is(err, ErrLastAdmin) {
		return httpx.Conflict("ini satu-satunya admin — angkat admin lain lebih dulu")
	}
	return err
}

// EnsureSeedAdmin creates the first administrator when the table is empty, so a
// fresh database is reachable without hand-writing a password hash. It is a
// no-op once any user exists.
func (s *Service) EnsureSeedAdmin(ctx context.Context, email, password, name string) (created bool, err error) {
	users, err := s.repo.List(ctx)
	if err != nil {
		return false, err
	}
	if len(users) > 0 {
		return false, nil
	}
	in := domain.CreateUserInput{Email: email, Password: password, Name: name}
	admin := domain.RoleAdmin
	in.Role = &admin
	if _, err := s.Create(ctx, in); err != nil {
		return false, err
	}
	return true, nil
}

// --- quick login -------------------------------------------------------------

// Quick login signs a caller in WITHOUT a password. It exists so demos and
// local development don't retype seeded credentials on every reload.
//
// The obvious alternative — shipping the password to the browser through a
// VITE_* variable — bakes it into the public JS bundle, where anyone who opens
// devtools can read it. Here the credential never leaves the server.
//
// The guard is an explicit allow-list of emails, not a boolean. A boolean would
// mean that switching the feature on in production turns EVERY account into a
// passwordless one; naming the accounts makes that impossible by construction.
// An empty list disables the feature outright, which is the default.

// QuickLoginEnabled reports whether any account was allow-listed.
func (s *Service) QuickLoginEnabled() bool { return len(s.quickLogin) > 0 }

// QuickLoginAccounts lists the allow-listed accounts that actually exist, so
// the sign-in page can render one button each. Never returns password hashes.
func (s *Service) QuickLoginAccounts(ctx context.Context) ([]domain.AuthUser, error) {
	if !s.QuickLoginEnabled() {
		return nil, httpx.NotFound("quick login tidak aktif")
	}
	out := make([]domain.AuthUser, 0, len(s.quickLogin))
	for _, email := range s.quickLogin {
		user, err := s.repo.GetByEmail(ctx, email)
		if err != nil {
			// A configured account that was never seeded is a misconfiguration,
			// not a request failure — skip it rather than breaking the page.
			if errors.Is(err, httpx.ErrNotFound) {
				continue
			}
			return nil, err
		}
		out = append(out, user.AsAuthUser())
	}
	return out, nil
}

// QuickLogin issues a token for an allow-listed account.
func (s *Service) QuickLogin(ctx context.Context, email string) (*domain.LoginResult, error) {
	if !s.QuickLoginEnabled() {
		return nil, httpx.NotFound("quick login tidak aktif")
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if !s.isQuickLoginAllowed(email) {
		// Deliberately not "unknown account": the caller learns only that this
		// email is not on the list, never whether it exists.
		return nil, httpx.Forbidden("akun ini tidak terdaftar untuk quick login")
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, httpx.ErrNotFound) {
			return nil, httpx.NotFound("akun quick login belum ada di database")
		}
		return nil, err
	}

	token, expires, err := s.signer.Sign(*user, s.now())
	if err != nil {
		return nil, err
	}
	return &domain.LoginResult{Token: token, ExpiresAt: expires, User: user.AsAuthUser()}, nil
}

func (s *Service) isQuickLoginAllowed(email string) bool {
	for _, allowed := range s.quickLogin {
		if allowed == email {
			return true
		}
	}
	return false
}

// --- reset kata sandi lewat email --------------------------------------------

// PengirimEmail adalah bagian mailer yang dibutuhkan service ini.
//
// Antarmuka sempit, bukan *mailer.Mailer langsung: paket auth jadi tidak perlu
// tahu apa pun soal cara pengirimannya, dan tesnya tidak perlu jaringan.
type PengirimEmail interface {
	Siap() bool
	Kirim(ctx context.Context, p mailer.Pesan) error
}

// MintaResetKataSandi membuat token, menyimpan hash-nya, dan mengirim tautannya.
//
// SELALU MENGEMBALIKAN nil UNTUK EMAIL YANG TIDAK DIKENAL.
//
// Membalas "email tidak terdaftar" mengubah formulir ini menjadi alat pemeriksa
// keanggotaan: siapa pun bisa mencoba daftar alamat dan mengetahui mana yang
// punya akun di sini. Yang berhasil dan yang tidak dijawab persis sama, dan
// pemanggilnya tidak diberi cara untuk membedakannya.
func (s *Service) MintaResetKataSandi(ctx context.Context, email, baseURL string) error {
	if s.mail == nil || !s.mail.Siap() {
		return httpx.NewError(http.StatusServiceUnavailable,
			"pengiriman email belum dikonfigurasi — hubungi administrator", nil)
	}

	user, err := s.repo.GetByEmail(ctx, strings.TrimSpace(email))
	if err != nil || user == nil {
		// Sengaja diam. Lihat catatan di atas.
		return nil
	}

	tok, err := BuatTokenReset(time.Now())
	if err != nil {
		return err
	}
	if err := s.repo.SimpanTokenReset(ctx, user.ID, tok.Hash, tok.Berlaku); err != nil {
		return err
	}

	tautan := strings.TrimRight(baseURL, "/") + "/reset-password?token=" + url.QueryEscape(tok.Token)
	menit := int(UmurTokenReset.Minutes())

	return s.mail.Kirim(ctx, mailer.Pesan{
		Ke:     user.Email,
		Subjek: "Atur ulang kata sandi — BNI Finance Hub",
		Teks: "Halo " + user.Name + ",\r\n\r\n" +
			"Ada permintaan untuk mengatur ulang kata sandi akun Anda.\r\n" +
			"Buka tautan berikut untuk membuat kata sandi baru:\r\n\r\n" +
			tautan + "\r\n\r\n" +
			"Tautan ini berlaku " + strconv.Itoa(menit) + " menit dan hanya bisa dipakai sekali.\r\n\r\n" +
			"Kalau bukan Anda yang meminta, abaikan saja email ini — " +
			"kata sandi Anda tidak berubah selama tautannya tidak dibuka.\r\n",
		HTML: `<div style="font:15px/1.6 -apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;color:#1a1a1a">` +
			`<p>Halo ` + htmlAman(user.Name) + `,</p>` +
			`<p>Ada permintaan untuk mengatur ulang kata sandi akun Anda.</p>` +
			`<p style="margin:22px 0"><a href="` + htmlAman(tautan) + `" ` +
			`style="display:inline-block;background:#c8102e;color:#fff;text-decoration:none;` +
			`padding:11px 22px;border-radius:8px;font-weight:600">Buat kata sandi baru</a></p>` +
			`<p style="color:#666;font-size:13px">Tautan ini berlaku ` + strconv.Itoa(menit) +
			` menit dan hanya bisa dipakai sekali.</p>` +
			`<p style="color:#666;font-size:13px">Kalau bukan Anda yang meminta, abaikan saja email ini — ` +
			`kata sandi Anda tidak berubah selama tautannya tidak dibuka.</p></div>`,
	})
}

// ResetKataSandi menukar token yang sah dengan kata sandi baru.
func (s *Service) ResetKataSandi(ctx context.Context, token, kataSandiBaru string) error {
	if strings.TrimSpace(token) == "" {
		return httpx.BadRequest("token tidak disertakan")
	}
	if len(strings.TrimSpace(kataSandiBaru)) < domain.MinPasswordLength {
		return httpx.BadRequest(fmt.Sprintf("kata sandi minimal %d karakter", domain.MinPasswordLength))
	}

	hash := HashToken(token)
	sekarang := time.Now()

	userID, err := s.repo.AmbilTokenReset(ctx, hash, sekarang)
	if err != nil {
		// Kedaluwarsa, sudah dipakai, dan tidak pernah ada dijawab SAMA.
		// Membedakannya memberi tahu penyerang bahwa tokennya pernah sah.
		return httpx.BadRequest("tautan reset tidak berlaku lagi — minta yang baru")
	}

	baru, err := HashPassword(kataSandiBaru)
	if err != nil {
		return err
	}
	if err := s.repo.PakaiTokenReset(ctx, hash, userID, baru, sekarang); err != nil {
		return httpx.BadRequest("tautan reset tidak berlaku lagi — minta yang baru")
	}
	return nil
}

// htmlAman meloloskan teks yang masuk ke badan HTML email.
func htmlAman(v string) string {
	return strings.NewReplacer(
		"&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;",
	).Replace(v)
}
