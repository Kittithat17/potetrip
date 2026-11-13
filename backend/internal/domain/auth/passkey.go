package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/raisin/travel-split/backend/internal/platform/apperr"
	"github.com/raisin/travel-split/backend/internal/platform/db"
)

// Passkeys signs people in with WebAuthn passkeys (Face ID, fingerprint,
// device PIN). Nothing involved goes through Google, so it keeps working
// where Google is blocked. A passkey is bound to the site's domain.
//
// Every ceremony is two calls: Begin* returns options for the browser plus a
// one-time challenge token (kept in an HttpOnly cookie by the HTTP layer);
// Finish* checks the browser's signed answer against that challenge.
type Passkeys struct {
	svc *Service
	wa  *webauthn.WebAuthn
}

const (
	challengeTTL = 5 * time.Minute
	maxPasskeys  = 20 // per account
)

var (
	errPasskeyExpired = apperr.Invalid("passkey_expired", "หมดเวลายืนยันตัวตน ลองใหม่อีกครั้ง")
	errPasskeyFailed  = apperr.Invalid("passkey_failed", "ยืนยัน passkey ไม่สำเร็จ")
	errPasskeyUnknown = apperr.Invalid("passkey_unknown", "ไม่พบ passkey นี้ในระบบ อาจถูกลบไปแล้ว")
)

// NewPasskeys configures WebAuthn for the site at publicURL; the passkeys'
// relying party is its host name (e.g. split.example, or localhost in dev).
func NewPasskeys(svc *Service, publicURL string) (*Passkeys, error) {
	u, err := url.Parse(publicURL)
	if err != nil || u.Hostname() == "" {
		return nil, fmt.Errorf("passkeys: bad public URL %q", publicURL)
	}
	wa, err := webauthn.New(&webauthn.Config{
		RPID:          u.Hostname(),
		RPDisplayName: "Travel Split",
		RPOrigins:     []string{u.Scheme + "://" + u.Host},
	})
	if err != nil {
		return nil, fmt.Errorf("passkeys: %w", err)
	}
	return &Passkeys{svc: svc, wa: wa}, nil
}

// waUser adapts a user and their passkeys to go-webauthn. The user handle is
// the account's random UUID: stable, opaque, and never shown to anyone.
type waUser struct {
	u     db.User
	creds []webauthn.Credential
}

func (w waUser) WebAuthnID() []byte { return w.u.ID[:] }
func (w waUser) WebAuthnName() string {
	if w.u.Email != nil {
		return *w.u.Email
	}
	return w.u.Name
}
func (w waUser) WebAuthnDisplayName() string                { return w.u.Name }
func (w waUser) WebAuthnCredentials() []webauthn.Credential { return w.creds }

func (p *Passkeys) loadUser(ctx context.Context, id uuid.UUID) (waUser, error) {
	u, err := p.svc.q.GetUser(ctx, id)
	if err != nil {
		return waUser{}, err
	}
	rows, err := p.svc.q.ListPasskeys(ctx, id)
	if err != nil {
		return waUser{}, err
	}
	w := waUser{u: u}
	for _, r := range rows {
		var c webauthn.Credential
		if err := json.Unmarshal(r.Credential, &c); err != nil {
			return waUser{}, fmt.Errorf("passkey %x: %w", r.ID, err)
		}
		w.creds = append(w.creds, c)
	}
	return w, nil
}

// Options are what the browser passes to navigator.credentials.create/get.
type Options = any

// ---- sign up: a new account with only a passkey ----

func (p *Passkeys) BeginSignup(ctx context.Context, name string) (Options, string, error) {
	name, ok := cleanName(name)
	if !ok {
		return nil, "", apperr.Invalid("invalid_name", "ใส่ชื่อ 1–50 ตัวอักษร")
	}
	// The account doesn't exist until the passkey is made; its ID is fixed
	// now because it becomes the passkey's user handle.
	pending := waUser{u: db.User{ID: uuid.New(), Name: name}}
	creation, session, err := p.wa.BeginRegistration(pending, registrationOpts(nil)...)
	if err != nil {
		return nil, "", err
	}
	token, err := p.saveChallenge(ctx, "signup", nil, &name, session)
	return creation.Response, token, err
}

