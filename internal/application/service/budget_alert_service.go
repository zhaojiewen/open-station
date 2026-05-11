package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
)

// BudgetAlertService handles budget alert management and triggering
type BudgetAlertService struct {
	alertRepo        repository.BudgetAlertRepository
	tenantRepo       repository.TenantRepository
	userRepo         repository.UserRepository
	apiKeyRepo       repository.APIKeyRepository
	notificationSvc  *NotificationService
}

// NewBudgetAlertService creates a new budget alert service
func NewBudgetAlertService(
	alertRepo repository.BudgetAlertRepository,
	tenantRepo repository.TenantRepository,
	userRepo repository.UserRepository,
	apiKeyRepo repository.APIKeyRepository,
	notificationSvc *NotificationService,
) *BudgetAlertService {
	return &BudgetAlertService{
		alertRepo:       alertRepo,
		tenantRepo:      tenantRepo,
		userRepo:        userRepo,
		apiKeyRepo:      apiKeyRepo,
		notificationSvc: notificationSvc,
	}
}

// Create creates a new budget alert
func (s *BudgetAlertService) Create(ctx context.Context, scope string, scopeID uuid.UUID, alertType string, thresholdPercent int, notifyEmails []string, notifySlack, notifyWebhook string) (*entity.BudgetAlert, error) {
	// Validate scope
	if scope != "tenant" && scope != "user" && scope != "api_key" {
		return nil, fmt.Errorf("invalid scope: must be tenant, user, or api_key")
	}

	// Validate threshold
	if thresholdPercent < 1 || thresholdPercent > 100 {
		return nil, fmt.Errorf("invalid threshold: must be between 1 and 100")
	}

	// Serialize notify emails
	emailsJSON, _ := json.Marshal(notifyEmails)

	alert := &entity.BudgetAlert{
		Scope:            scope,
		ScopeID:          scopeID,
		AlertType:        alertType,
		ThresholdPercent: thresholdPercent,
		NotifyEmails:     string(emailsJSON),
		NotifySlack:      notifySlack,
		NotifyWebhook:    notifyWebhook,
		IsEnabled:        true,
	}

	if err := s.alertRepo.Create(ctx, alert); err != nil {
		return nil, err
	}

	return alert, nil
}

// GetByID gets an alert by ID
func (s *BudgetAlertService) GetByID(ctx context.Context, id uuid.UUID) (*entity.BudgetAlert, error) {
	return s.alertRepo.GetByID(ctx, id)
}

// List lists alerts with pagination
func (s *BudgetAlertService) List(ctx context.Context, page, pageSize int) ([]entity.BudgetAlert, int64, error) {
	return s.alertRepo.List(ctx, page, pageSize)
}

// GetByScope gets alerts for a specific scope
func (s *BudgetAlertService) GetByScope(ctx context.Context, scope string, scopeID uuid.UUID) ([]entity.BudgetAlert, error) {
	return s.alertRepo.GetByScope(ctx, scope, scopeID)
}

// Update updates an alert
func (s *BudgetAlertService) Update(ctx context.Context, id uuid.UUID, thresholdPercent int, notifyEmails []string, notifySlack, notifyWebhook string) error {
	alert, err := s.alertRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	emailsJSON, _ := json.Marshal(notifyEmails)
	alert.ThresholdPercent = thresholdPercent
	alert.NotifyEmails = string(emailsJSON)
	alert.NotifySlack = notifySlack
	alert.NotifyWebhook = notifyWebhook

	return s.alertRepo.Update(ctx, alert)
}

// Enable enables an alert
func (s *BudgetAlertService) Enable(ctx context.Context, id uuid.UUID) error {
	return s.alertRepo.Enable(ctx, id)
}

// Disable disables an alert
func (s *BudgetAlertService) Disable(ctx context.Context, id uuid.UUID) error {
	return s.alertRepo.Disable(ctx, id)
}

// Delete deletes an alert
func (s *BudgetAlertService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.alertRepo.Delete(ctx, id)
}

// CheckAndTriggerAlerts checks budget usage and triggers alerts if thresholds exceeded
func (s *BudgetAlertService) CheckAndTriggerAlerts(ctx context.Context, apiKeyID, userID, tenantID uuid.UUID) {
	// Check tenant alerts
	s.checkTenantAlerts(ctx, tenantID)

	// Check user alerts
	s.checkUserAlerts(ctx, userID)

	// Check API key alerts (built-in threshold)
	s.checkAPIKeyAlerts(ctx, apiKeyID)
}

