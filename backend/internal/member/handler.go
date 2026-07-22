package member

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
	mux.HandleFunc("GET /api/v1/members", h.list)
	mux.HandleFunc("POST /api/v1/members", h.create)
	// Literal patterns beat wildcards in Go 1.22 routing, so this stays
	// reachable alongside /members/{id}.
	mux.HandleFunc("GET /api/v1/members/renewal-due", h.renewalDue)
	mux.HandleFunc("GET /api/v1/members/{id}", h.get)
	mux.HandleFunc("PATCH /api/v1/members/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/members/{id}", h.remove)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	f := domain.MemberFilter{
		ChapterID:   httpx.Query(r, "chapterId"),
		Status:      httpx.Query(r, "status"),
		Search:      httpx.Query(r, "q"),
		RenewalFrom: httpx.Query(r, "renewalFrom"),
		RenewalTo:   httpx.Query(r, "renewalTo"),
		Limit:       httpx.QueryInt(r, "limit", 50, 1, 200),
		Offset:      httpx.QueryInt(r, "offset", 0, 0, 1_000_000),
	}

	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.List(items, total, f.Limit, f.Offset))
}

func (h *Handler) renewalDue(w http.ResponseWriter, r *http.Request) {
	days := httpx.QueryInt(r, "days", 30, 1, 365)
	limit := httpx.QueryInt(r, "limit", 100, 1, 500)

	items, err := h.svc.RenewalDue(r.Context(), days, limit)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.List(items, len(items), limit, 0))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	m, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, m)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in domain.CreateMemberInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	m, err := h.svc.Create(r.Context(), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, m)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var in domain.UpdateMemberInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	m, err := h.svc.Update(r.Context(), r.PathValue("id"), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, m)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		httpx.Fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