func (p *Passkeys) FinishSignup(ctx context.Context, token string, body []byte, device string) (User, string, time.Time, error) {
	ch, session, err := p.takeChallenge(ctx, token, "signup")
	if err != nil {
		return User{}, "", time.Time{}, err
	}
	id, err := uuid.FromBytes(session.UserID)
	if err != nil || ch.Name == nil {
		return User{}, "", time.Time{}, errPasskeyExpired
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(body)
	if err != nil {
		return User{}, "", time.Time{}, errPasskeyFailed
	}
	cred, err := p.wa.CreateCredential(waUser{u: db.User{ID: id, Name: *ch.Name}}, *session, parsed)
	if err != nil {
		return User{}, "", time.Time{}, errPasskeyFailed
	}

	var out User
	var sessToken string
	var exp time.Time
	err = pgx.BeginFunc(ctx, p.svc.pool, func(tx pgx.Tx) error {
		q := p.svc.q.WithTx(tx)
		u, err := q.CreatePasskeyUser(ctx, db.CreatePasskeyUserParams{ID: id, Name: *ch.Name})
		if err != nil {
			return err
		}
		if err := savePasskey(ctx, q, id, cred, device); err != nil {
			return err
		}
		out = view(u)
		sessToken, exp, err = p.svc.startSession(ctx, q, id)
		return err
	})
	return out, sessToken, exp, err
}

// ---- add a passkey to the signed-in account ----

func (p *Passkeys) BeginRegister(ctx context.Context, userID uuid.UUID) (Options, string, error) {
	u, err := p.loadUser(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	if len(u.creds) >= maxPasskeys {
		return nil, "", apperr.Conflict("too_many_passkeys", "มี passkey ครบจำนวนแล้ว ลบอันที่ไม่ใช้ก่อน")
	}
	creation, session, err := p.wa.BeginRegistration(u, registrationOpts(u.creds)...)
	if err != nil {
		return nil, "", err
	}
	token, err := p.saveChallenge(ctx, "register", &userID, nil, session)
	return creation.Response, token, err
}

func (p *Passkeys) FinishRegister(ctx context.Context, userID uuid.UUID, token string, body []byte, device string) error {
	ch, session, err := p.takeChallenge(ctx, token, "register")
	if err != nil {
		return err
	}
	if ch.UserID == nil || *ch.UserID != userID {
		return errPasskeyExpired // started by another account
	}
	u, err := p.loadUser(ctx, userID)
	if err != nil {
		return err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(body)
	if err != nil {
		return errPasskeyFailed
	}
	cred, err := p.wa.CreateCredential(u, *session, parsed)
	if err != nil {
		return errPasskeyFailed
	}
	err = savePasskey(ctx, p.svc.q, userID, cred, device)
	if pgCode(err) == "23505" {
		return apperr.Conflict("passkey_exists", "เครื่องนี้มี passkey ของบัญชีนี้อยู่แล้ว")
	}
	return err
}

// registrationOpts asks for a discoverable credential (so sign-in needs no
// username) verified by the device's biometrics or PIN.
func registrationOpts(existing []webauthn.Credential) []webauthn.RegistrationOption {
	exclude := make([]protocol.CredentialDescriptor, len(existing))
	for i := range existing {
		exclude[i] = existing[i].Descriptor()
	}
	return []webauthn.RegistrationOption{
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			ResidentKey:        protocol.ResidentKeyRequirementRequired,
			RequireResidentKey: protocol.ResidentKeyRequired(),
			UserVerification:   protocol.VerificationRequired,
		}),
		webauthn.WithExclusions(exclude),
	}
}

// ---- sign in ----

func (p *Passkeys) BeginLogin(ctx context.Context) (Options, string, error) {
	assertion, session, err := p.wa.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		return nil, "", err
	}
	token, err := p.saveChallenge(ctx, "login", nil, nil, session)
	return assertion.Response, token, err
}

func (p *Passkeys) FinishLogin(ctx context.Context, token string, body []byte) (User, string, time.Time, error) {
	_, session, err := p.takeChallenge(ctx, token, "login")
	if err != nil {
		return User{}, "", time.Time{}, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(body)
	if err != nil {
		return User{}, "", time.Time{}, errPasskeyFailed
	}
	found := false
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		id, err := uuid.FromBytes(userHandle)
		if err != nil {
			return nil, errPasskeyUnknown
		}
		u, err := p.loadUser(ctx, id)
		if err != nil {
			return nil, errPasskeyUnknown
		}
		for _, c := range u.creds {
			if string(c.ID) == string(rawID) {
				found = true
			}
		}
		return u, nil
	}
	who, cred, err := p.wa.ValidatePasskeyLogin(handler, *session, parsed)
	if err != nil {
		if !found {
			return User{}, "", time.Time{}, errPasskeyUnknown
		}
		return User{}, "", time.Time{}, errPasskeyFailed
	}
	u := who.(waUser).u

	raw, err := json.Marshal(cred) // keeps the updated signature counter
	if err != nil {
		return User{}, "", time.Time{}, err
	}
	if err := p.svc.q.UpdatePasskeyUse(ctx, db.UpdatePasskeyUseParams{ID: cred.ID, Credential: raw}); err != nil {
		return User{}, "", time.Time{}, err
	}
	_ = p.svc.q.TouchUserLogin(ctx, u.ID)
	sessToken, exp, err := p.svc.startSession(ctx, p.svc.q, u.ID)
	return view(u), sessToken, exp, err
}

