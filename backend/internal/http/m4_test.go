package http

import (
	"context"
	"testing"
	"time"

	"github.com/raisin/travel-split/backend/internal/platform/apperr"
	"github.com/raisin/travel-split/backend/internal/domain/expense"
	"github.com/raisin/travel-split/backend/internal/domain/fx"
)

// fakeRates serves fixed rates and records the day it was asked about.
type fakeRates struct {
	rates map[string]string
	asked []string
}

func (f *fakeRates) Rate(_ context.Context, from, to string, date *time.Time) (fx.Quote, error) {
	f.asked = append(f.asked, from+to+"@"+date.Format(time.DateOnly))
	r, ok := f.rates[from+to]
	if !ok {
		return fx.Quote{}, apperr.Invalid("unsupported_currency", "no rate")
	}
	return fx.Quote{From: from, To: to, Rate: r}, nil
}

type fullExpense struct {
	expenseResp
	Amount    int64          `json:"amount"`
	Currency  string         `json:"currency"`
	SplitType string         `json:"split_type"`
	Items     []expense.Item `json:"items"`
}

func TestForeignCurrencyExpenses(t *testing.T) {
	rates := &fakeRates{rates: map[string]string{"JPYTHB": "0.2345678912"}}
	c := newAPIWithRates(t, rates)
	tr, a, owner := c.newTrip("Tokyo", "A")
	b, _ := c.addMember(tr.ID, owner, "B")
	c.me = owner
	base := "/trips/" + tr.ID

	// ¥1,500 on a past date: looked up for that day, rounded to 8 decimals.
	var e fullExpense
	c.do("POST", base+"/expenses", map[string]any{
		"title": "Ramen", "category": "food", "paid_by": a.ID, "amount": 1500, "currency": "jpy",
		"spent_at": "2026-11-01T12:00:00Z", "split_type": "equal", "member_ids": []string{a.ID, b.ID},
	}, 201, &e)
	if e.Currency != "JPY" || e.FxRate != "0.23456789" || e.AmountBase != 35185 {
		t.Fatalf("converted expense %+v", e)
	}
	if e.Shares[0].AmountBase+e.Shares[1].AmountBase != 35185 {
		t.Fatalf("shares %+v", e.Shares)
	}
	if rates.asked[0] != "JPYTHB@2026-11-01" {
		t.Fatalf("asked %v", rates.asked)
	}

	// A future date asks for today's rate instead.
	future := time.Now().Add(72 * time.Hour).UTC().Format(time.RFC3339)
	c.do("POST", base+"/expenses", map[string]any{
		"title": "Hotel", "category": "lodging", "paid_by": a.ID, "amount": 100, "currency": "JPY",
		"spent_at": future, "split_type": "equal", "member_ids": []string{a.ID},
	}, 201, nil)
	if want := "JPYTHB@" + time.Now().UTC().Format(time.DateOnly); rates.asked[1] != want {
		t.Fatalf("asked %v want %s", rates.asked, want)
	}

	// A user-entered rate wins, even for a currency the provider lacks.
	c.do("POST", base+"/expenses", map[string]any{
		"title": "Pho", "category": "food", "paid_by": b.ID, "amount": 150000, "currency": "VND", "fx_rate": "0.00136",
		"split_type": "exact", "shares": []map[string]any{{"member_id": a.ID, "amount": 100000}, {"member_id": b.ID, "amount": 50000}},
	}, 201, &e)
	if e.AmountBase != 20400 || e.FxRate != "0.00136" || e.Shares[0].AmountBase != 13600 || e.Shares[1].AmountBase != 6800 {
		t.Fatalf("manual rate %+v", e)
	}

	for body, code := range map[string]map[string]any{
		"unsupported_currency": {"currency": "VND"}, // provider has no rate, none given
		"invalid_fx_rate":      {"currency": "VND", "fx_rate": "-1"},
		"amount_too_small":     {"currency": "VND", "fx_rate": "0.00001", "amount": 1},
	} {
		req := map[string]any{"title": "x", "category": "food", "paid_by": a.ID, "amount": 1000, "split_type": "equal", "member_ids": []string{a.ID}}
		for k, v := range code {
			req[k] = v
		}
		if got := c.errCode("POST", base+"/expenses", req, 400); got != body {
			t.Errorf("%v: got %s want %s", code, got, body)
		}
	}
}

