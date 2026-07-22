package audit

import (
	"net/http"

	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register nests the timeline under the invoice it belongs to.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/invoices/{id}/audit", h.list)
	mux.HandleFunc("POST /api/v1/invoices/{id}/audit", h.create)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	limit := httpx.QueryInt(r, "limit", 50, 1, 200)

	items, err := h.svc.ListByInvoice(r.Context(), r.PathValue("id"), limit)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.List(items, len(items), limit, 0))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in domain.CreateAuditEntryInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	entry, err := h.svc.Create(r.Context(), r.PathValue("id"), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, entry)
}
