package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver for goose
	"github.com/pressly/goose/v3"

	"github.com/raisin/travel-split/backend/internal/domain/auth"
	"github.com/raisin/travel-split/backend/internal/domain/expense"
	"github.com/raisin/travel-split/backend/internal/domain/fx"
	"github.com/raisin/travel-split/backend/internal/domain/settlement"
	"github.com/raisin/travel-split/backend/internal/domain/slip"
	"github.com/raisin/travel-split/backend/internal/domain/trip"
	apihttp "github.com/raisin/travel-split/backend/internal/http"
	"github.com/raisin/travel-split/backend/internal/platform/config"
	"github.com/raisin/travel-split/backend/internal/platform/db"
	"github.com/raisin/travel-split/backend/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.MigrateOnStart {
		if err := migrate(ctx, cfg.DatabaseURL); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	reader, err := newSlipReader(ctx, cfg)
	if err != nil {
		return err
	}
	var slips *slip.Service
	if reader != nil {
		slips = slip.NewService(reader, db.New(pool), cfg.SlipConcurrency)
	}

	authCfg := apihttp.AuthConfig{Provider: cfg.AuthProvider, PublicURL: cfg.PublicURL}
	switch cfg.AuthProvider {
	case "google":
		authCfg.Google = auth.NewGoogle(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.PublicURL+"/api/v1/auth/google/callback")
		slog.Info("sign-in with Google", "redirect_url", cfg.PublicURL+"/api/v1/auth/google/callback")
	case "mock":
		slog.Warn("MOCK sign-in (AUTH_PROVIDER=mock): anyone can sign in as anyone; local dev only")
	}

	authSvc := auth.NewService(pool)
	passkeys, err := auth.NewPasskeys(authSvc, cfg.PublicURL)
	if err != nil {
		return err
	}

	rates := fx.NewService(fx.NewFrankfurter(cfg.FXBaseURL), db.New(pool))
	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: apihttp.NewServer(apihttp.Deps{
			DB:          pool,
			Auth:        authSvc,
			Passkeys:    passkeys,
			AuthCfg:     authCfg,
			Trips:       trip.NewService(pool),
			Expenses:    expense.NewService(pool, rates),
			Settlements: settlement.NewService(pool),
			Slips:       slips,
			FX:          rates,
			CORSOrigins: cfg.CORSOrigins,
			StaticDir:   cfg.StaticDir,
			TrustProxy:  cfg.TrustProxy,
		}).Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// migrate applies pending migrations (embedded in the binary) with goose.
func migrate(ctx context.Context, dsn string) error {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer func() { _ = sqlDB.Close() }()
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		return err
	}
	results, err := provider.Up(ctx)
	for _, r := range results {
		slog.Info("migration applied", "version", r.Source.Version, "file", r.Source.Path, "took", r.Duration)
	}
	return err
}

// newSlipReader picks the AI provider from SLIP_PROVIDER. A missing API key
// disables slip reading instead of failing startup, so the rest of the app
// still works in dev.
func newSlipReader(ctx context.Context, cfg config.Config) (slip.Reader, error) {
	switch cfg.SlipProvider {
	case "gemini":
		if cfg.GeminiAPIKey == "" || cfg.GeminiModel == "" {
			slog.Warn("slip reading disabled: set GEMINI_API_KEY and GEMINI_MODEL to enable it")
			return nil, nil
		}
		r, err := slip.NewGeminiReader(ctx, cfg.GeminiAPIKey, cfg.GeminiModel)
		if err != nil {
			return nil, err
		}
		slog.Info("slip reading enabled", "provider", "gemini", "model", cfg.GeminiModel, "concurrency", cfg.SlipConcurrency)
		// Spec 6.3: retry 429s with exponential backoff, at most 3 times.
		return slip.NewRetryReader(r, 3, 2*time.Second), nil
	case "mock":
		slog.Warn("slip reading uses MOCK results (SLIP_PROVIDER=mock); do not use in production")
		return slip.MockReader{Delay: 1500 * time.Millisecond}, nil
	case "", "none":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown SLIP_PROVIDER %q", cfg.SlipProvider)
	}
}
