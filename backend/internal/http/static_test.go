package http

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSPA(t *testing.T) {
	dir := t.TempDir()
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html>app"), 0o644))
	must(os.MkdirAll(filepath.Join(dir, "assets"), 0o755))
	must(os.WriteFile(filepath.Join(dir, "assets", "index-abc.js"), []byte("js"), 0o644))
	must(os.WriteFile(filepath.Join(dir, "sw.js"), []byte("sw"), 0o644))
	secret := filepath.Join(filepath.Dir(dir), "secret.txt")
	must(os.WriteFile(secret, []byte("nope"), 0o644))

	h := NewServer(Deps{DB: fakePinger{}, StaticDir: dir, AuthCfg: AuthConfig{PublicURL: "https://split.example"}}).Routes()
	get := func(method, p string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, p, nil))
		return rec
	}

	for _, p := range []string{"/", "/t/ABCDEFGH23", "/t/ABCDEFGH23/settle", "/assets"} {
		rec := get("GET", p)
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), "app") || rec.Header().Get("Cache-Control") != "no-cache" {
			t.Fatalf("%s: %d %q %q", p, rec.Code, rec.Body, rec.Header().Get("Cache-Control"))
		}
		if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") || rec.Header().Get("Strict-Transport-Security") == "" {
			t.Fatalf("%s: security headers missing: %v", p, rec.Header())
		}
	}
	if rec := get("GET", "/assets/index-abc.js"); rec.Body.String() != "js" || !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("asset: %q %q", rec.Body, rec.Header().Get("Cache-Control"))
	}
	if rec := get("GET", "/sw.js"); rec.Body.String() != "sw" || rec.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("sw: %q %q", rec.Body, rec.Header().Get("Cache-Control"))
	}
	// Nothing outside the web root is reachable.
	for _, p := range []string{"/../secret.txt", "/%2e%2e/secret.txt", "/assets/../../secret.txt"} {
		if rec := get("GET", p); strings.Contains(rec.Body.String(), "nope") {
			t.Fatalf("%s leaked a file outside the web root", p)
		}
	}
	// Unknown API paths stay JSON, writes to pages are refused, health still works.
	if rec := get("GET", "/api/v1/nope"); rec.Code != 404 || !strings.Contains(rec.Body.String(), `"not_found"`) {
		t.Fatalf("api 404: %d %s", rec.Code, rec.Body)
	}
	if rec := get("POST", "/t/x"); rec.Code != 405 {
		t.Fatalf("post page: %d", rec.Code)
	}
	if rec := get("GET", "/healthz"); rec.Code != 200 {
		t.Fatalf("healthz: %d", rec.Code)
	}
}
