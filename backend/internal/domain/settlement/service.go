package settlement

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raisin/travel-split/backend/internal/platform/apperr"
	"github.com/raisin/travel-split/backend/internal/platform/db"
)

// Service loads trip totals from the database and applies the pure
// Balances/Settle functions.
type Service struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool, q: db.New(pool)} }

func (s *Service) Balances(ctx context.Context, tripID uuid.UUID) ([]Balance, error) {
	members, err := s.q.ListMembers(ctx, tripID)
	if err != nil {
		return nil, err
	}
	paidRows, err := s.q.SumPaidByMember(ctx, tripID)
	if err != nil {
		return nil, err
	}
	owedRows, err := s.q.SumOwedByMember(ctx, tripID)
	if err != nil {
		return nil, err
	}
	settledRows, err := s.q.ListSettlementsPaid(ctx, tripID)
	if err != nil {
		return nil, err
	}

	in := Input{
		Members: make([]uuid.UUID, len(members)),
		Paid:    make(map[uuid.UUID]int64, len(paidRows)),
		Owed:    make(map[uuid.UUID]int64, len(owedRows)),
		Settled: make([]Transfer, len(settledRows)),
	}
	for i, m := range members {
		in.Members[i] = m.ID
	}
	for _, r := range paidRows {
		in.Paid[r.MemberID] = r.Total
	}
	for _, r := range owedRows {
		in.Owed[r.MemberID] = r.Total
	}
	for i, r := range settledRows {
		in.Settled[i] = Transfer{From: r.FromMember, To: r.ToMember, AmountBase: r.AmountBase}
	}
	return Balances(in), nil
}

func (s *Service) Plan(ctx context.Context, tripID uuid.UUID) ([]Transfer, error) {
	bs, err := s.Balances(ctx, tripID)
	if err != nil {
		return nil, err
	}
	ts := Settle(bs)
	if ts == nil {
		ts = []Transfer{}
	}
	return ts, nil
}

// MarkPaid records that from paid to amountBase. Both must be trip members.
// MarkPaid records a repayment. Only the payer, the receiver or the trip
// owner may record it, so nobody can wipe their own debt by saying someone
// else paid it, and the recorder is kept.
func (s *Service) MarkPaid(ctx context.Context, tripID uuid.UUID, actor Actor, from, to uuid.UUID, amountBase int64) (db.SettlementsPaid, error) {
	if amountBase <= 0 {
		return db.SettlementsPaid{}, apperr.Invalid("invalid_amount", "จำนวนเงินต้องมากกว่า 0")
	}
	if from == to {
		return db.SettlementsPaid{}, apperr.Invalid("invalid_transfer", "ผู้โอนและผู้รับต้องเป็นคนละคน")
	}
	if actor.MemberID != from && actor.MemberID != to && !actor.IsOwner {
		return db.SettlementsPaid{}, apperr.Forbidden("not_allowed", "บันทึกการโอนได้เฉพาะคนโอน คนรับ หรือเจ้าของทริป")
	}
	members, err := s.q.ListMembers(ctx, tripID)
	if err != nil {
		return db.SettlementsPaid{}, err
	}
	names := map[uuid.UUID]string{}
	for _, m := range members {
		names[m.ID] = m.Name
	}
	fromName, okFrom := names[from]
	toName, okTo := names[to]
	if !okFrom || !okTo {
		return db.SettlementsPaid{}, apperr.Invalid("invalid_transfer", "ผู้โอนหรือผู้รับไม่ได้อยู่ในทริปนี้")
	}
	var out db.SettlementsPaid
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		out, err = q.CreateSettlementPaid(ctx, db.CreateSettlementPaidParams{
			TripID: tripID, FromMember: from, ToMember: to, AmountBase: amountBase, CreatedBy: &actor.MemberID,
		})
		if err != nil {
			return err
		}
		return q.CreateActivity(ctx, db.CreateActivityParams{
			TripID: tripID, MemberID: &actor.MemberID, Action: "repayment_recorded",
			Title: fromName + " → " + toName, AmountBase: amountBase,
		})
	})
	return out, err
}

// Actor is the member making a change, and whether they own the trip.
type Actor struct {
	MemberID uuid.UUID
	IsOwner  bool
}
