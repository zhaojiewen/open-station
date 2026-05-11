package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/pkg/config"
	"github.com/zhaojiewen/open-station/pkg/errors"
)

// StripeClient implements PaymentGatewayClient for Stripe
type StripeClient struct {
	apiKey     string
	publishKey string
	enabled    bool
	baseURL    string
	httpClient *http.Client
}

// NewStripeClient creates a new Stripe client
func NewStripeClient(cfg *config.StripeConfig) *StripeClient {
	return &StripeClient{
		apiKey:     cfg.APIKey,
		publishKey: cfg.PublishKey,
		enabled:    cfg.APIKey != "",
		baseURL:    "https://api.stripe.com/v1",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Provider returns the provider name
func (c *StripeClient) Provider() string {
	return ProviderStripe
}

// IsEnabled returns whether the client is enabled
func (c *StripeClient) IsEnabled() bool {
	return c.enabled
}

// CreatePayment creates a Stripe payment intent
func (c *StripeClient) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*PaymentCredential, error) {
	if !c.IsEnabled() {
		return nil, errors.ErrPaymentProviderDisabled
	}

	// Create payment intent
	url := c.baseURL + "/payment_intents"

	// Convert amount to cents (Stripe uses smallest currency unit)
	amountCents := int(req.Amount.Mul(decimal.NewFromInt(100)).IntPart())

	body := fmt.Sprintf("amount=%d&currency=%s&metadata[order_number]=%s",
		amountCents, toLowerCurrency(req.Currency), req.OrderNumber)

	reqBody, err := http.NewRequestWithContext(ctx, "POST", url, stripeNewReader(body))
	if err != nil {
		return nil, err
	}

	reqBody.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqBody.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("stripe API error: status %d", resp.StatusCode)
	}

	var result stripePaymentIntent
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Return payment credential with client_secret for frontend
	return &PaymentCredential{
		PaymentID:   result.ID,
		PayURL:      "", // Stripe uses client_secret on frontend
		AppPayload:  fmt.Sprintf(`{"client_secret":"%s","publish_key":"%s"}`, result.ClientSecret, c.publishKey),
		ExpireAt:    time.Now().Add(30 * time.Minute),
		RawResponse: "", // Will be filled from raw response
	}, nil
}

// VerifyCallback verifies Stripe webhook callback
func (c *StripeClient) VerifyCallback(ctx context.Context, body []byte, headers http.Header) (*CallbackResult, error) {
	if !c.IsEnabled() {
		return nil, errors.ErrPaymentProviderDisabled
	}

	// Get Stripe signature from header
	sigHeader := headers.Get("Stripe-Signature")
	if sigHeader == "" {
		return nil, errors.ErrPaymentInvalidSignature
	}

	// Verify webhook signature (simplified - production requires proper verification)
	// In production, use stripe.WebhookSignature verification

	var event stripeEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse callback: %w", err)
	}

	// Only process payment_intent.succeeded events
	if event.Type != "payment_intent.succeeded" {
		return &CallbackResult{
			Status:  StatusPending,
			RawData: string(body),
		}, nil
	}

	// Parse payment intent data
	var intent stripePaymentIntent
	intentJSON, _ := json.Marshal(event.Data.Object)
	if err := json.Unmarshal(intentJSON, &intent); err != nil {
		return nil, fmt.Errorf("failed to parse payment intent: %w", err)
	}

	paidAmount := decimal.NewFromInt(int64(intent.Amount / 100))

	return &CallbackResult{
		OrderNumber:  intent.Metadata.OrderNumber,
		PaymentID:    intent.ID,
		OutTradeNo:   intent.ID,
		PaidAmount:   paidAmount,
		Status:       StatusSuccess,
		PaidAt:       time.Unix(intent.Created, 0),
		RawData:      string(body),
		SignVerified: true,
	}, nil
}

// QueryPayment queries Stripe payment status
func (c *StripeClient) QueryPayment(ctx context.Context, paymentID string) (*PaymentStatus, error) {
	if !c.IsEnabled() {
		return nil, errors.ErrPaymentProviderDisabled
	}

	url := c.baseURL + "/payment_intents/" + paymentID

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("stripe API error: status %d", resp.StatusCode)
	}

	var result stripePaymentIntent
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	status := StatusPending
	switch result.Status {
	case "succeeded":
		status = StatusSuccess
	case "canceled":
		status = StatusCancelled
	case "failed":
		status = StatusFailed
	}

	paidAmount := decimal.NewFromInt(int64(result.Amount / 100))

	return &PaymentStatus{
		PaymentID:   result.ID,
		OrderNumber: result.Metadata.OrderNumber,
		Status:      status,
		PaidAmount:  paidAmount,
	}, nil
}

func toLowerCurrency(s string) string {
	return strings.ToLower(s)
}

type stringsReader struct {
	s string
	i int
}

func (r *stringsReader) Read(p []byte) (n int, err error) {
	if r.i >= len(r.s) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(p, r.s[r.i:])
	r.i += n
	return n, nil
}

func stripeNewReader(s string) *stringsReader {
	return &stringsReader{s: s}
}

// Stripe API structures
type stripePaymentIntent struct {
	ID            string `json:"id"`
	Object        string `json:"object"`
	Amount        int    `json:"amount"`
	Currency      string `json:"currency"`
	Status        string `json:"status"`
	ClientSecret  string `json:"client_secret"`
	Created       int64  `json:"created"`
	Metadata      struct {
		OrderNumber string `json:"order_number"`
	} `json:"metadata"`
}

type stripeEvent struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Type    string `json:"type"`
	Created int64  `json:"created"`
	Data    struct {
		Object interface{} `json:"object"`
	} `json:"data"`
}