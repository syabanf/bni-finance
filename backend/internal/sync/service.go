package sync

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/auth"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

// TokenSettingKey is where the BNI VM token lives in app_settings. The name
// contains "token", so the settings API reads it back masked — it can be set
// through the UI but never read out of it.
const TokenSettingKey = "bni_vm_token"

type Store interface {
	Apply(ctx context.Context, members []RemoteMember, now time.Time) (*Result, error)
	Setting(ctx context.Context, key string) (string, error)
}

var _ Store = (*Repository)(nil)

// Fetcher is the upstream, kept an interface so the service can be tested
// against a stub instead of the real BNI VM.
type Fetcher interface {
	FetchMembers(ctx context.Context) ([]RemoteMember, error)
}

type Service struct {
	repo     Store
	baseURL  string
	envToken string
	now      func() time.Time

	// newFetcher is swapped in tests.
	newFetcher func(baseURL, token string) Fetcher
}

func NewService(repo Store, baseURL, envToken string) *Service {
	return &Service{
		repo:     repo,
		baseURL:  baseURL,
		envToken: envToken,
		now:      time.Now,
		newFetcher: func(baseURL, token string) Fetcher {
			return NewClient(baseURL, token)
		},
	}
}

// Run pulls the whole member list and applies it.
//
// The token comes from app_settings first so it can be rotated from the
// settings page without a redeploy, falling back to the environment.
func (s *Service) Run(ctx context.Context) (*Result, error) {
	token, err := s.repo.Setting(ctx, TokenSettingKey)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(token) == "" {
		token = s.envToken
	}
	if strings.TrimSpace(token) == "" {
		return nil, httpx.NewError(http.StatusServiceUnavailable,
			"token BNI VM belum dikonfigurasi — isi 'bni_vm_token' di Pengaturan", nil)
	}

	members, err := s.newFetcher(s.baseURL, token).FetchMembers(ctx)
	if err != nil {
		return nil, httpx.NewError(http.StatusBadGateway, err.Error(), err)
	}
	if len(members) == 0 {
		// Applying an empty snapshot would deactivate everyone. Far more likely
		// the upstream had a bad day than that the organisation lost every
		// member at once.
		return nil, httpx.NewError(http.StatusBadGateway,
			"BNI VM tidak mengembalikan satu member pun — sinkronisasi dibatalkan agar data lokal tidak dinonaktifkan massal", nil)
	}

	return s.repo.Apply(ctx, members, s.now())
}

// --- HTTP -------------------------------------------------------------------

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/sync", auth.RequireAdmin(h.run))
}

func (h *Handler) run(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Run(r.Context())
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}
