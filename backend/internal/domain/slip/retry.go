package slip

import (
	"context"
	"errors"
	"time"
)

// RetryReader retries transient failures (429 rate limit, 503 overload)
// with exponential backoff.
type RetryReader struct {
	Next       Reader
	MaxRetries int           // retries after the first attempt
	BaseDelay  time.Duration // doubled after each retry
	sleep      func(context.Context, time.Duration) error
}

func NewRetryReader(next Reader, maxRetries int, baseDelay time.Duration) *RetryReader {
	return &RetryReader{Next: next, MaxRetries: maxRetries, BaseDelay: baseDelay, sleep: sleepCtx}
}

func (r *RetryReader) Read(ctx context.Context, image []byte, mimeType string) (*Result, error) {
	delay := r.BaseDelay
	for attempt := 0; ; attempt++ {
		res, err := r.Next.Read(ctx, image, mimeType)
		if err == nil || !retryable(err) || attempt >= r.MaxRetries {
			return res, err
		}
		if err := r.sleep(ctx, delay); err != nil {
			return nil, err
		}
		delay *= 2
	}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func retryable(err error) bool {
	return errors.Is(err, ErrRateLimited) || errors.Is(err, ErrUnavailable)
}
