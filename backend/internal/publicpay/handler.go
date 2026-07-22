package publicpay

import (
	"net/http"

	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register wires the routes that must stay reachable WITHOUT a token: the
// person paying an invoice is not a user of the app.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/public/invoices/{id}", h.view)
	mux.HandleFunc("POST /api/v1/public/invoices/{id}/payment", h.createPayment)
	mux.HandleFunc("POST /api/v1/webhooks/xendit", h.webhook)
}

func (h *Handler) view(w http.ResponseWriter, r *http.Request) {
	view, err := h.svc.View(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, view)
}

func (h *Handler) createPayment(w http.ResponseWriter, r *http.Request) {
	var in CreatePaymentInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	result, err := h.svc.CreatePayment(r.Context(), r.PathValue("id"), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, result)
}

func (h *Handler) webhook(w http.ResponseWriter, r *http.Request) {
	var in WebhookInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}

	settled, err := h.svc.HandleWebhook(r.Context(), r.Header.Get("x-callback-token"), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	// Always 200 on a well-formed, authentic callback — including the ignored
	// ones. A non-2xx makes Xendit retry, and retrying an event we chose not to
	// act on achieves nothing.
	httpx.JSON(w, http.StatusOK, map[string]bool{"settled": settled})
}
