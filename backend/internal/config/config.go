// Package config loads runtime configuration from environment variables.
package config

import (
	"errors"
	"fmt"
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
}

func Load() (Config, error) {
	cfg := Config{
		Port:         getenv("PORT", "8080"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		SlipProvider: getenv("SLIP_PROVIDER", "gemini"),
		GeminiAPIKey: os.Getenv("GEMINI_API_KEY"),
		GeminiModel:  os.Getenv("GEMINI_MODEL"),
		FXBaseURL:    getenv("FX_BASE_URL", "https://api.frankfurter.dev/v1"),
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL is required")
	}

	conc, err := strconv.Atoi(getenv("SLIP_CONCURRENCY", "3"))
	if err != nil || conc < 1 {
		return cfg, fmt.Errorf("SLIP_CONCURRENCY must be a positive integer")
	}
	cfg.SlipConcurrency = conc

	for _, o := range strings.Split(getenv("CORS_ORIGINS", "http://localhost:5173"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			cfg.CORSOrigins = append(cfg.CORSOrigins, o)
		}
	}
	return cfg, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
