package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/raisin/travel-split/backend/internal/domain/auth"
	"github.com/raisin/travel-split/backend/internal/domain/expense"
	"github.com/raisin/travel-split/backend/internal/domain/settlement"
	"github.com/raisin/travel-split/backend/internal/platform/testdb"
	"github.com/raisin/travel-split/backend/internal/domain/trip"
)

type apiClient struct {
	t     *testing.T
	h     http.Handler
	me    string // session token sent as the session cookie
	users int
}

// testDeps is a server with mock sign-in, as in local dev.
func testDeps(pool *pgxpool.Pool) Deps {
	return Deps{
		DB:          pool,
		Auth:        auth.NewService(pool),
		AuthCfg:     AuthConfig{Provider: "mock", PublicURL: "http://localhost:5173"},
		Trips:       trip.NewService(pool),
		Expenses:    expense.NewService(pool, nil),
		Settlements: settlement.NewService(pool),
	}
}

func newAPI(t *testing.T) *apiClient { return newAPIWithRates(t, nil) }

func newAPIWithRates(t *testing.T, rater expense.Rater) *apiClient {
	pool := testdb.New(t)
	d := testDeps(pool)
	d.Expenses = expense.NewService(pool, rater)
	return &apiClient{t: t, h: NewServer(d).Routes()}
}

// authorize adds what the app sends on every request: the session cookie
// and the anti-CSRF header.
func (c *apiClient) authorize(req *http.Request, session string) {
	req.Header.Set(CSRFHeader, CSRFValue)
	if session != "" {
		req.AddCookie(&http.Cookie{Name: sessionCookie, Value: session})
	}
}

// login signs a brand-new user in (mock provider) and returns their session.
func (c *apiClient) login(name string) string {
	c.t.Helper()
	c.users++
	req := httptest.NewRequest("POST", "/api/v1/auth/mock", strings.NewReader(fmt.Sprintf(`{"name":%q}`, fmt.Sprintf("%s #%d", name, c.users))))
	c.authorize(req, "")
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		c.t.Fatalf("mock login: %d %s", rec.Code, rec.Body)
	}
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == sessionCookie && ck.Value != "" {
			return ck.Value
		}
	}
	c.t.Fatal("mock login set no session cookie")
	return ""
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
	c.authorize(req, c.me)
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
	ID            string  `json:"id"`
	InviteCode    string  `json:"invite_code"`
	BaseCurrency  string  `json:"base_currency"`
	OwnerMemberID *string `json:"owner_member_id"`
}

type memberResp struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// newTrip signs in a new user, who creates a trip and stays signed in (c.me);
// the returned session belongs to the owner.
func (c *apiClient) newTrip(name, creator string) (tripResp, memberResp, string) {
	c.t.Helper()
	c.me = c.login(creator)
	var out struct {
		Trip   tripResp   `json:"trip"`
		Member memberResp `json:"member"`
	}
	c.do("POST", "/trips", map[string]any{"name": name, "creator_name": creator}, 201, &out)
	return out.Trip, out.Member, c.me
}

// join signs in a new user who files a join request; returns their session
// and the request id. c.me is left as it was.
func (c *apiClient) join(tripID string, body map[string]any) (string, string) {
	c.t.Helper()
	var out struct {
		Request struct {
			ID string `json:"id"`
		} `json:"request"`
	}
	saved := c.me
	c.me = c.login("joiner")
	session := c.me
	c.do("POST", "/trips/"+tripID+"/join", body, 201, &out)
	c.me = saved
	return session, out.Request.ID
}

// addMember joins and gets approved by ownerToken; returns the member and its token.
func (c *apiClient) addMember(tripID, ownerToken, name string) (memberResp, string) {
	c.t.Helper()
	token, reqID := c.join(tripID, map[string]any{"name": name})
	saved := c.me
	c.me = ownerToken
	var m memberResp
	c.do("POST", "/trips/"+tripID+"/join-requests/"+reqID+"/approve", nil, 200, &m)
	c.me = saved
	return m, token
}

