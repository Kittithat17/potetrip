package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("PORT", "")
	t.Setenv("SLIP_CONCURRENCY", "")
	t.Setenv("CORS_ORIGINS", "http://a.test, http://b.test")
	t.Setenv("AUTH_PROVIDER", "mock")
	t.Setenv("PUBLIC_URL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8080" || cfg.SlipConcurrency != 3 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.PublicURL != "http://localhost:5173" {
		t.Fatalf("public url: %q", cfg.PublicURL)
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
	t.Setenv("AUTH_PROVIDER", "mock")
	if _, err := Load(); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadAuth(t *testing.T) {
	cases := []struct {
		name, provider, id, secret, public string
		ok                                 bool
	}{
		{"google needs a client", "google", "", "", "https://split.example", false},
		{"google", "google", "id", "secret", "https://split.example/", true},
		{"mock in dev", "mock", "", "", "http://192.168.1.5:5173", true},
		{"mock never on https", "mock", "", "", "https://split.example", false},
		{"unknown provider", "github", "", "", "http://localhost:5173", false},
		{"public url with a path", "mock", "", "", "http://localhost:5173/app", false},
		{"public url without scheme", "mock", "", "", "localhost:5173", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://x")
			t.Setenv("AUTH_PROVIDER", tc.provider)
			t.Setenv("GOOGLE_CLIENT_ID", tc.id)
			t.Setenv("GOOGLE_CLIENT_SECRET", tc.secret)
			t.Setenv("PUBLIC_URL", tc.public)
			cfg, err := Load()
			if (err == nil) != tc.ok {
				t.Fatalf("err = %v, want ok=%v", err, tc.ok)
			}
			if tc.ok && cfg.PublicURL[len(cfg.PublicURL)-1] == '/' {
				t.Fatalf("trailing slash kept: %q", cfg.PublicURL)
			}
		})
	}
}

func TestLoadTrimsPastedValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x?sslmode=require\n\n")
	t.Setenv("PUBLIC_URL", "https://split.example/\n")
	t.Setenv("AUTH_PROVIDER", "google")
	t.Setenv("GOOGLE_CLIENT_ID", " id\n")
	t.Setenv("GOOGLE_CLIENT_SECRET", "secret\r\n")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != "postgres://x?sslmode=require" || cfg.PublicURL != "https://split.example" || cfg.GoogleClientID != "id" || cfg.GoogleClientSecret != "secret" {
		t.Fatalf("%+v", cfg)
	}
}