func TestItemSplitAndSummary(t *testing.T) {
	c := newAPI(t)
	tr, a, owner := c.newTrip("Seoul", "A")
	b, _ := c.addMember(tr.ID, owner, "B")
	cc, _ := c.addMember(tr.ID, owner, "C")
	c.me = owner
	base := "/trips/" + tr.ID

	// ฿1,070 bill: items add up to ฿1,000, the extra 7% VAT spreads by price.
	var e fullExpense
	c.do("POST", base+"/expenses", map[string]any{
		"title": "BBQ", "category": "food", "paid_by": a.ID, "amount": 107000, "split_type": "items",
		"items": []map[string]any{
			{"name": " Pork ", "amount": 60000, "member_ids": []string{a.ID, b.ID}},
			{"name": "Beer", "amount": 30000, "member_ids": []string{cc.ID}},
			{"name": "Rice", "amount": 10000, "member_ids": []string{cc.ID, b.ID, a.ID, a.ID}},
		},
	}, 201, &e)
	got := map[string]int64{}
	for _, s := range e.Shares {
		got[s.MemberID.String()] = s.AmountBase
	}
	// Pork 64200 -> 32100 each; beer 32100 -> C; rice 10700 -> 3567/3567/3566 in trip order.
	if got[a.ID] != 32100+3567 || got[b.ID] != 32100+3567 || got[cc.ID] != 32100+3566 {
		t.Fatalf("item shares %v", got)
	}
	if len(e.Items) != 3 || e.Items[0].Name != "Pork" || len(e.Items[2].MemberIDs) != 3 || e.Items[2].MemberIDs[0].String() != a.ID {
		t.Fatalf("items %+v", e.Items)
	}

	// Items come back on GET, in order; editing replaces them.
	c.do("GET", base+"/expenses/"+e.ID, nil, 200, &e)
	if e.SplitType != "items" || len(e.Items) != 3 || e.Items[1].Name != "Beer" {
		t.Fatalf("reloaded %+v", e)
	}
	c.do("PUT", base+"/expenses/"+e.ID, map[string]any{
		"title": "BBQ", "category": "food", "paid_by": a.ID, "amount": 107000, "split_type": "equal", "member_ids": []string{a.ID, b.ID},
	}, 200, &e)
	if len(e.Items) != 0 {
		t.Fatalf("items kept after switching split: %+v", e.Items)
	}

	if code := c.errCode("POST", base+"/expenses", map[string]any{
		"title": "x", "category": "food", "paid_by": a.ID, "amount": 100, "split_type": "items",
		"items": []map[string]any{{"name": "x", "amount": 100, "member_ids": []string{}}},
	}, 400); code != "item_no_members" {
		t.Errorf("got %s", code)
	}

	c.do("POST", base+"/expenses", map[string]any{
		"title": "Taxi", "category": "transport", "paid_by": b.ID, "amount": 30000, "split_type": "equal", "member_ids": []string{a.ID, b.ID, cc.ID},
	}, 201, nil)

	var sum struct {
		Currency   string `json:"currency"`
		Total      int64  `json:"total"`
		Count      int    `json:"count"`
		ByCategory []struct {
			Category   string `json:"category"`
			AmountBase int64  `json:"amount_base"`
			Count      int    `json:"count"`
		} `json:"by_category"`
		ByMember []struct {
			MemberID string `json:"member_id"`
			Paid     int64  `json:"paid"`
			Consumed int64  `json:"consumed"`
		} `json:"by_member"`
	}
	c.do("GET", base+"/summary", nil, 200, &sum)
	if sum.Currency != "THB" || sum.Total != 137000 || sum.Count != 2 || len(sum.ByCategory) != 2 ||
		sum.ByCategory[0].Category != "food" || sum.ByCategory[0].AmountBase != 107000 {
		t.Fatalf("summary %+v", sum)
	}
	if len(sum.ByMember) != 3 || sum.ByMember[0].Paid != 107000 || sum.ByMember[2].Consumed != 10000 {
		t.Fatalf("by member %+v", sum.ByMember)
	}
}

func TestEditOwnPromptPay(t *testing.T) {
	c := newAPI(t)
	tr, _, owner := c.newTrip("Krabi", "A")
	_, bTok := c.addMember(tr.ID, owner, "B")
	c.me = bTok
	var m struct {
		Name      string  `json:"name"`
		PromptPay *string `json:"promptpay"`
	}
	c.do("PATCH", "/trips/"+tr.ID+"/me", map[string]any{"promptpay": "081-234-5678"}, 200, &m)
	if m.Name != "B" || m.PromptPay == nil || *m.PromptPay != "0812345678" {
		t.Fatalf("member %+v", m)
	}
	if code := c.errCode("PATCH", "/trips/"+tr.ID+"/me", map[string]any{"promptpay": "12"}, 400); code != "invalid_promptpay" {
		t.Errorf("got %s", code)
	}
	c.do("PATCH", "/trips/"+tr.ID+"/me", map[string]any{"promptpay": ""}, 200, &m)
	if m.PromptPay != nil {
		t.Fatalf("not cleared: %v", *m.PromptPay)
	}
}
