package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/raisin/travel-split/backend/internal/expense"
	"github.com/raisin/travel-split/backend/internal/settlement"
	"github.com/raisin/travel-split/backend/internal/testdb"
	"github.com/raisin/travel-split/backend/internal/trip"
)

type apiClient struct {
	t  *testing.T
	h  http.Handler
	me string
}

func newAPI(t *testing.T) *apiClient {
	pool := testdb.New(t)
	h := NewServer(Deps{
		DB:          pool,
		Trips:       trip.NewService(pool),
		Expenses:    expense.NewService(pool),
		Settlements: settlement.NewService(pool),
	}).Routes()
	return &apiClient{t: t, h: h}
}

// do sends a request and decodes the JSON response into out (if non-nil).
func (c *apiClient) do(method, path string, body any, wantStatus int, out any) {
	c.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			c.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, "/api/v1"+path, &buf)
	if c.me != "" {
		req.Header.Set("X-Member-Id", c.me)
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		c.t.Fatalf("%s %s: status %d want %d; body %s", method, path, rec.Code, wantStatus, rec.Body)
	}
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			c.t.Fatalf("decode %s: %v", rec.Body, err)
		}
	}
}

func (c *apiClient) errCode(method, path string, body any, wantStatus int) string {
	c.t.Helper()
	var e errorBody
	c.do(method, path, body, wantStatus, &e)
	return e.Error.Code
}

type tripResp struct {
	ID           string `json:"id"`
	InviteCode   string `json:"invite_code"`
	BaseCurrency string `json:"base_currency"`
}

type memberResp struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestTripAndMembers(t *testing.T) {
	c := newAPI(t)

	var tr tripResp
	c.do("POST", "/trips", map[string]any{"name": "ญี่ปุ่น 2026"}, 201, &tr)
	if tr.BaseCurrency != "THB" || len(tr.InviteCode) != 10 {
		t.Fatalf("trip %+v", tr)
	}

	var got tripResp
	c.do("GET", "/trips/"+strings.ToLower(tr.InviteCode), nil, 200, &got)
	if got.ID != tr.ID {
		t.Fatal("invite lookup returned a different trip")
	}
	if code := c.errCode("GET", "/trips/NOPE", nil, 404); code != "trip_not_found" {
		t.Fatal(code)
	}
	if code := c.errCode("POST", "/trips", map[string]any{"name": " "}, 400); code != "invalid_name" {
		t.Fatal(code)
	}
	if code := c.errCode("POST", "/trips", map[string]any{"name": "x", "base_currency": "ZZZ"}, 400); code != "unsupported_currency" {
		t.Fatal(code)
	}

	var m memberResp
	c.do("POST", "/trips/"+tr.ID+"/members", map[string]any{"name": "แบงก์", "promptpay": "081-234-5678"}, 201, &m)
	if code := c.errCode("POST", "/trips/"+tr.ID+"/members", map[string]any{"name": " แบงก์ "}, 409); code != "member_name_taken" {
		t.Fatal(code)
	}
	var ms []memberResp
	c.do("GET", "/trips/"+tr.ID+"/members", nil, 200, &ms)
	if len(ms) != 1 || ms[0].ID != m.ID {
		t.Fatalf("members %+v", ms)
	}
	if code := c.errCode("GET", "/trips/00000000-0000-0000-0000-000000000000/members", nil, 404); code != "trip_not_found" {
		t.Fatal(code)
	}
}

func TestMemberHeaderIsChecked(t *testing.T) {
	c := newAPI(t)
	var t1, t2 tripResp
	c.do("POST", "/trips", map[string]any{"name": "A"}, 201, &t1)
	c.do("POST", "/trips", map[string]any{"name": "B"}, 201, &t2)
	var outsider memberResp
	c.do("POST", "/trips/"+t2.ID+"/members", map[string]any{"name": "คนนอก"}, 201, &outsider)

	path := "/trips/" + t1.ID + "/expenses"
	if code := c.errCode("GET", path, nil, 401); code != "member_required" {
		t.Fatal(code)
	}
	c.me = "not-a-uuid"
	if code := c.errCode("GET", path, nil, 403); code != "not_a_member" {
		t.Fatal(code)
	}
	c.me = outsider.ID
	if code := c.errCode("GET", path, nil, 403); code != "not_a_member" {
		t.Fatal(code)
	}
}

