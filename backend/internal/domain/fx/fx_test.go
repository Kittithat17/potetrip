package fx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/raisin/travel-split/backend/internal/platform/apperr"
	"github.com/raisin/travel-split/backend/internal/platform/db"
)

func TestFrankfurterClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest":
			if r.URL.Query().Get("from") == "TWD" {
				http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
				return
			}
			_, _ = w.Write([]byte(`{"amount":1.0,"base":"CNY","date":"2026-11-11","rates":{"THB":5.0505}}`))
		case "/2026-11-08":
			_, _ = w.Write([]byte(`{"amount":1.0,"base":"JPY","date":"2026-11-06","rates":{"THB":0.21191}}`))
		case "/currencies":
			_, _ = w.Write([]byte(`{"USD":"US Dollar","THB":"Thai Baht","CNY":"Chinese Renminbi Yuan"}`))
		default:
			http.Error(w, "boom", http.StatusInternalServerError)
		}
	}))
	defer srv.Close()
	f := NewFrankfurter(srv.URL)
	ctx := context.Background()

	rate, d, err := f.Rate(ctx, "CNY", "THB", nil)
	if err != nil || rate != "5.0505" || d.Format(time.DateOnly) != "2026-11-11" {
		t.Fatalf("latest: %q %v %v", rate, d, err)
	}
	day := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	rate, d, err = f.Rate(ctx, "JPY", "THB", &day)
	if err != nil || rate != "0.21191" || d.Format(time.DateOnly) != "2026-11-06" {
		t.Fatalf("dated: %q %v %v", rate, d, err)
	}
	if _, _, err := f.Rate(ctx, "TWD", "THB", nil); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("unsupported: %v", err)
	}
	cs, err := f.Currencies(ctx)
	if err != nil || len(cs) != 3 || cs[0] != "CNY" {
		t.Fatalf("currencies %v %v", cs, err)
	}
	broken := NewFrankfurter(srv.URL + "/nope")
	if _, _, err := broken.Rate(ctx, "CNY", "THB", nil); err == nil || errors.Is(err, ErrUnsupported) {
		t.Fatalf("server error should not look unsupported: %v", err)
	}
}

// fakeProvider counts calls and can be switched to fail.
type fakeProvider struct {
	calls int
	fail  error
	rate  string
}

func (p *fakeProvider) Name() string { return "fake" }
func (p *fakeProvider) Rate(_ context.Context, _, _ string, date *time.Time) (string, time.Time, error) {
	p.calls++
	if p.fail != nil {
		return "", time.Time{}, p.fail
	}
	d := time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)
	if date != nil {
		d = *date
	}
	return p.rate, d, nil
}
func (p *fakeProvider) Currencies(context.Context) ([]string, error) {
	p.calls++
	if p.fail != nil {
		return nil, p.fail
	}
	return []string{"CNY", "THB"}, nil
}

type memStore struct {
	rows map[db.GetFxRateParams]db.FxRate
}

func (m *memStore) GetFxRate(_ context.Context, k db.GetFxRateParams) (db.FxRate, error) {
	r, ok := m.rows[k]
	if !ok {
		return r, pgx.ErrNoRows
	}
	return r, nil
}

func (m *memStore) UpsertFxRate(_ context.Context, a db.UpsertFxRateParams) error {
	m.rows[db.GetFxRateParams{Base: a.Base, Quote: a.Quote, RequestedDate: a.RequestedDate}] = db.FxRate{
		Base: a.Base, Quote: a.Quote, RequestedDate: a.RequestedDate, RateDate: a.RateDate, Rate: a.Rate, FetchedAt: now,
	}
	return nil
}

var now = time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

func newSvc(p Provider) (*Service, *memStore) {
	st := &memStore{rows: map[db.GetFxRateParams]db.FxRate{}}
	return &Service{p: p, store: st, now: func() time.Time { return now }}, st
}

