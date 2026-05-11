package payment

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// BankClient implements PaymentGatewayClient for bank transfer (manual processing)
type BankClient struct {
	enabled bool
}

// NewBankClient creates a new bank transfer client
func NewBankClient() *BankClient {
	return &BankClient{enabled: true}
}

// Provider returns the provider name
func (c *BankClient) Provider() string {
	return ProviderBank
}

// IsEnabled returns whether the client is enabled
func (c *BankClient) IsEnabled() bool {
	return c.enabled
}

// CreatePayment creates a bank transfer payment order
// Bank transfer doesn't have an API - returns bank account info for manual payment
func (c *BankClient) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*PaymentCredential, error) {
	// Bank transfer is manual - return placeholder credential
	// The actual bank account info should be configured and displayed to user
	return &PaymentCredential{
		PaymentID: req.OrderNumber, // Use order number as payment ID
		// In production, these should be populated with actual bank account info:
		// QRCodeURL:   could be a QR code for bank account info
		// PayURL:      could be a page showing bank transfer instructions
		ExpireAt:    time.Now().Add(24 * time.Hour), // Bank transfer has longer expiry
		RawResponse: fmt.Sprintf(`{"method":"bank_transfer","order_number":"%s","amount":"%s","currency":"%s"}`,
			req.OrderNumber, req.Amount.StringFixed(2), req.Currency),
	}, nil
}

// VerifyCallback verifies bank transfer callback
// Bank transfer doesn't have automatic callback - handled manually
func (c *BankClient) VerifyCallback(ctx context.Context, body []byte, headers http.Header) (*CallbackResult, error) {
	// Bank transfer callback is manual confirmation
	// This should be called when admin manually confirms payment
	return nil, fmt.Errorf("bank transfer callback not supported - use manual confirmation")
}

// QueryPayment queries bank transfer status
// Bank transfer status is tracked internally, not via API
func (c *BankClient) QueryPayment(ctx context.Context, paymentID string) (*PaymentStatus, error) {
	// Bank transfer doesn't have an API to query
	// Status is tracked via internal PaymentOrder entity
	return nil, fmt.Errorf("bank transfer query not supported - use internal order status")
}