package paperid

import (
	"net/http"

	"github.com/syabanf/bni-finance/backend/internal/auth"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterProtected wires the admin send action and the test console onto the
// authenticated mux.
func (h *Handler) RegisterProtected(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/invoices/{id}/send", auth.RequireAdmin(h.send))
	mux.HandleFunc("GET /api/v1/paperid/status", auth.RequireAdmin(h.status))
	mux.HandleFunc("POST /api/v1/paperid/test-invoice", auth.RequireAdmin(h.testInvoice))
	mux.HandleFunc("POST /api/v1/paperid/test-callback", auth.RequireAdmin(h.testCallback))
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.Status(r.Context())
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}

func (h *Handler) testInvoice(w http.ResponseWriter, r *http.Request) {
	var in TestInvoiceInput
	if r.ContentLength != 0 {
		if err := httpx.Decode(r, &in); err != nil {
			httpx.Fail(w, err)
			return
		}
	}
	res, err := h.svc.TestInvoice(r.Context(), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) testCallback(w http.ResponseWriter, r *http.Request) {
	var in TestCallbackInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	res, err := h.svc.TestCallback(r.Context(), in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

// RegisterPublic wires the payment callback. Paper.id calls it with no login,
// so it must sit outside the auth middleware — it authenticates itself with the
// shared secret carried in the callback URL.
func (h *Handler) RegisterPublic(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/webhooks/paperid", h.webhook)
}

func (h *Handler) send(w http.ResponseWriter, r *http.Request) {
	// Body is optional — default is to send no email/WhatsApp.
	var opts SendOptions
	if r.ContentLength != 0 {
		if err := httpx.Decode(r, &opts); err != nil {
			httpx.Fail(w, err)
			return
		}
	}
	inv, err := h.svc.Send(r.Context(), r.PathValue("id"), opts)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, inv)
}

func (h *Handler) webhook(w http.ResponseWriter, r *http.Request) {
	var in WebhookInput
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}

	// The secret arrives via header or ?token= — whichever the callback URL
	// registered in the Paper.id dashboard uses.
	token := r.Header.Get("x-paper-callback-token")
	if token == "" {
		token = httpx.Query(r, "token")
	}

	settled, err := h.svc.HandleWebhook(r.Context(), token, in)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	// Always 200 on an authentic, well-formed callback — including ignored
	// events. A non-2xx just makes Paper.id retry something we chose to skip.
	httpx.JSON(w, http.StatusOK, map[string]bool{"settled": settled})
}
