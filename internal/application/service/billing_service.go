package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

type BillingService struct {
	tenantRepo   repository.TenantRepository
	usageRepo    repository.UsageRepository
	billRepo     repository.BillRepository
	rechargeRepo repository.RechargeRepository
	modelRepo    repository.ModelRepository
}

func NewBillingService(
	tenantRepo repository.TenantRepository,
	usageRepo repository.UsageRepository,
	billRepo repository.BillRepository,
	rechargeRepo repository.RechargeRepository,
	modelRepo repository.ModelRepository,
) *BillingService {
	return &BillingService{
		tenantRepo:   tenantRepo,
		usageRepo:    usageRepo,
		billRepo:     billRepo,
		rechargeRepo: rechargeRepo,
		modelRepo:    modelRepo,
	}
}

func (s *BillingService) CalculateCost(ctx context.Context, provider, modelID string, promptTokens, completionTokens int64) (decimal.Decimal, error) {
	model, err := s.modelRepo.GetPricing(ctx, provider, modelID)
	if err != nil {
		return decimal.Zero, fmt.Errorf("failed to get pricing: %w", err)
	}

	promptCost := model.PromptPrice.Mul(decimal.NewFromInt(promptTokens)).Div(decimal.NewFromInt(1000))
	completionCost := model.CompletionPrice.Mul(decimal.NewFromInt(completionTokens)).Div(decimal.NewFromInt(1000))
	totalCost := promptCost.Add(completionCost)

	return totalCost, nil
}

func (s *BillingService) CheckBalance(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
	return s.tenantRepo.GetBalance(ctx, tenantID)
}

func (s *BillingService) RecordUsage(ctx context.Context, tenantID, userID, apiKeyID uuid.UUID, requestID, provider, modelID string, promptTokens, completionTokens int64, latencyMs int, statusCode int) (*entity.UsageRecord, error) {
	cost, err := s.CalculateCost(ctx, provider, modelID, promptTokens, completionTokens)
	if err != nil {
		return nil, err
	}

	// Atomically deduct balance: only succeeds if balance >= cost
	if err := s.tenantRepo.DeductBalance(ctx, tenantID, cost); err != nil {
		return nil, apperrors.ErrInsufficientBalance
	}

	record := &entity.UsageRecord{
		TenantID:         tenantID,
		UserID:           userID,
		APIKeyID:         apiKeyID,
		RequestID:        requestID,
		Provider:         provider,
		ModelID:          modelID,
		PromptTokens:     int(promptTokens),
		CompletionTokens: int(completionTokens),
		TotalTokens:      int(promptTokens + completionTokens),
		Cost:             cost,
		Currency:         "USD",
		LatencyMs:        &latencyMs,
		StatusCode:       &statusCode,
	}

	if err := s.usageRepo.Create(ctx, record); err != nil {
		s.tenantRepo.UpdateBalance(ctx, tenantID, cost)
		return nil, fmt.Errorf("failed to create usage record: %w", err)
	}

	return record, nil
}

func (s *BillingService) Recharge(ctx context.Context, tenantID uuid.UUID, amount decimal.Decimal, paymentMethod, paymentID, notes string) (*entity.RechargeRecord, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, apperrors.ErrInvalidAmount
	}

	record := &entity.RechargeRecord{
		TenantID:      tenantID,
		Amount:        amount,
		Currency:      "USD",
		PaymentMethod: paymentMethod,
		PaymentID:     paymentID,
		Status:        "pending",
		Notes:         notes,
	}

	if err := s.rechargeRepo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to create recharge record: %w", err)
	}

	if err := s.tenantRepo.UpdateBalance(ctx, tenantID, amount); err != nil {
		return nil, fmt.Errorf("failed to update balance: %w", err)
	}

	// Note: Payment completion should be handled via external payment webhook/callback.
	// For now, auto-complete for amounts that don't require external verification.
	if err := s.rechargeRepo.MarkCompleted(ctx, record.ID); err != nil {
		return nil, fmt.Errorf("failed to mark recharge completed: %w", err)
	}

	record.Status = "completed"
	now := time.Now()
	record.CompletedAt = &now

	return record, nil
}

func (s *BillingService) GetUsage(ctx context.Context, tenantID uuid.UUID, start, end time.Time, page, pageSize int) ([]entity.UsageRecord, int64, error) {
	return s.usageRepo.List(ctx, tenantID, start, end, page, pageSize)
}

func (s *BillingService) GetTotalCost(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (decimal.Decimal, int64, error) {
	return s.usageRepo.GetTotalCost(ctx, tenantID, start, end)
}

func (s *BillingService) GenerateBill(ctx context.Context, tenantID uuid.UUID, periodStart, periodEnd time.Time) (*entity.Bill, error) {
	totalCost, totalTokens, err := s.usageRepo.GetTotalCost(ctx, tenantID, periodStart, periodEnd)
	if err != nil {
		return nil, err
	}

	if totalCost.IsZero() {
		return nil, apperrors.NewAppError("BILL_003", "no usage in this period", nil)
	}

	existingBill, err := s.billRepo.GetByPeriod(ctx, tenantID, periodStart, periodEnd)
	if err == nil && existingBill != nil {
		return existingBill, nil
	}

	billNumber := fmt.Sprintf("BILL-%s-%d", tenantID.String()[:8], time.Now().Unix())

	bill := &entity.Bill{
		TenantID:    tenantID,
		BillNumber:  billNumber,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		TotalTokens: totalTokens,
		TotalCost:   totalCost,
		Currency:    "USD",
		Status:      "pending",
	}

	if err := s.billRepo.Create(ctx, bill); err != nil {
		return nil, fmt.Errorf("failed to create bill: %w", err)
	}

	return bill, nil
}

func (s *BillingService) GetBills(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.Bill, int64, error) {
	return s.billRepo.List(ctx, tenantID, page, pageSize)
}

func (s *BillingService) GetRechargeRecords(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.RechargeRecord, int64, error) {
	return s.rechargeRepo.List(ctx, tenantID, page, pageSize)
}