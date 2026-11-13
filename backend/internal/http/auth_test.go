package http

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/raisin/travel-split/backend/internal/domain/auth"
	"github.com/raisin/travel-split/backend/internal/platform/testdb"
)

func TestSafeNext(t *testing.T) {
	ok := []string{"/", "/t/ABC234", "/t/ABC234/settle?x=1"}
	for _, n := range ok {
		if got := safeNext(n); got != n {
			t.Errorf("%q -> %q", n, got)
		}
	}
	bad := []string{"", "t/ABC", "//evil.example", "/\\evil.example", "https://evil.example", "/api/v1/auth/logout", "/a\r\nSet-Cookie: x", strings.Repeat("/a", 300)}
	for _, n := range bad {
		if got := safeNext(n); got != "/" {
			t.Errorf("%q -> %q, want /", n, got)
		}
	}
}

func TestSessionFlow(t *testing.T) {
	c := newAPI(t)

	// Signed out: every app endpoint refuses; health and sign-in config don't.
	for _, path := range []string{"/auth/me", "/me/trips", "/trips/ABCDEFGHJK", "/fx/currencies"} {
		if code := c.errCode("GET", path, nil, 401); code != "login_required" {
			t.Fatalf("%s: %s", path, code)
		}
	}
	var cfg map[string]any
	c.do("GET", "/auth/config", nil, 200, &cfg)
	if cfg["provider"] != "mock" || cfg["passkeys"] != true {
		t.Fatalf("%v", cfg)
	}

	// The session cookie is HttpOnly, Lax, site-wide.
	req := httptest.NewRequest("POST", "/api/v1/auth/mock", strings.NewReader(`{"name":"แบงก์"}`))
	c.authorize(req, "")
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	var ck *http.Cookie
	for _, x := range rec.Result().Cookies() {
		if x.Name == sessionCookie {
			ck = x
		}
	}
	if ck == nil || !ck.HttpOnly || ck.SameSite != http.SameSiteLaxMode || ck.Path != "/" || ck.MaxAge <= 0 || ck.Secure {
		t.Fatalf("cookie %+v", ck)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("auth responses must not be cached")
	}
	c.me = ck.Value

	var me auth.User
	c.do("GET", "/auth/me", nil, 200, &me)
	if me.Name != "แบงก์" {
		t.Fatalf("%+v", me)
	}

	// Writes need the anti-CSRF header, and are refused from other sites.
	for _, h := range []map[string]string{{}, {CSRFHeader: CSRFValue, "Sec-Fetch-Site": "cross-site"}} {
		req := httptest.NewRequest("POST", "/api/v1/trips", strings.NewReader(`{"name":"x"}`))
		req.AddCookie(&http.Cookie{Name: sessionCookie, Value: c.me})
		for k, v := range h {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		c.h.ServeHTTP(rec, req)
		if rec.Code != 403 || !strings.Contains(rec.Body.String(), `"csrf"`) {
			t.Fatalf("%v: %d %s", h, rec.Code, rec.Body)
		}
	}

	// The creator's name defaults to the account name; the trip is listed as theirs.
	var created struct {
		Member memberResp `json:"member"`
	}
	c.do("POST", "/trips", map[string]any{"name": "เชียงใหม่"}, 201, &created)
	if created.Member.Name != "แบงก์" {
		t.Fatalf("%+v", created)
	}
	var trips []map[string]any
	c.do("GET", "/me/trips", nil, 200, &trips)
	if len(trips) != 1 || trips[0]["name"] != "เชียงใหม่" || trips[0]["is_owner"] != true || trips[0]["status"] != "approved" {
		t.Fatalf("%+v", trips)
	}

	// The owner can't leave; a member can, and is then out.
	if code := c.errCode("DELETE", "/trips/"+trips[0]["id"].(string)+"/me", nil, 409); code != "owner_cannot_leave" {
		t.Fatal(code)
	}
	owner := c.me
	friend, reqID := c.join(trips[0]["id"].(string), map[string]any{"name": "ฝน"})
	c.do("POST", "/trips/"+trips[0]["id"].(string)+"/join-requests/"+reqID+"/approve", nil, 200, nil)
	c.me = friend
	c.do("DELETE", "/trips/"+trips[0]["id"].(string)+"/me", nil, 204, nil)
	if code := c.errCode("GET", "/trips/"+trips[0]["id"].(string)+"/expenses", nil, 403); code != "not_a_member" {
		t.Fatal(code)
	}
	c.do("GET", "/me/trips", nil, 200, &trips)
	if len(trips) != 0 {
		t.Fatalf("left trip still listed: %+v", trips)
	}
	c.me = owner

	// Logout kills the session server-side, not just the cookie.
	c.do("POST", "/auth/logout", nil, 204, nil)
	if code := c.errCode("GET", "/auth/me", nil, 401); code != "login_required" {
		t.Fatal(code)
	}
}

func TestClaimLegacyDevice(t *testing.T) {
	pool := testdb.New(t)
	c := &apiClient{t: t, h: NewServer(testDeps(pool)).Routes()}
	tr, _, _ := c.newTrip("เก่า", "แบงก์")

	// A device token from before sign-in existed, approved into the trip.
	token, hash := auth.NewToken()
	if _, err := pool.Exec(t.Context(), `INSERT INTO devices (trip_id, token_hash, status, member_id, requested_name)
		SELECT id, $2, 'approved', owner_member_id, 'แบงก์' FROM trips WHERE id = $1`, tr.ID, hash); err != nil {
		t.Fatal(err)
	}

	c.me = c.login("แบงก์ใหม่")
	var out map[string]int
	c.do("POST", "/auth/claim", map[string]any{"tokens": []string{token, "garbage", ""}}, 200, &out)
	if out["claimed"] != 1 {
		t.Fatalf("%v", out)
	}
	var me map[string]any
	c.do("GET", "/trips/"+tr.ID+"/me", nil, 200, &me)
	if me["status"] != "approved" || me["is_owner"] != true {
		t.Fatalf("%+v", me)
	}
	// The token is spent: nobody can claim it again.
	c.me = c.login("someone")
	c.do("POST", "/auth/claim", map[string]any{"tokens": []string{token}}, 200, &out)
	if out["claimed"] != 0 {
		t.Fatalf("claimed twice: %v", out)
	}
}

func TestGoogleRoutes(t *testing.T) {
	pool := testdb.New(t)
	d := testDeps(pool)
	d.AuthCfg = AuthConfig{Provider: "google", PublicURL: "https://split.example",
		Google: auth.NewGoogle("client-1", "secret", "https://split.example/api/v1/auth/google/callback")}
	h := NewServer(d).Routes()

	// Mock sign-in is off when Google is configured.
	req := httptest.NewRequest("POST", "/api/v1/auth/mock", strings.NewReader(`{"name":"x"}`))
	req.Header.Set(CSRFHeader, CSRFValue)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Fatalf("mock login with google: %d", rec.Code)
	}

	// Start: redirect to Google with state, and an HttpOnly Secure cookie holding it.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/auth/google/start?next=//evil.example", nil))
	if rec.Code != 302 || !strings.HasPrefix(rec.Header().Get("Location"), "https://accounts.google.com/") {
		t.Fatalf("start: %d %s", rec.Code, rec.Header().Get("Location"))
	}
	loc, _ := url.Parse(rec.Header().Get("Location"))
	var st *http.Cookie
	for _, x := range rec.Result().Cookies() {
		if x.Name == oauthCookie {
			st = x
		}
	}
	if st == nil || !st.HttpOnly || !st.Secure || st.Path != oauthCookiePath {
		t.Fatalf("state cookie %+v", st)
	}
	raw, _ := base64.RawURLEncoding.DecodeString(st.Value)
	var saved oauthState
	_ = json.Unmarshal(raw, &saved)
	if saved.State != loc.Query().Get("state") || saved.Next != "/" {
		t.Fatalf("state %+v vs %s", saved, loc.Query().Get("state"))
	}

	// Callback with a forged state never reaches Google and signs nobody in.
	req = httptest.NewRequest("GET", "/api/v1/auth/google/callback?code=x&state=forged", nil)
	req.AddCookie(st)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 302 || rec.Header().Get("Location") != "/?login_error=expired" {
		t.Fatalf("forged state: %d %s", rec.Code, rec.Header().Get("Location"))
	}
	for _, x := range rec.Result().Cookies() {
		if x.Name == sessionCookieSecure && x.Value != "" {
			t.Fatal("session issued on forged state")
		}
	}
	// Pressing Cancel at Google.
	req = httptest.NewRequest("GET", "/api/v1/auth/google/callback?error=access_denied&state="+saved.State, nil)
	req.AddCookie(st)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Location") != "/?login_error=cancelled" {
		t.Fatalf("cancel: %s", rec.Header().Get("Location"))
	}
}
