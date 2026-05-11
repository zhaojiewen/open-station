package payment

import (
	"bytes"
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

// PayPalClient implements PaymentGatewayClient for PayPal
type PayPalClient struct {
	clientID     string
	clientSecret string
	isSandbox    bool
	enabled      bool
	baseURL      string
	httpClient   *http.Client
	accessToken  string
	tokenExpiry  time.Time
}

// NewPayPalClient creates a new PayPal client
func NewPayPalClient(cfg *config.PayPalConfig) *PayPalClient {
	baseURL := "https://api-m.paypal.com"
	if cfg.IsSandbox {
		baseURL = "https://api-m.sandbox.paypal.com"
	}

	return &PayPalClient{
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		isSandbox:    cfg.IsSandbox,
		enabled:      cfg.ClientID != "" && cfg.ClientSecret != "",
		baseURL:      baseURL,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// Provider returns the provider name
func (c *PayPalClient) Provider() string {
	return ProviderPayPal
}

// IsEnabled returns whether the client is enabled
func (c *PayPalClient) IsEnabled() bool {
	return c.enabled
}

// getAccessToken gets OAuth2 access token
func (c *PayPalClient) getAccessToken(ctx context.Context) (string, error) {
	// Check if we have a valid cached token
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	url := c.baseURL + "/v1/oauth2/token"

	body := "grant_type=client_credentials"

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.clientID, c.clientSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("paypal auth error: status %d", resp.StatusCode)
	}

	var result paypalTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	c.accessToken = result.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn - 60) * time.Second)

	return c.accessToken, nil
}

// CreatePayment creates a PayPal order
func (c *PayPalClient) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*PaymentCredential, error) {
	if !c.IsEnabled() {
		return nil, errors.ErrPaymentProviderDisabled
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	url := c.baseURL + "/v2/checkout/orders"

	// Build order request
	orderReq := paypalOrderRequest{
		Intent: "CAPTURE",
		PurchaseUnits: []paypalPurchaseUnit{
			{
				ReferenceID: req.OrderNumber,
				Amount: paypalAmount{
					Value:    req.Amount.StringFixed(2),
					Currency: req.Currency,
				},
				Description: req.Subject,
			},
		},
		ApplicationContext: paypalApplicationContext{
			ReturnURL: req.ReturnURL,
			CancelURL: req.ReturnURL + "?cancel=true",
			NotifyURL: req.NotifyURL,
		},
	}

	orderJSON, _ := json.Marshal(orderReq)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(orderJSON))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("paypal API error: status %d", resp.StatusCode)
	}

	var result paypalOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Find approval URL
	var payURL string
	for _, link := range result.Links {
		if link.Rel == "approve" {
			payURL = link.Href
			break
		}
	}

	return &PaymentCredential{
		PaymentID:   result.ID,
		PayURL:      payURL,
		ExpireAt:    time.Now().Add(30 * time.Minute),
		RawResponse: "",
	}, nil
}

