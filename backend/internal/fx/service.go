package fx

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/raisin/travel-split/backend/internal/apperr"
	"github.com/raisin/travel-split/backend/internal/db"
)

const (
	latestTTL     = time.Hour          // how long a "latest" or very recent rate is reused
	finalAfter    = 7 * 24 * time.Hour // rates for days this old never change
	currenciesTTL = 24 * time.Hour
)

var ErrUnavailable = apperr.Invalid("fx_unavailable", "ดึงอัตราแลกเปลี่ยนไม่ได้ ลองใหม่อีกครั้ง")

// Quote is an exchange rate as returned by the API.
type Quote struct {
	From          string `json:"from"`
	To            string `json:"to"`
	Rate          string `json:"rate"`      // units of To per 1 From, decimal string
	RateDate      string `json:"rate_date"` // YYYY-MM-DD the rate is from
	RequestedDate string `json:"requested_date"`
	Source        string `json:"source"`
	Stale         bool   `json:"stale"` // provider unreachable; served from an older cache entry
}

type store interface {
	GetFxRate(ctx context.Context, arg db.GetFxRateParams) (db.FxRate, error)
	UpsertFxRate(ctx context.Context, arg db.UpsertFxRateParams) error
}

type Service struct {
	p     Provider
	store store
	now   func() time.Time

	mu         sync.Mutex
	currencies []string
	curFetched time.Time
}

func NewService(p Provider, q *db.Queries) *Service {
	return &Service{p: p, store: q, now: time.Now}
}

// Rate returns the from->to rate for date (nil = latest), using the cache
// when it is still valid and falling back to a stale entry if the provider fails.
func (s *Service) Rate(ctx context.Context, from, to string, date *time.Time) (Quote, error) {
	from, to = strings.ToUpper(strings.TrimSpace(from)), strings.ToUpper(strings.TrimSpace(to))
	if len(from) != 3 || len(to) != 3 {
		return Quote{}, apperr.Invalid("invalid_currency", "ต้องระบุสกุลเงินเป็นรหัส 3 ตัว เช่น THB")
	}
	today := truncateDay(s.now())
	day := today
	if date != nil {
		day = truncateDay(*date)
		if day.After(today) {
			return Quote{}, apperr.Invalid("invalid_date", "ยังไม่มีอัตราแลกเปลี่ยนของวันในอนาคต")
		}
	}
	q := Quote{From: from, To: to, RequestedDate: day.Format(time.DateOnly), Source: s.p.Name()}
	if from == to {
		q.Rate, q.RateDate = "1", q.RequestedDate
		return q, nil
	}

	key := db.GetFxRateParams{Base: from, Quote: to, RequestedDate: day}
	cached, err := s.store.GetFxRate(ctx, key)
	hasCache := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Quote{}, err
	}
	if hasCache && s.fresh(cached, day) {
		return fill(q, cached), nil
	}

	var reqDate *time.Time
	if day.Before(today) {
		reqDate = &day
	}
	rate, rateDate, err := s.p.Rate(ctx, from, to, reqDate)
	if errors.Is(err, ErrUnsupported) {
		return Quote{}, apperr.Invalid("unsupported_currency", "ยังไม่มีอัตราแลกเปลี่ยนสำหรับ "+from+" → "+to)
	}
	if err != nil {
		slog.WarnContext(ctx, "fx provider failed", "from", from, "to", to, "err", err)
		if hasCache {
			q = fill(q, cached)
			q.Stale = true
			return q, nil
		}
		return Quote{}, ErrUnavailable
	}

	var num pgtype.Numeric
	if err := num.Scan(rate); err != nil {
		return Quote{}, err
	}
	if err := s.store.UpsertFxRate(ctx, db.UpsertFxRateParams{Base: from, Quote: to, RequestedDate: day, RateDate: rateDate, Rate: num}); err != nil {
		slog.WarnContext(ctx, "fx cache write failed", "err", err)
	}
	q.Rate, q.RateDate = rate, rateDate.Format(time.DateOnly)
	return q, nil
}

// Currencies lists codes the provider supports, cached for a day.
func (s *Service) Currencies(ctx context.Context) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.currencies != nil && s.now().Sub(s.curFetched) < currenciesTTL {
		return s.currencies, nil
	}
	list, err := s.p.Currencies(ctx)
	if err != nil {
		if s.currencies != nil {
			return s.currencies, nil
		}
		slog.WarnContext(ctx, "fx currencies failed", "err", err)
		return nil, ErrUnavailable
	}
	s.currencies, s.curFetched = list, s.now()
	return list, nil
}

// fresh reports whether a cached rate can be reused without asking the provider.
// Old dates are final; recent ones are refreshed hourly because the day's
// rate may not have been published yet when it was first fetched.
func (s *Service) fresh(r db.FxRate, day time.Time) bool {
	if s.now().Sub(day) > finalAfter {
		return true
	}
	return s.now().Sub(r.FetchedAt) < latestTTL
}

func fill(q Quote, r db.FxRate) Quote {
	if v, err := r.Rate.Value(); err == nil {
		str, _ := v.(string)
		q.Rate = trimZeros(str)
	}
	q.RateDate = r.RateDate.Format(time.DateOnly)
	return q
}

func truncateDay(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// trimZeros turns the numeric(18,8) text "5.05050000" back into "5.0505".
func trimZeros(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}
