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
)

func (r UserRole) Valid() bool { return r == RoleAdmin || r == RoleUser }

// User mirrors the `users` table. PasswordHash is never serialised — it has no
// JSON tag pair that would let it escape.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      UserRole  `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	PasswordHash string `json:"-"`
}

// AuthUser is the shape the frontend's AuthUser type expects.
type AuthUser struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Role  UserRole `json:"role"`
}

func (u User) AsAuthUser() AuthUser {
	return AuthUser{ID: u.ID, Name: u.Name, Email: u.Email, Role: u.Role}
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
	Email    string    `json:"email"`
	Password string    `json:"password"`
	Name     string    `json:"name"`
	Role     *UserRole `json:"role"`
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
		return fmt.Errorf("role harus 'admin' atau 'user'")
	}
	return nil
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
