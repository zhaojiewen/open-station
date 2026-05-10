package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/application/service"
)

// BillingHandler handles billing-related requests
type BillingHandler struct {
	billingService *service.BillingService
}

func NewBillingHandler(billingService *service.BillingService) *BillingHandler {
	return &BillingHandler{
		billingService: billingService,
	}
}

func (h *BillingHandler) GetBalance(c *gin.Context) {
	tenantIDStr := c.Param("tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
		return
	}

	balance, err := h.billingService.CheckBalance(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant_id": tenantID,
		"balance":   balance.String(),
		"currency":  "USD",
	})
}

func (h *BillingHandler) Recharge(c *gin.Context) {
	var req struct {
		TenantID      string          `json:"tenant_id" binding:"required"`
		Amount        decimal.Decimal `json:"amount" binding:"required"`
		PaymentMethod string          `json:"payment_method"`
		PaymentID     string          `json:"payment_id"`
		Notes         string          `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
		return
	}

	record, err := h.billingService.Recharge(c.Request.Context(), tenantID, req.Amount, req.PaymentMethod, req.PaymentID, req.Notes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *BillingHandler) GetUsage(c *gin.Context) {
	tenantIDStr := c.Query("tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
		return
	}

	startStr := c.Query("start_date")
	endStr := c.Query("end_date")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		start = time.Now().AddDate(0, -1, 0)
	}
	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		end = time.Now()
	}

	pageInt := 1
	pageSizeInt := 20
	if p, err := strconv.Atoi(page); err == nil {
		pageInt = p
	}
	if ps, err := strconv.Atoi(pageSize); err == nil {
		pageSizeInt = ps
	}

	records, total, err := h.billingService.GetUsage(c.Request.Context(), tenantID, start, end, pageInt, pageSizeInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"records":    records,
		"total":      total,
		"page":       pageInt,
		"page_size":  pageSizeInt,
	})
}

func (h *BillingHandler) GetBills(c *gin.Context) {
	tenantIDStr := c.Query("tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant id"})
		return
	}

	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	pageInt := 1
	pageSizeInt := 20
	if p, err := strconv.Atoi(page); err == nil {
		pageInt = p
	}
	if ps, err := strconv.Atoi(pageSize); err == nil {
		pageSizeInt = ps
	}

	bills, total, err := h.billingService.GetBills(c.Request.Context(), tenantID, pageInt, pageSizeInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bills":      bills,
		"total":      total,
		"page":       pageInt,
		"page_size":  pageSizeInt,
	})
}