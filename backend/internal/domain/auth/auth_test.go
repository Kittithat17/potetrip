package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewToken(t *testing.T) {
	a, ha := NewToken()
	b, hb := NewToken()
	if a == b || len(a) < 40 {
		t.Fatalf("weak tokens %q %q", a, b)
	}
	if string(HashToken(a)) != string(ha) || string(ha) == string(hb) {
		t.Fatal("hash mismatch")
	}
}

func idToken(claims map[string]any) string {
	enc := func(v any) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	return enc(map[string]string{"alg": "RS256"}) + "." + enc(claims) + ".sig"
}

func TestGoogleProfile(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	g := NewGoogle("client-1", "secret", "http://localhost:5173/api/v1/auth/google/callback")
	g.now = func() time.Time { return now }
	good := func() map[string]any {
		return map[string]any{
			"iss": "https://accounts.google.com", "aud": "client-1", "sub": "1234", "exp": now.Unix() + 60,
			"email": "bank@gmail.com", "email_verified": true, "name": "Bank", "picture": "https://lh3.googleusercontent.com/a",
		}
	}

	p, err := g.profile(idToken(good()))
	if err != nil || p.Subject != "1234" || p.Email != "bank@gmail.com" || p.Name != "Bank" {
		t.Fatalf("%+v %v", p, err)
	}

	bad := map[string]func(map[string]any){
		"other issuer":   func(c map[string]any) { c["iss"] = "https://evil.example" },
		"other client":   func(c map[string]any) { c["aud"] = "client-2" },
		"expired":        func(c map[string]any) { c["exp"] = now.Unix() },
		"no subject":     func(c map[string]any) { c["sub"] = "" },
		"unverified":     func(c map[string]any) { c["email_verified"] = false },
		"missing verify": func(c map[string]any) { delete(c, "email_verified") },
	}
	for name, mutate := range bad {
		c := good()
		mutate(c)
		if _, err := g.profile(idToken(c)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	for _, junk := range []string{"", "a.b", "a.!!!.c", "a." + base64.RawURLEncoding.EncodeToString([]byte("[]")) + ".c"} {
		if _, err := g.profile(junk); err == nil {
			t.Errorf("%q: accepted", junk)
		}
	}
}

func TestAuthURL(t *testing.T) {
	g := NewGoogle("client-1", "secret", "http://localhost:5173/cb")
	u := g.AuthURL("state-1", "verifier-verifier-verifier-verifier-verifier")
	for _, want := range []string{"accounts.google.com", "client_id=client-1", "state=state-1", "code_challenge_method=S256", "code_challenge=", "scope=openid+email+profile", "redirect_uri=http%3A%2F%2Flocalhost%3A5173%2Fcb"} {
		if !strings.Contains(u, want) {
			t.Errorf("auth url %s lacks %s", u, want)
		}
	}
	if strings.Contains(u, "verifier-verifier") || strings.Contains(u, "secret") {
		t.Fatal("auth url leaks the PKCE verifier or client secret")
	}
}