// VerifyCallback verifies PayPal webhook callback
func (c *PayPalClient) VerifyCallback(ctx context.Context, body []byte, headers http.Header) (*CallbackResult, error) {
	if !c.IsEnabled() {
		return nil, errors.ErrPaymentProviderDisabled
	}

	// Verify webhook signature (simplified)
	// In production, use PayPal's webhook signature verification
	authAlgo := headers.Get("PAYPAL-TRANSMISSION-SIG-ALGO")
	if authAlgo == "" {
		return nil, errors.ErrPaymentInvalidSignature
	}

	var event paypalWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse callback: %w", err)
	}

	// Only process CHECKOUT.ORDER.APPROVED events
	if event.EventType != "CHECKOUT.ORDER.APPROVED" {
		return &CallbackResult{
			Status:  StatusPending,
			RawData: string(body),
		}, nil
	}

	// Capture the order to complete payment
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	captureURL := c.baseURL + "/v2/checkout/orders/" + event.Resource.ID + "/capture"

	captureReq, err := http.NewRequestWithContext(ctx, "POST", captureURL, bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, err
	}

	captureReq.Header.Set("Content-Type", "application/json")
	captureReq.Header.Set("Authorization", "Bearer "+token)

	captureResp, err := c.httpClient.Do(captureReq)
	if err != nil {
		return nil, err
	}
	defer captureResp.Body.Close()

	if captureResp.StatusCode >= 400 {
		return nil, fmt.Errorf("paypal capture error: status %d", captureResp.StatusCode)
	}

	var captured paypalOrderResponse
	if err := json.NewDecoder(captureResp.Body).Decode(&captured); err != nil {
		return nil, fmt.Errorf("failed to parse capture response: %w", err)
	}

	// Get payment details
	var paidAmount decimal.Decimal
	var orderNumber string
	for _, unit := range captured.PurchaseUnits {
		paidAmount, _ = decimal.NewFromString(unit.Amount.Value)
		orderNumber = unit.ReferenceID
	}

	return &CallbackResult{
		OrderNumber:  orderNumber,
		PaymentID:    captured.ID,
		OutTradeNo:   captured.ID,
		PaidAmount:   paidAmount,
		Status:       StatusSuccess,
		PaidAt:       time.Now(),
		RawData:      string(body),
		SignVerified: true,
	}, nil
}

// QueryPayment queries PayPal order status
func (c *PayPalClient) QueryPayment(ctx context.Context, paymentID string) (*PaymentStatus, error) {
	if !c.IsEnabled() {
		return nil, errors.ErrPaymentProviderDisabled
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	url := c.baseURL + "/v2/checkout/orders/" + paymentID

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("paypal API error: status %d", resp.StatusCode)
	}

	var result paypalOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	status := StatusPending
	switch result.Status {
	case "COMPLETED":
		status = StatusSuccess
	case "CANCELLED":
		status = StatusCancelled
	case "DENIED":
		status = StatusFailed
	}

	var paidAmount decimal.Decimal
	var orderNumber string
	for _, unit := range result.PurchaseUnits {
		paidAmount, _ = decimal.NewFromString(unit.Amount.Value)
		orderNumber = unit.ReferenceID
	}

	return &PaymentStatus{
		PaymentID:   result.ID,
		OrderNumber: orderNumber,
		Status:      status,
		PaidAmount:  paidAmount,
	}, nil
}

type bytesReader struct {
	data []byte
	pos  int
}

func paypalNewReader(b []byte) *bytesReader {
	return &bytesReader{data: b}
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// PayPal API structures
type paypalTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type paypalOrderRequest struct {
	Intent           string                    `json:"intent"`
	PurchaseUnits    []paypalPurchaseUnit      `json:"purchase_units"`
	ApplicationContext paypalApplicationContext `json:"application_context"`
}

type paypalPurchaseUnit struct {
	ReferenceID string        `json:"reference_id"`
	Amount      paypalAmount  `json:"amount"`
	Description string        `json:"description"`
}

type paypalAmount struct {
	Value    string `json:"value"`
	Currency string `json:"currency_code"`
}

type paypalApplicationContext struct {
	ReturnURL string `json:"return_url"`
	CancelURL string `json:"cancel_url"`
	NotifyURL string `json:"notify_url,omitempty"`
}

type paypalOrderResponse struct {
	ID            string             `json:"id"`
	Status        string             `json:"status"`
	PurchaseUnits []paypalPurchaseUnit `json:"purchase_units"`
	Links         []paypalLink       `json:"links"`
}

type paypalLink struct {
	Href   string `json:"href"`
	Rel    string `json:"rel"`
	Method string `json:"method"`
}

type paypalWebhookEvent struct {
	ID        string `json:"id"`
	EventType string `json:"event_type"`
	Resource  struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	} `json:"resource"`
}