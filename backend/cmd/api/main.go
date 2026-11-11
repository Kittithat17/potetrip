package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raisin/travel-split/backend/internal/config"
	"github.com/raisin/travel-split/backend/internal/db"
	"github.com/raisin/travel-split/backend/internal/expense"
	apihttp "github.com/raisin/travel-split/backend/internal/http"
	"github.com/raisin/travel-split/backend/internal/settlement"
	"github.com/raisin/travel-split/backend/internal/slip"
	"github.com/raisin/travel-split/backend/internal/trip"
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

	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: apihttp.NewServer(apihttp.Deps{
			DB:          pool,
			Trips:       trip.NewService(pool),
			Expenses:    expense.NewService(pool),
			Settlements: settlement.NewService(pool),
			Slips:       slips,
			CORSOrigins: cfg.CORSOrigins,
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
