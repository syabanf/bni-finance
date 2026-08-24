// Command api serves the BNI Finance API: CRUD for every resource, local
// accounts, file uploads, and the public payment page.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/syabanf/bni-finance/backend/internal/api"
	"github.com/syabanf/bni-finance/backend/internal/audit"
	"github.com/syabanf/bni-finance/backend/internal/auth"
	"github.com/syabanf/bni-finance/backend/internal/blackbox"
	"github.com/syabanf/bni-finance/backend/internal/chapter"
	"github.com/syabanf/bni-finance/backend/internal/config"
	"github.com/syabanf/bni-finance/backend/internal/dashboard"
	"github.com/syabanf/bni-finance/backend/internal/database"
	"github.com/syabanf/bni-finance/backend/internal/importer"
	"github.com/syabanf/bni-finance/backend/internal/invoice"
	"github.com/syabanf/bni-finance/backend/internal/member"
	"github.com/syabanf/bni-finance/backend/internal/metrics"
	"github.com/syabanf/bni-finance/backend/internal/paperid"
	"github.com/syabanf/bni-finance/backend/internal/payment"
	"github.com/syabanf/bni-finance/backend/internal/reminder"
	"github.com/syabanf/bni-finance/backend/internal/renewal"
	"github.com/syabanf/bni-finance/backend/internal/settings"
	"github.com/syabanf/bni-finance/backend/internal/sync"
	"github.com/syabanf/bni-finance/backend/internal/upload"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	// httpx.Fail logs unexpected causes through the default logger, so point it
	// at ours before serving anything.
	slog.SetDefault(log)

	if err := run(log); err != nil {
		log.Error("server berhenti", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	signer, err := auth.NewSigner(cfg.JWTSecret, cfg.TokenTTL)
	if err != nil {
		return err
	}

	uploads, err := upload.NewStore(cfg.UploadDir, cfg.MaxUploadSize)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		return err
	}
	defer pool.Close()
	log.Info("terhubung ke database")

	// One recorder shared by every integration, so the blackbox page shows all
	// traffic in a single timeline.
	//
	// Riwayatnya disimpan ke Postgres: pertanyaan yang benar-benar diajukan
	// orang — "invoice ini dikirim kapan, dan Paper.id menjawab apa" — hampir
	// selalu datang berhari-hari setelah prosesnya di-restart.
	recorder := blackbox.New(cfg.BlackboxSize).
		WithStore(blackbox.NewRepository(pool, cfg.BlackboxRetain), log)

	// Metrics. The registry is per-process and passed explicitly rather than
	// kept in a package global, so tests get a clean one each time.
	reg := metrics.NewRegistry()
	metrics.RegisterRuntime(reg)
	metrics.RegisterPool(reg, func() metrics.PoolStats { return pool.Stat() })

	// The blackbox keeps the last N calls; these counters survive it and can
	// raise an alert, which a ring buffer cannot.
	integrationMetrics := metrics.NewIntegrations(reg)
	recorder.WithObserver(func(c blackbox.Call) {
		integrationMetrics.Record(c.Integration, c.Direction, c.Success, c.Duration.Seconds())
	})

	// Wire: repository → service → handler, one chain per resource.
	authSvc := auth.NewService(auth.NewRepository(pool), signer, cfg.QuickLoginEmails...)

	// Passwordless sign-in is a deliberate hole in authentication. Say so at
	// every start, naming the accounts, so it can never be on unnoticed.
	if authSvc.QuickLoginEnabled() {
		log.Warn("QUICK LOGIN AKTIF — akun berikut bisa masuk tanpa kata sandi; jangan dipakai di produksi",
			"akun", strings.Join(cfg.QuickLoginEmails, ", "))
	}

	if cfg.HasSeedAdmin() {
		created, err := authSvc.EnsureSeedAdmin(ctx, cfg.SeedAdminEmail, cfg.SeedAdminPassword, cfg.SeedAdminName)
		if err != nil {
			return err
		}
		if created {
			log.Info("admin awal dibuat", "email", cfg.SeedAdminEmail)
		}
	}

	paperSvc := paperid.NewService(paperid.NewRepository(pool),
		cfg.PaperIDBaseURL, cfg.PaperIDClientID, cfg.PaperIDClientSecret,
		cfg.PaperIDCallbackToken, recorder)

	handler := api.NewHandler(log, cfg, signer, api.Services{
		Auth:      authSvc,
		Invoice:   invoice.NewService(invoice.NewRepository(pool)),
		Payment:   payment.NewService(payment.NewRepository(pool)),
		Member:    member.NewService(member.NewRepository(pool)),
		Chapter:   chapter.NewService(chapter.NewRepository(pool)),
		Settings:  settings.NewService(settings.NewRepository(pool)),
		Audit:     audit.NewService(audit.NewRepository(pool)),
		Dashboard: dashboard.NewService(dashboard.NewRepository(pool)),
		Upload:    uploads,
		Sync:      sync.NewService(sync.NewRepository(pool), cfg.BNIVMURL, cfg.BNIVMToken, recorder),
		Importer:  importer.NewService(importer.NewRepository(pool)),
		Renewal:   renewal.NewService(renewal.NewRepository(pool)),
		PaperID:   paperSvc,
		Blackbox:  recorder,
		Metrics:   reg,
	}, pool.Ping)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Worker pengingat berjalan sebagai goroutine di proses yang sama, dan
	// berhenti saat context aplikasi dibatalkan — sama seperti server-nya.
	//
	// Bawaannya MATI (app_settings.reminder_worker_enabled = false). Worker ini
	// mengirim pesan sungguhan ke member dan membakar nomor invoice Paper.id
	// secara permanen, jadi menyalakannya harus keputusan sadar, bukan efek
	// samping dari sebuah deploy.
	workerCtx, hentikanWorker := context.WithCancel(ctx)
	defer hentikanWorker()
	go reminder.NewWorker(
		reminder.NewRepository(pool),
		reminder.NewPaperPengirim(paperSvc),
		log,
	).Jalankan(workerCtx)

	errCh := make(chan error, 1)
	go func() {
		log.Info("server berjalan",
			"port", cfg.Port,
			"uploads", uploads.Dir())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("mematikan server…")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
