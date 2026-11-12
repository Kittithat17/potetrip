// Package expense records expenses and how each one is split.
package expense

import (
	"context"
	"errors"
	"math/big"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raisin/travel-split/backend/internal/apperr"
	"github.com/raisin/travel-split/backend/internal/db"
	"github.com/raisin/travel-split/backend/internal/money"
)

const (
	SplitEqualType = "equal"
	SplitExactType = "exact"
	SplitItemsType = "items"

	maxTitle = 200
)

var Categories = []string{"food", "transport", "lodging", "shopping", "activity", "other"}

var ErrNotFound = apperr.NotFound("expense_not_found", "ไม่พบรายการนี้")

// Input is what a client sends to create or replace an expense.
type Input struct {
	Title     string       `json:"title"`
	Category  string       `json:"category"`
	PaidBy    uuid.UUID    `json:"paid_by"`
	Amount    int64        `json:"amount"`   // minor units of Currency
	Currency  string       `json:"currency"` // defaults to the trip base currency
	SplitType string       `json:"split_type"`
	SpentAt   *time.Time   `json:"spent_at"` // defaults to now
	MemberIDs []uuid.UUID  `json:"member_ids"`
	Shares    []ShareInput `json:"shares"`
	// SlipID links a parsed slip the user just reviewed (create only).
	// Each slip can back at most one expense.
	SlipID *uuid.UUID `json:"slip_id"`
}

// ShareInput is one member's portion for split_type=exact, in Currency minor units.
type ShareInput struct {
	MemberID uuid.UUID `json:"member_id"`
	Amount   int64     `json:"amount"`
}

type Share struct {
	MemberID   uuid.UUID `json:"member_id"`
	AmountBase int64     `json:"amount_base"`
}