// checkTenantAlerts checks and triggers alerts for a tenant
func (s *BudgetAlertService) checkTenantAlerts(ctx context.Context, tenantID uuid.UUID) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return
	}

	// Get alerts for tenant
	alerts, err := s.alertRepo.GetByScope(ctx, "tenant", tenantID)
	if err != nil || len(alerts) == 0 {
		return
	}

	// Calculate usage percentage
	var percentUsed int
	if tenant.MonthlyBudgetLimit != nil && !tenant.MonthlyBudgetLimit.IsZero() {
		percentUsed = int(tenant.BudgetUsedMonth.Div(*tenant.MonthlyBudgetLimit).Mul(decimal.NewFromInt(100)).IntPart())
	}

	// Check each alert
	for _, alert := range alerts {
		if !alert.IsEnabled {
			continue
		}

		if percentUsed >= alert.ThresholdPercent {
			s.triggerAlert(ctx, &alert, "tenant", tenantID, percentUsed, tenant.BudgetUsedMonth, *tenant.MonthlyBudgetLimit)
		}
	}
}

// checkUserAlerts checks and triggers alerts for a user
func (s *BudgetAlertService) checkUserAlerts(ctx context.Context, userID uuid.UUID) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return
	}

	// Get alerts for user
	alerts, err := s.alertRepo.GetByScope(ctx, "user", userID)
	if err != nil || len(alerts) == 0 {
		return
	}

	// Calculate monthly usage percentage
	var monthlyPercentUsed int
	if user.MonthlyBudget != nil && !user.MonthlyBudget.IsZero() {
		monthlyPercentUsed = int(user.BudgetUsedMonth.Div(*user.MonthlyBudget).Mul(decimal.NewFromInt(100)).IntPart())
	}

	// Calculate daily usage percentage
	var dailyPercentUsed int
	if user.DailyBudget != nil && !user.DailyBudget.IsZero() {
		dailyPercentUsed = int(user.BudgetUsedToday.Div(*user.DailyBudget).Mul(decimal.NewFromInt(100)).IntPart())
	}

	// Check each alert
	for _, alert := range alerts {
		if !alert.IsEnabled {
			continue
		}

		// Use the higher of monthly or daily percentage
		percentUsed := monthlyPercentUsed
		if dailyPercentUsed > percentUsed {
			percentUsed = dailyPercentUsed
		}

		if percentUsed >= alert.ThresholdPercent {
			var used, limit decimal.Decimal
			if user.MonthlyBudget != nil {
				used = user.BudgetUsedMonth
				limit = *user.MonthlyBudget
			} else if user.DailyBudget != nil {
				used = user.BudgetUsedToday
				limit = *user.DailyBudget
			}
			s.triggerAlert(ctx, &alert, "user", userID, percentUsed, used, limit)
		}
	}
}

// checkAPIKeyAlerts checks built-in alerts for an API key
func (s *BudgetAlertService) checkAPIKeyAlerts(ctx context.Context, apiKeyID uuid.UUID) {
	apiKey, err := s.apiKeyRepo.GetByID(ctx, apiKeyID)
	if err != nil {
		return
	}

	// Check monthly cost percentage
	if apiKey.MonthlyCostLimit != nil && !apiKey.MonthlyCostLimit.IsZero() {
		percentUsed := int(apiKey.MonthlyCostUsed.Div(*apiKey.MonthlyCostLimit).Mul(decimal.NewFromInt(100)).IntPart())

		// Check thresholds (80, 90, 100)
		if percentUsed >= apiKey.AlertThreshold1 || percentUsed >= apiKey.AlertThreshold2 || percentUsed >= apiKey.AlertThreshold3 {
			// Get tenant to find notification recipients
			tenant, err := s.tenantRepo.GetByID(ctx, apiKey.TenantID)
			if err != nil {
				return
			}

			// Build alert info
			alertInfo := AlertInfo{
				Scope:         "api_key",
				ScopeID:       apiKeyID,
				PercentUsed:   percentUsed,
				UsedAmount:    apiKey.MonthlyCostUsed,
				LimitAmount:   *apiKey.MonthlyCostLimit,
				ResourceName:  apiKey.Name,
				TenantName:    tenant.Name,
				TenantEmail:   tenant.BillingEmail,
				Timestamp:     time.Now(),
			}

			// Send notification to tenant billing email
			if tenant.BillingEmail != "" {
				s.notificationSvc.SendEmail([]string{tenant.BillingEmail}, alertInfo)
			}
		}
	}

	// Check daily cost percentage
	if apiKey.DailyCostLimit != nil && !apiKey.DailyCostLimit.IsZero() {
		percentUsed := int(apiKey.DailyCostUsed.Div(*apiKey.DailyCostLimit).Mul(decimal.NewFromInt(100)).IntPart())

		if percentUsed >= 100 {
			tenant, err := s.tenantRepo.GetByID(ctx, apiKey.TenantID)
			if err != nil {
				return
			}

			alertInfo := AlertInfo{
				Scope:         "api_key",
				ScopeID:       apiKeyID,
				PercentUsed:   percentUsed,
				UsedAmount:    apiKey.DailyCostUsed,
				LimitAmount:   *apiKey.DailyCostLimit,
				ResourceName:  apiKey.Name + " (daily)",
				TenantName:    tenant.Name,
				TenantEmail:   tenant.BillingEmail,
				Timestamp:     time.Now(),
			}

			if tenant.BillingEmail != "" {
				s.notificationSvc.SendEmail([]string{tenant.BillingEmail}, alertInfo)
			}
		}
	}
}

