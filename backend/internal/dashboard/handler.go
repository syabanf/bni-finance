package dashboard

import (
	"context"
	"net/http"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type Store interface {
	Summary(ctx context.Context, months int) (*domain.DashboardSummary, error)
}

var _ Store = (*Repository)(nil)

type Service struct {
	repo Store
}

func NewService(repo Store) *Service { return &Service{repo: repo} }

func (s *Service) Summary(ctx context.Context, months int) (*domain.DashboardSummary, error) {
	return s.repo.Summary(ctx, months)
}

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/dashboard/summary", h.summary)
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	months := httpx.QueryInt(r, "months", 6, 1, 24)

	sum, err := h.svc.Summary(r.Context(), months)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, sum)
}