// View is an expense as returned by the API.
type View struct {
	ID         uuid.UUID  `json:"id"`
	TripID     uuid.UUID  `json:"trip_id"`
	Title      string     `json:"title"`
	Category   string     `json:"category"`
	PaidBy     uuid.UUID  `json:"paid_by"`
	Amount     int64      `json:"amount"`
	Currency   string     `json:"currency"`
	FxRate     string     `json:"fx_rate"`
	AmountBase int64      `json:"amount_base"`
	SplitType  string     `json:"split_type"`
	SpentAt    time.Time  `json:"spent_at"`
	SlipID     *uuid.UUID `json:"slip_id"`
	CreatedBy  uuid.UUID  `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
	Shares     []Share    `json:"shares"`
}

type Filter struct {
	Category *string
	MemberID *uuid.UUID
}

type Service struct {
	pool *pgxpool.Pool
	q    *db.Queries
	now  func() time.Time
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, q: db.New(pool), now: time.Now}
}

// prepared is a validated Input ready to be written.
type prepared struct {
	in         Input
	fxRate     pgtype.Numeric
	amountBase int64
	shares     []Share
}

func (s *Service) Create(ctx context.Context, trip db.Trip, createdBy uuid.UUID, in Input) (View, error) {
	p, err := s.prepare(ctx, trip, in)
	if err != nil {
		return View{}, err
	}
	var out View
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := checkCurrency(ctx, q, trip.ID, p.in.Currency); err != nil {
			return err
		}
		if p.in.SlipID != nil {
			n, err := q.ConfirmSlip(ctx, db.ConfirmSlipParams{ID: *p.in.SlipID, TripID: trip.ID})
			if err != nil {
				return err
			}
			if n == 0 {
				return apperr.Conflict("slip_unavailable", "สลิปนี้ถูกบันทึกไปแล้ว หรือไม่ได้อยู่ในทริปนี้")
			}
		}
		e, err := q.CreateExpense(ctx, db.CreateExpenseParams{
			TripID:     trip.ID,
			Title:      p.in.Title,
			Category:   p.in.Category,
			PaidBy:     p.in.PaidBy,
			Amount:     p.in.Amount,
			Currency:   p.in.Currency,
			FxRate:     p.fxRate,
			AmountBase: p.amountBase,
			SplitType:  p.in.SplitType,
			SpentAt:    *p.in.SpentAt,
			SlipID:     p.in.SlipID,
			CreatedBy:  createdBy,
		})
		if err != nil {
			return err
		}
		if err := insertShares(ctx, q, e.ID, p.shares); err != nil {
			return err
		}
		out = toView(e, p.shares)
		return nil
	})
	return out, err
}

func (s *Service) Update(ctx context.Context, trip db.Trip, id uuid.UUID, in Input) (View, error) {
	p, err := s.prepare(ctx, trip, in)
	if err != nil {
		return View{}, err
	}
	var out View
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := checkCurrency(ctx, q, trip.ID, p.in.Currency); err != nil {
			return err
		}
		e, err := q.UpdateExpense(ctx, db.UpdateExpenseParams{
			ID:         id,
			TripID:     trip.ID,
			Title:      p.in.Title,
			Category:   p.in.Category,
			PaidBy:     p.in.PaidBy,
			Amount:     p.in.Amount,
			Currency:   p.in.Currency,
			FxRate:     p.fxRate,
			AmountBase: p.amountBase,
			SplitType:  p.in.SplitType,
			SpentAt:    *p.in.SpentAt,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := q.DeleteShares(ctx, e.ID); err != nil {
			return err
		}
		if err := insertShares(ctx, q, e.ID, p.shares); err != nil {
			return err
		}
		out = toView(e, p.shares)
		return nil
	})
	return out, err
}

func (s *Service) Delete(ctx context.Context, tripID, id uuid.UUID) error {
	n, err := s.q.DeleteExpense(ctx, db.DeleteExpenseParams{ID: id, TripID: tripID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) Get(ctx context.Context, tripID, id uuid.UUID) (View, error) {
	e, err := s.q.GetExpense(ctx, db.GetExpenseParams{ID: id, TripID: tripID})
	if errors.Is(err, pgx.ErrNoRows) {
		return View{}, ErrNotFound
	}
	if err != nil {
		return View{}, err
	}
	views, err := s.withShares(ctx, []db.Expense{e})
	if err != nil {
		return View{}, err
	}
	return views[0], nil
}

func (s *Service) List(ctx context.Context, tripID uuid.UUID, f Filter) ([]View, error) {
	if f.Category != nil && !slices.Contains(Categories, *f.Category) {
		return nil, apperr.Invalid("invalid_category", "หมวดหมู่ไม่ถูกต้อง")
	}
	rows, err := s.q.ListExpenses(ctx, db.ListExpensesParams{TripID: tripID, Category: f.Category, MemberID: f.MemberID})
	if err != nil {
		return nil, err
	}
	return s.withShares(ctx, rows)
}

func (s *Service) withShares(ctx context.Context, rows []db.Expense) ([]View, error) {
	ids := make([]uuid.UUID, len(rows))
	for i, e := range rows {
		ids[i] = e.ID
	}
	shareRows, err := s.q.ListSharesForExpenses(ctx, ids)
	if err != nil {
		return nil, err
	}
	byExpense := map[uuid.UUID][]Share{}
	for _, r := range shareRows {
		byExpense[r.ExpenseID] = append(byExpense[r.ExpenseID], Share{MemberID: r.MemberID, AmountBase: r.AmountBase})
	}
	out := make([]View, len(rows))
	for i, e := range rows {
		out[i] = toView(e, byExpense[e.ID])
	}
	return out, nil
}

func (s *Service) prepare(ctx context.Context, trip db.Trip, in Input) (prepared, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" || utf8.RuneCountInString(in.Title) > maxTitle {
		return prepared{}, apperr.Invalid("invalid_title", "ชื่อรายการต้องมี 1–200 ตัวอักษร")
	}
	if !slices.Contains(Categories, in.Category) {
		return prepared{}, apperr.Invalid("invalid_category", "หมวดหมู่ไม่ถูกต้อง")
	}
	if in.Amount <= 0 {
		return prepared{}, apperr.Invalid("invalid_amount", "จำนวนเงินต้องมากกว่า 0")
	}
	in.Currency = money.Normalize(in.Currency)
	if in.Currency == "" {
		in.Currency = trip.BaseCurrency
	}
	// Multi-currency arrives in M4; until then only the base currency is accepted.
	if in.Currency != trip.BaseCurrency {
		return prepared{}, apperr.Invalid("unsupported_currency", "ตอนนี้รองรับเฉพาะสกุลเงินหลักของทริป ("+trip.BaseCurrency+")")
	}
	if in.SpentAt == nil || in.SpentAt.IsZero() {
		now := s.now()
		in.SpentAt = &now
	}

	members, err := s.q.ListMembers(ctx, trip.ID)
	if err != nil {
		return prepared{}, err
	}
	order := make(map[uuid.UUID]int, len(members))
	for i, m := range members {
		order[m.ID] = i
	}
	if _, ok := order[in.PaidBy]; !ok {
		return prepared{}, apperr.Invalid("invalid_paid_by", "ผู้จ่ายไม่ได้อยู่ในทริปนี้")
	}

	fxRate := pgtype.Numeric{Int: big.NewInt(1), Exp: 0, Valid: true}
	amountBase := in.Amount

	shares, err := computeShares(in, amountBase, order)
	if err != nil {
		return prepared{}, err
	}
	return prepared{in: in, fxRate: fxRate, amountBase: amountBase, shares: shares}, nil
}

// computeShares turns the split request into base-currency shares that sum
// exactly to amountBase. Members are processed in trip order so remainder
// distribution is stable regardless of request order.
func computeShares(in Input, amountBase int64, order map[uuid.UUID]int) ([]Share, error) {
	byOrder := func(a, b uuid.UUID) int { return order[a] - order[b] }

	switch in.SplitType {
	case SplitEqualType:
		ids, err := uniqueMembers(in.MemberIDs, order)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return nil, apperr.Invalid("invalid_split", "ต้องเลือกคนหารอย่างน้อย 1 คน")
		}
		slices.SortFunc(ids, byOrder)
		parts, err := SplitEqual(amountBase, len(ids))
		if err != nil {
			return nil, err
		}
		return zip(ids, parts), nil

	case SplitExactType:
		ids := make([]uuid.UUID, len(in.Shares))
		amounts := map[uuid.UUID]int64{}
		var sum int64
		for i, sh := range in.Shares {
			if sh.Amount < 0 {
				return nil, apperr.Invalid("invalid_split", "ยอดของแต่ละคนต้องไม่ติดลบ")
			}
			ids[i] = sh.MemberID
			amounts[sh.MemberID] = sh.Amount
			sum += sh.Amount
		}
		ids, err := uniqueMembers(ids, order)
		if err != nil {
			return nil, err
		}
		if len(ids) != len(in.Shares) {
			return nil, apperr.Invalid("invalid_split", "มีสมาชิกซ้ำในรายการหาร")
		}
		if sum != in.Amount {
			return nil, apperr.Invalid("split_mismatch", "ผลรวมของแต่ละคนต้องเท่ากับยอดทั้งหมด")
		}
		slices.SortFunc(ids, byOrder)
		var kept []uuid.UUID
		var weights []int64
		for _, id := range ids {
			if amounts[id] > 0 {
				kept = append(kept, id)
				weights = append(weights, amounts[id])
			}
		}
		// Shares are entered in the expense currency; allocate base amounts
		// proportionally so a converted total still splits exactly.
		parts, err := Allocate(amountBase, weights)
		if err != nil {
			return nil, err
		}
		return zip(kept, parts), nil

	case SplitItemsType:
		return nil, apperr.Invalid("unsupported_split", "การหารแบบรายเมนูจะมาในเวอร์ชันถัดไป")
	default:
		return nil, apperr.Invalid("invalid_split", "วิธีหารไม่ถูกต้อง")
	}
}

// checkCurrency locks the trip row for share and confirms its base currency
// still matches what the expense was validated against.
func checkCurrency(ctx context.Context, q *db.Queries, tripID uuid.UUID, currency string) error {
	cur, err := q.TripCurrencyForShare(ctx, tripID)
	if err != nil {
		return err
	}
	if cur != currency {
		return apperr.Conflict("currency_changed", "สกุลเงินหลักของทริปเพิ่งถูกเปลี่ยน ลองบันทึกใหม่อีกครั้ง")
	}
	return nil
}

func uniqueMembers(ids []uuid.UUID, order map[uuid.UUID]int) ([]uuid.UUID, error) {
	seen := map[uuid.UUID]bool{}
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := order[id]; !ok {
			return nil, apperr.Invalid("invalid_split", "มีคนที่ไม่ได้อยู่ในทริปนี้")
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out, nil
}

func zip(ids []uuid.UUID, parts []int64) []Share {
	out := make([]Share, 0, len(ids))
	for i, id := range ids {
		if parts[i] > 0 {
			out = append(out, Share{MemberID: id, AmountBase: parts[i]})
		}
	}
	return out
}

func insertShares(ctx context.Context, q *db.Queries, expenseID uuid.UUID, shares []Share) error {
	for _, sh := range shares {
		if err := q.CreateShare(ctx, db.CreateShareParams{ExpenseID: expenseID, MemberID: sh.MemberID, AmountBase: sh.AmountBase}); err != nil {
			return err
		}
	}
	return nil
}

func toView(e db.Expense, shares []Share) View {
	if shares == nil {
		shares = []Share{}
	}
	return View{
		ID: e.ID, TripID: e.TripID, Title: e.Title, Category: e.Category, PaidBy: e.PaidBy,
		Amount: e.Amount, Currency: e.Currency, FxRate: numericString(e.FxRate), AmountBase: e.AmountBase,
		SplitType: e.SplitType, SpentAt: e.SpentAt, SlipID: e.SlipID, CreatedBy: e.CreatedBy,
		CreatedAt: e.CreatedAt, Shares: shares,
	}
}

func numericString(n pgtype.Numeric) string {
	v, err := n.Value()
	if err != nil || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}
