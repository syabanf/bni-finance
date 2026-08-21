// Package api assembles the HTTP surface: routes + middleware chain.
// Keeping it here (instead of inline in main) means tests exercise the real
// wiring rather than a re-implementation of it.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/apidocs"
	"github.com/syabanf/bni-finance/backend/internal/audit"
	"github.com/syabanf/bni-finance/backend/internal/auth"
	"github.com/syabanf/bni-finance/backend/internal/blackbox"
	"github.com/syabanf/bni-finance/backend/internal/chapter"
	"github.com/syabanf/bni-finance/backend/internal/config"
	"github.com/syabanf/bni-finance/backend/internal/dashboard"
	"github.com/syabanf/bni-finance/backend/internal/httpx"
	"github.com/syabanf/bni-finance/backend/internal/invoice"
	"github.com/syabanf/bni-finance/backend/internal/member"
	"github.com/syabanf/bni-finance/backend/internal/metrics"
	"github.com/syabanf/bni-finance/backend/internal/paperid"
	"github.com/syabanf/bni-finance/backend/internal/payment"
	"github.com/syabanf/bni-finance/backend/internal/settings"
	"github.com/syabanf/bni-finance/backend/internal/sync"
	"github.com/syabanf/bni-finance/backend/internal/upload"
)

// Pinger reports database health for /healthz.
type Pinger func(ctx context.Context) error

// Services collects everything the API can expose. A nil field simply means
// that resource isn't registered — which is what lets tests bring up a subset.
type Services struct {
	Auth      *auth.Service
	Invoice   *invoice.Service
	Payment   *payment.Service
	Member    *member.Service
	Chapter   *chapter.Service
	Settings  *settings.Service
	Audit     *audit.Service
	Dashboard *dashboard.Service
	Upload    *upload.Store
	Sync      *sync.Service
	PaperID   *paperid.Service
	Blackbox  *blackbox.Recorder

	// Metrics is optional; nil disables /metrics and all instrumentation.
	Metrics *metrics.Registry
}

// NewHandler builds the fully-wrapped HTTP handler.
//
// Three tiers of access:
//
//	public    — no token: login, the public payment page, health, stored files
//	protected — any signed-in user: reads
//	admin     — writes that change money or configuration
//
// Supabase enforced this with RLS in the database. The Go backend connects as a
// single trusted role, so the boundary lives here instead; see auth.RequireAuth.
func NewHandler(log *slog.Logger, cfg config.Config, signer *auth.Signer, svc Services, ping Pinger) http.Handler {
	protected := http.NewServeMux()
	registerProtected(protected, svc)

	root := http.NewServeMux()

	// Health check stays unauthenticated so probes work without credentials.
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

	// Documentation is unauthenticated: it has to be readable while you are
	// still working out how to authenticate.
	apidocs.NewHandler().Register(root)

	if svc.Auth != nil {
		auth.NewHandler(svc.Auth).RegisterPublic(root)
	}
	if svc.PaperID != nil {
		paperid.NewHandler(svc.PaperID).RegisterPublic(root)
	}
	if svc.Upload != nil {
		upload.NewHandler(svc.Upload).RegisterFileServer(root)
	}

	// Everything else under /api/ needs a valid token.
	root.Handle("/api/", auth.RequireAuth(signer)(protected))

	chain := []httpx.Middleware{
		httpx.RequestID,
		httpx.Logger(log),
		httpx.Recoverer(log),
		httpx.CORS(cfg.AllowedOrigins),
	}

	if svc.Metrics != nil {
		// Unauthenticated, like /healthz: a scraper has no session, and the
		// route inventory it reveals is already public via /openapi.json. Set
		// METRICS_TOKEN to require a bearer token.
		root.Handle("GET /metrics", metrics.Handler(svc.Metrics, cfg.MetricsToken))

		// Instrumentation sits INSIDE Recoverer so a panicking handler is still
		// counted — a 500 that never reaches the metrics is the one you most
		// need to see. It also runs after the mux has matched, which is what
		// makes r.Pattern available as the route label.
		chain = append(chain, metrics.NewHTTP(svc.Metrics).Middleware(protected, root))
	}

	return httpx.Chain(root, chain...)
}

func registerProtected(mux *http.ServeMux, svc Services) {
	if svc.Auth != nil {
		auth.NewHandler(svc.Auth).RegisterProtected(mux)
	}
	if svc.Invoice != nil {
		invoice.NewHandler(svc.Invoice).Register(mux)
	}
	if svc.Payment != nil {
		payment.NewHandler(svc.Payment).Register(mux)
	}
	if svc.Member != nil {
		member.NewHandler(svc.Member).Register(mux)
	}
	if svc.Chapter != nil {
		chapter.NewHandler(svc.Chapter).Register(mux)
	}
	if svc.Settings != nil {
		settings.NewHandler(svc.Settings).Register(mux)
	}
	if svc.Audit != nil {
		audit.NewHandler(svc.Audit).Register(mux)
	}
	if svc.Dashboard != nil {
		dashboard.NewHandler(svc.Dashboard).Register(mux)
	}
	if svc.Upload != nil {
		upload.NewHandler(svc.Upload).Register(mux)
	}
	if svc.Sync != nil {
		sync.NewHandler(svc.Sync).Register(mux)
	}
	if svc.PaperID != nil {
		paperid.NewHandler(svc.PaperID).RegisterProtected(mux)
	}
	if svc.Blackbox != nil {
		blackbox.NewHandler(svc.Blackbox).Register(mux)
	}
}
