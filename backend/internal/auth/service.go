package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type Store interface {
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
	Create(ctx context.Context, email, passwordHash, name string, role domain.UserRole) (*domain.User, error)
	UpdateName(ctx context.Context, id, name string) (*domain.User, error)
	UpdateRole(ctx context.Context, id string, role domain.UserRole) (*domain.User, error)
	// Guarded: pemeriksaan admin-terakhir dan penulisannya terjadi dalam satu
	// transaksi. Memisahkannya adalah balapan yang pernah menyisakan nol admin.
	UpdateRoleGuarded(ctx context.Context, id string, role domain.UserRole) (*domain.User, error)
	DeleteGuarded(ctx context.Context, id string) error
	UpdatePasswordHash(ctx context.Context, id, hash string) error
	Delete(ctx context.Context, id string) error
	CountAdmins(ctx context.Context) (int, error)
}

var _ Store = (*Repository)(nil)

type Service struct {
	repo   Store
	signer *Signer
	now    func() time.Time

	// quickLogin is the lower-cased allow-list for passwordless sign-in.
	// Empty means the feature is off — see the quick login section below.
	quickLogin []string
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
	user, err := s.repo.Create(ctx, in.Email, hash, strings.TrimSpace(in.Name), role)
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
