package slip

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/raisin/travel-split/backend/internal/db"
	"github.com/raisin/travel-split/backend/internal/money"
)

const (
	MaxFileSize  = 8 << 20 // 8 MB per image (spec 6.3)
	MaxFiles     = 10      // per request
	readTimeout  = 90 * time.Second
	errorMessage = "อ่านสลิปไม่สำเร็จ ลองใหม่หรือกรอกเอง"
)

// Upload is one image received from the client. It lives only in memory.
type Upload struct {
	Filename string
	Data     []byte
}

// Suggestion is the Result normalised for the expense form. Fields the
// model could not read reliably are nil so the user fills them in.
// Title is the merchant or, for a transfer, the receiver's name; the client
// adds wording around it in the user's language.
type Suggestion struct {
	Title     string     `json:"title"`
	TitleKind string     `json:"title_kind"` // merchant | transfer_to | none
	Amount    *int64     `json:"amount"`     // minor units of Currency
	Currency  string     `json:"currency"`
	SpentAt   *time.Time `json:"spent_at"`
	Category  string     `json:"category"`
	LowConfid bool       `json:"low_confidence"`
	// Items are receipt lines in Currency minor units (receipts only).
	Items []SuggestedItem `json:"items"`
}

type SuggestedItem struct {
	Name   string `json:"name"`
	Amount int64  `json:"amount"`
}

// Outcome is the per-image response.
type Outcome struct {
	Filename   string      `json:"filename"`
	SlipID     *uuid.UUID  `json:"slip_id"`
	Status     string      `json:"status"`               // parsed | failed
	ErrorCode  string      `json:"error_code,omitempty"` // unsupported_type | too_large | read_failed | rate_limited | ai_busy | cancelled
	Error      string      `json:"error,omitempty"`
	Result     *Result     `json:"result,omitempty"`
	Suggestion *Suggestion `json:"suggestion,omitempty"`
}

type store interface {
	CreateSlip(ctx context.Context, arg db.CreateSlipParams) (db.Slip, error)
	FinishSlip(ctx context.Context, arg db.FinishSlipParams) error
}

type Service struct {
	reader      Reader
	store       store
	concurrency int
}

func NewService(reader Reader, q *db.Queries, concurrency int) *Service {
	return &Service{reader: reader, store: q, concurrency: max(concurrency, 1)}
}

// Enabled reports whether a Reader is configured.
func (s *Service) Enabled() bool { return s != nil && s.reader != nil }

// Process reads every upload concurrently (bounded by the semaphore) and
// returns outcomes in input order. Nothing is saved as an expense here.
func (s *Service) Process(ctx context.Context, tripID, uploadedBy uuid.UUID, uploads []Upload) []Outcome {
	out := make([]Outcome, len(uploads))
	sem := make(chan struct{}, s.concurrency)
	var wg sync.WaitGroup
	for i, u := range uploads {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				out[i] = Outcome{Filename: u.Filename, Status: "failed", ErrorCode: "cancelled", Error: "ยกเลิกแล้ว"}
				return
			}
			out[i] = s.processOne(ctx, tripID, uploadedBy, u)
		}()
	}
	wg.Wait()
	return out
}

func (s *Service) processOne(ctx context.Context, tripID, uploadedBy uuid.UUID, u Upload) Outcome {
	o := Outcome{Filename: u.Filename, Status: "failed"}

	mime := DetectImageType(u.Data)
	if mime == "" {
		o.ErrorCode, o.Error = "unsupported_type", "รองรับเฉพาะรูป JPEG, PNG, WebP หรือ HEIC"
		return o
	}
	if len(u.Data) > MaxFileSize {
		o.ErrorCode, o.Error = "too_large", "ไฟล์ใหญ่เกิน 8MB"
		return o
	}

	row, err := s.store.CreateSlip(ctx, db.CreateSlipParams{TripID: tripID, UploadedBy: uploadedBy})
	if err != nil {
		slog.ErrorContext(ctx, "create slip", "err", err)
		o.ErrorCode, o.Error = "read_failed", errorMessage
		return o
	}
	o.SlipID = &row.ID

	rctx, cancel := context.WithTimeout(ctx, readTimeout)
	res, err := s.reader.Read(rctx, u.Data, mime)
	cancel()

	finish := db.FinishSlipParams{ID: row.ID, Status: "failed"}
	if err != nil {
		slog.WarnContext(ctx, "slip read failed", "slip", row.ID, "err", err)
		msg := err.Error()
		finish.Error = &msg
		o.ErrorCode, o.Error = "read_failed", errorMessage
		switch {
		case errors.Is(err, ErrRateLimited):
			o.ErrorCode, o.Error = "rate_limited", "AI ใช้งานเกินโควตาชั่วคราว รอสักครู่แล้วลองใหม่"
		case errors.Is(err, ErrUnavailable):
			o.ErrorCode, o.Error = "ai_busy", "AI ไม่ว่างชั่วคราว รอสักครู่แล้วลองใหม่"
		}
	} else {
		raw, _ := json.Marshal(res)
		finish.Status = "parsed"
		finish.RawResult = raw
		o.Status = "parsed"
		o.Result = res
		o.Suggestion = suggest(res)
	}
	// The read already happened; persist its outcome even if the client left.
	if err := s.store.FinishSlip(context.WithoutCancel(ctx), finish); err != nil {
		slog.ErrorContext(ctx, "finish slip", "slip", row.ID, "err", err)
	}
	return o
}

var categories = map[string]bool{"food": true, "transport": true, "lodging": true, "shopping": true, "activity": true, "other": true}

func suggest(r *Result) *Suggestion {
	s := &Suggestion{Currency: r.Currency, Category: "other", LowConfid: r.Confidence < 0.6, Items: []SuggestedItem{}}
	if categories[r.Category] {
		s.Category = r.Category
	}
	switch {
	case strings.TrimSpace(r.Merchant) != "":
		s.Title, s.TitleKind = strings.TrimSpace(r.Merchant), "merchant"
	case strings.TrimSpace(r.ToName) != "":
		s.Title, s.TitleKind = strings.TrimSpace(r.ToName), "transfer_to"
	default:
		s.TitleKind = "none"
	}
	if r.Currency != "" && r.Amount != "" {
		if v, err := money.ParseDecimal(r.Amount, r.Currency); err == nil && v > 0 {
			s.Amount = &v
		}
	}
	if t, err := time.Parse(time.RFC3339, strings.TrimSpace(r.DateTime)); err == nil {
		s.SpentAt = &t
	}
	// Lines that can't be read cleanly are dropped; the user can add them.
	for _, it := range r.Items {
		name := strings.TrimSpace(it.Name)
		if name == "" || r.Currency == "" {
			continue
		}
		if v, err := money.ParseDecimal(it.Amount, r.Currency); err == nil && v > 0 {
			s.Items = append(s.Items, SuggestedItem{Name: name, Amount: v})
		}
	}
	return s
}
