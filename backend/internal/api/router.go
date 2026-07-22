// Package api assembles the HTTP surface: routes + middleware chain.
// Keeping it here (instead of inline in main) means tests exercise the real
// wiring rather than a re-implementation of it.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/config"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
	"github.com/syabanf/bni-finance/backend/internal/invoice"
	"github.com/syabanf/bni-finance/backend/internal/payment"
)

// Pinger reports database health for /healthz.
type Pinger func(ctx context.Context) error

// NewHandler builds the fully-wrapped HTTP handler.
func NewHandler(
	log *slog.Logger,
	cfg config.Config,
	invoiceSvc *invoice.Service,
	paymentSvc *payment.Service,
	ping Pinger,
) http.Handler {
	apiMux := http.NewServeMux()
	invoice.NewHandler(invoiceSvc).Register(apiMux)
	payment.NewHandler(paymentSvc).Register(apiMux)

	root := http.NewServeMux()

	// Health check stays unauthenticated so probes work without a key.
	root.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if ping != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()
			if err := ping(ctx); err != nil {
				httpx.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "database tidak siap"})
				return
			}
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	root.Handle("/api/", httpx.APIKey(cfg.APIKey)(apiMux))

	return httpx.Chain(root,
		httpx.RequestID,
		httpx.Logger(log),
		httpx.Recoverer(log),
		httpx.CORS(cfg.AllowedOrigins),
	)
}
