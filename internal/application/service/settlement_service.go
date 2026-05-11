package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
)

// SettlementService handles credit settlement for approved tenants
type SettlementService struct {
	tenantRepo  repository.TenantRepository
	billRepo    repository.BillRepository
	notificationSvc *NotificationService
}

// NewSettlementService creates a new settlement service
func NewSettlementService(
	tenantRepo repository.TenantRepository,
	billRepo repository.BillRepository,
	notificationSvc *NotificationService,
) *SettlementService {
	return &SettlementService{
		tenantRepo:  tenantRepo,
		billRepo:    billRepo,
		notificationSvc: notificationSvc,
	}
}

// CheckSettlementTrigger checks if settlement should be triggered for a tenant
func (s *SettlementService) CheckSettlementTrigger(ctx context.Context, tenantID uuid.UUID) (*SettlementTriggerResult, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Only trigger settlement for approved credit tenants
	if tenant.CreditStatus != "approved" || tenant.CreditLimit == nil {
		return &SettlementTriggerResult{ShouldTrigger: false}, nil
	}

	// Check based on settlement cycle
	switch tenant.SettlementCycle {
	case "monthly":
		return s.checkMonthlySettlement(ctx, tenant)
	case "weekly":
		return s.checkWeeklySettlement(ctx, tenant)
	case "threshold":
		return s.checkThresholdSettlement(ctx, tenant)
	case "custom":
		return s.checkCustomSettlement(ctx, tenant)
	default:
		return &SettlementTriggerResult{ShouldTrigger: false}, nil
	}
}

// SettlementTriggerResult contains the result of settlement trigger check
type SettlementTriggerResult struct {
	ShouldTrigger bool
	Amount        decimal.Decimal
	Reason        string
}

// checkMonthlySettlement checks if monthly settlement should be triggered
func (s *SettlementService) checkMonthlySettlement(ctx context.Context, tenant *entity.Tenant) (*SettlementTriggerResult, error) {
	if tenant.SettlementDay == nil {
		return &SettlementTriggerResult{ShouldTrigger: false}, nil
	}

	now := time.Now()
	settlementDay := *tenant.SettlementDay

	// Check if today is the settlement day
	if now.Day() == settlementDay && tenant.CreditUsed.GreaterThan(decimal.Zero) {
		return &SettlementTriggerResult{
			ShouldTrigger: true,
			Amount:        tenant.CreditUsed,
			Reason:        fmt.Sprintf("monthly settlement on day %d", settlementDay),
		}, nil
	}

	return &SettlementTriggerResult{ShouldTrigger: false}, nil
}

// checkWeeklySettlement checks if weekly settlement should be triggered
func (s *SettlementService) checkWeeklySettlement(ctx context.Context, tenant *entity.Tenant) (*SettlementTriggerResult, error) {
	if tenant.SettlementDay == nil {
		return &SettlementTriggerResult{ShouldTrigger: false}, nil
	}

	now := time.Now()
	// SettlementDay represents day of week (0=Sunday, 1=Monday, etc.)
	settlementDayOfWeek := *tenant.SettlementDay

	// Check if today is the settlement day of week
	if int(now.Weekday()) == settlementDayOfWeek && tenant.CreditUsed.GreaterThan(decimal.Zero) {
		return &SettlementTriggerResult{
			ShouldTrigger: true,
			Amount:        tenant.CreditUsed,
			Reason:        fmt.Sprintf("weekly settlement on %s", now.Weekday().String()),
		}, nil
	}

	return &SettlementTriggerResult{ShouldTrigger: false}, nil
}

// checkThresholdSettlement checks if threshold settlement should be triggered
func (s *SettlementService) checkThresholdSettlement(ctx context.Context, tenant *entity.Tenant) (*SettlementTriggerResult, error) {
	if tenant.ThresholdAmount == nil {
		return &SettlementTriggerResult{ShouldTrigger: false}, nil
	}

	// Trigger settlement when credit used reaches threshold
	if tenant.CreditUsed.GreaterThanOrEqual(*tenant.ThresholdAmount) {
		return &SettlementTriggerResult{
			ShouldTrigger: true,
			Amount:        tenant.CreditUsed,
			Reason:        fmt.Sprintf("threshold settlement: credit used %s >= threshold %s", tenant.CreditUsed.String(), tenant.ThresholdAmount.String()),
		}, nil
	}

	return &SettlementTriggerResult{ShouldTrigger: false}, nil
}

// checkCustomSettlement checks custom settlement rules
func (s *SettlementService) checkCustomSettlement(ctx context.Context, tenant *entity.Tenant) (*SettlementTriggerResult, error) {
	// Custom settlement requires manual triggering
	// This can be extended for custom business logic
	return &SettlementTriggerResult{ShouldTrigger: false}, nil
}

