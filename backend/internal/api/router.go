// Package api assembles the HTTP surface: routes + middleware chain.
// Keeping it here (instead of inline in main) means tests exercise the real
// wiring rather than a re-implementation of it.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/audit"
	"github.com/syabanf/bni-finance/backend/internal/chapter"
	"github.com/syabanf/bni-finance/backend/internal/config"
	"github.com/syabanf/bni-finance/backend/internal/dashboard"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
	"github.com/syabanf/bni-finance/backend/internal/invoice"
	"github.com/syabanf/bni-finance/backend/internal/member"
	"github.com/syabanf/bni-finance/backend/internal/payment"
	"github.com/syabanf/bni-finance/backend/internal/settings"
)

// Pinger reports database health for /healthz.
type Pinger func(ctx context.Context) error

// Services collects everything the API can expose. A nil field simply means
// that resource isn't registered — which is what lets tests bring up a subset.
type Services struct {
	Invoice   *invoice.Service
	Payment   *payment.Service
	Member    *member.Service
	Chapter   *chapter.Service
	Settings  *settings.Service
	Audit     *audit.Service
	Dashboard *dashboard.Service
}

// NewHandler builds the fully-wrapped HTTP handler.
func NewHandler(log *slog.Logger, cfg config.Config, svc Services, ping Pinger) http.Handler {
	apiMux := http.NewServeMux()

	if svc.Invoice != nil {
		invoice.NewHandler(svc.Invoice).Register(apiMux)
	}
	if svc.Payment != nil {
		payment.NewHandler(svc.Payment).Register(apiMux)
	}
	if svc.Member != nil {
		member.NewHandler(svc.Member).Register(apiMux)
	}
	if svc.Chapter != nil {
		chapter.NewHandler(svc.Chapter).Register(apiMux)
	}
	if svc.Settings != nil {
		settings.NewHandler(svc.Settings).Register(apiMux)
	}
	if svc.Audit != nil {
		audit.NewHandler(svc.Audit).Register(apiMux)
	}
	if svc.Dashboard != nil {
		dashboard.NewHandler(svc.Dashboard).Register(apiMux)
	}

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
