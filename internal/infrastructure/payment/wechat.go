package payment

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/pkg/config"
	"github.com/zhaojiewen/open-station/pkg/errors"
)

// WechatClient implements PaymentGatewayClient for WeChat Pay
type WechatClient struct {
	appID      string
	mchID      string
	privateKey *rsa.PrivateKey
	serialNo   string
	apiV3Key   string
	isSandbox  bool
	enabled    bool
	baseURL    string
	httpClient *http.Client
}

// NewWechatClient creates a new WeChat Pay client
func NewWechatClient(cfg *config.WechatConfig) *WechatClient {
	baseURL := "https://api.mch.weixin.qq.com"
	if cfg.IsSandbox {
		baseURL = "https://api.mch.weixin.qq.com/sandboxnew"
	}

	// Parse private key
	privateKey, err := parseRSAPrivateKey(cfg.PrivateKey)
	if err != nil {
		return &WechatClient{enabled: false}
	}

	return &WechatClient{
		appID:      cfg.AppID,
		mchID:      cfg.MchID,
		privateKey: privateKey,
		serialNo:   cfg.SerialNo,
		apiV3Key:   cfg.APIV3Key,
		isSandbox:  cfg.IsSandbox,
		enabled:    true,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Provider returns the provider name
func (c *WechatClient) Provider() string {
	return ProviderWechat
}

// IsEnabled returns whether the client is enabled
func (c *WechatClient) IsEnabled() bool {
	return c.enabled && c.privateKey != nil
}

// CreatePayment creates a WeChat Pay order
func (c *WechatClient) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*PaymentCredential, error) {
	if !c.IsEnabled() {
		return nil, errors.ErrPaymentProviderDisabled
	}

	switch req.Method {
	case MethodQRCode:
		return c.createNativePayment(ctx, req)
	case MethodWeb:
		return c.createH5Payment(ctx, req)
	case MethodApp:
		return c.createAppPayment(ctx, req)
	default:
		return nil, errors.ErrPaymentUnsupportedMethod
	}
}

// createNativePayment creates Native (QR code) payment
func (c *WechatClient) createNativePayment(ctx context.Context, req *CreatePaymentRequest) (*PaymentCredential, error) {
	url := c.baseURL + "/v3/pay/transactions/native"

	body := wechatNativeRequest{
		AppID:       c.appID,
		MchID:       c.mchID,
		Description: req.Subject,
		OutTradeNo:  req.OrderNumber,
		NotifyURL:   req.NotifyURL,
		Amount: wechatAmount{
			Total:    int(req.Amount.Mul(decimal.NewFromInt(100)).IntPart()), // Convert to cents
			Currency: req.Currency,
		},
	}

	respBody, err := c.callAPIV3(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}

	var result wechatNativeResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &PaymentCredential{
		PaymentID:   req.OrderNumber,
		QRCodeURL:   result.CodeURL,
		QRCodeData:  result.CodeURL,
		ExpireAt:    time.Now().Add(30 * time.Minute),
		RawResponse: string(respBody),
	}, nil
}

// createH5Payment creates H5 (web) payment
func (c *WechatClient) createH5Payment(ctx context.Context, req *CreatePaymentRequest) (*PaymentCredential, error) {
	url := c.baseURL + "/v3/pay/transactions/h5"

	body := wechatH5Request{
		AppID:       c.appID,
		MchID:       c.mchID,
		Description: req.Subject,
		OutTradeNo:  req.OrderNumber,
		NotifyURL:   req.NotifyURL,
		Amount: wechatAmount{
			Total:    int(req.Amount.Mul(decimal.NewFromInt(100)).IntPart()),
			Currency: req.Currency,
		},
		Scene: wechatH5Scene{
			PayerClientIP: req.ClientIP,
			H5Info: wechatH5Info{
				Type: "Wap",
			},
		},
	}

	respBody, err := c.callAPIV3(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}

	var result wechatH5Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &PaymentCredential{
		PaymentID: req.OrderNumber,
		PayURL:    result.H5URL,
		ExpireAt:  time.Now().Add(30 * time.Minute),
	}, nil
}

// createAppPayment creates App payment
func (c *WechatClient) createAppPayment(ctx context.Context, req *CreatePaymentRequest) (*PaymentCredential, error) {
	url := c.baseURL + "/v3/pay/transactions/app"

	body := wechatAppRequest{
		AppID:       c.appID,
		MchID:       c.mchID,
		Description: req.Subject,
		OutTradeNo:  req.OrderNumber,
		NotifyURL:   req.NotifyURL,
		Amount: wechatAmount{
			Total:    int(req.Amount.Mul(decimal.NewFromInt(100)).IntPart()),
			Currency: req.Currency,
		},
	}

	respBody, err := c.callAPIV3(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}

	// App payment returns prepay_id, need to generate app payload
	var result wechatAppResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Generate app payload
	appPayload := c.generateAppPayload(result.PrepayID)

	return &PaymentCredential{
		PaymentID:  req.OrderNumber,
		AppPayload: appPayload,
		ExpireAt:   time.Now().Add(30 * time.Minute),
	}, nil
}

// generateAppPayload generates payload for WeChat App
func (c *WechatClient) generateAppPayload(prepayID string) string {
	timestamp := time.Now().Unix()
	nonceStr := generateNonceStr()
	message := fmt.Sprintf("%s\n%d\n%s\n%s\n", c.appID, timestamp, nonceStr, prepayID)

	signature := c.signMessage(message)

	payload := map[string]string{
		"appId":     c.appID,
		"partnerId": c.mchID,
		"prepayId":  prepayID,
		"package":   "Sign=WXPay",
		"nonceStr":  nonceStr,
		"timeStamp": fmt.Sprintf("%d", timestamp),
		"sign":      signature,
	}

	payloadJSON, _ := json.Marshal(payload)
	return string(payloadJSON)
}

// VerifyCallback verifies WeChat Pay callback
func (c *WechatClient) VerifyCallback(ctx context.Context, body []byte, headers http.Header) (*CallbackResult, error) {
	if !c.IsEnabled() {
		return nil, errors.ErrPaymentProviderDisabled
	}

	// Verify signature from headers
	signature := headers.Get("Wechatpay-Signature")
	timestamp := headers.Get("Wechatpay-Timestamp")
	nonce := headers.Get("Wechatpay-Nonce")
	serial := headers.Get("Wechatpay-Serial")

	// Build verification message
	message := fmt.Sprintf("%s\n%s\n%s\n", timestamp, nonce, string(body))

	// Verify signature (simplified - production requires proper verification)
	if err := c.verifySignature(message, signature, serial); err != nil {
		return nil, fmt.Errorf("%w: %v", errors.ErrPaymentInvalidSignature, err)
	}

	// Decrypt callback data using APIv3 key
	var callback wechatCallback
	if err := json.Unmarshal(body, &callback); err != nil {
		return nil, fmt.Errorf("failed to parse callback: %w", err)
	}

	// Decrypt resource
	decryptedData, err := c.decryptResource(callback.Resource.Ciphertext, callback.Resource.Nonce, callback.Resource.AssociatedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt callback: %w", err)
	}

	var resource wechatPaymentResource
	if err := json.Unmarshal([]byte(decryptedData), &resource); err != nil {
		return nil, fmt.Errorf("failed to parse decrypted data: %w", err)
	}

	status := StatusPending
	if resource.TradeState == "SUCCESS" {
		status = StatusSuccess
	} else if resource.TradeState == "CLOSED" {
		status = StatusClosed
	}

	paidAmount := decimal.NewFromInt(int64(resource.Amount.Total / 100))

	return &CallbackResult{
		OrderNumber:  resource.OutTradeNo,
		PaymentID:    resource.TransactionID,
		OutTradeNo:   resource.TransactionID,
		PaidAmount:   paidAmount,
		Status:       status,
		PaidAt:       time.Unix(resource.SuccessTime, 0),
		RawData:      string(body),
		SignVerified: true,
	}, nil
}

// QueryPayment queries WeChat Pay status
func (c *WechatClient) QueryPayment(ctx context.Context, paymentID string) (*PaymentStatus, error) {
	if !c.IsEnabled() {
		return nil, errors.ErrPaymentProviderDisabled
	}

	url := c.baseURL + "/v3/pay/transactions/out-trade-no/" + paymentID + "?mchid=" + c.mchID

	respBody, err := c.callAPIV3(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var result wechatQueryResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	status := StatusPending
	switch result.TradeState {
	case "SUCCESS":
		status = StatusSuccess
	case "CLOSED":
		status = StatusClosed
	case "REFUND":
		status = StatusFailed
	}

	paidAmount := decimal.NewFromInt(int64(result.Amount.Total / 100))

	return &PaymentStatus{
		PaymentID:   result.TransactionID,
		OrderNumber: result.OutTradeNo,
		Status:      status,
		PaidAmount:  paidAmount,
	}, nil
}

// signMessage signs a message with RSA-SHA256
func (c *WechatClient) signMessage(message string) string {
	hashed := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, c.privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(signature)
}

// verifySignature verifies the signature (simplified for development)
func (c *WechatClient) verifySignature(message, signature, serial string) error {
	// TODO: Implement proper signature verification using WeChat Pay public key
	// In production, this MUST be properly implemented
	return nil
}

// decryptResource decrypts callback resource using AES-256-GCM
func (c *WechatClient) decryptResource(ciphertext, nonce, associatedData string) (string, error) {
	// TODO: Implement AES-256-GCM decryption
	// This requires the APIv3 key
	return "", fmt.Errorf("decryption not implemented")
}

// callAPIV3 makes APIv3 call with signature
func (c *WechatClient) callAPIV3(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}

	// Build authorization header
	timestamp := time.Now().Unix()
	nonceStr := generateNonceStr()
	message := fmt.Sprintf("%s\n%s\n%d\n%s\n", method, url, timestamp, nonceStr)
	if len(bodyBytes) > 0 {
		message += string(bodyBytes) + "\n"
	}
	signature := c.signMessage(message)

	authHeader := fmt.Sprintf("WECHATPAY2-SHA256-RSA2048 mchid=\"%s\",nonce_str=\"%s\",signature=\"%s\",timestamp=\"%d\",serial_no=\"%s\"",
		c.mchID, nonceStr, signature, timestamp, c.serialNo)

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	if len(bodyBytes) > 0 {
		req.Body = newBodyReader(bodyBytes)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("wechat API error: status %d", resp.StatusCode)
	}

	// Read response body
	buf := make([]byte, 1024)
	n, _ := resp.Body.Read(buf)
	return buf[:n], nil
}

func generateNonceStr() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func newBodyReader(b []byte) *bodyReader {
	return &bodyReader{data: b}
}

type bodyReader struct {
	data []byte
	pos  int
}

func (r *bodyReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *bodyReader) Close() error { return nil }

// Request/Response structures
type wechatAmount struct {
	Total    int    `json:"total"`
	Currency string `json:"currency"`
}

type wechatNativeRequest struct {
	AppID       string        `json:"appid"`
	MchID       string        `json:"mchid"`
	Description string        `json:"description"`
	OutTradeNo  string        `json:"out_trade_no"`
	NotifyURL   string        `json:"notify_url"`
	Amount      wechatAmount  `json:"amount"`
}

type wechatNativeResponse struct {
	CodeURL string `json:"code_url"`
}

type wechatH5Request struct {
	AppID       string       `json:"appid"`
	MchID       string       `json:"mchid"`
	Description string       `json:"description"`
	OutTradeNo  string       `json:"out_trade_no"`
	NotifyURL   string       `json:"notify_url"`
	Amount      wechatAmount `json:"amount"`
	Scene       wechatH5Scene `json:"scene"`
}

type wechatH5Scene struct {
	PayerClientIP string      `json:"payer_client_ip"`
	H5Info        wechatH5Info `json:"h5_info"`
}

type wechatH5Info struct {
	Type string `json:"type"`
}

type wechatH5Response struct {
	H5URL string `json:"h5_url"`
}

type wechatAppRequest struct {
	AppID       string       `json:"appid"`
	MchID       string       `json:"mchid"`
	Description string       `json:"description"`
	OutTradeNo  string       `json:"out_trade_no"`
	NotifyURL   string       `json:"notify_url"`
	Amount      wechatAmount `json:"amount"`
}

type wechatAppResponse struct {
	PrepayID string `json:"prepay_id"`
}

type wechatQueryResponse struct {
	TransactionID string       `json:"transaction_id"`
	OutTradeNo    string       `json:"out_trade_no"`
	TradeState    string       `json:"trade_state"`
	Amount        wechatAmount `json:"amount"`
}

type wechatCallback struct {
	ID           string `json:"id"`
	CreateTime   string `json:"create_time"`
	ResourceType string `json:"resource_type"`
	EventType    string `json:"event_type"`
	Summary      string `json:"summary"`
	Resource     struct {
		OriginalType   string `json:"original_type"`
		Algorithm      string `json:"algorithm"`
		Ciphertext     string `json:"ciphertext"`
		Nonce          string `json:"nonce"`
		AssociatedData string `json:"associated_data"`
	} `json:"resource"`
}

type wechatPaymentResource struct {
	OutTradeNo    string `json:"out_trade_no"`
	TransactionID string `json:"transaction_id"`
	TradeState    string `json:"trade_state"`
	SuccessTime   int64  `json:"success_time"`
	Amount        struct {
		Total int `json:"total"`
	} `json:"amount"`
}