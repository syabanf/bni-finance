package period

import (
	"net/http"

	"github.com/syabanf/bni-finance/backend/internal/auth"
	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(mux *http.ServeMux) {
	// Membaca terbuka untuk semua peran: ST dan MC memilih periode di halaman
	// Konfirmasi Renewal. Mengubah hanya admin, karena periode berlaku nasional.
	mux.HandleFunc("GET /api/v1/renewal-periods", h.list)
	mux.HandleFunc("POST /api/v1/renewal-periods", auth.RequireAdmin(h.create))
	mux.HandleFunc("PATCH /api/v1/renewal-periods/{id}", auth.RequireAdmin(h.update))
	mux.HandleFunc("DELETE /api/v1/renewal-periods/{id}", auth.RequireAdmin(h.remove))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List(r.Context())
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in domain.SaveRenewalPeriodInput
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
	var in domain.SaveRenewalPeriodInput
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
