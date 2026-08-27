package invoice

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

// Register wires the CRUD routes onto the mux (Go 1.22 method+path patterns).
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/invoices", h.list)
	mux.HandleFunc("POST /api/v1/invoices", auth.RequireAdmin(h.create))
	mux.HandleFunc("GET /api/v1/invoices/{id}", h.get)
	mux.HandleFunc("PATCH /api/v1/invoices/{id}", auth.RequireAdmin(h.update))
	mux.HandleFunc("DELETE /api/v1/invoices/{id}", auth.RequireAdmin(h.remove))
}

type listMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	// Summary hanya terisi bila diminta lewat ?summary=true. Nil-nya dihilangkan
	// dari JSON, jadi pemanggil lama tidak melihat perubahan apa pun.
	Summary *domain.InvoiceSummary `json:"summary,omitempty"`
}

type listResponse struct {
	Data []domain.Invoice `json:"data"`
	Meta listMeta         `json:"meta"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	f := domain.InvoiceFilter{
		Status:     httpx.Query(r, "status"),
		Type:       httpx.Query(r, "type"),
		ChapterID:  httpx.Query(r, "chapterId"),
		MemberID:   httpx.Query(r, "memberId"),
		Search:     httpx.Query(r, "q"),
		DueFrom:    httpx.Query(r, "dueFrom"),
		DueTo:      httpx.Query(r, "dueTo"),
		IssuedFrom: httpx.Query(r, "issuedFrom"),
		IssuedTo:   httpx.Query(r, "issuedTo"),
		Aging:      httpx.Query(r, "aging"),
		Limit:      httpx.QueryInt(r, "limit", 50, 1, 200),
		Offset:     httpx.QueryInt(r, "offset", 0, 0, 1_000_000),
	}

	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	meta := listMeta{Total: total, Limit: f.Limit, Offset: f.Offset}

	// Ringkasan hanya dihitung bila diminta.
	//
	// Ia satu query agregat tambahan, dan sebagian besar pemanggil — dashboard,
	// laporan, autofill — tidak membutuhkannya. Menyalakannya untuk semua
	// berarti membebankan biaya itu pada jalur yang tidak memakainya.
	if httpx.Query(r, "summary") == "true" {
		ringkas, err := h.svc.Summary(r.Context(), f)
		if err != nil {
			httpx.Fail(w, err)
			return
		}
		meta.Summary = ringkas
	}

	httpx.JSON(w, http.StatusOK, listResponse{Data: items, Meta: meta})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	inv, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, inv)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in domain.CreateInvoiceInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	inv, err := h.svc.Create(r.Context(), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, inv)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var in domain.UpdateInvoiceInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	inv, err := h.svc.Update(r.Context(), r.PathValue("id"), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, inv)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		httpx.Fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
