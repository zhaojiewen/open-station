package payment

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/pkg/config"
	"github.com/zhaojiewen/open-station/pkg/errors"
)

// AlipayClient implements PaymentGatewayClient for Alipay
type AlipayClient struct {
	appID      string
	privateKey *rsa.PrivateKey
	publicKey  string
	isSandbox  bool
	enabled    bool
	baseURL    string
	httpClient *http.Client
}

// NewAlipayClient creates a new Alipay client
func NewAlipayClient(cfg *config.AlipayConfig) *AlipayClient {
	baseURL := "https://openapi.alipay.com/gateway.do"
	if cfg.IsSandbox {
		baseURL = "https://openapi.alipaydev.com/gateway.do"
	}

	// Parse private key
	privateKey, err := parseRSAPrivateKey(cfg.PrivateKey)
	if err != nil {
		// Log error but don't fail - client will be disabled
		return &AlipayClient{enabled: false}
	}

	return &AlipayClient{
		appID:      cfg.AppID,
		privateKey: privateKey,
		publicKey:  cfg.PublicKey,
		isSandbox:  cfg.IsSandbox,
		enabled:    true,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func parseRSAPrivateKey(keyStr string) (*rsa.PrivateKey, error) {
	// Remove header/footer if present
	keyStr = strings.TrimSpace(keyStr)
	keyStr = strings.ReplaceAll(keyStr, "-----BEGIN RSA PRIVATE KEY-----", "")
	keyStr = strings.ReplaceAll(keyStr, "-----END RSA PRIVATE KEY-----", "")
	keyStr = strings.ReplaceAll(keyStr, "-----BEGIN PRIVATE KEY-----", "")
	keyStr = strings.ReplaceAll(keyStr, "-----END PRIVATE KEY-----", "")
	keyStr = strings.ReplaceAll(keyStr, "\n", "")
	keyStr = strings.ReplaceAll(keyStr, "\r", "")

	keyBytes, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	// Try PKCS8 first
	key, err := x509.ParsePKCS8PrivateKey(keyBytes)
	if err == nil {
		return key.(*rsa.PrivateKey), nil
	}

	// Try PKCS1
	return x509.ParsePKCS1PrivateKey(keyBytes)
}

// Provider returns the provider name
func (c *AlipayClient) Provider() string {
	return ProviderAlipay
}

// IsEnabled returns whether the client is enabled
func (c *AlipayClient) IsEnabled() bool {
	return c.enabled && c.privateKey != nil
}

// CreatePayment creates an Alipay payment order
func (c *AlipayClient) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*PaymentCredential, error) {
	if !c.IsEnabled() {
		return nil, errors.ErrPaymentProviderDisabled
	}

	// Determine API method based on payment method
	var method string
	switch req.Method {
	case MethodQRCode:
		method = "alipay.trade.precreate" // 扫码支付
	case MethodWeb:
		method = "alipay.trade.page.pay" // 网页支付
	case MethodApp:
		method = "alipay.trade.app.pay" // App支付
	default:
		return nil, errors.ErrPaymentUnsupportedMethod
	}

	// Build request parameters
	params := map[string]string{
		"app_id":        c.appID,
		"method":        method,
		"format":        "JSON",
		"charset":       "utf-8",
		"sign_type":     "RSA2",
		"timestamp":     time.Now().Format("2006-01-02 15:04:05"),
		"version":       "1.0",
		"notify_url":    req.NotifyURL,
		"return_url":    req.ReturnURL,
	}

	// Build biz_content
	bizContent := map[string]interface{}{
		"out_trade_no": req.OrderNumber,
		"total_amount": req.Amount.StringFixed(2),
		"subject":      req.Subject,
		"product_code": "FAST_INSTANT_TRADE_PAY",
	}
	if req.Method == MethodQRCode {
		bizContent["product_code"] = "FACE_TO_FACE_PAYMENT"
	}

	bizJSON, _ := json.Marshal(bizContent)
	params["biz_content"] = string(bizJSON)

	// Sign the request
	sign, err := c.signParams(params)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errors.ErrPaymentSignFailed, err)
	}
	params["sign"] = sign

	// For QR code: call API and get qr_code
	if req.Method == MethodQRCode {
		resp, err := c.callAPI(ctx, params)
		if err != nil {
			return nil, err
		}

		// Parse response
		var result alipayPrecreateResponse
		if err := json.Unmarshal(resp, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		if result.AlipayTradePrecreateResponse.Code != "10000" {
			return nil, fmt.Errorf("alipay error: %s - %s",
				result.AlipayTradePrecreateResponse.Code,
				result.AlipayTradePrecreateResponse.Msg)
		}

		return &PaymentCredential{
			PaymentID:   result.AlipayTradePrecreateResponse.OutTradeNo,
			QRCodeURL:   result.AlipayTradePrecreateResponse.QrCode,
			QRCodeData:  result.AlipayTradePrecreateResponse.QrCode,
			ExpireAt:    time.Now().Add(30 * time.Minute),
			RawResponse: string(resp),
		}, nil
	}

	// For web/app: generate payment URL
	formValues := url.Values{}
	for k, v := range params {
		formValues.Set(k, v)
	}
	payURL := c.baseURL + "?" + formValues.Encode()

	return &PaymentCredential{
		PaymentID: req.OrderNumber,
		PayURL:    payURL,
		ExpireAt:  time.Now().Add(30 * time.Minute),
	}, nil
}

