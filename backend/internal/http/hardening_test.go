package http

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/raisin/travel-split/backend/internal/platform/testdb"
)

// Claiming an existing member works only for members nobody signs in as
// yet (from before accounts); once someone does, the name is theirs.
func TestClaimOnlyUnlinkedMembers(t *testing.T) {
	pool := testdb.New(t)
	c := &apiClient{t: t, h: NewServer(testDeps(pool)).Routes()}
	tr, owner, ownerSession := c.newTrip("ทริป", "แบงก์")
	base := "/trips/" + tr.ID
	var katie string
	if err := pool.QueryRow(t.Context(), `INSERT INTO members (trip_id, name) VALUES ($1, 'katie') RETURNING id`, tr.ID).Scan(&katie); err != nil {
		t.Fatal(err)
	}

	// The invite page says who can still be claimed.
	var inv struct {
		Members []struct {
			ID        string `json:"id"`
			Claimable bool   `json:"claimable"`
		} `json:"members"`
	}
	c.do("GET", "/trips/"+tr.InviteCode, nil, 200, &inv)
	claimable := map[string]bool{}
	for _, m := range inv.Members {
		claimable[m.ID] = m.Claimable
	}
	if claimable[owner.ID] || !claimable[katie] {
		t.Fatalf("claimable %+v", inv.Members)
	}

	real, reqID := c.join(tr.ID, map[string]any{"member_id": katie})
	c.me = ownerSession
	c.do("POST", base+"/join-requests/"+reqID+"/approve", nil, 200, nil)
	c.me = real
	var me map[string]any
	c.do("GET", base+"/me", nil, 200, &me)
	if me["member"].(map[string]any)["id"] != katie {
		t.Fatalf("katie %+v", me)
	}
	// Now katie is taken.
	c.me = c.login("fake katie")
	if code := c.errCode("POST", base+"/join", map[string]any{"member_id": katie}, 409); code != "member_linked" {
		t.Fatal(code)
	}
}

// A request filed before the member got linked can't be approved after.
func TestApproveRechecksClaim(t *testing.T) {
	pool := testdb.New(t)
	c := &apiClient{t: t, h: NewServer(testDeps(pool)).Routes()}
	tr, _, ownerSession := c.newTrip("ทริป", "แบงก์")
	base := "/trips/" + tr.ID
	var katie string
	if err := pool.QueryRow(t.Context(), `INSERT INTO members (trip_id, name) VALUES ($1, 'katie') RETURNING id`, tr.ID).Scan(&katie); err != nil {
		t.Fatal(err)
	}
	_, first := c.join(tr.ID, map[string]any{"member_id": katie})
	_, second := c.join(tr.ID, map[string]any{"member_id": katie})
	c.me = ownerSession
	c.do("POST", base+"/join-requests/"+first+"/approve", nil, 200, nil)
	if code := c.errCode("POST", base+"/join-requests/"+second+"/approve", nil, 409); code != "member_linked" {
		t.Fatal(code)
	}
	// Reject everything left in one go.
	c.join(tr.ID, map[string]any{"name": "spam1"})
	c.join(tr.ID, map[string]any{"name": "spam2"})
	var out map[string]int
	c.do("POST", base+"/join-requests/reject-all", nil, 200, &out)
	var reqs []any
	c.do("GET", base+"/join-requests", nil, 200, &reqs)
	if out["rejected"] != 3 || len(reqs) != 0 {
		t.Fatalf("reject all: %v, left %d", out, len(reqs))
	}
}

