package slip

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

const systemPrompt = `You read Thai and international payment slips and receipts. Extract the fields in the schema.
Amounts as decimal strings without thousand separators or currency symbols. Use the grand total actually paid.
Currency as an ISO 4217 code; Thai bank slips are THB.
Dates as RFC3339 with the timezone offset shown or implied by the document (Thailand is +07:00).
Thai slips often use the Buddhist calendar: subtract 543 from the year (2569 -> 2026).
from_name is the payer, to_name is the receiver. merchant is the shop name on a receipt.
category is your best guess of what was bought.
items lists line items only for itemised receipts, otherwise an empty list.
If a field is not visible, return an empty string. Never guess an amount. If unsure, lower confidence.`

// GeminiReader implements Reader with the Google Gemini API.
type GeminiReader struct {
	client *genai.Client
	model  string
}

func NewGeminiReader(ctx context.Context, apiKey, model string) (*GeminiReader, error) {
	if apiKey == "" {
		return nil, errors.New("GEMINI_API_KEY is empty")
	}
	if model == "" {
		return nil, errors.New("GEMINI_MODEL is empty")
	}
	c, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		return nil, err
	}
	return &GeminiReader{client: c, model: model}, nil
}

func (g *GeminiReader) Read(ctx context.Context, image []byte, mimeType string) (*Result, error) {
	contents := []*genai.Content{genai.NewContentFromParts([]*genai.Part{
		genai.NewPartFromBytes(image, mimeType),
		genai.NewPartFromText("Extract this slip or receipt."),
	}, genai.RoleUser)}

	resp, err := g.client.Models.GenerateContent(ctx, g.model, contents, &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
		ResponseMIMEType:  "application/json",
		ResponseSchema:    resultSchema(),
		Temperature:       genai.Ptr[float32](0),
	})
	if err != nil {
		var apiErr genai.APIError
		if errors.As(err, &apiErr) && apiErr.Code == 429 {
			return nil, fmt.Errorf("%w: %s", ErrRateLimited, apiErr.Message)
		}
		return nil, err
	}
	return decodeResult(resp.Text())
}

func decodeResult(text string) (*Result, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, errors.New("model returned no content")
	}
	var r Result
	if err := json.Unmarshal([]byte(text), &r); err != nil {
		return nil, fmt.Errorf("decode model output: %w", err)
	}
	if r.Items == nil {
		r.Items = []Item{}
	}
	r.Currency = strings.ToUpper(strings.TrimSpace(r.Currency))
	r.Confidence = min(max(r.Confidence, 0), 1)
	return &r, nil
}

func resultSchema() *genai.Schema {
	str := func(desc string) *genai.Schema { return &genai.Schema{Type: genai.TypeString, Description: desc} }
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"kind":      {Type: genai.TypeString, Enum: []string{"transfer_slip", "receipt", "unknown"}},
			"amount":    str("Total paid as a decimal string, e.g. 1250.00"),
			"currency":  str("ISO 4217 code"),
			"datetime":  str("RFC3339 date-time"),
			"from_name": str("Payer name"),
			"to_name":   str("Receiver name"),
			"merchant":  str("Shop or merchant name"),
			"reference": str("Transaction reference number"),
			"category":  {Type: genai.TypeString, Enum: []string{"food", "transport", "lodging", "shopping", "activity", "other"}},
			"items": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"name":   str("Line item name"),
						"amount": str("Line item total as a decimal string"),
					},
					Required: []string{"name", "amount"},
				},
			},
			"confidence": {Type: genai.TypeNumber, Description: "0 to 1"},
		},
		Required:         []string{"kind", "amount", "currency", "datetime", "from_name", "to_name", "merchant", "reference", "category", "items", "confidence"},
		PropertyOrdering: []string{"kind", "amount", "currency", "datetime", "from_name", "to_name", "merchant", "reference", "category", "items", "confidence"},
	}
}
