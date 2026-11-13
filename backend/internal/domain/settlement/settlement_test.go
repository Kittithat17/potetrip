package settlement

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/raisin/travel-split/backend/internal/domain/expense"
)

var (
	a = uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	b = uuid.MustParse("00000000-0000-0000-0000-00000000000b")
	c = uuid.MustParse("00000000-0000-0000-0000-00000000000c")
	d = uuid.MustParse("00000000-0000-0000-0000-00000000000d")
)

// ledger accumulates expenses the same way the DB aggregates do.
type ledger struct {
	members []uuid.UUID
	paid    map[uuid.UUID]int64
	owed    map[uuid.UUID]int64
	settled []Transfer
}

func newLedger(members ...uuid.UUID) *ledger {
	return &ledger{members: members, paid: map[uuid.UUID]int64{}, owed: map[uuid.UUID]int64{}}
}

func (l *ledger) expenseEqual(t *testing.T, payer uuid.UUID, amountBase int64, among ...uuid.UUID) {
	t.Helper()
	shares, err := expense.SplitEqual(amountBase, len(among))
	if err != nil {
		t.Fatal(err)
	}
	l.paid[payer] += amountBase
	for i, m := range among {
		l.owed[m] += shares[i]
	}
}

func (l *ledger) input() Input {
	return Input{Members: l.members, Paid: l.paid, Owed: l.owed, Settled: l.settled}
}

func nets(bs []Balance) []int64 {
	out := make([]int64, len(bs))
	for i, b := range bs {
		out[i] = b.Net
	}
	return out
}

// checkSettles asserts that applying the transfers zeroes every balance.
func checkSettles(t *testing.T, bs []Balance, ts []Transfer) {
	t.Helper()
	net := map[uuid.UUID]int64{}
	var total int64
	for _, x := range bs {
		net[x.MemberID] = x.Net
		total += x.Net
	}
	if total != 0 {
		t.Fatalf("balances do not sum to zero: %d", total)
	}
	for _, tr := range ts {
		if tr.AmountBase <= 0 {
			t.Fatalf("non-positive transfer %+v", tr)
		}
		net[tr.From] += tr.AmountBase
		net[tr.To] -= tr.AmountBase
	}
	for id, v := range net {
		if v != 0 {
			t.Fatalf("member %s left with %d", id, v)
		}
	}
	if n := len(bs); len(ts) > n-1 && n > 0 {
		t.Fatalf("%d transfers for %d members", len(ts), n)
	}
}

func TestThreeWaySplitWithRemainder(t *testing.T) {
	l := newLedger(a, b, c)
	l.expenseEqual(t, a, 10000, a, b, c) // 100.00 THB -> 33.34 / 33.33 / 33.33

	bs := Balances(l.input())
	if got, want := nets(bs), []int64{6666, -3333, -3333}; !slices.Equal(got, want) {
		t.Fatalf("nets = %v want %v", got, want)
	}
	ts := Settle(bs)
	checkSettles(t, bs, ts)
	want := []Transfer{{b, a, 3333}, {c, a, 3333}}
	if !slices.Equal(ts, want) {
		t.Fatalf("transfers = %v want %v", ts, want)
	}
}

func TestOnePersonPaysEverything(t *testing.T) {
	l := newLedger(a, b, c, d)
	l.expenseEqual(t, b, 120000, a, b, c, d)
	l.expenseEqual(t, b, 4000, a, b, c, d)
	l.expenseEqual(t, b, 9000, a, c) // only a and c ate dessert

	bs := Balances(l.input())
	ts := Settle(bs)
	checkSettles(t, bs, ts)
	for _, tr := range ts {
		if tr.To != b {
			t.Fatalf("everyone should pay b, got %+v", tr)
		}
	}
	if len(ts) != 3 {
		t.Fatalf("want 3 transfers, got %v", ts)
	}
}

func TestAllEvenNoTransfers(t *testing.T) {
	l := newLedger(a, b, c)
	l.expenseEqual(t, a, 3000, a, b, c)
	l.expenseEqual(t, b, 3000, a, b, c)
	l.expenseEqual(t, c, 3000, a, b, c)

	bs := Balances(l.input())
	if got := nets(bs); !slices.Equal(got, []int64{0, 0, 0}) {
		t.Fatalf("nets = %v", got)
	}
	if ts := Settle(bs); len(ts) != 0 {
		t.Fatalf("want no transfers, got %v", ts)
	}
	if ts := Settle(Balances(Input{})); len(ts) != 0 {
		t.Fatalf("empty trip: %v", ts)
	}
}

func TestMultiCurrency(t *testing.T) {
	// Expenses arrive already converted to base (THB satang); shares of a
	// converted expense are allocated from amount_base so they still sum up.
	l := newLedger(a, b, c)

	// a paid 3,000 JPY at 0.2263 THB/JPY -> 678.90 THB, split equally.
	l.expenseEqual(t, a, 67890, a, b, c)

	// b paid 25.00 USD at 32.95 -> 823.75 THB, exact split 10/15 USD between b and c.
	usdShares, err := expense.Allocate(82375, []int64{1000, 1500})
	if err != nil {
		t.Fatal(err)
	}
	l.paid[b] += 82375
	l.owed[b] += usdShares[0]
	l.owed[c] += usdShares[1]

	// c paid 450.00 THB directly, split equally.
	l.expenseEqual(t, c, 45000, a, b, c)

	bs := Balances(l.input())
	ts := Settle(bs)
	checkSettles(t, bs, ts)

	var totalPaid, totalOwed int64
	for _, x := range bs {
		totalPaid += x.Paid
		totalOwed += x.Owed
	}
	if totalPaid != totalOwed || totalPaid != 67890+82375+45000 {
		t.Fatalf("paid %d owed %d", totalPaid, totalOwed)
	}
}

func TestPartialSettlementAlreadyPaid(t *testing.T) {
	l := newLedger(a, b, c)
	l.expenseEqual(t, a, 90000, a, b, c) // b and c each owe a 300.00

	l.settled = []Transfer{
		{From: b, To: a, AmountBase: 30000}, // b paid in full
		{From: c, To: a, AmountBase: 10000}, // c paid a third
	}
	bs := Balances(l.input())
	if got, want := nets(bs), []int64{20000, 0, -20000}; !slices.Equal(got, want) {
		t.Fatalf("nets = %v want %v", got, want)
	}
	if bs[1].Sent != 30000 || bs[0].Received != 40000 {
		t.Fatalf("sent/received wrong: %+v", bs)
	}
	ts := Settle(bs)
	checkSettles(t, bs, ts)
	if want := []Transfer{{c, a, 20000}}; !slices.Equal(ts, want) {
		t.Fatalf("transfers = %v want %v", ts, want)
	}
}

func TestGreedyPicksLargest(t *testing.T) {
	bs := []Balance{
		{MemberID: a, Net: 500},
		{MemberID: b, Net: 300},
		{MemberID: c, Net: -100},
		{MemberID: d, Net: -700},
	}
	ts := Settle(bs)
	checkSettles(t, bs, ts)
	want := []Transfer{{d, a, 500}, {d, b, 200}, {c, b, 100}}
	if !slices.Equal(ts, want) {
		t.Fatalf("transfers = %v want %v", ts, want)
	}
}