type expenseResp struct {
	ID         string          `json:"id"`
	Title      string          `json:"title"`
	AmountBase int64           `json:"amount_base"`
	FxRate     string          `json:"fx_rate"`
	CreatedBy  string          `json:"created_by"`
	Shares     []expense.Share `json:"shares"`
}

func TestExpensesBalancesAndSettlement(t *testing.T) {
	c := newAPI(t)
	var tr tripResp
	c.do("POST", "/trips", map[string]any{"name": "เชียงใหม่"}, 201, &tr)
	base := "/trips/" + tr.ID
	ids := map[string]string{}
	for _, n := range []string{"A", "B", "C"} {
		var m memberResp
		c.do("POST", base+"/members", map[string]any{"name": n}, 201, &m)
		ids[n] = m.ID
	}
	c.me = ids["B"]

	// A paid 100.00 split equally -> 33.34 / 33.33 / 33.33 (remainder to A, first in trip order).
	var dinner expenseResp
	c.do("POST", base+"/expenses", map[string]any{
		"title": "ข้าวเย็น", "category": "food", "paid_by": ids["A"], "amount": 10000,
		"split_type": "equal", "member_ids": []string{ids["C"], ids["B"], ids["A"]},
	}, 201, &dinner)
	if dinner.AmountBase != 10000 || dinner.FxRate != "1.00000000" || dinner.CreatedBy != ids["B"] {
		t.Fatalf("dinner %+v", dinner)
	}
	shares := map[string]int64{}
	for _, s := range dinner.Shares {
		shares[s.MemberID.String()] = s.AmountBase
	}
	if shares[ids["A"]] != 3334 || shares[ids["B"]] != 3333 || shares[ids["C"]] != 3333 {
		t.Fatalf("shares %v", shares)
	}

	// B paid 60.00 for a taxi, exact split: A 40.00, C 20.00.
	var taxi expenseResp
	c.do("POST", base+"/expenses", map[string]any{
		"title": "แท็กซี่", "category": "transport", "paid_by": ids["B"], "amount": 6000,
		"split_type": "exact", "shares": []map[string]any{
			{"member_id": ids["A"], "amount": 4000}, {"member_id": ids["C"], "amount": 2000},
		},
	}, 201, &taxi)

	// Validation failures.
	bad := []struct {
		body map[string]any
		code string
	}{
		{map[string]any{"title": "x", "category": "food", "paid_by": ids["A"], "amount": 100, "split_type": "exact",
			"shares": []map[string]any{{"member_id": ids["A"], "amount": 99}}}, "split_mismatch"},
		{map[string]any{"title": "x", "category": "food", "paid_by": ids["A"], "amount": 100, "currency": "JPY",
			"split_type": "equal", "member_ids": []string{ids["A"]}}, "unsupported_currency"},
		{map[string]any{"title": "x", "category": "nope", "paid_by": ids["A"], "amount": 100,
			"split_type": "equal", "member_ids": []string{ids["A"]}}, "invalid_category"},
		{map[string]any{"title": "x", "category": "food", "paid_by": ids["A"], "amount": 0,
			"split_type": "equal", "member_ids": []string{ids["A"]}}, "invalid_amount"},
		{map[string]any{"title": "x", "category": "food", "paid_by": ids["A"], "amount": 100,
			"split_type": "equal", "member_ids": []string{}}, "invalid_split"},
		{map[string]any{"title": "x", "category": "food", "paid_by": "00000000-0000-0000-0000-000000000001", "amount": 100,
			"split_type": "equal", "member_ids": []string{ids["A"]}}, "invalid_paid_by"},
	}
	for _, b := range bad {
		if code := c.errCode("POST", base+"/expenses", b.body, 400); code != b.code {
			t.Errorf("want %s got %s", b.code, code)
		}
	}

	// Filters.
	var list []expenseResp
	c.do("GET", base+"/expenses", nil, 200, &list)
	if len(list) != 2 {
		t.Fatalf("list %d", len(list))
	}
	c.do("GET", base+"/expenses?category=transport", nil, 200, &list)
	if len(list) != 1 || list[0].ID != taxi.ID {
		t.Fatalf("category filter %+v", list)
	}
	c.do("GET", base+"/expenses?member="+ids["C"], nil, 200, &list)
	if len(list) != 2 {
		t.Fatalf("member filter C %d", len(list))
	}

	type balancesResp struct {
		Currency string               `json:"currency"`
		Balances []settlement.Balance `json:"balances"`
	}
	var bal balancesResp
	c.do("GET", base+"/balances", nil, 200, &bal)
	// A: paid 10000, owes 3334+4000 -> +2666. B: paid 6000, owes 3333 -> +2667. C: owes 3333+2000 -> -5333.
	want := []int64{2666, 2667, -5333}
	for i, b := range bal.Balances {
		if b.Net != want[i] {
			t.Fatalf("balances %+v", bal.Balances)
		}
	}

	type planResp struct {
		Transfers []struct {
			From       string `json:"from"`
			To         string `json:"to"`
			AmountBase int64  `json:"amount_base"`
		} `json:"transfers"`
	}
	var plan planResp
	c.do("GET", base+"/settlement", nil, 200, &plan)
	if len(plan.Transfers) != 2 || plan.Transfers[0].From != ids["C"] || plan.Transfers[0].To != ids["B"] || plan.Transfers[0].AmountBase != 2667 {
		t.Fatalf("plan %+v", plan)
	}

	// C pays B; only C -> A remains.
	c.do("POST", base+"/settlements/paid", map[string]any{"from": ids["C"], "to": ids["B"], "amount_base": 2667}, 201, nil)
	if code := c.errCode("POST", base+"/settlements/paid", map[string]any{"from": ids["C"], "to": ids["C"], "amount_base": 1}, 400); code != "invalid_transfer" {
		t.Fatal(code)
	}
	c.do("GET", base+"/settlement", nil, 200, &plan)
	if len(plan.Transfers) != 1 || plan.Transfers[0].To != ids["A"] || plan.Transfers[0].AmountBase != 2666 {
		t.Fatalf("plan after paid %+v", plan)
	}

	// Update the taxi to an equal split between all three, then delete dinner.
	var updated expenseResp
	c.do("PUT", base+"/expenses/"+taxi.ID, map[string]any{
		"title": "แท็กซี่ (แก้)", "category": "transport", "paid_by": ids["B"], "amount": 6000,
		"split_type": "equal", "member_ids": []string{ids["A"], ids["B"], ids["C"]},
	}, 200, &updated)
	if updated.Title != "แท็กซี่ (แก้)" || len(updated.Shares) != 3 {
		t.Fatalf("updated %+v", updated)
	}
	c.do("DELETE", base+"/expenses/"+dinner.ID, nil, 204, nil)
	if code := c.errCode("DELETE", base+"/expenses/"+dinner.ID, nil, 404); code != "expense_not_found" {
		t.Fatal(code)
	}
	if code := c.errCode("PUT", base+"/expenses/"+dinner.ID, map[string]any{
		"title": "x", "category": "food", "paid_by": ids["A"], "amount": 1, "split_type": "equal", "member_ids": []string{ids["A"]},
	}, 404); code != "expense_not_found" {
		t.Fatal(code)
	}

	var one expenseResp
	c.do("GET", base+"/expenses/"+taxi.ID, nil, 200, &one)
	if len(one.Shares) != 3 {
		t.Fatalf("get one %+v", one)
	}
}
