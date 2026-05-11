package payment

import (
	"context"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

// PaymentGatewayClient defines the interface for payment provider clients
type PaymentGatewayClient interface {
	// CreatePayment creates a payment order and returns payment credentials
	CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*PaymentCredential, error)

	// VerifyCallback verifies the callback signature and parses the data
	VerifyCallback(ctx context.Context, body []byte, headers http.Header) (*CallbackResult, error)

	// QueryPayment queries the payment status from provider
	QueryPayment(ctx context.Context, paymentID string) (*PaymentStatus, error)

	// Provider returns the provider name
	Provider() string

	// IsEnabled returns whether the provider is enabled
	IsEnabled() bool
}

// CreatePaymentRequest represents the request to create a payment
type CreatePaymentRequest struct {
	OrderNumber string          // Internal order number (PAY-{timestamp}-{random})
	Amount      decimal.Decimal // Payment amount
	Currency    string          // Currency code (USD, CNY, etc.)
	Method      string          // Payment method: qr_code, web, app, bank_transfer
	Subject     string          // Payment subject/description
	ReturnURL   string          // Frontend redirect URL after payment
	NotifyURL   string          // Callback notification URL
	ClientIP    string          // Client IP address (for risk control)
}

// PaymentCredential represents the payment credentials returned by provider
type PaymentCredential struct {
	PaymentID   string    // External payment ID from provider
	QRCodeURL   string    // QR code URL (for qr_code method)
	QRCodeData  string    // QR code raw data (for scanning)
	PayURL      string    // Payment page URL (for web method)
	AppPayload  string    // App payment payload (for app method)
	ExpireAt    time.Time // Credential expiration time
	RawResponse string    // Raw response from provider (for debugging)
}

// CallbackResult represents the parsed callback data
type CallbackResult struct {
	OrderNumber   string          // Internal order number
	PaymentID     string          // External payment ID
	OutTradeNo    string          // Provider's trade number
	PaidAmount    decimal.Decimal // Actual paid amount
	Status        string          // Payment status: success, failed
	PaidAt        time.Time       // Payment completion time
	RawData       string          // Raw callback data (JSON)
	SignVerified  bool            // Whether signature was verified
}

// PaymentStatus represents the payment status from provider
type PaymentStatus struct {
	PaymentID     string          // External payment ID
	OrderNumber   string          // Internal order number (if available)
	Status        string          // Payment status: pending, success, failed, closed
	PaidAmount    decimal.Decimal // Paid amount (if successful)
	PaidAt        time.Time       // Payment time (if successful)
}

// PaymentMethod constants
const (
	MethodQRCode      = "qr_code"
	MethodWeb         = "web"
	MethodApp         = "app"
	MethodBankTransfer = "bank_transfer"
)

// PaymentStatus constants
const (
	StatusPending  = "pending"
	StatusSuccess  = "success"
	StatusFailed   = "failed"
	StatusClosed   = "closed"
	StatusCancelled = "cancelled"
)

// Provider constants
const (
	ProviderAlipay = "alipay"
	ProviderWechat = "wechat"
	ProviderStripe = "stripe"
	ProviderPayPal = "paypal"
	ProviderBank   = "bank"
)