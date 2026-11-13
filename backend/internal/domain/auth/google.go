package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

// Google runs the OAuth 2.0 authorization-code flow with PKCE against Google
// and reads the person from the returned OpenID Connect ID token.
type Google struct {
	cfg oauth2.Config
	now func() time.Time
}

// NewGoogle needs the OAuth client from Google Cloud Console and the exact
// redirect URL registered there (…/api/v1/auth/google/callback).
func NewGoogle(clientID, clientSecret, redirectURL string) *Google {
	return &Google{
		cfg: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     endpoints.Google,
			Scopes:       []string{"openid", "email", "profile"},
		},
		now: time.Now,
	}
}

// AuthURL is where to send the browser. state ties the callback to this
// browser (CSRF); the verifier's challenge ties the code to this server (PKCE).
func (g *Google) AuthURL(state, verifier string) string {
	return g.cfg.AuthCodeURL(state,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)
}

// Exchange trades the callback's code for the person's profile.
func (g *Google) Exchange(ctx context.Context, code, verifier string) (Profile, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	tok, err := g.cfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return Profile{}, fmt.Errorf("google token exchange: %w", err)
	}
	raw, _ := tok.Extra("id_token").(string)
	if raw == "" {
		return Profile{}, errors.New("google: no id_token in response")
	}
	return g.profile(raw)
}

type idClaims struct {
	Iss           string `json:"iss"`
	Aud           string `json:"aud"`
	Sub           string `json:"sub"`
	Exp           int64  `json:"exp"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// profile reads the ID token's claims. The token came straight from Google's
// token endpoint over TLS in exchange for our client secret, so per OpenID
// Connect Core §3.1.3.7 its signature need not be re-checked; we still check
// who issued it, who it is for, and that it hasn't expired.
func (g *Google) profile(idToken string) (Profile, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return Profile{}, errors.New("google: malformed id_token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Profile{}, fmt.Errorf("google: id_token payload: %w", err)
	}
	var c idClaims
	if err := json.Unmarshal(payload, &c); err != nil {
		return Profile{}, fmt.Errorf("google: id_token claims: %w", err)
	}
	switch {
	case c.Iss != "https://accounts.google.com" && c.Iss != "accounts.google.com":
		return Profile{}, fmt.Errorf("google: unexpected issuer %q", c.Iss)
	case c.Aud != g.cfg.ClientID:
		return Profile{}, errors.New("google: id_token is for another client")
	case g.now().Unix() >= c.Exp:
		return Profile{}, errors.New("google: id_token expired")
	case c.Sub == "":
		return Profile{}, errors.New("google: id_token has no subject")
	case !c.EmailVerified:
		return Profile{}, errors.New("google: email not verified")
	}
	return Profile{Subject: c.Sub, Email: c.Email, Name: c.Name, Picture: c.Picture}, nil
}
