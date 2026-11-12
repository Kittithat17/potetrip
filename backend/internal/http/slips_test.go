package http

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raisin/travel-split/backend/internal/db"
	"github.com/raisin/travel-split/backend/internal/expense"
	"github.com/raisin/travel-split/backend/internal/settlement"
	"github.com/raisin/travel-split/backend/internal/slip"
	"github.com/raisin/travel-split/backend/internal/testdb"
	"github.com/raisin/travel-split/backend/internal/trip"
)

type stubReader struct{}

func (stubReader) Read(context.Context, []byte, string) (*slip.Result, error) {
	return &slip.Result{Kind: "receipt", Amount: "1250.00", Currency: "THB", DateTime: "2026-11-10T19:30:00+07:00",
		Merchant: "Ichiran", Category: "food", Items: []slip.Item{}, Confidence: 0.9}, nil
}

var pngHeader = []byte("\x89PNG\r\n\x1a\n0000000000000000")

func multipartBody(t *testing.T, files map[string][]byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for name, data := range files {
		fw, err := mw.CreateFormFile("files", name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = fw.Write(data)
	}
	_ = mw.WriteField("note", "ignored")
	_ = mw.Close()
	return &buf, mw.FormDataContentType()
}

func TestSlipUploadAndConfirm(t *testing.T) {
	pool := testdb.New(t)
	h := NewServer(Deps{
		DB: pool, Trips: trip.NewService(pool), Expenses: expense.NewService(pool),
		Settlements: settlement.NewService(pool), Slips: slip.NewService(stubReader{}, db.New(pool), 2),
	}).Routes()
	c := &apiClient{t: t, h: h}

	tr, me, myToken := c.newTrip("สลิป", "แบงก์")
	other, _, outsiderToken := c.newTrip("อีกทริป", "คนนอก")

	upload := func(tripID, token string, files map[string][]byte, want int) map[string]any {
		body, ct := multipartBody(t, files)
		req := httptest.NewRequest("POST", "/api/v1/trips/"+tripID+"/slips", body)
		req.Header.Set("Content-Type", ct)
		req.Header.Set(TokenHeader, token)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("upload status %d want %d: %s", rec.Code, want, rec.Body)
		}
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return out
	}

	out := upload(tr.ID, myToken, map[string][]byte{"a.png": pngHeader, "b.txt": []byte("not an image")}, 200)
	results := out["results"].([]any)
	if len(results) != 2 {
		t.Fatalf("results %v", results)
	}
	var slipID string
	for _, r := range results {
		m := r.(map[string]any)
		switch m["filename"] {
		case "a.png":
			sug := m["suggestion"].(map[string]any)
			if m["status"] != "parsed" || sug["amount"].(float64) != 125000 || sug["title"] != "Ichiran" {
				t.Fatalf("png outcome %v", m)
			}
			slipID = m["slip_id"].(string)
		case "b.txt":
			if m["status"] != "failed" || m["slip_id"] != nil {
				t.Fatalf("txt outcome %v", m)
			}
		}
	}

	upload(tr.ID, myToken, map[string][]byte{}, 400)

	// Confirm the slip as an expense; it cannot be used twice or from another trip.
	c.me = myToken
	exp := map[string]any{"title": "Ichiran", "category": "food", "paid_by": me.ID, "amount": 125000,
		"split_type": "equal", "member_ids": []string{me.ID}, "slip_id": slipID}
	var created struct {
		SlipID string `json:"slip_id"`
	}
	c.do("POST", "/trips/"+tr.ID+"/expenses", exp, 201, &created)
	if created.SlipID != slipID {
		t.Fatalf("slip not linked: %+v", created)
	}
	if code := c.errCode("POST", "/trips/"+tr.ID+"/expenses", exp, 409); code != "slip_unavailable" {
		t.Fatal(code)
	}

	out = upload(other.ID, outsiderToken, map[string][]byte{"c.png": pngHeader}, 200)
	otherSlip := out["results"].([]any)[0].(map[string]any)["slip_id"].(string)
	exp["slip_id"] = otherSlip
	if code := c.errCode("POST", "/trips/"+tr.ID+"/expenses", exp, 409); code != "slip_unavailable" {
		t.Fatal(code)
	}

	var status string
	if err := pool.QueryRow(context.Background(), "SELECT status FROM slips WHERE id = $1", slipID).Scan(&status); err != nil || status != "confirmed" {
		t.Fatalf("slip status %q %v", status, err)
	}
}

func TestSlipUploadDisabled(t *testing.T) {
	pool := testdb.New(t)
	h := NewServer(Deps{DB: pool, Trips: trip.NewService(pool), Expenses: expense.NewService(pool), Settlements: settlement.NewService(pool)}).Routes()
	c := &apiClient{t: t, h: h}
	tr, _, token := c.newTrip("x", "a")

	body, ct := multipartBody(t, map[string][]byte{"a.png": pngHeader})
	req := httptest.NewRequest("POST", "/api/v1/trips/"+tr.ID+"/slips", body)
	req.Header.Set("Content-Type", ct)
	req.Header.Set(TokenHeader, token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d", rec.Code)
	}
}
