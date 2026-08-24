package renewal

import (
	"context"
	"net/http"
	"strings"

	"github.com/syabanf/bni-finance/backend/internal/auth"
	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

// Store adalah kontrak persistensi yang dibutuhkan service.
type Store interface {
	List(ctx context.Context, f domain.RenewalFilter) ([]domain.RenewalRequest, int, error)
	GetByID(ctx context.Context, id string) (*domain.RenewalRequest, error)
	Create(ctx context.Context, memberIDs []string, period, requestedBy string, assignedMC *string) (int, int, error)
	Answer(ctx context.Context, id string, in domain.AnswerRenewalInput, answeredBy string) (*domain.RenewalRequest, error)
}

var _ Store = (*Repository)(nil)

type Service struct{ repo Store }

func NewService(repo Store) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, f domain.RenewalFilter) ([]domain.RenewalRequest, int, error) {
	return s.repo.List(ctx, f)
}

// HasilMinta merangkum satu permintaan massal.
type HasilMinta struct {
	Dibuat   int `json:"dibuat"`
	Dilewati int `json:"dilewati"`
	Total    int `json:"total"`
}

// Minta membuat permintaan konfirmasi untuk sekumpulan member.
func (s *Service) Minta(ctx context.Context, in domain.CreateRenewalRequestInput) (*HasilMinta, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	user, ok := auth.UserFrom(ctx)
	if !ok {
		return nil, httpx.Unauthorized("token tidak disertakan")
	}

	dibuat, dilewati, err := s.repo.Create(ctx, dedup(in.MemberIDs),
		strings.TrimSpace(in.Period), user.ID, in.AssignedMC)
	if err != nil {
		return nil, err
	}
	return &HasilMinta{Dibuat: dibuat, Dilewati: dilewati, Total: dibuat + dilewati}, nil
}

// Jawab mencatat jawaban MC.
//
// SIAPA yang boleh menjawab dijaga di sini, bukan di middleware: middleware
// hanya tahu peran, sedangkan aturan ini juga menyangkut keadaan barisnya.
func (s *Service) Jawab(ctx context.Context, id string, in domain.AnswerRenewalInput) (*domain.RenewalRequest, error) {
	if err := in.Validate(); err != nil {
		return nil, httpx.BadRequest(err.Error())
	}
	user, ok := auth.UserFrom(ctx)
	if !ok {
		return nil, httpx.Unauthorized("token tidak disertakan")
	}
	// MC dan admin yang menjawab. ST TIDAK, dan itu inti alurnya: ST yang
	// menanyakan, jadi membiarkannya menjawab sendiri membuat konfirmasinya
	// tidak berarti apa-apa — ia hanya mencatat dugaannya sendiri sebagai
	// jawaban orang lain.
	if user.Role != domain.RoleMC && user.Role != domain.RoleAdmin {
		return nil, httpx.Forbidden("hanya MC yang menjawab konfirmasi renewal")
	}
	return s.repo.Answer(ctx, id, in, user.ID)
}

func dedup(ids []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// --- HTTP -------------------------------------------------------------------

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/renewal-requests", h.list)
	// RequireWrite: admin dan ST. MC tidak boleh MEMINTA konfirmasi kepada
	// dirinya sendiri.
	mux.HandleFunc("POST /api/v1/renewal-requests", auth.RequireWrite(h.minta))
	// Menjawab TIDAK memakai RequireWrite: MC hanya boleh membaca di tempat
	// lain, tapi harus boleh menjawab di sini. Perannya diperiksa di service.
	mux.HandleFunc("PATCH /api/v1/renewal-requests/{id}", h.jawab)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	f := domain.RenewalFilter{
		ChapterID: httpx.Query(r, "chapterId"),
		Answer:    httpx.Query(r, "answer"),
		Period:    httpx.Query(r, "period"),
		Limit:     httpx.QueryInt(r, "limit", 50, 1, 200),
		Offset:    httpx.QueryInt(r, "offset", 0, 0, 1_000_000),
	}
	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.List(items, total, f.Limit, f.Offset))
}

func (h *Handler) minta(w http.ResponseWriter, r *http.Request) {
	var in domain.CreateRenewalRequestInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	out, err := h.svc.Minta(r.Context(), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, out)
}

func (h *Handler) jawab(w http.ResponseWriter, r *http.Request) {
	var in domain.AnswerRenewalInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	out, err := h.svc.Jawab(r.Context(), r.PathValue("id"), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}
