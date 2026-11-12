package http

import (
	"context"
	"testing"
	"time"

	"github.com/raisin/travel-split/backend/internal/db"
	"github.com/raisin/travel-split/backend/internal/fx"
	"github.com/raisin/travel-split/backend/internal/testdb"
)

type stubFX struct{ calls int }

func (p *stubFX) Name() string { return "stub" }
func (p *stubFX) Rate(_ context.Context, from, _ string, date *time.Time) (string, time.Time, error) {
	p.calls++
	if from == "TWD" {
		return "", time.Time{}, fx.ErrUnsupported
	}
	d := time.Now().UTC().AddDate(0, 0, -1)
	if date != nil {
		d = *date
	}
	return "5.0505", d, nil
}
func (p *stubFX) Currencies(context.Context) ([]string, error) { return []string{"CNY", "THB"}, nil }

func TestFXEndpoints(t *testing.T) {
	pool := testdb.New(t)
	p := &stubFX{}
	c := &apiClient{t: t, h: NewServer(Deps{DB: pool, FX: fx.NewService(p, db.New(pool))}).Routes()}

	var q fx.Quote
	c.do("GET", "/fx?from=cny&to=THB", nil, 200, &q)
	if q.Rate != "5.0505" || q.From != "CNY" || q.Stale {
		t.Fatalf("%+v", q)
	}
	// Second read comes from the Postgres cache; numeric(18,8) padding is trimmed.
	c.do("GET", "/fx?from=CNY&to=THB", nil, 200, &q)
	if q.Rate != "5.0505" || p.calls != 1 {
		t.Fatalf("cache: %+v calls=%d", q, p.calls)
	}
	c.do("GET", "/fx?from=JPY&to=THB&date=2026-11-06", nil, 200, &q)
	if q.RateDate != "2026-11-06" {
		t.Fatalf("dated %+v", q)
	}
	if code := c.errCode("GET", "/fx?from=TWD&to=THB", nil, 400); code != "unsupported_currency" {
		t.Fatal(code)
	}
	if code := c.errCode("GET", "/fx?from=USD&to=THB&date=12-11-2026", nil, 400); code != "invalid_date" {
		t.Fatal(code)
	}
	var cur struct{ Currencies []string }
	c.do("GET", "/fx/currencies", nil, 200, &cur)
	if len(cur.Currencies) != 2 {
		t.Fatalf("%+v", cur)
	}
}
