package proof

import (
	"context"
	"net/http"

	"github.com/syabanf/bni-finance/backend/internal/auth"
	"github.com/syabanf/bni-finance/backend/internal/domain"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register memasang rute bukti bayar. ST mengirim dan menarik bukti di
// chapternya sendiri; hanya admin (finance nasional) yang menyetujui dan
// menolak, karena persetujuan itulah yang menandai invoice lunas.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/payment-proofs", h.list)
	mux.HandleFunc("GET /api/v1/payment-proofs/{id}", h.get)
	mux.HandleFunc("POST /api/v1/payment-proofs", auth.RequireWrite(h.kirim))
	mux.HandleFunc("POST /api/v1/payment-proofs/{id}/approve", auth.RequireAdmin(h.setujui))
	mux.HandleFunc("POST /api/v1/payment-proofs/{id}/reject", auth.RequireAdmin(h.tolak))
	mux.HandleFunc("DELETE /api/v1/payment-proofs/{id}", auth.RequireWrite(h.tarik))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	f := domain.ProofFilter{
		Status:    httpx.Query(r, "status"),
		InvoiceID: httpx.Query(r, "invoiceId"),
		ChapterID: httpx.Query(r, "chapterId"),
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

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handler) kirim(w http.ResponseWriter, r *http.Request) {
	var in domain.SubmitProofInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	p, err := h.svc.Kirim(r.Context(), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, p)
}

func (h *Handler) setujui(w http.ResponseWriter, r *http.Request) {
	h.periksa(w, r, h.svc.Setujui)
}

func (h *Handler) tolak(w http.ResponseWriter, r *http.Request) {
	h.periksa(w, r, h.svc.Tolak)
}

func (h *Handler) periksa(w http.ResponseWriter, r *http.Request,
	fn func(ctx context.Context, id string, in domain.ReviewProofInput) (*domain.PaymentProof, error)) {
	var in domain.ReviewProofInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	p, err := fn(r.Context(), r.PathValue("id"), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handler) tarik(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Tarik(r.Context(), r.PathValue("id")); err != nil {
		httpx.Fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
