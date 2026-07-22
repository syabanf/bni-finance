package chapter

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
	mux.HandleFunc("GET /api/v1/chapters", h.list)
	mux.HandleFunc("POST /api/v1/chapters", h.create)
	mux.HandleFunc("GET /api/v1/chapters/{id}", h.get)
	mux.HandleFunc("PATCH /api/v1/chapters/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/chapters/{id}", h.remove)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	f := domain.ChapterFilter{
		Search:   httpx.Query(r, "q"),
		CityName: httpx.Query(r, "cityName"),
		AreaName: httpx.Query(r, "areaName"),
		Limit:    httpx.QueryInt(r, "limit", 100, 1, 500),
		Offset:   httpx.QueryInt(r, "offset", 0, 0, 1_000_000),
	}

	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.List(items, total, f.Limit, f.Offset))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in domain.CreateChapterInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	c, err := h.svc.Create(r.Context(), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, c)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var in domain.UpdateChapterInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	c, err := h.svc.Update(r.Context(), r.PathValue("id"), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		httpx.Fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
