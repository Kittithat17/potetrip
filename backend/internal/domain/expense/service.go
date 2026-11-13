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

	"github.com/raisin/travel-split/backend/internal/domain/fx"
	"github.com/raisin/travel-split/backend/internal/platform/apperr"
	"github.com/raisin/travel-split/backend/internal/platform/db"
	"github.com/raisin/travel-split/backend/internal/platform/money"
)

const (
	SplitEqualType = "equal"
	SplitExactType = "exact"
	SplitItemsType = "items"

	maxTitle    = 200
	maxItems    = 100
	maxItemName = 100
)

var Categories = []string{"food", "transport", "lodging", "shopping", "activity", "other"}

var (
	ErrNotFound   = apperr.NotFound("expense_not_found", "ไม่พบรายการนี้")
	errNotAllowed = apperr.Forbidden("not_allowed", "แก้หรือลบได้เฉพาะคนที่บันทึก คนจ่าย หรือเจ้าของทริป")
)

// Actor is the member making a change, and whether they own the trip.
type Actor struct {
	MemberID uuid.UUID
	IsOwner  bool
}

// CanEdit: the member who logged an expense, the one who paid it, and the
// trip owner may change or delete it; everyone else sees it read-only.
func (a Actor) CanEdit(e db.Expense) bool {
	return a.IsOwner || a.MemberID == e.CreatedBy || a.MemberID == e.PaidBy
}

func logActivity(ctx context.Context, q *db.Queries, a Actor, action string, e db.Expense) error {
	return q.CreateActivity(ctx, db.CreateActivityParams{
		TripID: e.TripID, MemberID: &a.MemberID, Action: action, ExpenseID: &e.ID, Title: e.Title, AmountBase: e.AmountBase,
	})
}

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
	// FxRate overrides the looked-up rate (units of base per 1 Currency),
	// e.g. the rate the money changer actually gave. Ignored for the base currency.
	FxRate *string `json:"fx_rate"`
	// Items are receipt lines for split_type=items, in Currency minor units.
	Items []ItemInput `json:"items"`
}

// ItemInput is one receipt line and who shared it.
type ItemInput struct {
	Name      string      `json:"name"`
	Amount    int64       `json:"amount"`
	MemberIDs []uuid.UUID `json:"member_ids"`
}

type Item struct {
	Name      string      `json:"name"`
	Amount    int64       `json:"amount"`
	MemberIDs []uuid.UUID `json:"member_ids"`
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
	Items      []Item     `json:"items"`
}

type Filter struct {
	Category *string
	MemberID *uuid.UUID
}

// Rater looks up exchange rates; *fx.Service implements it.
type Rater interface {
	Rate(ctx context.Context, from, to string, date *time.Time) (fx.Quote, error)
}

type Service struct {
	pool  *pgxpool.Pool
	q     *db.Queries
	rater Rater // nil: only the base currency is accepted
	now   func() time.Time
}

func NewService(pool *pgxpool.Pool, rater Rater) *Service {
	return &Service{pool: pool, q: db.New(pool), rater: rater, now: time.Now}
}

// prepared is a validated Input ready to be written.
type prepared struct {
	in         Input
	fxRate     pgtype.Numeric
	amountBase int64
	shares     []Share
	items      []Item
}

func (s *Service) Create(ctx context.Context, trip db.Trip, actor Actor, in Input) (View, error) {
	createdBy := actor.MemberID
	p, err := s.prepare(ctx, trip, in)
	if err != nil {
		return View{}, err
	}
	var out View
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := checkCurrency(ctx, q, trip.ID, trip.BaseCurrency); err != nil {
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
		if err := insertItems(ctx, q, e.ID, p.items); err != nil {
			return err
		}
		out = toView(e, p.shares, p.items)
		return logActivity(ctx, q, actor, "expense_created", e)
	})
	return out, err
}

