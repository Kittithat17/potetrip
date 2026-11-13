package slip

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/raisin/travel-split/backend/internal/platform/db"
)

var (
	pngBytes  = []byte("\x89PNG\r\n\x1a\n0000000000000000")
	jpegBytes = []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00000000000")
	heicBytes = []byte("\x00\x00\x00\x18ftypheic\x00\x00\x00\x00mif1heic")
)

// mockReader returns canned results and records peak concurrency.
type mockReader struct {
	mu       sync.Mutex
	inFlight int
	peak     int
	calls    atomic.Int32
	delay    time.Duration
	fn       func(call int32, image []byte) (*Result, error)
}

func (m *mockReader) Read(ctx context.Context, image []byte, _ string) (*Result, error) {
	n := m.calls.Add(1)
	m.mu.Lock()
	m.inFlight++
	m.peak = max(m.peak, m.inFlight)
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.inFlight--
		m.mu.Unlock()
	}()
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return m.fn(n, image)
}

type fakeStore struct {
	mu       sync.Mutex
	created  int
	finished map[uuid.UUID]db.FinishSlipParams
}

func (f *fakeStore) CreateSlip(_ context.Context, arg db.CreateSlipParams) (db.Slip, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created++
	return db.Slip{ID: uuid.New(), TripID: arg.TripID, UploadedBy: arg.UploadedBy, Status: "pending"}, nil
}

func (f *fakeStore) FinishSlip(_ context.Context, arg db.FinishSlipParams) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.finished == nil {
		f.finished = map[uuid.UUID]db.FinishSlipParams{}
	}
	f.finished[arg.ID] = arg
	return nil
}

func TestProcessBoundedConcurrencyAndOrder(t *testing.T) {
	r := &mockReader{delay: 20 * time.Millisecond, fn: func(_ int32, img []byte) (*Result, error) {
		return &Result{Kind: "receipt", Amount: "100.00", Currency: "THB", Merchant: string(img[len(img)-1:]), Confidence: 0.9}, nil
	}}
	st := &fakeStore{}
	s := &Service{reader: r, store: st, concurrency: 3}

	var uploads []Upload
	for i := range 8 {
		uploads = append(uploads, Upload{Filename: fmt.Sprintf("%d.png", i), Data: append(append([]byte{}, pngBytes...), byte('a'+i))})
	}
	out := s.Process(context.Background(), uuid.New(), uuid.New(), uploads)

	if r.peak > 3 || r.peak < 2 {
		t.Fatalf("peak concurrency %d, want 2..3", r.peak)
	}
	for i, o := range out {
		if o.Filename != uploads[i].Filename || o.Status != "parsed" {
			t.Fatalf("outcome %d = %+v", i, o)
		}
		if o.Result.Merchant != string(rune('a'+i)) {
			t.Fatalf("outcome %d has another image's result: %q", i, o.Result.Merchant)
		}
		if f := st.finished[*o.SlipID]; f.Status != "parsed" || len(f.RawResult) == 0 {
			t.Fatalf("slip %d not stored as parsed: %+v", i, f)
		}
	}
}

func TestProcessRejectsBadFilesAndRecordsFailures(t *testing.T) {
	r := &mockReader{fn: func(_ int32, img []byte) (*Result, error) {
		if img[0] == 0xff {
			return nil, errors.New("model exploded")
		}
		return &Result{Kind: "transfer_slip", Amount: "50", Currency: "THB"}, nil
	}}
	st := &fakeStore{}
	s := &Service{reader: r, store: st, concurrency: 2}

	big := append(append([]byte{}, pngBytes...), make([]byte, MaxFileSize)...)
	out := s.Process(context.Background(), uuid.New(), uuid.New(), []Upload{
		{Filename: "note.txt", Data: []byte("hello, not an image")},
		{Filename: "huge.png", Data: big},
		{Filename: "broken.jpg", Data: jpegBytes},
		{Filename: "ok.heic", Data: heicBytes},
	})

	if out[0].Status != "failed" || out[0].SlipID != nil || out[0].ErrorCode != "unsupported_type" {
		t.Fatalf("text file: %+v", out[0])
	}
	if out[1].Status != "failed" || out[1].SlipID != nil || out[1].ErrorCode != "too_large" {
		t.Fatalf("huge file: %+v", out[1])
	}
	if out[2].Status != "failed" || out[2].SlipID == nil {
		t.Fatalf("model error: %+v", out[2])
	}
	if f := st.finished[*out[2].SlipID]; f.Status != "failed" || f.Error == nil || *f.Error != "model exploded" {
		t.Fatalf("failure not recorded: %+v", f)
	}
	if out[3].Status != "parsed" {
		t.Fatalf("heic: %+v", out[3])
	}
	if st.created != 2 {
		t.Fatalf("only readable images should create slip rows, got %d", st.created)
	}
	if r.calls.Load() != 2 {
		t.Fatalf("reader calls = %d", r.calls.Load())
	}
}