func TestCreateTripAndInvite(t *testing.T) {
	c := newAPI(t)

	tr, owner, token := c.newTrip("ญี่ปุ่น 2026", "แบงก์")
	if tr.BaseCurrency != "THB" || len(tr.InviteCode) != 10 || tr.OwnerMemberID == nil || *tr.OwnerMemberID != owner.ID || len(token) < 40 || c.me != token {
		t.Fatalf("trip %+v owner %+v token %q", tr, owner, token)
	}

	var got struct {
		Trip    tripResp `json:"trip"`
		Members []map[string]any
	}
	c.do("GET", "/trips/"+strings.ToLower(tr.InviteCode), nil, 200, &got)
	if got.Trip.ID != tr.ID || len(got.Members) != 1 || got.Members[0]["name"] != "แบงก์" {
		t.Fatalf("invite lookup %+v", got)
	}
	if _, leaked := got.Members[0]["promptpay"]; leaked {
		t.Fatal("public member list must not expose promptpay")
	}
	if code := c.errCode("GET", "/trips/NOPE", nil, 404); code != "trip_not_found" {
		t.Fatal(code)
	}
	for _, body := range []map[string]any{{"name": " ", "creator_name": "a"}, {"name": "x", "creator_name": strings.Repeat("ก", 51)}} {
		if code := c.errCode("POST", "/trips", body, 400); code != "invalid_name" {
			t.Fatal(code)
		}
	}
	if code := c.errCode("POST", "/trips", map[string]any{"name": "x", "creator_name": "a", "base_currency": "ZZZ"}, 400); code != "unsupported_currency" {
		t.Fatal(code)
	}
}

func TestJoinNeedsOwnerApproval(t *testing.T) {
	c := newAPI(t)
	tr, owner, ownerToken := c.newTrip("เชียงใหม่", "แบงก์")
	base := "/trips/" + tr.ID

	// A stranger with the invite code can only ask.
	strangerToken, reqID := c.join(tr.ID, map[string]any{"name": "ฝน", "promptpay": "081-234-5678"})
	c.me = strangerToken
	var me map[string]any
	c.do("GET", base+"/me", nil, 200, &me)
	if me["status"] != "pending" || me["member"] != nil {
		t.Fatalf("pending me %+v", me)
	}
	// Their trip list shows the trip as waiting (no member yet).
	var mine []map[string]any
	c.do("GET", "/me/trips", nil, 200, &mine)
	if len(mine) != 1 || mine[0]["status"] != "pending" || mine[0]["member_name"] != nil || mine[0]["is_owner"] != false {
		t.Fatalf("pending trip list %+v", mine)
	}
	for _, path := range []string{"/members", "/expenses", "/balances", "/settlement"} {
		if code := c.errCode("GET", base+path, nil, 403); code != "pending_approval" {
			t.Fatalf("%s: %s", path, code)
		}
	}
	if code := c.errCode("GET", base+"/join-requests", nil, 403); code != "pending_approval" {
		t.Fatal(code)
	}

	// Claiming an existing member also needs approval; names must be unique.
	if code := func() string {
		c.me = c.login("someone")
		return c.errCode("POST", base+"/join", map[string]any{"name": " แบงก์ "}, 409)
	}(); code != "member_name_taken" {
		t.Fatal(code)
	}
	// Asking twice is refused while the first request is open.
	c.me = strangerToken
	if code := c.errCode("POST", base+"/join", map[string]any{"name": "ฝน2"}, 409); code != "already_requested" {
		t.Fatal(code)
	}
	impostorToken, impostorReq := c.join(tr.ID, map[string]any{"member_id": owner.ID})

	// Only the owner sees and decides requests.
	c.me = ownerToken
	var reqs []map[string]any
	c.do("GET", base+"/join-requests", nil, 200, &reqs)
	if len(reqs) != 2 || reqs[1]["requested_member_id"] != owner.ID {
		t.Fatalf("requests %+v", reqs)
	}
	var fon memberResp
	c.do("POST", base+"/join-requests/"+reqID+"/approve", nil, 200, &fon)
	if fon.Name != "ฝน" {
		t.Fatalf("approved %+v", fon)
	}
	c.do("POST", base+"/join-requests/"+impostorReq+"/reject", nil, 204, nil)
	if code := c.errCode("POST", base+"/join-requests/"+reqID+"/approve", nil, 404); code != "request_not_found" {
		t.Fatal(code)
	}

	// Approved stranger is now a member but not the owner.
	c.me = strangerToken
	c.do("GET", base+"/me", nil, 200, &me)
	if me["status"] != "approved" || me["is_owner"] != false {
		t.Fatalf("approved me %+v", me)
	}
	var ms []map[string]any
	c.do("GET", base+"/members", nil, 200, &ms)
	if len(ms) != 2 || ms[1]["promptpay"] != "0812345678" {
		t.Fatalf("members %+v", ms)
	}
	if code := c.errCode("GET", base+"/join-requests", nil, 403); code != "owner_only" {
		t.Fatal(code)
	}

	c.me = impostorToken
	if code := c.errCode("GET", base+"/expenses", nil, 403); code != "join_rejected" {
		t.Fatal(code)
	}
	// A turned-down user may ask again (as a new name); a member may not.
	c.do("POST", base+"/join", map[string]any{"name": "มิ้นท์"}, 201, nil)
	c.do("GET", base+"/me", nil, 200, &me)
	if me["status"] != "pending" {
		t.Fatalf("re-request %+v", me)
	}
	c.me = strangerToken
	if code := c.errCode("POST", base+"/join", map[string]any{"name": "x"}, 409); code != "already_member" {
		t.Fatal(code)
	}
}

