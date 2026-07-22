package settings

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
	mux.HandleFunc("GET /api/v1/fee-settings", h.getFees)
	mux.HandleFunc("PATCH /api/v1/fee-settings", auth.RequireAdmin(h.updateFees))

	mux.HandleFunc("GET /api/v1/app-settings", h.listApp)
	mux.HandleFunc("GET /api/v1/app-settings/{key}", h.getApp)
	mux.HandleFunc("PUT /api/v1/app-settings/{key}", auth.RequireAdmin(h.setApp))
	mux.HandleFunc("DELETE /api/v1/app-settings/{key}", auth.RequireAdmin(h.deleteApp))
}

func (h *Handler) getFees(w http.ResponseWriter, r *http.Request) {
	fees, err := h.svc.GetFees(r.Context())
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, fees)
}

func (h *Handler) updateFees(w http.ResponseWriter, r *http.Request) {
	var in domain.UpdateFeeSettingsInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	fees, err := h.svc.UpdateFees(r.Context(), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, fees)
}

func (h *Handler) listApp(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListApp(r.Context())
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, httpx.List(items, len(items), len(items), 0))
}

func (h *Handler) getApp(w http.ResponseWriter, r *http.Request) {
	item, err := h.svc.GetApp(r.Context(), r.PathValue("key"))
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (h *Handler) setApp(w http.ResponseWriter, r *http.Request) {
	var in domain.SetAppSettingInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	item, err := h.svc.SetApp(r.Context(), r.PathValue("key"), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (h *Handler) deleteApp(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteApp(r.Context(), r.PathValue("key")); err != nil {
		httpx.Fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
