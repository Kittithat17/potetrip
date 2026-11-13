// Package settlement computes member balances and the minimal set of
// transfers that clears all debts. All amounts are in the trip's base
// currency minor units.
package settlement

import "github.com/google/uuid"

// Balance is one member's net position. Positive means the member should
// receive money; negative means the member owes.
type Balance struct {
	MemberID uuid.UUID `json:"member_id"`
	Paid     int64     `json:"paid"`
	Owed     int64     `json:"owed"`
	Sent     int64     `json:"sent"`
	Received int64     `json:"received"`
	Net      int64     `json:"net"`
}

// Transfer is a payment that settles (part of) a debt.
type Transfer struct {
	From       uuid.UUID `json:"from"`
	To         uuid.UUID `json:"to"`
	AmountBase int64     `json:"amount_base"`
}

// Input aggregates what each member paid and owes, plus transfers already made.
type Input struct {
	Members []uuid.UUID         // fixes output order
	Paid    map[uuid.UUID]int64 // sum of expenses.amount_base by payer
	Owed    map[uuid.UUID]int64 // sum of expense_shares.amount_base by member
	Settled []Transfer          // settlements_paid rows
}

// Balances returns one entry per member, in Input.Members order.
// Net = paid - owed + sent - received.
func Balances(in Input) []Balance {
	sent := map[uuid.UUID]int64{}
	received := map[uuid.UUID]int64{}
	for _, t := range in.Settled {
		sent[t.From] += t.AmountBase
		received[t.To] += t.AmountBase
	}
	out := make([]Balance, 0, len(in.Members))
	for _, id := range in.Members {
		b := Balance{
			MemberID: id,
			Paid:     in.Paid[id],
			Owed:     in.Owed[id],
			Sent:     sent[id],
			Received: received[id],
		}
		b.Net = b.Paid - b.Owed + b.Sent - b.Received
		out = append(out, b)
	}
	return out
}

// Settle greedily matches the largest debtor with the largest creditor and
// transfers min(debt, credit), repeating until everyone is even. Ties go to
// the member listed first, which keeps the result deterministic.
func Settle(balances []Balance) []Transfer {
	type entry struct {
		id  uuid.UUID
		amt int64
	}
	var creditors, debtors []entry
	for _, b := range balances {
		switch {
		case b.Net > 0:
			creditors = append(creditors, entry{b.MemberID, b.Net})
		case b.Net < 0:
			debtors = append(debtors, entry{b.MemberID, -b.Net})
		}
	}

	largest := func(es []entry) int {
		best := -1
		for i, e := range es {
			if e.amt > 0 && (best < 0 || e.amt > es[best].amt) {
				best = i
			}
		}
		return best
	}

	var out []Transfer
	for {
		d, c := largest(debtors), largest(creditors)
		if d < 0 || c < 0 {
			return out
		}
		amt := min(debtors[d].amt, creditors[c].amt)
		out = append(out, Transfer{From: debtors[d].id, To: creditors[c].id, AmountBase: amt})
		debtors[d].amt -= amt
		creditors[c].amt -= amt
	}
}
