package slip

import (
	"context"
	"time"
)

// MockReader returns a fixed, clearly labelled sample for local UI work
// without an AI key (SLIP_PROVIDER=mock). Never use it in production.
type MockReader struct {
	Delay time.Duration
	Now   func() time.Time
}

func (m MockReader) Read(ctx context.Context, _ []byte, _ string) (*Result, error) {
	select {
	case <-time.After(m.Delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	now := time.Now
	if m.Now != nil {
		now = m.Now
	}
	return &Result{
		Kind:       "receipt",
		Amount:     "1250.00",
		Currency:   "THB",
		DateTime:   now().Format(time.RFC3339),
		Merchant:   "ตัวอย่าง (mock)",
		Category:   "food",
		Items:      []Item{{Name: "ราเมง", Amount: "850.00"}, {Name: "เกี๊ยวซ่า", Amount: "400.00"}},
		Confidence: 0.5,
	}, nil
}
