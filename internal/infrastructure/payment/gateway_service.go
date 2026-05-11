package payment

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/zhaojiewen/open-station/pkg/config"
	"github.com/zhaojiewen/open-station/pkg/errors"
)

// PaymentGatewayService manages multiple payment provider clients
type PaymentGatewayService struct {
	clients       map[string]PaymentGatewayClient
	callbackURL   string
	defaultCurrency string
	mu            sync.RWMutex
}

// NewPaymentGatewayService creates a new payment gateway service
func NewPaymentGatewayService(cfg *config.PaymentConfig) *PaymentGatewayService {
	clients := make(map[string]PaymentGatewayClient)

	if cfg == nil {
		return &PaymentGatewayService{
			clients:       clients,
			callbackURL:   "",
			defaultCurrency: "USD",
		}
	}

	// Initialize Alipay client
	if cfg.Alipay.Enabled && cfg.Alipay.AppID != "" {
		clients[ProviderAlipay] = NewAlipayClient(&cfg.Alipay)
	}

	// Initialize Wechat client
	if cfg.Wechat.Enabled && cfg.Wechat.AppID != "" {
		clients[ProviderWechat] = NewWechatClient(&cfg.Wechat)
	}

	// Initialize Stripe client
	if cfg.Stripe.Enabled && cfg.Stripe.APIKey != "" {
		clients[ProviderStripe] = NewStripeClient(&cfg.Stripe)
	}

	// Initialize PayPal client
	if cfg.PayPal.Enabled && cfg.PayPal.ClientID != "" {
		clients[ProviderPayPal] = NewPayPalClient(&cfg.PayPal)
	}

	// Bank transfer is always available (manual processing)
	clients[ProviderBank] = NewBankClient()

	defaultCurrency := cfg.DefaultCurrency
	if defaultCurrency == "" {
		defaultCurrency = "USD"
	}

	return &PaymentGatewayService{
		clients:         clients,
		callbackURL:     cfg.CallbackBaseURL,
		defaultCurrency: defaultCurrency,
	}
}

// CreatePayment creates a payment order through the specified provider
func (s *PaymentGatewayService) CreatePayment(ctx context.Context, provider string, req *CreatePaymentRequest) (*PaymentCredential, error) {
	s.mu.RLock()
	client, ok := s.clients[provider]
	s.mu.RUnlock()

	if !ok {
		return nil, errors.ErrPaymentProviderDisabled
	}

	if !client.IsEnabled() {
		return nil, errors.ErrPaymentProviderDisabled
	}

	// Generate notify URL dynamically if not provided
	if req.NotifyURL == "" && s.callbackURL != "" {
		req.NotifyURL = s.callbackURL + "/" + provider
	}

	// Set default currency if not provided
	if req.Currency == "" {
		req.Currency = s.defaultCurrency
	}

	credential, err := client.CreatePayment(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errors.ErrPaymentCreateFailed, err)
	}

	return credential, nil
}

// VerifyCallback verifies the callback from a payment provider
func (s *PaymentGatewayService) VerifyCallback(ctx context.Context, provider string, body []byte, headers http.Header) (*CallbackResult, error) {
	s.mu.RLock()
	client, ok := s.clients[provider]
	s.mu.RUnlock()

	if !ok {
		return nil, errors.ErrPaymentProviderDisabled
	}

	result, err := client.VerifyCallback(ctx, body, headers)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errors.ErrPaymentVerifyFailed, err)
	}

	return result, nil
}

// QueryPayment queries the payment status from a provider
func (s *PaymentGatewayService) QueryPayment(ctx context.Context, provider string, paymentID string) (*PaymentStatus, error) {
	s.mu.RLock()
	client, ok := s.clients[provider]
	s.mu.RUnlock()

	if !ok {
		return nil, errors.ErrPaymentProviderDisabled
	}

	status, err := client.QueryPayment(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errors.ErrPaymentQueryFailed, err)
	}

	return status, nil
}

// GetClient returns a specific payment client
func (s *PaymentGatewayService) GetClient(provider string) (PaymentGatewayClient, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	client, ok := s.clients[provider]
	return client, ok
}

// GetCallbackURL returns the callback URL base
func (s *PaymentGatewayService) GetCallbackURL() string {
	return s.callbackURL
}

// GetEnabledProviders returns list of enabled providers
func (s *PaymentGatewayService) GetEnabledProviders() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	providers := make([]string, 0)
	for name, client := range s.clients {
		if client.IsEnabled() {
			providers = append(providers, name)
		}
	}
	return providers
}

// IsProviderEnabled checks if a provider is enabled
func (s *PaymentGatewayService) IsProviderEnabled(provider string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, ok := s.clients[provider]
	if !ok {
		return false
	}
	return client.IsEnabled()
}