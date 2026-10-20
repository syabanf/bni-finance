package domain

import (
	"fmt"
	"strings"
	"time"
)

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
	// RoleST: Secretary/Treasurer. Menulis invoice dan pembayaran, nasional
	// atau di satu chapter, mengikuti ChapterID.
	RoleST UserRole = "st"
	// RoleMC: Membership Committee. Membaca, dan menjawab permintaan
	// konfirmasi renewal, nasional atau di satu chapter.
	RoleMC UserRole = "mc"
)

func (r UserRole) Valid() bool {
	switch r {
	case RoleAdmin, RoleUser, RoleST, RoleMC:
		return true
	}
	return false
}

// Otorisasi di sistem ini punya DUA dimensi yang harus dipisah, dan
// mencampurnya adalah cara termudah membocorkan data antar chapter:
//
//	kemampuan — boleh menulis, atau hanya membaca
//	lingkup   — melihat seluruh chapter, atau hanya satu
//
// Peran menentukan kemampuan; ChapterID menentukan lingkup. Keduanya diatur
// terpisah: ST, MC, dan User masing-masing bisa nasional atau satu chapter.
// Hanya admin yang selalu nasional, karena rute khusus admin (pengaturan,
// pengguna, sinkron, impor) bekerja lintas chapter.

// CanWrite melaporkan peran ini boleh mengubah data.
func (r UserRole) CanWrite() bool { return r == RoleAdmin || r == RoleST }

// BolehBerlingkupChapter melaporkan peran ini boleh dibatasi ke satu chapter.
// Admin tidak: rute khusus admin bekerja lintas chapter dan tidak membaca
// lingkup, jadi admin berchapter adalah lingkup yang tidak ditegakkan.
func (r UserRole) BolehBerlingkupChapter() bool { return r != RoleAdmin }

// User mirrors the `users` table. PasswordHash is never serialised — it has no
// JSON tag pair that would let it escape.
type User struct {
	ID    string   `json:"id"`
	Email string   `json:"email"`
	Name  string   `json:"name"`
	Role  UserRole `json:"role"`
	// ChapterID mengikat pengguna ke satu chapter. Null berarti nasional;
	// admin selalu null.
	ChapterID *string   `json:"chapterId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	PasswordHash string `json:"-"`
}

// ValidateScope menolak admin yang berchapter.
//
// Peran lain bebas: tanpa chapter berarti nasional, dengan chapter berarti
// hanya chapter itu. Admin ditolak karena rute khusus admin tidak membaca
// lingkup; chapter pada admin akan tampil di layar tanpa membatasi apa pun.
func (u User) ValidateScope() error {
	if !u.Role.BolehBerlingkupChapter() && u.ChapterID != nil && strings.TrimSpace(*u.ChapterID) != "" {
		return fmt.Errorf("peran %s selalu nasional; kosongkan chapterId", u.Role)
	}
	return nil
}

// RapikanChapter mengubah chapterId kosong atau berisi spasi menjadi nil.
// Nilai kosong berarti nasional, dan menyimpannya sebagai "" akan ditolak
// foreign key sebagai chapter yang tidak ada.
func RapikanChapter(id *string) *string {
	if id == nil {
		return nil
	}
	v := strings.TrimSpace(*id)
	if v == "" {
		return nil
	}
	return &v
}

// AuthUser is the shape the frontend's AuthUser type expects.
type AuthUser struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Email     string   `json:"email"`
	Role      UserRole `json:"role"`
	ChapterID *string  `json:"chapterId"`
}

func (u User) AsAuthUser() AuthUser {
	return AuthUser{ID: u.ID, Name: u.Name, Email: u.Email, Role: u.Role, ChapterID: u.ChapterID}
}

// ChapterScope mengembalikan chapter yang boleh dilihat pengguna ini, dan
// apakah ia memang dibatasi.
//
// Satu-satunya tempat yang memutuskan. Repository memanggil ini, bukan membaca
// Role dan ChapterID sendiri-sendiri — supaya penambahan peran berlingkup
// berikutnya tidak perlu menyentuh setiap query.
func (u AuthUser) ChapterScope() (string, bool) {
	if !u.Role.BolehBerlingkupChapter() || u.ChapterID == nil {
		return "", false
	}
	id := strings.TrimSpace(*u.ChapterID)
	return id, id != ""
}

// LoginInput is the POST /auth/login body.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (in LoginInput) Validate() error {
	switch {
	case strings.TrimSpace(in.Email) == "":
		return fmt.Errorf("email wajib diisi")
	case in.Password == "":
		return fmt.Errorf("kata sandi wajib diisi")
	}
	return nil
}

// LoginResult carries the signed token alongside the profile, so the client
// needs only one round-trip to sign in.
type LoginResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	User      AuthUser  `json:"user"`
	// ButuhOTP menandai kata sandi sudah benar tapi belum cukup: kode sudah
	// dikirim ke email, dan token baru terbit setelah kode itu diverifikasi.
	// Token SENGAJA kosong saat ini true.
	ButuhOTP bool `json:"otpRequired,omitempty"`
}

// MinPasswordLength matches the rule the UI already enforces.
const MinPasswordLength = 6

type CreateUserInput struct {
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	Name      string    `json:"name"`
	Role      *UserRole `json:"role"`
	ChapterID *string   `json:"chapterId"`
}

func (in CreateUserInput) Validate() error {
	switch {
	case strings.TrimSpace(in.Email) == "":
		return fmt.Errorf("email wajib diisi")
	case !strings.Contains(in.Email, "@"):
		return fmt.Errorf("format email tidak valid")
	case strings.TrimSpace(in.Name) == "":
		return fmt.Errorf("nama wajib diisi")
	case len(strings.TrimSpace(in.Password)) < MinPasswordLength:
		return fmt.Errorf("kata sandi minimal %d karakter", MinPasswordLength)
	case in.Role != nil && !in.Role.Valid():
		return fmt.Errorf("role harus 'admin', 'st', 'mc', atau 'user'")
	}
	// Lingkupnya diperiksa lewat aturan yang sama dengan yang berlaku pada baris
	// tersimpan, supaya pembuatan lewat API tidak bisa menghasilkan kombinasi
	// yang ditolak di tempat lain.
	peran := RoleUser
	if in.Role != nil {
		peran = *in.Role
	}
	return User{Role: peran, ChapterID: in.ChapterID}.ValidateScope()
}

type UpdateProfileInput struct {
	Name *string `json:"name"`
}

func (in UpdateProfileInput) Validate() error {
	if in.Name != nil && strings.TrimSpace(*in.Name) == "" {
		return fmt.Errorf("nama tidak boleh kosong")
	}
	return nil
}

type UpdatePasswordInput struct {
	// CurrentPassword is required when a user changes their own password;
	// an admin resetting someone else's may omit it.
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (in UpdatePasswordInput) Validate() error {
	if len(strings.TrimSpace(in.NewPassword)) < MinPasswordLength {
		return fmt.Errorf("kata sandi minimal %d karakter", MinPasswordLength)
	}
	return nil
}

// NormalizeEmail is applied on both write and lookup so casing never causes a
// duplicate account or a failed sign-in.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
