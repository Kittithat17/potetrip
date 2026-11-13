// Package trip manages trips, members, and device access to a trip.
//
// Access model: everyone is a signed-in user (see package auth). A user's
// place in a trip is a devices row: the creator's is approved at creation and
// that member becomes the owner. Anyone else with the invite code can only
// file a join request, and acts as a member after the owner approves it.
// Rows from before sign-in existed carry a per-device token instead of a
// user until that device signs in and claims them.
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
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raisin/travel-split/backend/internal/domain/auth"
	"github.com/raisin/travel-split/backend/internal/platform/apperr"
	"github.com/raisin/travel-split/backend/internal/platform/db"
	"github.com/raisin/travel-split/backend/internal/platform/money"
)

const (
	maxTripName   = 100
	maxMemberName = 50
	maxPending    = 20 // open join requests per trip, to blunt spam
	maxClaims     = 50 // legacy device tokens one request may claim
	inviteLen     = 10
	// No 0/O/1/I/L so codes are easy to read aloud and type.
	inviteAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

var (
	ErrTripNotFound    = apperr.NotFound("trip_not_found", "ไม่พบทริปนี้")
	ErrMemberNotFound  = apperr.NotFound("member_not_found", "ไม่พบสมาชิกนี้")
	ErrDeviceNotFound  = apperr.Forbidden("not_a_member", "คุณยังไม่ได้เข้าร่วมทริปนี้")
	ErrRequestNotFound = apperr.NotFound("request_not_found", "ไม่พบคำขอนี้ หรือมีการตัดสินใจไปแล้ว")
	errNameTaken       = apperr.Conflict("member_name_taken", "มีคนชื่อนี้ในทริปแล้ว ถ้าเป็นคุณให้เลือกชื่อจากรายการ")
)

type Service struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool, q: db.New(pool)} }

// Created is the result of creating a trip: the creator is its owner.
type Created struct {
	Trip   db.Trip   `json:"trip"`
	Member db.Member `json:"member"`
}

// Create makes a trip, its owner member, and the creator's approved membership.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, name, baseCurrency, creatorName string, promptpay *string) (Created, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxTripName {
		return Created{}, apperr.Invalid("invalid_name", "ชื่อทริปต้องมี 1–100 ตัวอักษร")
	}
	cur := money.Normalize(baseCurrency)
	if cur == "" {
		cur = "THB"
	}
	if !money.Supported(cur) {
		return Created{}, apperr.Invalid("unsupported_currency", "ไม่รองรับสกุลเงิน "+cur)
	}
	creatorName, err := validName(creatorName)
	if err != nil {
		return Created{}, err
	}
	pp, err := normalizePromptPay(promptpay)
	if err != nil {
		return Created{}, err
	}
	var out Created
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var t db.Trip
		var err error
		for range 5 {
			t, err = q.CreateTrip(ctx, db.CreateTripParams{Name: name, BaseCurrency: cur, InviteCode: newInviteCode()})
			if !isUniqueViolation(err) {
				break
			}
		}
		if err != nil {
			return err
		}
		m, err := q.CreateMember(ctx, db.CreateMemberParams{TripID: t.ID, Name: creatorName, Promptpay: pp})
		if err != nil {
			return err
		}
		if err := q.SetTripOwner(ctx, db.SetTripOwnerParams{ID: t.ID, OwnerMemberID: &m.ID}); err != nil {
			return err
		}
		t.OwnerMemberID = &m.ID
		if _, err := q.CreateDevice(ctx, db.CreateDeviceParams{
			TripID: t.ID, UserID: &userID, Status: StatusApproved, MemberID: &m.ID, RequestedName: creatorName,
		}); err != nil {
			return err
		}
		out = Created{Trip: t, Member: m}
		return nil
	})
	return out, err
}

