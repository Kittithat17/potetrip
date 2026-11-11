// Package slip reads payment slips and receipts with an AI model. Images are
// processed in memory only and never stored; only the parsed result is kept.
package slip

import (
	"context"
	"errors"
)

// Result is what a Reader extracts from one image.
type Result struct {
	Kind       string  `json:"kind"`     // "transfer_slip" | "receipt" | "unknown"
	Amount     string  `json:"amount"`   // decimal string, e.g. "1250.00"
	Currency   string  `json:"currency"` // ISO 4217
	DateTime   string  `json:"datetime"` // RFC3339 if readable
	FromName   string  `json:"from_name"`
	ToName     string  `json:"to_name"`
	Merchant   string  `json:"merchant"`
	Reference  string  `json:"reference"`
	Category   string  `json:"category"` // hint: food | transport | lodging | shopping | activity | other
	Items      []Item  `json:"items"`
	Confidence float64 `json:"confidence"` // 0..1
}

type Item struct {
	Name   string `json:"name"`
	Amount string `json:"amount"`
}

// Reader extracts slip data from an image.
type Reader interface {
	Read(ctx context.Context, image []byte, mimeType string) (*Result, error)
}

// ErrRateLimited marks errors that are worth retrying after a pause (HTTP 429).
var ErrRateLimited = errors.New("rate limited")