func TestAccessIsCheckedPerTrip(t *testing.T) {
	c := newAPI(t)
	t1, _, _ := c.newTrip("A", "a")
	_, _, otherSession := c.newTrip("B", "b")

	path := "/trips/" + t1.ID + "/expenses"
	for _, s := range []string{"", "garbage"} {
		c.me = s
		if code := c.errCode("GET", path, nil, 401); code != "login_required" {
			t.Fatalf("%q: %s", s, code)
		}
	}
	c.me = otherSession
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
	tr, a, ownerToken := c.newTrip("เชียงใหม่", "A")
	base := "/trips/" + tr.ID
	ids := map[string]string{"A": a.ID}
	tokens := map[string]string{"A": ownerToken}
	for _, n := range []string{"B", "C"} {
		m, tok := c.addMember(tr.ID, ownerToken, n)
		ids[n], tokens[n] = m.ID, tok
	}
	c.me = tokens["B"]

	// A paid 100.00 split equally -> 33.34 / 33.33 / 33.33 (remainder to A, first in trip order).
	var dinner expenseResp
	c.do("POST", base+"/expenses", map[string]any{
		"title": "ข้าวเย็น", "category": "food", "paid_by": ids["A"], "amount": 10000,
		"split_type": "equal", "member_ids": []string{ids["C"], ids["B"], ids["A"]},
	}, 201, &dinner)
	if dinner.AmountBase != 10000 || dinner.FxRate != "1" || dinner.CreatedBy != ids["B"] {
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
			"split_type": "equal", "member_ids": []string{ids["A"]}}, "fx_unavailable"}, // no rate source configured
		{map[string]any{"title": "x", "category": "food", "paid_by": ids["A"], "amount": 100, "currency": "ZZZ",
			"split_type": "equal", "member_ids": []string{ids["A"]}}, "invalid_currency"},
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

func TestOwnerEditsTripCurrencyUntilFirstExpense(t *testing.T) {
	c := newAPI(t)
	tr, owner, ownerToken := c.newTrip("โตเกียว", "แบงก์")
	base := "/trips/" + tr.ID
	_, friendToken := c.addMember(tr.ID, ownerToken, "ฝน")

	c.me = friendToken
	if code := c.errCode("PATCH", base, map[string]any{"name": "x"}, 403); code != "owner_only" {
		t.Fatal(code)
	}

	c.me = ownerToken
	var got tripResp
	c.do("PATCH", base, map[string]any{"name": "  ญี่ปุ่น  ", "base_currency": "jpy"}, 200, &got)
	if got.BaseCurrency != "JPY" {
		t.Fatalf("%+v", got)
	}
	if code := c.errCode("PATCH", base, map[string]any{"base_currency": "ZZZ"}, 400); code != "unsupported_currency" {
		t.Fatal(code)
	}

	// After the first expense the currency is locked, but renaming still works.
	c.do("POST", base+"/expenses", map[string]any{"title": "ราเมง", "category": "food", "paid_by": owner.ID,
		"amount": 3000, "currency": "JPY", "split_type": "equal", "member_ids": []string{owner.ID}}, 201, nil)
	if code := c.errCode("PATCH", base, map[string]any{"base_currency": "THB"}, 409); code != "currency_locked" {
		t.Fatal(code)
	}
	c.do("PATCH", base, map[string]any{"name": "ญี่ปุ่น 2026", "base_currency": "JPY"}, 200, &got)
	var inv struct {
		Trip struct {
			Name         string `json:"name"`
			BaseCurrency string `json:"base_currency"`
		} `json:"trip"`
	}
	c.do("GET", "/trips/"+tr.InviteCode, nil, 200, &inv)
	if inv.Trip.Name != "ญี่ปุ่น 2026" || inv.Trip.BaseCurrency != "JPY" {
		t.Fatalf("%+v", inv)
	}
}
