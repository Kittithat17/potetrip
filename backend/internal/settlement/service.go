package settlement

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raisin/travel-split/backend/internal/apperr"
	"github.com/raisin/travel-split/backend/internal/db"
)

// Service loads trip totals from the database and applies the pure
// Balances/Settle functions.
type Service struct {
	q *db.Queries
}

func NewService(pool *pgxpool.Pool) *Service { return &Service{q: db.New(pool)} }

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
func (s *Service) MarkPaid(ctx context.Context, tripID, from, to uuid.UUID, amountBase int64) (db.SettlementsPaid, error) {
	if amountBase <= 0 {
		return db.SettlementsPaid{}, apperr.Invalid("invalid_amount", "จำนวนเงินต้องมากกว่า 0")
	}
	if from == to {
		return db.SettlementsPaid{}, apperr.Invalid("invalid_transfer", "ผู้โอนและผู้รับต้องเป็นคนละคน")
	}
	members, err := s.q.ListMembers(ctx, tripID)
	if err != nil {
		return db.SettlementsPaid{}, err
	}
	var okFrom, okTo bool
	for _, m := range members {
		okFrom = okFrom || m.ID == from
		okTo = okTo || m.ID == to
	}
	if !okFrom || !okTo {
		return db.SettlementsPaid{}, apperr.Invalid("invalid_transfer", "ผู้โอนหรือผู้รับไม่ได้อยู่ในทริปนี้")
	}
	return s.q.CreateSettlementPaid(ctx, db.CreateSettlementPaidParams{
		TripID: tripID, FromMember: from, ToMember: to, AmountBase: amountBase,
	})
}