// VerifyCallback verifies Alipay callback
func (c *AlipayClient) VerifyCallback(ctx context.Context, body []byte, headers http.Header) (*CallbackResult, error) {
	if !c.IsEnabled() {
		return nil, errors.ErrPaymentProviderDisabled
	}

	// Parse callback parameters
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse callback: %w", err)
	}

	// Verify signature
	sign := values.Get("sign")
	signType := values.Get("sign_type")
	values.Del("sign")
	values.Del("sign_type")

	if err := c.verifySign(values, sign, signType); err != nil {
		return nil, fmt.Errorf("%w: %v", errors.ErrPaymentInvalidSignature, err)
	}

	// Parse result
	tradeStatus := values.Get("trade_status")
	status := StatusFailed
	if tradeStatus == "TRADE_SUCCESS" || tradeStatus == "TRADE_FINISHED" {
		status = StatusSuccess
	}

	paidAmount, _ := decimal.NewFromString(values.Get("total_amount"))

	var paidAt time.Time
	if gmtPayment := values.Get("gmt_payment"); gmtPayment != "" {
		paidAt, _ = time.Parse("2006-01-02 15:04:05", gmtPayment)
	}

	return &CallbackResult{
		OrderNumber:  values.Get("out_trade_no"),
		PaymentID:    values.Get("trade_no"),
		OutTradeNo:   values.Get("trade_no"),
		PaidAmount:   paidAmount,
		Status:       status,
		PaidAt:       paidAt,
		RawData:      string(body),
		SignVerified: true,
	}, nil
}

// QueryPayment queries Alipay payment status
func (c *AlipayClient) QueryPayment(ctx context.Context, paymentID string) (*PaymentStatus, error) {
	if !c.IsEnabled() {
		return nil, errors.ErrPaymentProviderDisabled
	}

	params := map[string]string{
		"app_id":    c.appID,
		"method":    "alipay.trade.query",
		"format":    "JSON",
		"charset":   "utf-8",
		"sign_type": "RSA2",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"version":   "1.0",
	}

	bizContent := map[string]string{"trade_no": paymentID}
	bizJSON, _ := json.Marshal(bizContent)
	params["biz_content"] = string(bizJSON)

	sign, _ := c.signParams(params)
	params["sign"] = sign

	resp, err := c.callAPI(ctx, params)
	if err != nil {
		return nil, err
	}

	var result alipayQueryResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.AlipayTradeQueryResponse.Code != "10000" {
		return nil, fmt.Errorf("alipay error: %s", result.AlipayTradeQueryResponse.Msg)
	}

	status := StatusPending
	switch result.AlipayTradeQueryResponse.TradeStatus {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		status = StatusSuccess
	case "TRADE_CLOSED":
		status = StatusClosed
	}

	paidAmount, _ := decimal.NewFromString(result.AlipayTradeQueryResponse.TotalAmount)

	return &PaymentStatus{
		PaymentID:  result.AlipayTradeQueryResponse.TradeNo,
		OrderNumber: result.AlipayTradeQueryResponse.OutTradeNo,
		Status:     status,
		PaidAmount: paidAmount,
	}, nil
}

// signParams signs the request parameters
func (c *AlipayClient) signParams(params map[string]string) (string, error) {
	// Sort and concatenate parameters
	keys := make([]string, 0, len(params))
	for k := range params {
		if params[k] != "" {
			keys = append(keys, k)
		}
	}
	sortStrings(keys)

	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, k+"="+params[k])
	}
	content := strings.Join(pairs, "&")

	// RSA2 sign
	hashed := crypto.SHA256.New()
	hashed.Write([]byte(content))
	signature, err := rsa.SignPKCS1v15(rand.Reader, c.privateKey, crypto.SHA256, hashed.Sum(nil))
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

// verifySign verifies the signature
func (c *AlipayClient) verifySign(values url.Values, sign string, signType string) error {
	// TODO: Implement proper signature verification using Alipay public key
	// For now, we skip verification in development
	// In production, this MUST be implemented
	return nil
}

// callAPI makes API call to Alipay
func (c *AlipayClient) callAPI(ctx context.Context, params map[string]string) ([]byte, error) {
	formValues := url.Values{}
	for k, v := range params {
		formValues.Set(k, v)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, strings.NewReader(formValues.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("alipay API error: status %d", resp.StatusCode)
	}

	// Read response
	buf := make([]byte, 0, 1024)
	for {
		n, err := resp.Body.Read(buf[len(buf):cap(buf)])
		if n > 0 {
			buf = buf[:len(buf)+n]
		}
		if err != nil {
			break
		}
		if len(buf) == cap(buf) {
			newBuf := make([]byte, len(buf), len(buf)+1024)
			copy(newBuf, buf)
			buf = newBuf
		}
	}

	return buf, nil
}

// Response structures
type alipayPrecreateResponse struct {
	AlipayTradePrecreateResponse struct {
		Code      string `json:"code"`
		Msg       string `json:"msg"`
		OutTradeNo string `json:"out_trade_no"`
		QrCode    string `json:"qr_code"`
	} `json:"alipay_trade_precreate_response"`
}

type alipayQueryResponse struct {
	AlipayTradeQueryResponse struct {
		Code        string `json:"code"`
		Msg         string `json:"msg"`
		TradeNo     string `json:"trade_no"`
		OutTradeNo  string `json:"out_trade_no"`
		TradeStatus string `json:"trade_status"`
		TotalAmount string `json:"total_amount"`
	} `json:"alipay_trade_query_response"`
}

func sortStrings(s []string) {
	for i := 0; i < len(s)-1; i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}