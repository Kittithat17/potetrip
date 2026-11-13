package http

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/raisin/travel-split/backend/internal/platform/testdb"
)

// The cryptographic ceremony itself is covered by go-webauthn and by the
// browser test with a virtual authenticator; these check our side of it.

// post sends a raw body with the anti-CSRF header and optional cookies.
func post(h http.Handler, path, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/api/v1"+path, strings.NewReader(body))
	req.Header.Set(CSRFHeader, CSRFValue)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func cookieNamed(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestPasskeyCeremonies(t *testing.T) {
	pool := testdb.New(t)
	h := NewServer(testDeps(pool)).Routes()

	// Sign-up options: discoverable credential, verified by biometrics/PIN,
	// for this site, with a challenge kept in a short-lived HttpOnly cookie.
	rec := post(h, "/auth/passkey/signup/begin", `{"name":"ฝน"}`)
	if rec.Code != 200 {
		t.Fatalf("signup begin: %d %s", rec.Code, rec.Body)
	}
	var opts struct {
		Challenge string              `json:"challenge"`
		RP        struct{ ID string } `json:"rp"`
		User      struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"user"`
		AuthenticatorSelection struct {
			ResidentKey      string `json:"residentKey"`
			UserVerification string `json:"userVerification"`
		} `json:"authenticatorSelection"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &opts); err != nil {
		t.Fatal(err)
	}
	if opts.RP.ID != "localhost" || opts.User.DisplayName != "ฝน" || opts.Challenge == "" ||
		opts.AuthenticatorSelection.ResidentKey != "required" || opts.AuthenticatorSelection.UserVerification != "required" {
		t.Fatalf("options %+v", opts)
	}
	if id, _ := base64.RawURLEncoding.DecodeString(opts.User.ID); len(id) != 16 {
		t.Fatalf("user handle should be the 16-byte account id, got %q", opts.User.ID)
	}
	ch := cookieNamed(rec, passkeyCookie)
	if ch == nil || !ch.HttpOnly || ch.SameSite != http.SameSiteStrictMode || ch.Path != passkeyCookiePath || ch.MaxAge != 300 {
		t.Fatalf("challenge cookie %+v", ch)
	}

	// A forged answer fails, and the challenge is gone afterwards (single use).
	if rec := post(h, "/auth/passkey/signup/finish", `{"id":"x","rawId":"x","type":"public-key","response":{}}`, ch); !strings.Contains(rec.Body.String(), "passkey_failed") {
		t.Fatalf("forged signup: %d %s", rec.Code, rec.Body)
	}
	if rec := post(h, "/auth/passkey/signup/finish", `{"id":"x"}`, ch); !strings.Contains(rec.Body.String(), "passkey_expired") {
		t.Fatalf("reused challenge: %d %s", rec.Code, rec.Body)
	}
	// No challenge at all, or one from another ceremony, is refused.
	if rec := post(h, "/auth/passkey/login/finish", `{"id":"x"}`); !strings.Contains(rec.Body.String(), "passkey_expired") {
		t.Fatalf("no challenge: %d %s", rec.Code, rec.Body)
	}
	rec = post(h, "/auth/passkey/signup/begin", `{"name":"x"}`)
	if rec := post(h, "/auth/passkey/login/finish", `{"id":"x"}`, cookieNamed(rec, passkeyCookie)); !strings.Contains(rec.Body.String(), "passkey_expired") {
		t.Fatalf("signup challenge used for login: %d %s", rec.Code, rec.Body)
	}
	if rec := post(h, "/auth/passkey/signup/begin", `{"name":"  "}`); rec.Code != 400 {
		t.Fatalf("blank name: %d", rec.Code)
	}

	// Login options ask for any passkey of this site (no username needed).
	rec = post(h, "/auth/passkey/login/begin", "")
	var login struct {
		RPID             string `json:"rpId"`
		UserVerification string `json:"userVerification"`
		AllowCredentials []any  `json:"allowCredentials"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &login)
	if rec.Code != 200 || login.RPID != "localhost" || login.UserVerification != "required" || len(login.AllowCredentials) != 0 {
		t.Fatalf("login begin: %d %s", rec.Code, rec.Body)
	}

	// Adding a passkey needs a signed-in user.
	if rec := post(h, "/auth/passkey/register/begin", ""); rec.Code != 401 {
		t.Fatalf("register signed out: %d", rec.Code)
	}
}

func TestPasskeyListAndLastMethodGuard(t *testing.T) {
	pool := testdb.New(t)
	c := &apiClient{t: t, h: NewServer(testDeps(pool)).Routes()}
	ctx := t.Context()

	// A passkey-only account (no Google) with one passkey.
	var userID string
	if err := pool.QueryRow(ctx, `INSERT INTO users (name) VALUES ('แบงก์') RETURNING id`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO passkeys (id, user_id, credential, name) VALUES ('\x0102', $1, '{}', 'iPhone')`, userID); err != nil {
		t.Fatal(err)
	}
	token := "test-session-token-0123456789012345678901234567"
	if _, err := pool.Exec(ctx, `INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (sha256($1::bytea), $2, now() + interval '1 day')`, token, userID); err != nil {
		t.Fatal(err)
	}
	c.me = token

	var me map[string]any
	c.do("GET", "/auth/me", nil, 200, &me)
	if me["email"] != nil || me["has_google"] != false || me["name"] != "แบงก์" {
		t.Fatalf("passkey-only user %+v", me)
	}
	var list []map[string]any
	c.do("GET", "/auth/passkeys", nil, 200, &list)
	if len(list) != 1 || list[0]["id"] != "AQI" || list[0]["name"] != "iPhone" {
		t.Fatalf("list %+v", list)
	}
	// Its only passkey can't be removed: the account would be locked out.
	if code := c.errCode("DELETE", "/auth/passkeys/AQI", nil, 409); code != "last_sign_in_method" {
		t.Fatal(code)
	}
	if code := c.errCode("DELETE", "/auth/passkeys/nope", nil, 400); code != "passkey_unknown" {
		t.Fatal(code)
	}

	// An account that also has Google can remove its passkeys.
	c.me = c.login("google user")
	var gid string
	c.do("GET", "/auth/me", nil, 200, &me)
	gid = me["id"].(string)
	if _, err := pool.Exec(ctx, `INSERT INTO passkeys (id, user_id, credential, name) VALUES ('\x0304', $1, '{}', 'Mac')`, gid); err != nil {
		t.Fatal(err)
	}
	// Someone else's passkey id is simply unknown to this account.
	if code := c.errCode("DELETE", "/auth/passkeys/AQI", nil, 400); code != "passkey_unknown" {
		t.Fatal(code)
	}
	c.do("DELETE", "/auth/passkeys/AwQ", nil, 204, nil)
	c.do("GET", "/auth/passkeys", nil, 200, &list)
	if len(list) != 0 {
		t.Fatalf("after delete %+v", list)
	}
}
