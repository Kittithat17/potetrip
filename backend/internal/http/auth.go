package http

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/raisin/travel-split/backend/internal/domain/auth"
	"github.com/raisin/travel-split/backend/internal/platform/apperr"
)

const (
	// The session cookie. Over https it gets the __Host- prefix, which makes
	// browsers insist on Secure, Path=/ and no Domain (no sibling subdomain
	// can plant or overwrite it).
	sessionCookie       = "ts_session"
	sessionCookieSecure = "__Host-ts_session"
	// Short-lived cookie carrying the OAuth state and PKCE verifier between
	// /google/start and /google/callback.
	oauthCookie     = "ts_oauth"
	oauthCookiePath = "/api/v1/auth/google"

	// CSRFHeader must accompany every state-changing request. Browsers can't
	// add a custom header to a cross-site form post, and cross-origin fetch
	// with it needs a CORS preflight we don't grant.
	CSRFHeader = "X-Requested-With"
	CSRFValue  = "travel-split"
)

// AuthConfig selects how people sign in.
type AuthConfig struct {
	Provider  string       // "google" or "mock"
	Google    *auth.Google // set when Provider is "google"
	PublicURL string       // e.g. https://split.example; https makes cookies Secure
}

func (c AuthConfig) secure() bool { return strings.HasPrefix(c.PublicURL, "https://") }

func (c AuthConfig) cookieName() string {
	if c.secure() {
		return sessionCookieSecure
	}
	return sessionCookie
}

const userKey ctxKey = 100

func userFrom(ctx context.Context) auth.User { return ctx.Value(userKey).(auth.User) }

// ---- middleware ----

// apiHeaders keeps API responses out of caches and stops browsers from
// guessing content types or leaking URLs (the OAuth callback carries a code).
func apiHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Cache-Control", "no-store")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

// csrf rejects state-changing requests that didn't come from our own pages.
func csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if r.Header.Get(CSRFHeader) != CSRFValue || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				writeErr(w, r, apperr.Forbidden("csrf", "คำขอนี้ไม่ได้มาจากแอป"))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// requireUser lets only signed-in people through and slides their session.
func (s *Server) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string
		if c, err := r.Cookie(s.AuthCfg.cookieName()); err == nil {
			token = c.Value
		}
		u, renewed, err := s.Auth.Session(r.Context(), token)
		if err != nil {
			if token != "" {
				s.clearSession(w) // stale cookie; stop sending it
			}
			writeErr(w, r, err)
			return
		}
		if !renewed.IsZero() {
			s.setSession(w, token, renewed)
		}
		next.ServeHTTP(w, with(r, userKey, u))
	})
}

func (s *Server) setSession(w http.ResponseWriter, token string, exp time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name: s.AuthCfg.cookieName(), Value: token, Path: "/",
		Expires: exp, MaxAge: int(time.Until(exp).Seconds()),
		HttpOnly: true, Secure: s.AuthCfg.secure(), SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: s.AuthCfg.cookieName(), Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: s.AuthCfg.secure(), SameSite: http.SameSiteLaxMode,
	})
}

// ---- handlers ----

func (s *Server) authConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"provider": s.AuthCfg.Provider, "passkeys": s.Passkeys != nil})
}

type oauthState struct {
	State    string `json:"s"`
	Verifier string `json:"v"`
	Next     string `json:"n"`
}