func TestServiceCachesLatestForAnHour(t *testing.T) {
	p := &fakeProvider{rate: "5.0505"}
	s, _ := newSvc(p)
	ctx := context.Background()

	q, err := s.Rate(ctx, "cny", "thb", nil)
	if err != nil || q.Rate != "5.0505" || q.From != "CNY" || q.RateDate != "2026-11-11" || q.RequestedDate != "2026-11-12" {
		t.Fatalf("%+v %v", q, err)
	}
	if _, err := s.Rate(ctx, "CNY", "THB", nil); err != nil || p.calls != 1 {
		t.Fatalf("second call should hit cache, calls=%d err=%v", p.calls, err)
	}
	now = now.Add(2 * time.Hour)
	defer func() { now = now.Add(-2 * time.Hour) }()
	if _, err := s.Rate(ctx, "CNY", "THB", nil); err != nil || p.calls != 2 {
		t.Fatalf("expired cache should refetch, calls=%d", p.calls)
	}
}

func TestServiceOldDatesAreFinal(t *testing.T) {
	p := &fakeProvider{rate: "0.21191"}
	s, st := newSvc(p)
	day := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	if _, err := s.Rate(context.Background(), "JPY", "THB", &day); err != nil {
		t.Fatal(err)
	}
	// Even a cache entry fetched long ago is reused for a month-old date.
	for k, r := range st.rows {
		r.FetchedAt = now.Add(-30 * 24 * time.Hour)
		st.rows[k] = r
	}
	q, err := s.Rate(context.Background(), "JPY", "THB", &day)
	if err != nil || p.calls != 1 || q.Rate != "0.21191" {
		t.Fatalf("calls=%d q=%+v err=%v", p.calls, q, err)
	}
}

func TestServiceFallsBackToStaleCache(t *testing.T) {
	p := &fakeProvider{rate: "5.0505"}
	s, _ := newSvc(p)
	ctx := context.Background()
	if _, err := s.Rate(ctx, "CNY", "THB", nil); err != nil {
		t.Fatal(err)
	}
	now = now.Add(3 * time.Hour)
	defer func() { now = now.Add(-3 * time.Hour) }()
	p.fail = errors.New("network down")

	q, err := s.Rate(ctx, "CNY", "THB", nil)
	if err != nil || !q.Stale || q.Rate != "5.0505" {
		t.Fatalf("want stale cached quote, got %+v %v", q, err)
	}
	if _, err := s.Rate(ctx, "USD", "THB", nil); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("no cache + provider down: %v", err)
	}
}

func TestServiceValidation(t *testing.T) {
	p := &fakeProvider{rate: "1", fail: ErrUnsupported}
	s, _ := newSvc(p)
	ctx := context.Background()

	q, err := s.Rate(ctx, "THB", "THB", nil)
	if err != nil || q.Rate != "1" || p.calls != 0 {
		t.Fatalf("same currency: %+v %v", q, err)
	}
	future := now.Add(48 * time.Hour)
	for _, tc := range []struct {
		from, to string
		date     *time.Time
		code     string
	}{
		{"TWD", "THB", nil, "unsupported_currency"},
		{"BAHT", "THB", nil, "invalid_currency"},
		{"USD", "THB", &future, "invalid_date"},
	} {
		_, err := s.Rate(ctx, tc.from, tc.to, tc.date)
		e, ok := apperr.As(err)
		if !ok || e.Code != tc.code {
			t.Errorf("%s->%s: want %s, got %v", tc.from, tc.to, tc.code, err)
		}
	}
}

func TestTrimZeros(t *testing.T) {
	for in, want := range map[string]string{"5.05050000": "5.0505", "1.00000000": "1", "12": "12", "0.00210000": "0.0021"} {
		if got := trimZeros(in); got != want {
			t.Errorf("trimZeros(%q) = %q want %q", in, got, want)
		}
	}
}
