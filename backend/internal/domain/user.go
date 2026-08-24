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
	// RoleST — Secretary/Treasurer sebuah chapter. Menulis, tapi hanya di
	// chapternya sendiri.
	RoleST UserRole = "st"
	// RoleMC — Membership Committee sebuah chapter. Membaca, dan menjawab
	// permintaan konfirmasi renewal.
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
// Peran `user` misalnya punya jangkauan nasional tapi tidak boleh menulis,
// sedangkan `st` boleh menulis tapi hanya di satu chapter. Satu pemeriksaan
// gabungan tidak bisa menyatakan keduanya tanpa salah pada salah satu.

// CanWrite melaporkan peran ini boleh mengubah data.
func (r UserRole) CanWrite() bool { return r == RoleAdmin || r == RoleST }

// ChapterScoped melaporkan peran ini terikat pada satu chapter saja.
func (r UserRole) ChapterScoped() bool { return r == RoleST || r == RoleMC }

// User mirrors the `users` table. PasswordHash is never serialised — it has no
// JSON tag pair that would let it escape.
type User struct {
	ID    string   `json:"id"`
	Email string   `json:"email"`
	Name  string   `json:"name"`
	Role  UserRole `json:"role"`
	// ChapterID mengikat pengguna ke satu chapter. Null berarti nasional —
	// itulah yang dipakai admin dan user.
	ChapterID *string   `json:"chapterId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	PasswordHash string `json:"-"`
}

// Validate menolak kombinasi peran dan lingkup yang tidak masuk akal.
//
// Keduanya diperiksa bersama karena kesalahannya berlawanan arah, dan keduanya
// berbahaya dengan cara yang berbeda:
//
//	ST/MC tanpa chapter   -> lingkupnya kosong; kalau kode di hilir membaca ini
//	                         sebagai "tanpa batas", ia melihat SELURUH chapter
//	admin dengan chapter  -> lingkup yang menyesatkan pada peran yang memang
//	                         nasional
func (u User) ValidateScope() error {
	scoped := u.Role.ChapterScoped()
	punya := u.ChapterID != nil && strings.TrimSpace(*u.ChapterID) != ""
	switch {
	case scoped && !punya:
		return fmt.Errorf("peran %s wajib punya chapterId", u.Role)
	case !scoped && punya:
		return fmt.Errorf("peran %s tidak boleh punya chapterId", u.Role)
	}
	return nil
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
	if !u.Role.ChapterScoped() || u.ChapterID == nil {
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
