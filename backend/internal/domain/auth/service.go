// Package auth signs people in with Google and keeps them signed in with a
// server-side session. The browser only ever holds a random session token in
// an HttpOnly cookie (JavaScript can't read it); the DB keeps its SHA-256.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raisin/travel-split/backend/internal/platform/apperr"
	"github.com/raisin/travel-split/backend/internal/platform/db"
)

const (
	// SessionTTL is how long a session lasts without use. Each use within
	// the last half of that window pushes the expiry out again.
	SessionTTL = 60 * 24 * time.Hour
	maxName    = 50
)

var ErrLoginRequired = apperr.Unauthorized("login_required", "กรุณาเข้าสู่ระบบก่อน")

// Profile is who the identity provider says the person is.
type Profile struct {
	Subject string // stable account ID
	Email   string
	Name    string
	Picture string
}

// User is the public view of a signed-in person. Email is nil for an
// account made with a passkey only.
type User struct {
	ID        uuid.UUID `json:"id"`
	Email     *string   `json:"email"`
	Name      string    `json:"name"`
	Picture   *string   `json:"picture"`
	HasGoogle bool      `json:"has_google"`
}

func view(u db.User) User {
	return User{ID: u.ID, Email: u.Email, Name: u.Name, Picture: u.Picture, HasGoogle: u.GoogleSub != nil}
}

type Service struct {
	pool *pgxpool.Pool
	q    *db.Queries
	now  func() time.Time
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, q: db.New(pool), now: time.Now}
}

// Login records the profile and starts a session. It returns the session
// token for the cookie and when it expires.
func (s *Service) Login(ctx context.Context, p Profile) (User, string, time.Time, error) {
	if p.Subject == "" {
		return User{}, "", time.Time{}, errors.New("auth: empty subject")
	}
	name, _ := cleanName(p.Name)
	if name == "" {
		name, _ = cleanName(strings.SplitN(p.Email, "@", 2)[0])
	}
	var pic *string
	if strings.HasPrefix(p.Picture, "https://") {
		pic = &p.Picture
	}
	email := p.Email
	u, err := s.q.UpsertUser(ctx, db.UpsertUserParams{GoogleSub: &p.Subject, Email: &email, Name: name, Picture: pic})
	if err != nil {
		return User{}, "", time.Time{}, err
	}
	token, exp, err := s.startSession(ctx, s.q, u.ID)
	return view(u), token, exp, err
}

// startSession issues a session for userID (any sign-in method).
func (s *Service) startSession(ctx context.Context, q *db.Queries, userID uuid.UUID) (string, time.Time, error) {
	// Housekeeping: sign-ins are rare enough to sweep expired rows here.
	_ = q.DeleteExpiredSessions(ctx)
	_ = q.DeleteExpiredChallenges(ctx)

	token, hash := NewToken()
	exp := s.now().Add(SessionTTL)
	if err := q.CreateSession(ctx, db.CreateSessionParams{TokenHash: hash, UserID: userID, ExpiresAt: exp}); err != nil {
		return "", time.Time{}, err
	}
	return token, exp, nil
}

// cleanName trims a display name and caps its length; "" means unusable.
func cleanName(n string) (string, bool) {
	n = strings.TrimSpace(n)
	if utf8.RuneCountInString(n) > maxName {
		n = string([]rune(n)[:maxName])
	}
	return n, n != ""
}

// Session resolves a session token. renewed is non-zero when the expiry was
// pushed out, so the caller can refresh the cookie too.
func (s *Service) Session(ctx context.Context, token string) (u User, renewed time.Time, err error) {
	if token == "" || len(token) > 100 {
		return User{}, time.Time{}, ErrLoginRequired
	}
	hash := HashToken(token)
	row, err := s.q.GetSessionUser(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, time.Time{}, ErrLoginRequired
	}
	if err != nil {
		return User{}, time.Time{}, err
	}
	now := s.now()
	if row.SessionExpiresAt.Sub(now) < SessionTTL/2 {
		renewed = now.Add(SessionTTL)
		if err := s.q.ExtendSession(ctx, db.ExtendSessionParams{TokenHash: hash, ExpiresAt: renewed}); err != nil {
			return User{}, time.Time{}, err
		}
	}
	return view(row.User), renewed, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.q.DeleteSession(ctx, HashToken(token))
}

// NewToken returns a 256-bit random token and its SHA-256 for storage.
func NewToken() (string, []byte) {
	b := make([]byte, 32)
	_, _ = rand.Read(b) // never fails per crypto/rand docs
	t := base64.RawURLEncoding.EncodeToString(b)
	return t, HashToken(t)
}

func HashToken(t string) []byte {
	h := sha256.Sum256([]byte(t))
	return h[:]
}
