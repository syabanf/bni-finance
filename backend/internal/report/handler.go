package report

import (
	"net/http"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/reports/summary", h.summary)
	mux.HandleFunc("GET /api/v1/reports/detail", h.detail)
}

func filter(r *http.Request) domain.ReportFilter {
	return domain.ReportFilter{
		From: httpx.Query(r, "from"),
		To:   httpx.Query(r, "to"),
		Type: httpx.Query(r, "type"),
	}
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	sum, err := h.svc.Summary(r.Context(), filter(r))
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, sum)
}

func (h *Handler) detail(w http.ResponseWriter, r *http.Request) {
	limit := httpx.QueryInt(r, "limit", 50, 1, 200)
	offset := httpx.QueryInt(r, "offset", 0, 0, 1_000_000)

	items, total, err := h.svc.Detail(r.Context(), filter(r), limit, offset)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.List(items, total, limit, offset))
}
