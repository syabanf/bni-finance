package payment

import (
	"net/http"

	"github.com/syabanf/bni-finance/backend/internal/auth"
	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/payments", h.list)
	mux.HandleFunc("POST /api/v1/payments", auth.RequireAdmin(h.create))
	mux.HandleFunc("GET /api/v1/payments/{id}", h.get)
	mux.HandleFunc("PATCH /api/v1/payments/{id}", auth.RequireAdmin(h.update))
	mux.HandleFunc("DELETE /api/v1/payments/{id}", auth.RequireAdmin(h.remove))
}

type listMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type listResponse struct {
	Data []domain.Payment `json:"data"`
	Meta listMeta         `json:"meta"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	f := domain.PaymentFilter{
		InvoiceID: httpx.Query(r, "invoiceId"),
		Method:    httpx.Query(r, "method"),
		PaidFrom:  httpx.Query(r, "paidFrom"),
		PaidTo:    httpx.Query(r, "paidTo"),
		Limit:     httpx.QueryInt(r, "limit", 50, 1, 200),
		Offset:    httpx.QueryInt(r, "offset", 0, 0, 1_000_000),
	}

	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, listResponse{
		Data: items,
		Meta: listMeta{Total: total, Limit: f.Limit, Offset: f.Offset},
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in domain.CreatePaymentInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	p, err := h.svc.Create(r.Context(), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, p)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var in domain.UpdatePaymentInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	p, err := h.svc.Update(r.Context(), r.PathValue("id"), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		httpx.Fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