// Update renames the trip and/or changes its base currency. The currency is
// locked once the trip has any expense or recorded repayment, because every
// stored amount is in that currency.
func (s *Service) Update(ctx context.Context, tripID uuid.UUID, name *string, baseCurrency *string) (db.Trip, error) {
	var out db.Trip
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		t, err := q.LockTrip(ctx, tripID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTripNotFound
		}
		if err != nil {
			return err
		}
		newName, newCur := t.Name, t.BaseCurrency
		if name != nil {
			n := strings.TrimSpace(*name)
			if n == "" || utf8.RuneCountInString(n) > maxTripName {
				return apperr.Invalid("invalid_name", "ชื่อทริปต้องมี 1–100 ตัวอักษร")
			}
			newName = n
		}
		if baseCurrency != nil {
			c := money.Normalize(*baseCurrency)
			if !money.Supported(c) {
				return apperr.Invalid("unsupported_currency", "ไม่รองรับสกุลเงิน "+c)
			}
			if c != t.BaseCurrency {
				n, err := q.CountTripActivity(ctx, tripID)
				if err != nil {
					return err
				}
				if n > 0 {
					return apperr.Conflict("currency_locked", "เปลี่ยนสกุลเงินหลักไม่ได้แล้ว เพราะทริปนี้มีรายจ่ายหรือการโอนคืนแล้ว")
				}
			}
			newCur = c
		}
		out, err = q.UpdateTrip(ctx, db.UpdateTripParams{ID: tripID, Name: newName, BaseCurrency: newCur})
		return err
	})
	return out, err
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

func (s *Service) ListMembers(ctx context.Context, tripID uuid.UUID) ([]db.Member, error) {
	return s.q.ListMembers(ctx, tripID)
}

// RequestJoin files a pending join request for the user. claim names an
// existing member when the person is already in the trip (e.g. they joined
// before sign-in existed); otherwise name is a new member's name. A user who
// was turned down may ask again.
func (s *Service) RequestJoin(ctx context.Context, userID, tripID uuid.UUID, name string, claim *uuid.UUID, promptpay *string) (db.Device, error) {
	switch d, err := s.q.GetDeviceForUser(ctx, db.GetDeviceForUserParams{TripID: tripID, UserID: &userID}); {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return db.Device{}, err
	case d.Status == StatusApproved:
		return db.Device{}, apperr.Conflict("already_member", "คุณอยู่ในทริปนี้แล้ว")
	case d.Status == StatusPending:
		return db.Device{}, apperr.Conflict("already_requested", "ส่งคำขอไปแล้ว รอเจ้าของทริปอนุมัติ")
	}

	members, err := s.q.ListMembers(ctx, tripID)
	if err != nil {
		return db.Device{}, err
	}
	arg := db.CreateDeviceParams{TripID: tripID, UserID: &userID, Status: StatusPending}
	if claim != nil {
		m, ok := findMember(members, *claim)
		if !ok {
			return db.Device{}, ErrMemberNotFound
		}
		arg.RequestedMemberID, arg.RequestedName = &m.ID, m.Name
	} else {
		n, err := validName(name)
		if err != nil {
			return db.Device{}, err
		}
		if nameTaken(members, n) {
			return db.Device{}, errNameTaken
		}
		pp, err := normalizePromptPay(promptpay)
		if err != nil {
			return db.Device{}, err
		}
		arg.RequestedName, arg.Promptpay = n, pp
	}

	pending, err := s.q.CountPendingDevices(ctx, tripID)
	if err != nil {
		return db.Device{}, err
	}
	if pending >= maxPending {
		return db.Device{}, apperr.Conflict("too_many_requests", "มีคำขอรออนุมัติเยอะเกินไป ให้เจ้าของทริปจัดการก่อน")
	}

	var d db.Device
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := q.DeleteRejectedDevice(ctx, db.DeleteRejectedDeviceParams{TripID: tripID, UserID: &userID}); err != nil {
			return err
		}
		var err error
		d, err = q.CreateDevice(ctx, arg)
		return err
	})
	switch {
	case isForeignKeyViolation(err):
		return db.Device{}, ErrTripNotFound
	case isUniqueViolation(err): // a second request raced this one
		return db.Device{}, apperr.Conflict("already_requested", "ส่งคำขอไปแล้ว รอเจ้าของทริปอนุมัติ")
	}
	return d, err
}

// DeviceForUser is the user's membership (or join request) in tripID.
func (s *Service) DeviceForUser(ctx context.Context, tripID, userID uuid.UUID) (db.Device, error) {
	d, err := s.q.GetDeviceForUser(ctx, db.GetDeviceForUserParams{TripID: tripID, UserID: &userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Device{}, ErrDeviceNotFound
	}
	return d, err
}

// ClaimDevices moves memberships held by pre-sign-in device tokens onto the
// user and invalidates those tokens. Tokens that are unknown, already
// claimed, or for a trip the user is already in are skipped.
func (s *Service) ClaimDevices(ctx context.Context, userID uuid.UUID, tokens []string) (int, error) {
	if len(tokens) > maxClaims {
		tokens = tokens[:maxClaims]
	}
	n := 0
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if t == "" || len(t) > 100 {
			continue
		}
		_, err := s.q.ClaimDevice(ctx, db.ClaimDeviceParams{TokenHash: auth.HashToken(t), UserID: &userID})
		switch {
		case errors.Is(err, pgx.ErrNoRows), isUniqueViolation(err):
			continue
		case err != nil:
			return n, err
		}
		n++
	}
	return n, nil
}

