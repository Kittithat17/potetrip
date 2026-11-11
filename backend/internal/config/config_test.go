package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("PORT", "")
	t.Setenv("SLIP_CONCURRENCY", "")
	t.Setenv("CORS_ORIGINS", "http://a.test, http://b.test")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8080" || cfg.SlipConcurrency != 3 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if len(cfg.CORSOrigins) != 2 || cfg.CORSOrigins[1] != "http://b.test" {
		t.Fatalf("cors origins: %v", cfg.CORSOrigins)
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadRejectsBadConcurrency(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("SLIP_CONCURRENCY", "0")
	if _, err := Load(); err == nil {
		t.Fatal("expected error")
	}
}