// ---- managing passkeys ----

type PasskeyView struct {
	ID         string     `json:"id"` // base64url credential ID
	Name       string     `json:"name"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

func (p *Passkeys) List(ctx context.Context, userID uuid.UUID) ([]PasskeyView, error) {
	rows, err := p.svc.q.ListPasskeys(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]PasskeyView, len(rows))
	for i, r := range rows {
		out[i] = PasskeyView{ID: base64.RawURLEncoding.EncodeToString(r.ID), Name: r.Name, CreatedAt: r.CreatedAt, LastUsedAt: r.LastUsedAt}
	}
	return out, nil
}

// Delete removes a passkey, but never an account's last way to sign in.
func (p *Passkeys) Delete(ctx context.Context, userID uuid.UUID, id string) error {
	rawID, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		return errPasskeyUnknown
	}
	// Delete, then check what's left; the transaction rolls back if that
	// would leave no way in.
	return pgx.BeginFunc(ctx, p.svc.pool, func(tx pgx.Tx) error {
		q := p.svc.q.WithTx(tx)
		gone, err := q.DeletePasskey(ctx, db.DeletePasskeyParams{ID: rawID, UserID: userID})
		if err != nil {
			return err
		}
		if gone == 0 {
			return errPasskeyUnknown
		}
		u, err := q.GetUser(ctx, userID)
		if err != nil {
			return err
		}
		left, err := q.CountPasskeys(ctx, userID)
		if err != nil {
			return err
		}
		if u.GoogleSub == nil && left == 0 {
			return apperr.Conflict("last_sign_in_method", "ลบไม่ได้ เพราะเป็นวิธีเข้าสู่ระบบสุดท้ายของบัญชีนี้")
		}
		return nil
	})
}

// ---- helpers ----

func savePasskey(ctx context.Context, q *db.Queries, userID uuid.UUID, cred *webauthn.Credential, device string) error {
	raw, err := json.Marshal(cred)
	if err != nil {
		return err
	}
	return q.CreatePasskey(ctx, db.CreatePasskeyParams{ID: cred.ID, UserID: userID, Credential: raw, Name: device})
}

func (p *Passkeys) saveChallenge(ctx context.Context, kind string, userID *uuid.UUID, name *string, s *webauthn.SessionData) (string, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	token, hash := NewToken()
	err = p.svc.q.CreateChallenge(ctx, db.CreateChallengeParams{
		TokenHash: hash, Kind: kind, UserID: userID, Name: name, Session: raw,
		ExpiresAt: p.svc.now().Add(challengeTTL),
	})
	return token, err
}

// takeChallenge consumes a challenge (single use) and checks it hasn't expired.
func (p *Passkeys) takeChallenge(ctx context.Context, token, kind string) (db.AuthChallenge, *webauthn.SessionData, error) {
	if token == "" || len(token) > 100 {
		return db.AuthChallenge{}, nil, errPasskeyExpired
	}
	ch, err := p.svc.q.TakeChallenge(ctx, db.TakeChallengeParams{TokenHash: HashToken(token), Kind: kind})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.AuthChallenge{}, nil, errPasskeyExpired
	}
	if err != nil {
		return db.AuthChallenge{}, nil, err
	}
	if !p.svc.now().Before(ch.ExpiresAt) {
		return db.AuthChallenge{}, nil, errPasskeyExpired
	}
	var s webauthn.SessionData
	if err := json.Unmarshal(ch.Session, &s); err != nil {
		return db.AuthChallenge{}, nil, err
	}
	return ch, &s, nil
}

// DeviceName gives a passkey a friendly label from the browser's User-Agent.
func DeviceName(userAgent string) string {
	ua := strings.ToLower(userAgent)
	for _, d := range []struct{ key, name string }{
		{"iphone", "iPhone"}, {"ipad", "iPad"}, {"android", "Android"},
		{"macintosh", "Mac"}, {"windows", "Windows"}, {"cros", "Chromebook"}, {"linux", "Linux"},
	} {
		if strings.Contains(ua, d.key) {
			return d.name
		}
	}
	return "อุปกรณ์"
}

func pgCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}