// Leave removes the user's membership or cancels their join request. Their
// member (and its expenses) stays in the trip. The owner can't leave: nobody
// else could ever approve anyone again.
func (s *Service) Leave(ctx context.Context, t db.Trip, d db.Device) error {
	if d.Status == StatusApproved && d.MemberID != nil && t.OwnerMemberID != nil && *t.OwnerMemberID == *d.MemberID {
		return apperr.Conflict("owner_cannot_leave", "เจ้าของทริปออกจากทริปไม่ได้")
	}
	return s.q.DeleteDeviceForUser(ctx, db.DeleteDeviceForUserParams{TripID: t.ID, UserID: d.UserID})
}

// MyTrips lists the trips the user is in or waiting to join.
func (s *Service) MyTrips(ctx context.Context, userID uuid.UUID) ([]db.ListUserTripsRow, error) {
	return s.q.ListUserTrips(ctx, &userID)
}

// Member returns the member only if it belongs to tripID.
func (s *Service) Member(ctx context.Context, tripID, memberID uuid.UUID) (db.Member, error) {
	m, err := s.q.GetMember(ctx, memberID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && m.TripID != tripID) {
		return db.Member{}, ErrMemberNotFound
	}
	return m, err
}

// UpdatePromptPay sets (or clears, with nil/empty) a member's PromptPay ID.
func (s *Service) UpdatePromptPay(ctx context.Context, tripID, memberID uuid.UUID, promptpay *string) (db.Member, error) {
	pp, err := normalizePromptPay(promptpay)
	if err != nil {
		return db.Member{}, err
	}
	m, err := s.q.UpdateMemberPromptPay(ctx, db.UpdateMemberPromptPayParams{ID: memberID, TripID: tripID, Promptpay: pp})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Member{}, ErrMemberNotFound
	}
	return m, err
}

func (s *Service) PendingRequests(ctx context.Context, tripID uuid.UUID) ([]db.Device, error) {
	return s.q.ListPendingDevices(ctx, tripID)
}

// Approve lets a pending device into the trip, creating its member first
// when the request was for a new name.
func (s *Service) Approve(ctx context.Context, tripID, requestID uuid.UUID) (db.Member, error) {
	var out db.Member
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		d, err := q.GetDevice(ctx, db.GetDeviceParams{ID: requestID, TripID: tripID})
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && d.Status != StatusPending) {
			return ErrRequestNotFound
		}
		if err != nil {
			return err
		}
		if d.RequestedMemberID != nil {
			out, err = q.GetMember(ctx, *d.RequestedMemberID)
			if err != nil {
				return err
			}
		} else {
			members, err := q.ListMembers(ctx, tripID)
			if err != nil {
				return err
			}
			// Someone with the same name may have joined since the request.
			if nameTaken(members, d.RequestedName) {
				return errNameTaken
			}
			out, err = q.CreateMember(ctx, db.CreateMemberParams{TripID: tripID, Name: d.RequestedName, Promptpay: d.Promptpay})
			if err != nil {
				return err
			}
		}
		if _, err := q.ApproveDevice(ctx, db.ApproveDeviceParams{ID: d.ID, TripID: tripID, MemberID: &out.ID}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrRequestNotFound
			}
			return err
		}
		return nil
	})
	return out, err
}

func (s *Service) Reject(ctx context.Context, tripID, requestID uuid.UUID) error {
	_, err := s.q.RejectDevice(ctx, db.RejectDeviceParams{ID: requestID, TripID: tripID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRequestNotFound
	}
	return err
}

func validName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxMemberName {
		return "", apperr.Invalid("invalid_name", "ชื่อต้องมี 1–50 ตัวอักษร")
	}
	return name, nil
}

func nameTaken(members []db.Member, name string) bool {
	for _, m := range members {
		if strings.EqualFold(m.Name, name) {
			return true
		}
	}
	return false
}

func findMember(members []db.Member, id uuid.UUID) (db.Member, bool) {
	for _, m := range members {
		if m.ID == id {
			return m, true
		}
	}
	return db.Member{}, false
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
	_, _ = rand.Read(buf)
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
