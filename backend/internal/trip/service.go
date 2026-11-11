// Package trip manages trips and their members.
package trip

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/raisin/travel-split/backend/internal/apperr"
	"github.com/raisin/travel-split/backend/internal/db"
	"github.com/raisin/travel-split/backend/internal/money"
)

const (
	maxTripName   = 100
	maxMemberName = 50
	inviteLen     = 10
	// No 0/O/1/I/L so codes are easy to read aloud and type.
	inviteAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"
)

var (
	ErrTripNotFound   = apperr.NotFound("trip_not_found", "ไม่พบทริปนี้")
	ErrMemberNotFound = apperr.NotFound("member_not_found", "ไม่พบสมาชิกนี้")
)

type Service struct {
	q *db.Queries
}

func NewService(dbtx db.DBTX) *Service { return &Service{q: db.New(dbtx)} }

func (s *Service) Create(ctx context.Context, name, baseCurrency string) (db.Trip, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxTripName {
		return db.Trip{}, apperr.Invalid("invalid_name", "ชื่อทริปต้องมี 1–100 ตัวอักษร")
	}
	cur := money.Normalize(baseCurrency)
	if cur == "" {
		cur = "THB"
	}
	if !money.Supported(cur) {
		return db.Trip{}, apperr.Invalid("unsupported_currency", "ไม่รองรับสกุลเงิน "+cur)
	}

	for range 5 {
		t, err := s.q.CreateTrip(ctx, db.CreateTripParams{Name: name, BaseCurrency: cur, InviteCode: newInviteCode()})
		if isUniqueViolation(err) {
			continue
		}
		return t, err
	}
	return db.Trip{}, errors.New("could not generate a unique invite code")
}

func (s *Service) GetByInviteCode(ctx context.Context, code string) (db.Trip, error) {
	t, err := s.q.GetTripByInviteCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrTripNotFound
	}
	return t, err
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (db.Trip, error) {
	t, err := s.q.GetTrip(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrTripNotFound
	}
	return t, err
}

func (s *Service) AddMember(ctx context.Context, tripID uuid.UUID, name string, promptpay *string) (db.Member, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxMemberName {
		return db.Member{}, apperr.Invalid("invalid_name", "ชื่อต้องมี 1–50 ตัวอักษร")
	}
	pp, err := normalizePromptPay(promptpay)
	if err != nil {
		return db.Member{}, err
	}

	existing, err := s.q.ListMembers(ctx, tripID)
	if err != nil {
		return db.Member{}, err
	}
	for _, m := range existing {
		if strings.EqualFold(m.Name, name) {
			return db.Member{}, apperr.Conflict("member_name_taken", "มีคนชื่อนี้ในทริปแล้ว")
		}
	}

	m, err := s.q.CreateMember(ctx, db.CreateMemberParams{TripID: tripID, Name: name, Promptpay: pp})
	if isForeignKeyViolation(err) {
		return m, ErrTripNotFound
	}
	return m, err
}

func (s *Service) ListMembers(ctx context.Context, tripID uuid.UUID) ([]db.Member, error) {
	return s.q.ListMembers(ctx, tripID)
}

// Member returns the member only if it belongs to tripID.
func (s *Service) Member(ctx context.Context, tripID, memberID uuid.UUID) (db.Member, error) {
	m, err := s.q.GetMember(ctx, memberID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && m.TripID != tripID) {
		return db.Member{}, ErrMemberNotFound
	}
	return m, err
}

// normalizePromptPay keeps digits only; PromptPay IDs are a 10-digit phone
// number, a 13-digit national/tax ID, or a 15-digit e-wallet ID.
func normalizePromptPay(p *string) (*string, error) {
	if p == nil {
		return nil, nil
	}
	var b strings.Builder
	for _, r := range *p {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == ' ':
		default:
			return nil, apperr.Invalid("invalid_promptpay", "พร้อมเพย์ต้องเป็นตัวเลขเท่านั้น")
		}
	}
	digits := b.String()
	if digits == "" {
		return nil, nil
	}
	if n := len(digits); n != 10 && n != 13 && n != 15 {
		return nil, apperr.Invalid("invalid_promptpay", "พร้อมเพย์ต้องเป็นเบอร์โทร 10 หลัก หรือเลข 13/15 หลัก")
	}
	return &digits, nil
}

func newInviteCode() string {
	buf := make([]byte, inviteLen)
	_, _ = rand.Read(buf) // never fails per crypto/rand docs
	for i, v := range buf {
		buf[i] = inviteAlphabet[int(v)%len(inviteAlphabet)]
	}
	return string(buf)
}

func isUniqueViolation(err error) bool     { return pgCode(err) == "23505" }
func isForeignKeyViolation(err error) bool { return pgCode(err) == "23503" }

func pgCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}
