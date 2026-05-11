package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/application/service"
	"github.com/zhaojiewen/open-station/internal/infrastructure/payment"
	"github.com/zhaojiewen/open-station/internal/interfaces/http/middleware"
)

type PaymentHandler struct {
	paymentSvc *service.PaymentService
	gatewaySvc *payment.PaymentGatewayService
}

func NewPaymentHandler(paymentSvc *service.PaymentService, gatewaySvc *payment.PaymentGatewayService) *PaymentHandler {
	return &PaymentHandler{
		paymentSvc: paymentSvc,
		gatewaySvc: gatewaySvc,
	}
}

// CreateOrder handles payment order creation
// POST /user/payments (individual mode) or POST /admin/payments (organization mode)
func (h *PaymentHandler) CreateOrder(c *gin.Context) {
	var req struct {
		PaymentMode     string          `json:"payment_mode" binding:"required"`    // individual, organization
		OrderType       string          `json:"order_type" binding:"required"`      // recharge, subscription, credit_settlement
		Amount          decimal.Decimal `json:"amount" binding:"required"`
		Currency        string          `json:"currency"`
		PaymentProvider string          `json:"payment_provider" binding:"required"` // alipay, wechat, stripe, paypal, bank
		PaymentMethod   string          `json:"payment_method" binding:"required"`   // qr_code, web, app, bank_transfer
		ReturnURL       string          `json:"return_url"`                          // frontend redirect URL
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)

	var userQuotaID *uuid.UUID
	var tenantID *uuid.UUID

	if req.PaymentMode == "individual" {
		// Individual mode: user quota will be looked up by service
	} else if req.PaymentMode == "organization" {
		tenantIDFromCtx := middleware.GetTenantID(c)
		tenantID = &tenantIDFromCtx
	}

	if req.Currency == "" {
		req.Currency = "USD"
	}

	orderReq := &service.PaymentOrderRequest{
		PaymentMode:     req.PaymentMode,
		UserID:          userID,
		UserQuotaID:     userQuotaID,
		TenantID:        tenantID,
		OrderType:       req.OrderType,
		Amount:          req.Amount,
		Currency:        req.Currency,
		PaymentProvider: req.PaymentProvider,
		PaymentMethod:   req.PaymentMethod,
		ReturnURL:       req.ReturnURL,
		ClientIP:        c.ClientIP(),
	}

	result, err := h.paymentSvc.CreatePaymentOrder(c.Request.Context(), orderReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order := result.Order
	credential := result.Credential

	response := gin.H{
		"order": gin.H{
			"id":              order.ID,
			"order_number":    order.OrderNumber,
			"amount":          order.Amount.String(),
			"currency":        order.Currency,
			"status":          order.Status,
			"expire_at":       order.ExpireAt,
			"payment_mode":    order.PaymentMode,
			"payment_provider": order.PaymentProvider,
		},
	}

	// Include payment credential if available
	if credential != nil {
		response["credential"] = gin.H{
			"payment_id":  credential.PaymentID,
			"qr_code_url": credential.QRCodeURL,
			"qr_code":     credential.QRCodeData,
			"pay_url":     credential.PayURL,
			"app_payload": credential.AppPayload,
			"expire_at":   credential.ExpireAt,
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetOrder retrieves a payment order by ID
// GET /user/payments/:id or GET /admin/payments/:id
func (h *PaymentHandler) GetOrder(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	order, err := h.paymentSvc.GetPaymentOrder(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	userID := middleware.GetUserID(c)
	if order.UserID != userID {
		if order.TenantID != nil {
			tenantID := middleware.GetTenantID(c)
			if *order.TenantID != tenantID {
				c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
				return
			}
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"order": order})
}

// GetOrderByNumber retrieves a payment order by order number (public for payment page)
// GET /payments/:order_number
func (h *PaymentHandler) GetOrderByNumber(c *gin.Context) {
	orderNumber := c.Param("order_number")

	order, err := h.paymentSvc.GetPaymentOrderByNumber(c.Request.Context(), orderNumber)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order": gin.H{
			"order_number":     order.OrderNumber,
			"amount":           order.Amount.String(),
			"currency":         order.Currency,
			"status":           order.Status,
			"payment_provider": order.PaymentProvider,
			"payment_method":   order.PaymentMethod,
			"expire_at":        order.ExpireAt,
		},
	})
}

// CancelOrder cancels a pending payment order
// POST /user/payments/:id/cancel
func (h *PaymentHandler) CancelOrder(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	order, err := h.paymentSvc.GetPaymentOrder(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	userID := middleware.GetUserID(c)
	if order.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	err = h.paymentSvc.CancelPaymentOrder(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order cancelled"})
}

// ProcessCallback handles payment provider callback (public endpoint)
// POST /payments/callback/:provider
func (h *PaymentHandler) ProcessCallback(c *gin.Context) {
	provider := c.Param("provider")

	// Read raw body for signature verification
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// Verify signature and parse callback using gateway service
	if h.gatewaySvc != nil {
		result, err := h.gatewaySvc.VerifyCallback(c.Request.Context(), provider, body, c.Request.Header)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Process verified payment
		callbackReq := &service.PaymentCallbackRequest{
			OrderNumber:   result.OrderNumber,
			PaymentID:     result.PaymentID,
			PaidAmount:    result.PaidAmount,
			CallbackData:  result.RawData,
			PaymentStatus: result.Status,
		}

		order, err := h.paymentSvc.ProcessPaymentCallback(c.Request.Context(), callbackReq)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":      "payment processed",
			"order_id":     order.ID,
			"order_status": order.Status,
		})
		return
	}

	// Fallback: process without gateway verification (for testing)
	var req struct {
		OrderNumber   string          `json:"order_number" binding:"required"`
		PaymentID     string          `json:"payment_id" binding:"required"`
		PaidAmount    decimal.Decimal `json:"paid_amount" binding:"required"`
		PaymentStatus string          `json:"payment_status" binding:"required"` // success, failed
		CallbackData  string          `json:"callback_data"`
	}

	// Try JSON parsing first, then form parsing
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.PaymentStatus != "success" {
		c.JSON(http.StatusOK, gin.H{"message": "payment not successful, ignored"})
		return
	}

	callbackReq := &service.PaymentCallbackRequest{
		OrderNumber:   req.OrderNumber,
		PaymentID:     req.PaymentID,
		PaidAmount:    req.PaidAmount,
		CallbackData:  req.CallbackData,
		PaymentStatus: req.PaymentStatus,
	}

	order, err := h.paymentSvc.ProcessPaymentCallback(c.Request.Context(), callbackReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "payment processed",
		"order_id":     order.ID,
		"order_status": order.Status,
	})
}

// ListOrders lists payment orders with pagination
// GET /user/payments or GET /admin/payments
func (h *PaymentHandler) ListOrders(c *gin.Context) {
	status := c.Query("status")
	paymentMode := c.Query("payment_mode")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	if paymentMode == "" {
		if c.FullPath() != "" && len(c.FullPath()) >= 5 && c.FullPath()[:5] == "/user" {
			paymentMode = "individual"
		} else {
			paymentMode = "organization"
		}
	}

	listReq := &service.PaymentOrderListRequest{
		UserID:      userID,
		TenantID:    tenantID,
		PaymentMode: paymentMode,
		Status:      status,
		Page:        page,
		PageSize:    pageSize,
	}

	orders, total, err := h.paymentSvc.ListPaymentOrders(c.Request.Context(), listReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders":    orders,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetPendingOrders returns pending orders for current user/tenant
// GET /user/payments/pending
func (h *PaymentHandler) GetPendingOrders(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tenantID := middleware.GetTenantID(c)

	var tenantIDPtr *uuid.UUID
	userObj := middleware.GetUser(c)
	if userObj != nil && userObj.Role == "admin" {
		tenantIDPtr = &tenantID
	}

	orders, err := h.paymentSvc.GetPendingOrders(c.Request.Context(), userID, tenantIDPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"orders": orders})
}