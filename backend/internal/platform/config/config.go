// Package config loads runtime configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port            string
	DatabaseURL     string
	SlipProvider    string
	GeminiAPIKey    string
	GeminiModel     string
	SlipConcurrency int
	CORSOrigins     []string
	FXBaseURL       string

	// AuthProvider is "google" (real sign-in) or "mock" (dev only: sign in
	// with any name, no Google account needed).
	AuthProvider       string
	GoogleClientID     string
	GoogleClientSecret string
	// PublicURL is where people open the app (scheme://host[:port], no path).
	// The Google redirect URL is built from it, and an https URL makes the
	// session cookie Secure.
	PublicURL string

	// StaticDir is the built web app the API serves (production image);
	// empty in dev, where Vite serves the frontend.
	StaticDir string
	// MigrateOnStart applies pending DB migrations before serving.
	MigrateOnStart bool
}

func Load() (Config, error) {
	cfg := Config{
		Port:         getenv("PORT", "8080"),
		DatabaseURL:  env("DATABASE_URL"),
		SlipProvider: getenv("SLIP_PROVIDER", "gemini"),
		GeminiAPIKey: env("GEMINI_API_KEY"),
		GeminiModel:  env("GEMINI_MODEL"),
		FXBaseURL:    getenv("FX_BASE_URL", "https://api.frankfurter.dev/v1"),

		AuthProvider:       getenv("AUTH_PROVIDER", "google"),
		GoogleClientID:     env("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: env("GOOGLE_CLIENT_SECRET"),
		PublicURL:          strings.TrimRight(getenv("PUBLIC_URL", "http://localhost:5173"), "/"),
		StaticDir:          env("STATIC_DIR"),
		MigrateOnStart:     env("MIGRATE_ON_START") == "true",
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL is required")
	}

	conc, err := strconv.Atoi(getenv("SLIP_CONCURRENCY", "3"))
	if err != nil || conc < 1 {
		return cfg, fmt.Errorf("SLIP_CONCURRENCY must be a positive integer")
	}
	cfg.SlipConcurrency = conc

	u, err := url.Parse(cfg.PublicURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || (u.Path != "" && u.Path != "/") {
		return cfg, fmt.Errorf("PUBLIC_URL must look like https://example.com")
	}
	switch cfg.AuthProvider {
	case "google":
		if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
			return cfg, errors.New("AUTH_PROVIDER=google needs GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET (or AUTH_PROVIDER=mock for local dev)")
		}
	case "mock":
		if u.Scheme == "https" {
			return cfg, errors.New("AUTH_PROVIDER=mock lets anyone sign in as anyone; it is refused when PUBLIC_URL is https")
		}
	default:
		return cfg, fmt.Errorf("unknown AUTH_PROVIDER %q (google | mock)", cfg.AuthProvider)
	}

	for _, o := range strings.Split(getenv("CORS_ORIGINS", "http://localhost:5173"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			cfg.CORSOrigins = append(cfg.CORSOrigins, o)
		}
	}
	return cfg, nil
}

// env reads a variable without surrounding spaces or newlines, which easily
// sneak in when values are pasted into a hosting dashboard.
func env(key string) string { return strings.TrimSpace(os.Getenv(key)) }

func getenv(key, def string) string {
	if v := env(key); v != "" {
		return v
	}
	return def
}