func TestRetryReaderBacksOffOnRateLimit(t *testing.T) {
	r := &mockReader{fn: func(call int32, _ []byte) (*Result, error) {
		switch call {
		case 1:
			return nil, fmt.Errorf("%w: slow down", ErrRateLimited)
		case 2:
			return nil, fmt.Errorf("%w: high demand", ErrUnavailable)
		}
		return &Result{Kind: "receipt"}, nil
	}}
	var sleeps []time.Duration
	rr := &RetryReader{Next: r, MaxRetries: 3, BaseDelay: time.Second, sleep: func(_ context.Context, d time.Duration) error {
		sleeps = append(sleeps, d)
		return nil
	}}
	res, err := rr.Read(context.Background(), pngBytes, "image/png")
	if err != nil || res.Kind != "receipt" {
		t.Fatalf("got %v, %v", res, err)
	}
	if fmt.Sprint(sleeps) != "[1s 2s]" {
		t.Fatalf("sleeps %v", sleeps)
	}
}

func TestRetryReaderGivesUpAfterMaxRetries(t *testing.T) {
	r := &mockReader{fn: func(int32, []byte) (*Result, error) { return nil, ErrRateLimited }}
	var sleeps []time.Duration
	rr := &RetryReader{Next: r, MaxRetries: 3, BaseDelay: 500 * time.Millisecond, sleep: func(_ context.Context, d time.Duration) error {
		sleeps = append(sleeps, d)
		return nil
	}}
	if _, err := rr.Read(context.Background(), pngBytes, "image/png"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v", err)
	}
	if r.calls.Load() != 4 || fmt.Sprint(sleeps) != "[500ms 1s 2s]" {
		t.Fatalf("calls %d sleeps %v", r.calls.Load(), sleeps)
	}
}

func TestRetryReaderDoesNotRetryOtherErrors(t *testing.T) {
	r := &mockReader{fn: func(int32, []byte) (*Result, error) { return nil, errors.New("bad request") }}
	rr := NewRetryReader(r, 3, time.Hour)
	if _, err := rr.Read(context.Background(), pngBytes, "image/png"); err == nil {
		t.Fatal("want error")
	}
	if r.calls.Load() != 1 {
		t.Fatalf("calls %d", r.calls.Load())
	}
}

func TestSuggest(t *testing.T) {
	s := suggest(&Result{Kind: "receipt", Amount: "1,250.50", Currency: "THB", DateTime: "2026-11-10T19:30:00+07:00",
		Merchant: "Ichiran", Category: "food", Confidence: 0.95})
	if s.Title != "Ichiran" || s.TitleKind != "merchant" || s.Amount == nil || *s.Amount != 125050 || s.SpentAt == nil || s.Category != "food" || s.LowConfid {
		t.Fatalf("receipt suggestion %+v", s)
	}

	s = suggest(&Result{Kind: "transfer_slip", Amount: "12.345", Currency: "THB", ToName: "นาย ก", Category: "weird", Confidence: 0.3})
	if s.Title != "นาย ก" || s.TitleKind != "transfer_to" || s.Amount != nil || s.SpentAt != nil || s.Category != "other" || !s.LowConfid {
		t.Fatalf("slip suggestion %+v", s)
	}
	if len(s.Items) != 0 {
		t.Fatalf("items should be an empty list, got %+v", s.Items)
	}

	s = suggest(&Result{Kind: "receipt", Amount: "1250", Currency: "JPY", Items: []Item{
		{Name: " ラーメン ", Amount: "850"}, {Name: "", Amount: "100"}, {Name: "ぎょうざ", Amount: "abc"}, {Name: "ビール", Amount: "400"},
	}})
	if len(s.Items) != 2 || s.Items[0] != (SuggestedItem{Name: "ラーメン", Amount: 850}) || s.Items[1].Amount != 400 {
		t.Fatalf("items %+v", s.Items)
	}
}

func TestDecodeResult(t *testing.T) {
	r, err := decodeResult(`{"kind":"receipt","amount":"10","currency":" jpy ","confidence":1.7}`)
	if err != nil || r.Currency != "JPY" || r.Confidence != 1 || r.Items == nil {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := decodeResult("  "); err == nil {
		t.Fatal("want error for empty")
	}
	if _, err := decodeResult("not json"); err == nil {
		t.Fatal("want error for junk")
	}
}

func TestDetectImageType(t *testing.T) {
	cases := map[string][]byte{
		"image/png":  pngBytes,
		"image/jpeg": jpegBytes,
		"image/heic": heicBytes,
		"image/webp": []byte("RIFF\x00\x00\x00\x00WEBPVP8 "),
		"":           []byte("%PDF-1.7 not an image"),
	}
	for want, b := range cases {
		if got := DetectImageType(b); got != want {
			t.Errorf("DetectImageType = %q want %q", got, want)
		}
	}
}