// triggerAlert triggers an alert notification
func (s *BudgetAlertService) triggerAlert(ctx context.Context, alert *entity.BudgetAlert, scope string, scopeID uuid.UUID, percentUsed int, used, limit decimal.Decimal) {
	// Get resource name
	var resourceName, tenantName, tenantEmail string
	var user *entity.User
	var tenant *entity.Tenant

	switch scope {
	case "tenant":
		tenant, _ = s.tenantRepo.GetByID(ctx, scopeID)
		if tenant != nil {
			resourceName = tenant.Name
			tenantName = tenant.Name
			tenantEmail = tenant.BillingEmail
		}
	case "user":
		user, _ = s.userRepo.GetByID(ctx, scopeID)
		if user != nil {
			resourceName = user.Name + " (" + user.Email + ")"
			tenant, _ = s.tenantRepo.GetByID(ctx, user.TenantID)
			if tenant != nil {
				tenantName = tenant.Name
				tenantEmail = tenant.BillingEmail
			}
		}
	case "api_key":
		apiKey, _ := s.apiKeyRepo.GetByID(ctx, scopeID)
		if apiKey != nil {
			resourceName = apiKey.Name
			tenant, _ = s.tenantRepo.GetByID(ctx, apiKey.TenantID)
			if tenant != nil {
				tenantName = tenant.Name
				tenantEmail = tenant.BillingEmail
			}
		}
	}

	// Build alert info
	alertInfo := AlertInfo{
		Scope:         scope,
		ScopeID:       scopeID,
		PercentUsed:   percentUsed,
		UsedAmount:    used,
		LimitAmount:   limit,
		ResourceName:  resourceName,
		TenantName:    tenantName,
		TenantEmail:   tenantEmail,
		Timestamp:     time.Now(),
	}

	// Parse notify emails
	var notifyEmails []string
	if alert.NotifyEmails != "" {
		json.Unmarshal([]byte(alert.NotifyEmails), &notifyEmails)
	}

	// If no custom emails, use tenant billing email
	if len(notifyEmails) == 0 && tenantEmail != "" {
		notifyEmails = []string{tenantEmail}
	}

	// Send notifications
	if len(notifyEmails) > 0 {
		s.notificationSvc.SendEmail(notifyEmails, alertInfo)
	}

	if alert.NotifySlack != "" {
		s.notificationSvc.SendSlack(alert.NotifySlack, alertInfo)
	}

	if alert.NotifyWebhook != "" {
		s.notificationSvc.SendWebhook(alert.NotifyWebhook, alertInfo)
	}

	// Mark alert as triggered
	s.alertRepo.MarkTriggered(ctx, alert.ID)
}

// AlertInfo contains information for alert notifications
type AlertInfo struct {
	Scope         string
	ScopeID       uuid.UUID
	PercentUsed   int
	UsedAmount    decimal.Decimal
	LimitAmount   decimal.Decimal
	ResourceName  string
	TenantName    string
	TenantEmail   string
	Timestamp     time.Time
}