func (s *Service) Update(ctx context.Context, trip db.Trip, actor Actor, id uuid.UUID, in Input) (View, error) {
	p, err := s.prepare(ctx, trip, in)
	if err != nil {
		return View{}, err
	}
	var out View
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		if err := checkCurrency(ctx, q, trip.ID, trip.BaseCurrency); err != nil {
			return err
		}
		old, err := q.GetExpense(ctx, db.GetExpenseParams{ID: id, TripID: trip.ID})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if !actor.CanEdit(old) {
			return errNotAllowed
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
		if err := q.DeleteItems(ctx, e.ID); err != nil {
			return err
		}
		if err := insertShares(ctx, q, e.ID, p.shares); err != nil {
			return err
		}
		if err := insertItems(ctx, q, e.ID, p.items); err != nil {
			return err
		}
		out = toView(e, p.shares, p.items)
		return logActivity(ctx, q, actor, "expense_updated", e)
	})
	return out, err
}

func (s *Service) Delete(ctx context.Context, tripID uuid.UUID, actor Actor, id uuid.UUID) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		e, err := q.GetExpense(ctx, db.GetExpenseParams{ID: id, TripID: tripID})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if !actor.CanEdit(e) {
			return errNotAllowed
		}
		if _, err := q.DeleteExpense(ctx, db.DeleteExpenseParams{ID: id, TripID: tripID}); err != nil {
			return err
		}
		return logActivity(ctx, q, actor, "expense_deleted", e)
	})
}

