package http

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/raisin/travel-split/backend/internal/domain/auth"
	"github.com/raisin/travel-split/backend/internal/platform/apperr"
)

// The WebAuthn challenge token lives in an HttpOnly cookie between a Begin
// and its Finish call, scoped to the passkey endpoints and five minutes.
const (
	passkeyCookie     = "ts_webauthn"
	passkeyCookiePath = "/api/v1/auth/passkey"
	maxPasskeyBody    = 64 << 10
)

func (s *Server) setChallenge(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: passkeyCookie, Value: token, Path: passkeyCookiePath, MaxAge: 300,
		HttpOnly: true, Secure: s.AuthCfg.secure(), SameSite: http.SameSiteStrictMode,
	})
}

// challenge reads and clears the challenge cookie (each challenge is single use).
func (s *Server) challenge(w http.ResponseWriter, r *http.Request) string {
	http.SetCookie(w, &http.Cookie{
		Name: passkeyCookie, Value: "", Path: passkeyCookiePath, MaxAge: -1,
		HttpOnly: true, Secure: s.AuthCfg.secure(), SameSite: http.SameSiteStrictMode,
	})
	if c, err := r.Cookie(passkeyCookie); err == nil {
		return c.Value
	}
	return ""
}

func readCredential(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxPasskeyBody))
	if err != nil || len(body) == 0 {
		return nil, apperr.Invalid("invalid_json", "ต้องส่งข้อมูล passkey")
	}
	return body, nil
}

func (s *Server) passkeysEnabled(w http.ResponseWriter, r *http.Request) bool {
	if s.Passkeys == nil {
		http.NotFound(w, r)
		return false
	}
	return true
}

// POST /auth/passkey/signup/begin {name}: options for creating a passkey for a new account.
func (s *Server) passkeySignupBegin(w http.ResponseWriter, r *http.Request) {
	if !s.passkeysEnabled(w, r) {
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, r, err)
		return
	}
	opts, token, err := s.Passkeys.BeginSignup(r.Context(), req.Name)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	s.setChallenge(w, token)
	writeJSON(w, http.StatusOK, opts)
}

// POST /auth/passkey/signup/finish <RegistrationResponseJSON>: makes the account and signs in.
func (s *Server) passkeySignupFinish(w http.ResponseWriter, r *http.Request) {
	if !s.passkeysEnabled(w, r) {
		return
	}
	token := s.challenge(w, r)
	body, err := readCredential(w, r)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	u, sess, exp, err := s.Passkeys.FinishSignup(r.Context(), token, body, auth.DeviceName(r.UserAgent()))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	s.setSession(w, sess, exp)
	writeJSON(w, http.StatusCreated, u)
}

// POST /auth/passkey/login/begin: options for signing in with any passkey of this site.
func (s *Server) passkeyLoginBegin(w http.ResponseWriter, r *http.Request) {
	if !s.passkeysEnabled(w, r) {
		return
	}
	opts, token, err := s.Passkeys.BeginLogin(r.Context())
	if err != nil {
		writeErr(w, r, err)
		return
	}
	s.setChallenge(w, token)
	writeJSON(w, http.StatusOK, opts)
}

// POST /auth/passkey/login/finish <AuthenticationResponseJSON>: signs in.
func (s *Server) passkeyLoginFinish(w http.ResponseWriter, r *http.Request) {
	if !s.passkeysEnabled(w, r) {
		return
	}
	token := s.challenge(w, r)
	body, err := readCredential(w, r)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	u, sess, exp, err := s.Passkeys.FinishLogin(r.Context(), token, body)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	s.setSession(w, sess, exp)
	writeJSON(w, http.StatusOK, u)
}

// POST /auth/passkey/register/begin (signed in): options for adding a passkey to this account.
func (s *Server) passkeyRegisterBegin(w http.ResponseWriter, r *http.Request) {
	if !s.passkeysEnabled(w, r) {
		return
	}
	opts, token, err := s.Passkeys.BeginRegister(r.Context(), userFrom(r.Context()).ID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	s.setChallenge(w, token)
	writeJSON(w, http.StatusOK, opts)
}

// POST /auth/passkey/register/finish (signed in) <RegistrationResponseJSON>.
func (s *Server) passkeyRegisterFinish(w http.ResponseWriter, r *http.Request) {
	if !s.passkeysEnabled(w, r) {
		return
	}
	token := s.challenge(w, r)
	body, err := readCredential(w, r)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	if err := s.Passkeys.FinishRegister(r.Context(), userFrom(r.Context()).ID, token, body, auth.DeviceName(r.UserAgent())); err != nil {
		writeErr(w, r, err)
		return
	}
	s.listPasskeys(w, r)
}

// GET /auth/passkeys (signed in).
func (s *Server) listPasskeys(w http.ResponseWriter, r *http.Request) {
	if !s.passkeysEnabled(w, r) {
		return
	}
	list, err := s.Passkeys.List(r.Context(), userFrom(r.Context()).ID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// DELETE /auth/passkeys/{id} (signed in).
func (s *Server) deletePasskey(w http.ResponseWriter, r *http.Request) {
	if !s.passkeysEnabled(w, r) {
		return
	}
	if err := s.Passkeys.Delete(r.Context(), userFrom(r.Context()).ID, chi.URLParam(r, "id")); err != nil {
		writeErr(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