// TriggerSettlement creates a settlement bill for a tenant
func (s *SettlementService) TriggerSettlement(ctx context.Context, tenantID uuid.UUID) (*entity.Bill, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Validate tenant has approved credit
	if tenant.CreditStatus != "approved" {
		return nil, fmt.Errorf("tenant credit not approved")
	}

	if tenant.CreditUsed.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("no credit usage to settle")
	}

	// Create settlement bill
	bill := &entity.Bill{
		TenantID:    tenantID,
		Type:        "credit_settlement",
		TotalCost:   tenant.CreditUsed,
		Currency:    "USD",
		Status:      "pending",
		Description: fmt.Sprintf("Credit settlement - %s cycle", tenant.SettlementCycle),
		DueDate:     calculateDueDate(tenant.SettlementCycle),
		PeriodStart: time.Now(),
		PeriodEnd:   time.Now(),
	}

	if err := s.billRepo.Create(ctx, bill); err != nil {
		return nil, fmt.Errorf("failed to create settlement bill: %w", err)
	}

	// Reset credit used after creating bill
	tenant.CreditUsed = decimal.Zero
	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to reset credit used: %w", err)
	}

	// Send notification
	if s.notificationSvc != nil {
		// TODO: Send settlement notification to tenant admin
	}

	return bill, nil
}

// calculateDueDate calculates the due date for a settlement bill
func calculateDueDate(settlementCycle string) *time.Time {
	now := time.Now()
	var dueDate time.Time

	switch settlementCycle {
	case "monthly":
		// Due in 15 days
		dueDate = now.AddDate(0, 0, 15)
	case "weekly":
		// Due in 7 days
		dueDate = now.AddDate(0, 0, 7)
	case "threshold":
		// Due in 10 days
		dueDate = now.AddDate(0, 0, 10)
	default:
		// Due in 15 days
		dueDate = now.AddDate(0, 0, 15)
	}

	return &dueDate
}

// ProcessSettlementBill processes payment for a settlement bill
func (s *SettlementService) ProcessSettlementBill(ctx context.Context, billID uuid.UUID, paymentAmount decimal.Decimal) error {
	bill, err := s.billRepo.GetByID(ctx, billID)
	if err != nil {
		return fmt.Errorf("failed to get bill: %w", err)
	}

	if bill.Type != "credit_settlement" {
		return fmt.Errorf("bill is not a settlement bill")
	}

	// Update bill status
	now := time.Now()
	if paymentAmount.GreaterThanOrEqual(bill.TotalCost) {
		bill.Status = "paid"
		bill.PaidAt = &now
	} else {
		bill.Status = "partial_paid"
		bill.TotalCost = bill.TotalCost.Sub(paymentAmount) // remaining amount
	}

	if err := s.billRepo.Update(ctx, bill); err != nil {
		return fmt.Errorf("failed to update bill: %w", err)
	}

	// If fully paid, optionally convert to prepaid balance
	if bill.Status == "paid" {
		_, err := s.tenantRepo.GetByID(ctx, bill.TenantID)
		if err == nil {
			// Option: Convert payment to prepaid balance
			// tenant.Balance = tenant.Balance.Add(paymentAmount)
			// s.tenantRepo.Update(ctx, tenant)
		}
	}

	return nil
}

// CheckOverdueBills checks for overdue settlement bills and suspends tenants
func (s *SettlementService) CheckOverdueBills(ctx context.Context) ([]OverdueBillResult, error) {
	now := time.Now()
	var overdueBills []OverdueBillResult

	// Get all pending bills with past due date
	bills, _, err := s.billRepo.ListByStatus(ctx, "pending", 1, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending bills: %w", err)
	}

	for _, bill := range bills {
		if bill.Type != "credit_settlement" {
			continue
		}

		if bill.DueDate != nil && now.After(*bill.DueDate) {
			// Bill is overdue, suspend tenant
			tenant, err := s.tenantRepo.GetByID(ctx, bill.TenantID)
			if err != nil {
				continue
			}

			tenant.Status = "suspended"
			if err := s.tenantRepo.Update(ctx, tenant); err != nil {
				continue
			}

			overdueBills = append(overdueBills, OverdueBillResult{
				BillID:   bill.ID,
				TenantID: bill.TenantID,
				Amount:   bill.TotalCost,
				DueDate:  *bill.DueDate,
			})

			// Send overdue notification
			if s.notificationSvc != nil {
				// TODO: Send overdue notification
			}
		}
	}

	return overdueBills, nil
}

// OverdueBillResult contains overdue bill information
type OverdueBillResult struct {
	BillID   uuid.UUID
	TenantID uuid.UUID
	Amount   decimal.Decimal
	DueDate  time.Time
}

// RunScheduledSettlement runs scheduled settlement checks for all tenants
func (s *SettlementService) RunScheduledSettlement(ctx context.Context) ([]SettlementResult, error) {
	var results []SettlementResult

	// Get all tenants with approved credit
	tenants, _, err := s.tenantRepo.ListByCreditStatus(ctx, "approved", 1, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to list approved credit tenants: %w", err)
	}

	for _, tenant := range tenants {
		triggerResult, err := s.CheckSettlementTrigger(ctx, tenant.ID)
		if err != nil {
			results = append(results, SettlementResult{
				TenantID: tenant.ID,
				Error:    err.Error(),
			})
			continue
		}

		if triggerResult.ShouldTrigger {
			bill, err := s.TriggerSettlement(ctx, tenant.ID)
			if err != nil {
				results = append(results, SettlementResult{
					TenantID: tenant.ID,
					Error:    err.Error(),
				})
				continue
			}

			results = append(results, SettlementResult{
				TenantID:  tenant.ID,
				BillID:    bill.ID,
				Amount:    bill.TotalCost,
				Triggered: true,
			})
		}
	}

	return results, nil
}

// SettlementResult contains settlement result for a tenant
type SettlementResult struct {
	TenantID  uuid.UUID
	BillID    uuid.UUID
	Amount    decimal.Decimal
	Triggered bool
	Error     string
}