// Activity is the trip's change history, newest first.
func (s *Service) Activity(ctx context.Context, tripID uuid.UUID) ([]db.ListActivityRow, error) {
	return s.q.ListActivity(ctx, tripID)
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

// Summary totals a trip's spending by category and by original currency.
type Summary struct {
	Total      int64                     `json:"total"`
	Count      int                       `json:"count"`
	ByCategory []db.SummaryByCategoryRow `json:"by_category"`
	ByCurrency []db.SummaryByCurrencyRow `json:"by_currency"`
}

func (s *Service) Summary(ctx context.Context, tripID uuid.UUID) (Summary, error) {
	cats, err := s.q.SummaryByCategory(ctx, tripID)
	if err != nil {
		return Summary{}, err
	}
	curs, err := s.q.SummaryByCurrency(ctx, tripID)
	if err != nil {
		return Summary{}, err
	}
	out := Summary{ByCategory: cats, ByCurrency: curs}
	for _, c := range cats {
		out.Total += c.AmountBase
		out.Count += int(c.Count)
	}
	return out, nil
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
	itemRows, err := s.q.ListItemsForExpenses(ctx, ids)
	if err != nil {
		return nil, err
	}
	items := map[uuid.UUID][]Item{}
	for _, r := range itemRows {
		items[r.ExpenseID] = append(items[r.ExpenseID], Item{Name: r.Name, Amount: r.Amount, MemberIDs: r.MemberIds})
	}
	out := make([]View, len(rows))
	for i, e := range rows {
		out[i] = toView(e, byExpense[e.ID], items[e.ID])
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
	if !money.Supported(in.Currency) {
		return prepared{}, apperr.Invalid("invalid_currency", "ไม่รองรับสกุลเงินนี้")
	}
	if in.SpentAt == nil || in.SpentAt.IsZero() {
		now := s.now()
		in.SpentAt = &now
	}

	rate, err := s.rate(ctx, in, trip.BaseCurrency)
	if err != nil {
		return prepared{}, err
	}
	amountBase, err := money.Convert(in.Amount, in.Currency, trip.BaseCurrency, rate)
	if err != nil {
		return prepared{}, apperr.Invalid("invalid_amount", "จำนวนเงินมากเกินไป")
	}
	if amountBase <= 0 {
		return prepared{}, apperr.Invalid("amount_too_small", "ยอดนี้แปลงเป็นสกุลหลักแล้วน้อยกว่า 1 หน่วย")
	}
	var fxRate pgtype.Numeric
	if err := fxRate.Scan(money.RateString(rate)); err != nil {
		return prepared{}, err
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

	var items []Item
	if in.SplitType == SplitItemsType {
		if items, err = cleanItems(in.Items, order); err != nil {
			return prepared{}, err
		}
	}
	shares, err := computeShares(in, amountBase, items, order)
	if err != nil {
		return prepared{}, err
	}
	return prepared{in: in, fxRate: fxRate, amountBase: amountBase, shares: shares, items: items}, nil
}

// rate picks the conversion rate: 1 for the base currency, the user's own
// rate if given, otherwise the reference rate for the day of the expense.
func (s *Service) rate(ctx context.Context, in Input, base string) (*big.Rat, error) {
	if in.Currency == base {
		return big.NewRat(1, 1), nil
	}
	if in.FxRate != nil && strings.TrimSpace(*in.FxRate) != "" {
		r, err := money.ParseRate(*in.FxRate)
		if err != nil {
			return nil, apperr.Invalid("invalid_fx_rate", "อัตราแลกเปลี่ยนต้องเป็นตัวเลขมากกว่า 0 ทศนิยมไม่เกิน 8 ตำแหน่ง")
		}
		return r, nil
	}
	if s.rater == nil {
		return nil, apperr.Invalid("fx_unavailable", "ดึงอัตราแลกเปลี่ยนไม่ได้ ใส่เรตเองได้")
	}
	day := *in.SpentAt
	if now := s.now(); day.After(now) {
		day = now // no rates exist for the future
	}
	q, err := s.rater.Rate(ctx, in.Currency, base, &day)
	if err != nil {
		return nil, err
	}
	return money.NormalizeRate(q.Rate)
}

// cleanItems validates receipt lines and dedupes their members.
func cleanItems(in []ItemInput, order map[uuid.UUID]int) ([]Item, error) {
	if len(in) == 0 || len(in) > maxItems {
		return nil, apperr.Invalid("invalid_items", "ต้องมีรายการ 1–100 รายการ")
	}
	out := make([]Item, len(in))
	for i, it := range in {
		name := strings.TrimSpace(it.Name)
		if name == "" || utf8.RuneCountInString(name) > maxItemName {
			return nil, apperr.Invalid("invalid_items", "ชื่อรายการต้องมี 1–100 ตัวอักษร")
		}
		if it.Amount <= 0 {
			return nil, apperr.Invalid("invalid_items", "ราคาของแต่ละรายการต้องมากกว่า 0")
		}
		ids, err := uniqueMembers(it.MemberIDs, order)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return nil, apperr.Invalid("item_no_members", "ทุกรายการต้องมีคนกินอย่างน้อย 1 คน")
		}
		slices.SortFunc(ids, func(a, b uuid.UUID) int { return order[a] - order[b] })
		out[i] = Item{Name: name, Amount: it.Amount, MemberIDs: ids}
	}
	return out, nil
}

// computeShares turns the split request into base-currency shares that sum
// exactly to amountBase. Members are processed in trip order so remainder
// distribution is stable regardless of request order.
func computeShares(in Input, amountBase int64, items []Item, order map[uuid.UUID]int) ([]Share, error) {
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
		return SplitItems(amountBase, items, order)
	default:
		return nil, apperr.Invalid("invalid_split", "วิธีหารไม่ถูกต้อง")
	}
}

// checkCurrency locks the trip row for share and confirms its base currency
// still matches the one amount_base was converted into.
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

func insertItems(ctx context.Context, q *db.Queries, expenseID uuid.UUID, items []Item) error {
	for i, it := range items {
		if err := q.CreateItem(ctx, db.CreateItemParams{ExpenseID: expenseID, Position: int32(i), Name: it.Name, Amount: it.Amount, MemberIds: it.MemberIDs}); err != nil {
			return err
		}
	}
	return nil
}

func toView(e db.Expense, shares []Share, items []Item) View {
	if shares == nil {
		shares = []Share{}
	}
	if items == nil {
		items = []Item{}
	}
	return View{
		ID: e.ID, TripID: e.TripID, Title: e.Title, Category: e.Category, PaidBy: e.PaidBy,
		Amount: e.Amount, Currency: e.Currency, FxRate: numericString(e.FxRate), AmountBase: e.AmountBase,
		SplitType: e.SplitType, SpentAt: e.SpentAt, SlipID: e.SlipID, CreatedBy: e.CreatedBy,
		CreatedAt: e.CreatedAt, Shares: shares, Items: items,
	}
}

func numericString(n pgtype.Numeric) string {
	v, err := n.Value()
	if err != nil || v == nil {
		return ""
	}
	s, _ := v.(string)
	// numeric(18,8) comes back as "0.23450000"; show "0.2345".
	if strings.Contains(s, ".") {
		s = strings.TrimSuffix(strings.TrimRight(s, "0"), ".")
	}
	return s
}
