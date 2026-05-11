package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/application/service"
	"github.com/zhaojiewen/open-station/internal/interfaces/http/middleware"
)

type SettlementHandler struct {
	settlementSvc *service.SettlementService
}

func NewSettlementHandler(settlementSvc *service.SettlementService) *SettlementHandler {
	return &SettlementHandler{
		settlementSvc: settlementSvc,
	}
}

// CheckTrigger checks if settlement should be triggered for tenant
// GET /tenant/settlement/check
func (h *SettlementHandler) CheckTrigger(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	result, err := h.settlementSvc.CheckSettlementTrigger(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"should_trigger": result.ShouldTrigger,
		"amount":         result.Amount.String(),
		"reason":         result.Reason,
	})
}

// TriggerSettlement manually triggers settlement for tenant
// POST /tenant/settlement/trigger
func (h *SettlementHandler) TriggerSettlement(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	user := middleware.GetUser(c)
	if user == nil || user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin permission required"})
		return
	}

	bill, err := h.settlementSvc.TriggerSettlement(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bill": gin.H{
			"id":          bill.ID,
			"bill_number": bill.BillNumber,
			"type":        bill.Type,
			"total_cost":  bill.TotalCost.String(),
			"currency":    bill.Currency,
			"status":      bill.Status,
			"due_date":    bill.DueDate,
		},
	})
}

// ProcessBillPayment processes payment for a settlement bill
// POST /admin/settlement/:bill_id/pay
func (h *SettlementHandler) ProcessBillPayment(c *gin.Context) {
	billIDStr := c.Param("bill_id")
	billID, err := uuid.Parse(billIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bill id"})
		return
	}

	var req struct {
		PaymentAmount decimal.Decimal `json:"payment_amount" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.settlementSvc.ProcessSettlementBill(c.Request.Context(), billID, req.PaymentAmount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "bill payment processed"})
}

// CheckOverdue checks for overdue settlement bills (platform admin)
// GET /platform/settlement/overdue
func (h *SettlementHandler) CheckOverdue(c *gin.Context) {
	if !middleware.HasPlatformPermission(c, "billing_manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	results, err := h.settlementSvc.CheckOverdueBills(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"overdue_bills": results,
		"count":         len(results),
	})
}

// RunScheduledSettlement runs scheduled settlement for all tenants (platform admin)
// POST /platform/settlement/run
func (h *SettlementHandler) RunScheduledSettlement(c *gin.Context) {
	if !middleware.HasPlatformPermission(c, "billing_manage") {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	results, err := h.settlementSvc.RunScheduledSettlement(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	triggeredCount := 0
	errorCount := 0
	for _, r := range results {
		if r.Triggered {
			triggeredCount++
		}
		if r.Error != "" {
			errorCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"results":         results,
		"triggered_count": triggeredCount,
		"error_count":     errorCount,
	})
}