// googleStart sends the browser to Google's account chooser.
func (s *Server) googleStart(w http.ResponseWriter, r *http.Request) {
	if s.AuthCfg.Provider != "google" || s.AuthCfg.Google == nil {
		http.NotFound(w, r)
		return
	}
	st := oauthState{State: randomString(), Verifier: oauth2.GenerateVerifier(), Next: safeNext(r.URL.Query().Get("next"))}
	raw, _ := json.Marshal(st)
	http.SetCookie(w, &http.Cookie{
		Name: oauthCookie, Value: base64.RawURLEncoding.EncodeToString(raw), Path: oauthCookiePath,
		MaxAge: 600, HttpOnly: true, Secure: s.AuthCfg.secure(), SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, s.AuthCfg.Google.AuthURL(st.State, st.Verifier), http.StatusFound)
}

// googleCallback finishes sign-in and returns the person to where they were.
// Failures go back to the login screen with a reason, never to an error page.
func (s *Server) googleCallback(w http.ResponseWriter, r *http.Request) {
	if s.AuthCfg.Provider != "google" || s.AuthCfg.Google == nil {
		http.NotFound(w, r)
		return
	}
	fail := func(reason string) { http.Redirect(w, r, "/?login_error="+reason, http.StatusFound) }

	var st oauthState
	c, err := r.Cookie(oauthCookie)
	if err == nil {
		raw, derr := base64.RawURLEncoding.DecodeString(c.Value)
		if derr != nil || json.Unmarshal(raw, &st) != nil {
			st = oauthState{}
		}
	}
	// One use only.
	http.SetCookie(w, &http.Cookie{Name: oauthCookie, Value: "", Path: oauthCookiePath, MaxAge: -1, HttpOnly: true, Secure: s.AuthCfg.secure(), SameSite: http.SameSiteLaxMode})

	q := r.URL.Query()
	if q.Get("error") != "" { // e.g. the person pressed Cancel
		fail("cancelled")
		return
	}
	if st.State == "" || subtle.ConstantTimeCompare([]byte(st.State), []byte(q.Get("state"))) != 1 {
		fail("expired")
		return
	}
	profile, err := s.AuthCfg.Google.Exchange(r.Context(), q.Get("code"), st.Verifier)
	if err != nil {
		slog.WarnContext(r.Context(), "google sign-in failed", "err", err)
		fail("google")
		return
	}
	_, token, exp, err := s.Auth.Login(r.Context(), profile)
	if err != nil {
		slog.ErrorContext(r.Context(), "login failed", "err", err)
		fail("server")
		return
	}
	s.setSession(w, token, exp)
	http.Redirect(w, r, safeNext(st.Next), http.StatusFound)
}

// mockLogin signs in as anyone by name. Only enabled with AUTH_PROVIDER=mock,
// which config refuses on https.
func (s *Server) mockLogin(w http.ResponseWriter, r *http.Request) {
	if s.AuthCfg.Provider != "mock" {
		http.NotFound(w, r)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, r, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len(name) > 100 {
		writeErr(w, r, apperr.Invalid("invalid_name", "ใส่ชื่อ"))
		return
	}
	key := strings.ToLower(name)
	u, token, exp, err := s.Auth.Login(r.Context(), auth.Profile{
		Subject: "mock:" + key, Email: url.PathEscape(key) + "@mock.invalid", Name: name,
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	s.setSession(w, token, exp)
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) authMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, userFrom(r.Context()))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(s.AuthCfg.cookieName()); err == nil {
		if err := s.Auth.Logout(r.Context(), c.Value); err != nil {
			writeErr(w, r, err)
			return
		}
	}
	s.clearSession(w)
	w.WriteHeader(http.StatusNoContent)
}

// claimDevices moves trips this browser joined before sign-in (device tokens
// the app kept locally) onto the signed-in account.
func (s *Server) claimDevices(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tokens []string `json:"tokens"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, r, err)
		return
	}
	n, err := s.Trips.ClaimDevices(r.Context(), userFrom(r.Context()).ID, req.Tokens)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"claimed": n})
}

func (s *Server) myTrips(w http.ResponseWriter, r *http.Request) {
	ts, err := s.Trips.MyTrips(r.Context(), userFrom(r.Context()).ID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ts)
}

// safeNext keeps post-login redirects on our own site: a plain absolute path,
// never "//evil.example", "/\evil.example" or a URL with a scheme.
func safeNext(next string) string {
	if len(next) > 512 || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") ||
		strings.ContainsAny(next, "\\\r\n\t") || strings.HasPrefix(next, "/api/") {
		return "/"
	}
	u, err := url.Parse(next)
	if err != nil || u.Scheme != "" || u.Host != "" {
		return "/"
	}
	return next
}

func randomString() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