// Members can only change what's theirs; every change is in the history.
func TestEditRightsAndHistory(t *testing.T) {
	c := newAPI(t)
	tr, owner, ownerSession := c.newTrip("ทริป", "แบงก์")
	base := "/trips/" + tr.ID
	fon, fonSession := c.addMember(tr.ID, ownerSession, "ฝน")
	mint, mintSession := c.addMember(tr.ID, ownerSession, "มิ้นท์")
	all := []string{owner.ID, fon.ID, mint.ID}

	c.me = fonSession
	var e expenseResp
	c.do("POST", base+"/expenses", map[string]any{"title": "ข้าว", "category": "food", "paid_by": fon.ID, "amount": 30000, "split_type": "equal", "member_ids": all}, 201, &e)
	edit := map[string]any{"title": "ข้าว", "category": "food", "paid_by": mint.ID, "amount": 30000, "split_type": "equal", "member_ids": []string{owner.ID}}

	// มิ้นท์ neither logged nor paid it: read-only for her.
	c.me = mintSession
	c.do("GET", base+"/expenses/"+e.ID, nil, 200, nil)
	if code := c.errCode("PUT", base+"/expenses/"+e.ID, edit, 403); code != "not_allowed" {
		t.Fatal(code)
	}
	if code := c.errCode("DELETE", base+"/expenses/"+e.ID, nil, 403); code != "not_allowed" {
		t.Fatal(code)
	}
	// Repayments: only payer, receiver or owner may record them.
	if code := c.errCode("POST", base+"/settlements/paid", map[string]any{"from": owner.ID, "to": fon.ID, "amount_base": 10000}, 403); code != "not_allowed" {
		t.Fatal(code)
	}
	c.do("POST", base+"/settlements/paid", map[string]any{"from": mint.ID, "to": fon.ID, "amount_base": 10000}, 201, nil)

	// The creator may edit; the owner may delete.
	c.me = fonSession
	c.do("PUT", base+"/expenses/"+e.ID, map[string]any{"title": "ข้าวมันไก่", "category": "food", "paid_by": fon.ID, "amount": 30000, "split_type": "equal", "member_ids": all}, 200, nil)
	c.me = ownerSession
	c.do("DELETE", base+"/expenses/"+e.ID, nil, 204, nil)

	var log []struct {
		Action     string `json:"action"`
		Title      string `json:"title"`
		MemberName string `json:"member_name"`
		AmountBase int64  `json:"amount_base"`
	}
	c.do("GET", base+"/activity", nil, 200, &log)
	got := []string{}
	for _, a := range log {
		got = append(got, a.Action+":"+a.MemberName+":"+a.Title)
	}
	want := []string{"expense_deleted:แบงก์:ข้าวมันไก่", "expense_updated:ฝน:ข้าวมันไก่", "repayment_recorded:มิ้นท์:มิ้นท์ → ฝน", "expense_created:ฝน:ข้าว"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("history\n got %v\nwant %v", got, want)
	}
}

func TestRateLimits(t *testing.T) {
	l := newLimiter(3, time.Minute)
	now := time.Unix(1_800_000_000, 0)
	l.now = func() time.Time { return now }
	for i := range 3 {
		if ok, _ := l.allow("a"); !ok {
			t.Fatalf("hit %d refused", i)
		}
	}
	if ok, wait := l.allow("a"); ok || wait != time.Minute {
		t.Fatalf("4th: ok=%v wait=%v", ok, wait)
	}
	if ok, _ := l.allow("b"); !ok {
		t.Fatal("other keys have their own budget")
	}
	now = now.Add(time.Minute)
	if ok, _ := l.allow("a"); !ok {
		t.Fatal("window should reset")
	}

	// Through the API: signed-out sign-in steps are limited per IP.
	pool := testdb.New(t)
	srv := NewServer(testDeps(pool))
	srv.lim.authIP = newLimiter(5, time.Minute)
	h := srv.Routes()
	codes := []int{}
	for range 7 {
		rec := post(h, "/auth/passkey/login/begin", "")
		codes = append(codes, rec.Code)
		if rec.Code == 429 && rec.Header().Get("Retry-After") == "" {
			t.Fatal("429 without Retry-After")
		}
	}
	if codes[4] != 200 || codes[5] != 429 || codes[6] != 429 {
		t.Fatalf("codes %v", codes)
	}
}

func TestClientIP(t *testing.T) {
	s := &Server{}
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.7:5555"
	r.Header.Set("CF-Connecting-IP", "203.0.113.9")
	if got := s.clientIP(r); got != "ip:10.0.0.7" {
		t.Fatalf("untrusted proxy header used: %s", got)
	}
	s.TrustProxy = true
	if got := s.clientIP(r); got != "ip:203.0.113.9" {
		t.Fatalf("got %s", got)
	}
	r.Header.Set("CF-Connecting-IP", "not-an-ip")
	if got := s.clientIP(r); got != "ip:10.0.0.7" {
		t.Fatalf("garbage header used: %s", got)
	}
}
