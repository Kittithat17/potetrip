// Package fx provides currency exchange rates with a database cache.
package fx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"
)

// ErrUnsupported means the provider has no rate for this currency pair/date.
var ErrUnsupported = errors.New("currency or date not supported by rate provider")

// Provider fetches exchange rates from an external source.
type Provider interface {
	// Rate returns how many `to` one unit of `from` buys, as a decimal string,
	// and the date the rate is from. A nil date means the latest rate.
	Rate(ctx context.Context, from, to string, date *time.Time) (rate string, rateDate time.Time, err error)
	Currencies(ctx context.Context) ([]string, error)
	Name() string
}

// Frankfurter reads ECB reference rates from frankfurter.dev (free, no key).
// Rates are published once per working day.
type Frankfurter struct {
	BaseURL string // e.g. https://api.frankfurter.dev/v1
	Client  *http.Client
}

func NewFrankfurter(baseURL string) *Frankfurter {
	return &Frankfurter{BaseURL: baseURL, Client: &http.Client{Timeout: 10 * time.Second}}
}

func (f *Frankfurter) Name() string { return "ECB via Frankfurter" }

func (f *Frankfurter) Rate(ctx context.Context, from, to string, date *time.Time) (string, time.Time, error) {
	path := "latest"
	if date != nil {
		path = date.Format(time.DateOnly)
	}
	u := fmt.Sprintf("%s/%s?from=%s&to=%s", f.BaseURL, path, url.QueryEscape(from), url.QueryEscape(to))
	body, err := f.get(ctx, u)
	if err != nil {
		return "", time.Time{}, err
	}
	var out struct {
		Date  string                 `json:"date"`
		Rates map[string]json.Number `json:"rates"`
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber() // keep the exact decimal text, never a float
	if err := dec.Decode(&out); err != nil {
		return "", time.Time{}, fmt.Errorf("frankfurter: decode: %w", err)
	}
	rate, ok := out.Rates[to]
	if !ok {
		return "", time.Time{}, ErrUnsupported
	}
	d, err := time.Parse(time.DateOnly, out.Date)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("frankfurter: bad date %q", out.Date)
	}
	return rate.String(), d, nil
}

func (f *Frankfurter) Currencies(ctx context.Context) ([]string, error) {
	body, err := f.get(ctx, f.BaseURL+"/currencies")
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("frankfurter: decode currencies: %w", err)
	}
	out := make([]string, 0, len(m))
	for code := range m {
		out = append(out, code)
	}
	sort.Strings(out)
	return out, nil
}

func (f *Frankfurter) get(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("frankfurter: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(http.MaxBytesReader(nil, resp.Body, 1<<20)); err != nil {
		return nil, fmt.Errorf("frankfurter: read: %w", err)
	}
	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnprocessableEntity:
		return nil, ErrUnsupported
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("frankfurter: HTTP %d", resp.StatusCode)
	}
	return buf.Bytes(), nil
}
