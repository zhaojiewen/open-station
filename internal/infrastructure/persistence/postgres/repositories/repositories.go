package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"gorm.io/gorm"
)

type TenantRepoImpl struct {
	db *gorm.DB
}

func NewTenantRepository(db *gorm.DB) *TenantRepoImpl {
	return &TenantRepoImpl{db: db}
}

func (r *TenantRepoImpl) Create(ctx context.Context, tenant *entity.Tenant) error {
	return r.db.WithContext(ctx).Create(tenant).Error
}

func (r *TenantRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error) {
	var tenant entity.Tenant
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&tenant).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *TenantRepoImpl) GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error) {
	var tenant entity.Tenant
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&tenant).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *TenantRepoImpl) Update(ctx context.Context, tenant *entity.Tenant) error {
	return r.db.WithContext(ctx).Save(tenant).Error
}

func (r *TenantRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.Tenant{}, id).Error
}

func (r *TenantRepoImpl) List(ctx context.Context, page, pageSize int) ([]entity.Tenant, int64, error) {
	var tenants []entity.Tenant
	var total int64

	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).Model(&entity.Tenant{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = r.db.WithContext(ctx).Offset(offset).Limit(pageSize).Find(&tenants).Error
	if err != nil {
		return nil, 0, err
	}

	return tenants, total, nil
}

func (r *TenantRepoImpl) ListByCreditStatus(ctx context.Context, creditStatus string, page, pageSize int) ([]entity.Tenant, int64, error) {
	var tenants []entity.Tenant
	var total int64

	offset := (page - 1) * pageSize
	query := r.db.WithContext(ctx).Model(&entity.Tenant{}).Where("credit_status = ?", creditStatus)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset(offset).Limit(pageSize).Find(&tenants).Error
	if err != nil {
		return nil, 0, err
	}

	return tenants, total, nil
}

func (r *TenantRepoImpl) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return r.db.WithContext(ctx).Model(&entity.Tenant{}).
		Where("id = ?", id).
		Update("balance", gorm.Expr("balance + ?", amount)).Error
}

func (r *TenantRepoImpl) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	var tenant entity.Tenant
	err := r.db.WithContext(ctx).Select("balance").Where("id = ?", id).First(&tenant).Error
	if err != nil {
		return decimal.Zero, err
	}
	return tenant.Balance, nil
}

func (r *TenantRepoImpl) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	result := r.db.WithContext(ctx).Model(&entity.Tenant{}).
		Where("id = ? AND balance >= ?", id, amount).
		Update("balance", gorm.Expr("balance - ?", amount))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("insufficient balance or tenant not found")
	}
	return nil
}

// IncrementBudgetUsed increments the monthly budget used for a tenant
func (r *TenantRepoImpl) IncrementBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return r.db.WithContext(ctx).Model(&entity.Tenant{}).
		Where("id = ?", id).
		Update("budget_used_month", gorm.Expr("budget_used_month + ?", amount)).Error
}

// ResetBudgetUsed resets the monthly budget used for a tenant
func (r *TenantRepoImpl) ResetBudgetUsed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.Tenant{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"budget_used_month": decimal.Zero,
			"tokens_used_month": 0,
		}).Error
}

// GetBudgetUsage returns budget usage for a tenant
func (r *TenantRepoImpl) GetBudgetUsage(ctx context.Context, id uuid.UUID) (monthlyUsed decimal.Decimal, tokensUsed int64, err error) {
	var tenant entity.Tenant
	err = r.db.WithContext(ctx).Select("budget_used_month, tokens_used_month").Where("id = ?", id).First(&tenant).Error
	return tenant.BudgetUsedMonth, tenant.TokensUsedMonth, err
}

// IncrementTokensUsed increments the monthly tokens used for a tenant
func (r *TenantRepoImpl) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
	return r.db.WithContext(ctx).Model(&entity.Tenant{}).
		Where("id = ?", id).
		Update("tokens_used_month", gorm.Expr("tokens_used_month + ?", tokens)).Error
}

// ResetTokensUsed resets the monthly tokens used for a tenant
func (r *TenantRepoImpl) ResetTokensUsed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.Tenant{}).
		Where("id = ?", id).
		Update("tokens_used_month", 0).Error
}

type UserRepoImpl struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepoImpl {
	return &UserRepoImpl{db: db}
}

func (r *UserRepoImpl) Create(ctx context.Context, user *entity.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepoImpl) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepoImpl) GetByVerificationToken(ctx context.Context, token string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("email_verification_token = ?", token).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepoImpl) Update(ctx context.Context, user *entity.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *UserRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.User{}, id).Error
}

func (r *UserRepoImpl) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.User, int64, error) {
	var users []entity.User
	var total int64

	offset := (page - 1) * pageSize
	query := r.db.WithContext(ctx).Model(&entity.User{}).Where("tenant_id = ?", tenantID)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Offset(offset).Limit(pageSize).Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *UserRepoImpl) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).
		Where("id = ?", id).
		Update("last_login_at", time.Now()).Error
}

// IncrementMonthlyBudgetUsed increments the monthly budget used for a user
func (r *UserRepoImpl) IncrementMonthlyBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).
		Where("id = ?", id).
		Update("budget_used_month", gorm.Expr("budget_used_month + ?", amount)).Error
}

// IncrementDailyBudgetUsed increments the daily budget used for a user
func (r *UserRepoImpl) IncrementDailyBudgetUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).
		Where("id = ?", id).
		Update("budget_used_today", gorm.Expr("budget_used_today + ?", amount)).Error
}

// ResetMonthlyBudgetUsed resets the monthly budget used for a user
func (r *UserRepoImpl) ResetMonthlyBudgetUsed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"budget_used_month": decimal.Zero,
			"tokens_used_month": 0,
		}).Error
}

// ResetDailyBudgetUsed resets the daily budget used for a user
func (r *UserRepoImpl) ResetDailyBudgetUsed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"budget_used_today": decimal.Zero,
		}).Error
}

// GetBudgetUsage returns budget usage for a user
func (r *UserRepoImpl) GetBudgetUsage(ctx context.Context, id uuid.UUID) (monthlyUsed decimal.Decimal, dailyUsed decimal.Decimal, tokensUsed int64, err error) {
	var user entity.User
	err = r.db.WithContext(ctx).Select("budget_used_month, budget_used_today, tokens_used_month").Where("id = ?", id).First(&user).Error
	return user.BudgetUsedMonth, user.BudgetUsedToday, user.TokensUsedMonth, err
}

// IncrementTokensUsed increments the monthly tokens used for a user
func (r *UserRepoImpl) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).
		Where("id = ?", id).
		Update("tokens_used_month", gorm.Expr("tokens_used_month + ?", tokens)).Error
}

// IncrementActiveAPIKeys increments the active API keys count for a user
func (r *UserRepoImpl) IncrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).
		Where("id = ?", id).
		Update("active_api_keys", gorm.Expr("active_api_keys + 1")).Error
}

// DecrementActiveAPIKeys decrements the active API keys count for a user
func (r *UserRepoImpl) DecrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).
		Where("id = ?", id).
		Update("active_api_keys", gorm.Expr("active_api_keys - 1")).Error
}

func (r *UserRepoImpl) UpdateBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).
		Where("id = ?", id).
		Update("balance", gorm.Expr("balance + ?", amount)).Error
}

func (r *UserRepoImpl) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Select("balance").Where("id = ?", id).First(&user).Error
	if err != nil {
		return decimal.Zero, err
	}
	return user.Balance, nil
}

func (r *UserRepoImpl) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	result := r.db.WithContext(ctx).Model(&entity.User{}).
		Where("id = ? AND balance >= ?", id, amount).
		Update("balance", gorm.Expr("balance - ?", amount))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("insufficient balance or user not found")
	}
	return nil
}

type APIKeyRepoImpl struct {
	db *gorm.DB
}

func NewAPIKeyRepository(db *gorm.DB) *APIKeyRepoImpl {
	return &APIKeyRepoImpl{db: db}
}

func (r *APIKeyRepoImpl) Create(ctx context.Context, apiKey *entity.APIKey) error {
	return r.db.WithContext(ctx).Create(apiKey).Error
}

func (r *APIKeyRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.APIKey, error) {
	var apiKey entity.APIKey
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&apiKey).Error
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

func (r *APIKeyRepoImpl) GetByHash(ctx context.Context, keyHash string) (*entity.APIKey, error) {
	var apiKey entity.APIKey
	err := r.db.WithContext(ctx).Where("key_hash = ?", keyHash).First(&apiKey).Error
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

func (r *APIKeyRepoImpl) GetWithRelations(ctx context.Context, keyHash string) (*entity.APIKey, *entity.User, *entity.Tenant, error) {
	var apiKey entity.APIKey
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("User.Tenant").
		Preload("Tenant").
		Where("key_hash = ?", keyHash).
		First(&apiKey).Error
	if err != nil {
		return nil, nil, nil, err
	}

	var user *entity.User
	var tenant *entity.Tenant

	if apiKey.User != nil {
		user = apiKey.User
		if user.Tenant != nil {
			tenant = user.Tenant
		}
	}

	if tenant == nil && apiKey.Tenant != nil {
		tenant = apiKey.Tenant
	}

	if user == nil {
		var u entity.User
		err = r.db.WithContext(ctx).Where("id = ?", apiKey.UserID).First(&u).Error
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to get user: %w", err)
		}
		user = &u
	}

	if tenant == nil {
		var t entity.Tenant
		err = r.db.WithContext(ctx).Where("id = ?", apiKey.TenantID).First(&t).Error
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to get tenant: %w", err)
		}
		tenant = &t
	}

	return &apiKey, user, tenant, nil
}

func (r *APIKeyRepoImpl) GetByKeyPrefix(ctx context.Context, prefix string) ([]entity.APIKey, error) {
	var apiKeys []entity.APIKey
	err := r.db.WithContext(ctx).Where("key_prefix = ?", prefix).Find(&apiKeys).Error
	if err != nil {
		return nil, err
	}
	return apiKeys, nil
}

func (r *APIKeyRepoImpl) Update(ctx context.Context, apiKey *entity.APIKey) error {
	return r.db.WithContext(ctx).Save(apiKey).Error
}

func (r *APIKeyRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.APIKey{}, id).Error
}

func (r *APIKeyRepoImpl) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&entity.APIKey{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     "revoked",
			"revoked_at": now,
		}).Error
}

func (r *APIKeyRepoImpl) List(ctx context.Context, userID uuid.UUID) ([]entity.APIKey, error) {
	var apiKeys []entity.APIKey
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&apiKeys).Error
	if err != nil {
		return nil, err
	}
	return apiKeys, nil
}

func (r *APIKeyRepoImpl) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.APIKey{}).
		Where("id = ?", id).
		Update("last_used_at", time.Now()).Error
}

func (r *APIKeyRepoImpl) UpdateTokenUsage(ctx context.Context, id uuid.UUID, tokens int64) error {
	return r.db.WithContext(ctx).Model(&entity.APIKey{}).
		Where("id = ?", id).
		Update("used_tokens_this_month", gorm.Expr("used_tokens_this_month + ?", tokens)).Error
}

// IncrementMonthlyCostUsed increments the monthly cost used for an API key
func (r *APIKeyRepoImpl) IncrementMonthlyCostUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return r.db.WithContext(ctx).Model(&entity.APIKey{}).
		Where("id = ?", id).
		Update("monthly_cost_used", gorm.Expr("monthly_cost_used + ?", amount)).Error
}

// IncrementDailyCostUsed increments the daily cost used for an API key
func (r *APIKeyRepoImpl) IncrementDailyCostUsed(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
	return r.db.WithContext(ctx).Model(&entity.APIKey{}).
		Where("id = ?", id).
		Update("daily_cost_used", gorm.Expr("daily_cost_used + ?", amount)).Error
}

// ResetMonthlyCostUsed resets the monthly cost used for an API key
func (r *APIKeyRepoImpl) ResetMonthlyCostUsed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.APIKey{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"monthly_cost_used":     decimal.Zero,
			"used_tokens_this_month": 0,
		}).Error
}

// ResetDailyCostUsed resets the daily cost used for an API key
func (r *APIKeyRepoImpl) ResetDailyCostUsed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.APIKey{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"daily_cost_used":  decimal.Zero,
			"tokens_used_today": 0,
		}).Error
}

// GetCostUsage returns cost usage for an API key
func (r *APIKeyRepoImpl) GetCostUsage(ctx context.Context, id uuid.UUID) (monthlyUsed decimal.Decimal, dailyUsed decimal.Decimal, tokensMonth int64, tokensToday int64, err error) {
	var apiKey entity.APIKey
	err = r.db.WithContext(ctx).Select("monthly_cost_used, daily_cost_used, used_tokens_this_month, tokens_used_today").Where("id = ?", id).First(&apiKey).Error
	return apiKey.MonthlyCostUsed, apiKey.DailyCostUsed, apiKey.UsedTokensThisMonth, apiKey.TokensUsedToday, err
}

// IncrementDailyTokens increments the daily tokens used for an API key
func (r *APIKeyRepoImpl) IncrementDailyTokens(ctx context.Context, id uuid.UUID, tokens int64) error {
	return r.db.WithContext(ctx).Model(&entity.APIKey{}).
		Where("id = ?", id).
		Update("tokens_used_today", gorm.Expr("tokens_used_today + ?", tokens)).Error
}

// ResetDailyTokens resets the daily tokens used for an API key
func (r *APIKeyRepoImpl) ResetDailyTokens(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.APIKey{}).
		Where("id = ?", id).
		Update("tokens_used_today", 0).Error
}

// UpdateProviderUsage updates per-provider token and cost usage for an API key.
// Also updates the aggregated total counters.
func (r *APIKeyRepoImpl) UpdateProviderUsage(ctx context.Context, id uuid.UUID, provider string, tokens int64, cost decimal.Decimal) error {
	var apiKey entity.APIKey
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&apiKey).Error; err != nil {
		return err
	}

	if apiKey.ProviderUsage == nil {
		usage := make(entity.ProviderUsage)
		apiKey.ProviderUsage = &usage
	}
	stats := apiKey.ProviderUsage.EnsureProvider(provider)
	stats.UsedTokensThisMonth += tokens
	stats.TokensUsedToday += tokens
	stats.MonthlyCostUsed = stats.MonthlyCostUsed.Add(cost)
	stats.DailyCostUsed = stats.DailyCostUsed.Add(cost)

	return r.db.WithContext(ctx).Model(&apiKey).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"provider_usage":        apiKey.ProviderUsage,
			"used_tokens_this_month": gorm.Expr("used_tokens_this_month + ?", tokens),
			"tokens_used_today":      gorm.Expr("tokens_used_today + ?", tokens),
			"monthly_cost_used":      gorm.Expr("monthly_cost_used + ?", cost),
			"daily_cost_used":        gorm.Expr("daily_cost_used + ?", cost),
		}).Error
}

func (r *APIKeyRepoImpl) ListByTenant(ctx context.Context, tenantID uuid.UUID, status string) ([]entity.APIKey, error) {
	var apiKeys []entity.APIKey
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&apiKeys).Error
	if err != nil {
		return nil, err
	}
	return apiKeys, nil
}

func (r *APIKeyRepoImpl) ListAll(ctx context.Context) ([]entity.APIKey, error) {
	var apiKeys []entity.APIKey
	err := r.db.WithContext(ctx).Find(&apiKeys).Error
	if err != nil {
		return nil, err
	}
	return apiKeys, nil
}

type ModelRepoImpl struct {
	db *gorm.DB
}

func NewModelRepository(db *gorm.DB) *ModelRepoImpl {
	return &ModelRepoImpl{db: db}
}

func (r *ModelRepoImpl) Create(ctx context.Context, model *entity.Model) error {
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *ModelRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.Model, error) {
	var model entity.Model
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		return nil, err
	}
	return &model, nil
}

func (r *ModelRepoImpl) GetByProviderModel(ctx context.Context, provider, modelID string) (*entity.Model, error) {
	var model entity.Model
	err := r.db.WithContext(ctx).
		Where("provider = ? AND model_id = ?", provider, modelID).
		First(&model).Error
	if err != nil {
		return nil, err
	}
	return &model, nil
}

func (r *ModelRepoImpl) Update(ctx context.Context, model *entity.Model) error {
	return r.db.WithContext(ctx).Save(model).Error
}

func (r *ModelRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.Model{}, id).Error
}

func (r *ModelRepoImpl) List(ctx context.Context, provider string) ([]entity.Model, error) {
	var models []entity.Model
	query := r.db.WithContext(ctx)
	if provider != "" {
		query = query.Where("provider = ?", provider)
	}
	err := query.Find(&models).Error
	if err != nil {
		return nil, err
	}
	return models, nil
}

func (r *ModelRepoImpl) ListActive(ctx context.Context) ([]entity.Model, error) {
	var models []entity.Model
	err := r.db.WithContext(ctx).Where("status = ?", "active").Find(&models).Error
	if err != nil {
		return nil, err
	}
	return models, nil
}

func (r *ModelRepoImpl) GetPricing(ctx context.Context, provider, modelID string) (*entity.Model, error) {
	return r.GetByProviderModel(ctx, provider, modelID)
}

type UsageRepoImpl struct {
	db *gorm.DB
}

func NewUsageRepository(db *gorm.DB) *UsageRepoImpl {
	return &UsageRepoImpl{db: db}
}

func (r *UsageRepoImpl) Create(ctx context.Context, record *entity.UsageRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// CreateBatch creates multiple usage records in a single transaction (high-throughput optimization)
func (r *UsageRepoImpl) CreateBatch(ctx context.Context, records []*entity.UsageRecord) error {
	if len(records) == 0 {
		return nil
	}

	// Use GORM batch insert with chunking for large datasets
	batchSize := 100
	return r.db.WithContext(ctx).CreateInBatches(records, batchSize).Error
}

func (r *UsageRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.UsageRecord, error) {
	var record entity.UsageRecord
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *UsageRepoImpl) GetByRequestID(ctx context.Context, requestID string) (*entity.UsageRecord, error) {
	var record entity.UsageRecord
	err := r.db.WithContext(ctx).Where("request_id = ?", requestID).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *UsageRepoImpl) List(ctx context.Context, tenantID uuid.UUID, start, end time.Time, page, pageSize int) ([]entity.UsageRecord, int64, error) {
	var records []entity.UsageRecord
	var total int64

	offset := (page - 1) * pageSize
	query := r.db.WithContext(ctx).Model(&entity.UsageRecord{}).
		Where("tenant_id = ? AND created_at >= ? AND created_at <= ?", tenantID, start, end)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error
	if err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

func (r *UsageRepoImpl) ListByUser(ctx context.Context, userID uuid.UUID, start, end time.Time, page, pageSize int) ([]entity.UsageRecord, int64, error) {
	var records []entity.UsageRecord
	var total int64

	offset := (page - 1) * pageSize
	query := r.db.WithContext(ctx).Model(&entity.UsageRecord{}).
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, start, end)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error
	if err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

func (r *UsageRepoImpl) GetTotalCost(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (decimal.Decimal, int64, error) {
	var result struct {
		TotalCost   decimal.Decimal
		TotalTokens int64
	}

	err := r.db.WithContext(ctx).Model(&entity.UsageRecord{}).
		Select("SUM(cost) as total_cost, SUM(total_tokens) as total_tokens").
		Where("tenant_id = ? AND created_at >= ? AND created_at <= ?", tenantID, start, end).
		Scan(&result).Error
	if err != nil {
		return decimal.Zero, 0, err
	}

	return result.TotalCost, result.TotalTokens, nil
}

type BillRepoImpl struct {
	db *gorm.DB
}

func NewBillRepository(db *gorm.DB) *BillRepoImpl {
	return &BillRepoImpl{db: db}
}

func (r *BillRepoImpl) Create(ctx context.Context, bill *entity.Bill) error {
	return r.db.WithContext(ctx).Create(bill).Error
}

func (r *BillRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.Bill, error) {
	var bill entity.Bill
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&bill).Error
	if err != nil {
		return nil, err
	}
	return &bill, nil
}

func (r *BillRepoImpl) GetByBillNumber(ctx context.Context, billNumber string) (*entity.Bill, error) {
	var bill entity.Bill
	err := r.db.WithContext(ctx).Where("bill_number = ?", billNumber).First(&bill).Error
	if err != nil {
		return nil, err
	}
	return &bill, nil
}

func (r *BillRepoImpl) Update(ctx context.Context, bill *entity.Bill) error {
	return r.db.WithContext(ctx).Save(bill).Error
}

func (r *BillRepoImpl) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.Bill, int64, error) {
	var bills []entity.Bill
	var total int64

	offset := (page - 1) * pageSize
	query := r.db.WithContext(ctx).Model(&entity.Bill{}).Where("tenant_id = ?", tenantID)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&bills).Error
	if err != nil {
		return nil, 0, err
	}

	return bills, total, nil
}

func (r *BillRepoImpl) GetByPeriod(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (*entity.Bill, error) {
	var bill entity.Bill
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND period_start = ? AND period_end = ?", tenantID, start, end).
		First(&bill).Error
	if err != nil {
		return nil, err
	}
	return &bill, nil
}

func (r *BillRepoImpl) MarkPaid(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.Bill{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":  "paid",
			"paid_at": time.Now(),
		}).Error
}

func (r *BillRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.Bill{}, id).Error
}

func (r *BillRepoImpl) ListByStatus(ctx context.Context, status string, page, pageSize int) ([]entity.Bill, int64, error) {
	var bills []entity.Bill
	var total int64

	offset := (page - 1) * pageSize
	query := r.db.WithContext(ctx).Model(&entity.Bill{}).Where("status = ?", status)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&bills).Error
	if err != nil {
		return nil, 0, err
	}

	return bills, total, nil
}

func (r *BillRepoImpl) ListByType(ctx context.Context, billType string, page, pageSize int) ([]entity.Bill, int64, error) {
	var bills []entity.Bill
	var total int64

	offset := (page - 1) * pageSize
	query := r.db.WithContext(ctx).Model(&entity.Bill{}).Where("type = ?", billType)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&bills).Error
	if err != nil {
		return nil, 0, err
	}

	return bills, total, nil
}

func (r *BillRepoImpl) MarkPartialPaid(ctx context.Context, id uuid.UUID, remainingAmount decimal.Decimal) error {
	return r.db.WithContext(ctx).Model(&entity.Bill{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":    "partial_paid",
			"total_cost": remainingAmount,
		}).Error
}

type RechargeRepoImpl struct {
	db *gorm.DB
}

func NewRechargeRepository(db *gorm.DB) *RechargeRepoImpl {
	return &RechargeRepoImpl{db: db}
}

func (r *RechargeRepoImpl) Create(ctx context.Context, record *entity.RechargeRecord) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *RechargeRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.RechargeRecord, error) {
	var record entity.RechargeRecord
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *RechargeRepoImpl) Update(ctx context.Context, record *entity.RechargeRecord) error {
	return r.db.WithContext(ctx).Save(record).Error
}

func (r *RechargeRepoImpl) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.RechargeRecord, int64, error) {
	var records []entity.RechargeRecord
	var total int64

	offset := (page - 1) * pageSize
	query := r.db.WithContext(ctx).Model(&entity.RechargeRecord{}).Where("tenant_id = ?", tenantID)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error
	if err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

func (r *RechargeRepoImpl) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.RechargeRecord{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      "completed",
			"completed_at": time.Now(),
		}).Error
}

type AuditLogRepoImpl struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) *AuditLogRepoImpl {
	return &AuditLogRepoImpl{db: db}
}

func (r *AuditLogRepoImpl) Create(ctx context.Context, log *entity.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *AuditLogRepoImpl) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.AuditLog, int64, error) {
	var logs []entity.AuditLog
	var total int64

	offset := (page - 1) * pageSize
	query := r.db.WithContext(ctx).Model(&entity.AuditLog{}).Where("tenant_id = ?", tenantID)

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// ProviderAccountRepoImpl 实现 Provider 多账户管理
type ProviderAccountRepoImpl struct {
	db *gorm.DB
}

func NewProviderAccountRepository(db *gorm.DB) *ProviderAccountRepoImpl {
	return &ProviderAccountRepoImpl{db: db}
}

func (r *ProviderAccountRepoImpl) Create(ctx context.Context, account *entity.ProviderAccount) error {
	return r.db.WithContext(ctx).Create(account).Error
}

func (r *ProviderAccountRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.ProviderAccount, error) {
	var account entity.ProviderAccount
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *ProviderAccountRepoImpl) GetByProvider(ctx context.Context, provider string) ([]entity.ProviderAccount, error) {
	var accounts []entity.ProviderAccount
	err := r.db.WithContext(ctx).Where("provider = ?", provider).Order("priority ASC").Find(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *ProviderAccountRepoImpl) GetActiveByProvider(ctx context.Context, provider string) ([]entity.ProviderAccount, error) {
	var accounts []entity.ProviderAccount
	err := r.db.WithContext(ctx).
		Where("provider = ? AND status IN ?", provider, []string{"active", "limited"}).
		Order("priority ASC").
		Find(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *ProviderAccountRepoImpl) GetDefaultByProvider(ctx context.Context, provider string) (*entity.ProviderAccount, error) {
	var account entity.ProviderAccount
	err := r.db.WithContext(ctx).
		Where("provider = ? AND is_default = ? AND status = ?", provider, true, "active").
		First(&account).Error
	if err != nil {
		// 如果没有默认账户，返回第一个活跃账户
		err = r.db.WithContext(ctx).
			Where("provider = ? AND status = ?", provider, "active").
			Order("priority ASC").
			First(&account).Error
		if err != nil {
			return nil, err
		}
	}
	return &account, nil
}

func (r *ProviderAccountRepoImpl) GetNextAvailable(ctx context.Context, provider string, excludeID uuid.UUID) (*entity.ProviderAccount, error) {
	var account entity.ProviderAccount
	err := r.db.WithContext(ctx).
		Where("provider = ? AND status = ? AND id != ?", provider, "active", excludeID).
		Order("priority ASC").
		First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *ProviderAccountRepoImpl) Update(ctx context.Context, account *entity.ProviderAccount) error {
	return r.db.WithContext(ctx).Save(account).Error
}

func (r *ProviderAccountRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entity.ProviderAccount{}, id).Error
}

func (r *ProviderAccountRepoImpl) List(ctx context.Context, page, pageSize int) ([]entity.ProviderAccount, int64, error) {
	var accounts []entity.ProviderAccount
	var total int64

	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).Model(&entity.ProviderAccount{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = r.db.WithContext(ctx).Order("provider ASC, priority ASC").Offset(offset).Limit(pageSize).Find(&accounts).Error
	if err != nil {
		return nil, 0, err
	}

	return accounts, total, nil
}

func (r *ProviderAccountRepoImpl) ListByStatus(ctx context.Context, status string) ([]entity.ProviderAccount, error) {
	var accounts []entity.ProviderAccount
	query := r.db.WithContext(ctx)
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	err := query.Order("provider ASC, priority ASC").Find(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *ProviderAccountRepoImpl) SetDefault(ctx context.Context, provider string, id uuid.UUID) error {
	// 先清除该 provider 的所有默认标记
	err := r.db.WithContext(ctx).Model(&entity.ProviderAccount{}).
		Where("provider = ?", provider).
		Update("is_default", false).Error
	if err != nil {
		return err
	}

	// 设置指定账户为默认
	return r.db.WithContext(ctx).Model(&entity.ProviderAccount{}).
		Where("id = ?", id).
		Update("is_default", true).Error
}

func (r *ProviderAccountRepoImpl) IncrementUsage(ctx context.Context, id uuid.UUID, cost decimal.Decimal) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&entity.ProviderAccount{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"used_this_month": gorm.Expr("used_this_month + ?", cost),
			"request_count":   gorm.Expr("request_count + 1"),
			"success_count":   gorm.Expr("success_count + 1"),
			"error_count":     0, // 成功后清零错误计数
			"total_requests":  gorm.Expr("total_requests + 1"),
			"total_success":   gorm.Expr("total_success + 1"),
			"total_cost":      gorm.Expr("total_cost + ?", cost),
			"last_used_at":    now,
			"updated_at":      now,
		}).Error
}

func (r *ProviderAccountRepoImpl) RecordError(ctx context.Context, id uuid.UUID, errMsg string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&entity.ProviderAccount{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"error_count":    gorm.Expr("error_count + 1"),
			"total_errors":   gorm.Expr("total_errors + 1"),
			"last_error":     errMsg,
			"last_error_at":  now,
			"updated_at":     now,
		}).Error
}

func (r *ProviderAccountRepoImpl) RecordSuccess(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&entity.ProviderAccount{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"error_count":  0, // 成功后清零错误计数
			"last_used_at": now,
			"updated_at":   now,
		}).Error
}

func (r *ProviderAccountRepoImpl) ResetMonthlyUsage(ctx context.Context) error {
	return r.db.WithContext(ctx).Model(&entity.ProviderAccount{}).
		Where("status IN ?", []string{"active", "limited", "exhausted"}).
		Updates(map[string]interface{}{
			"used_this_month": decimal.Zero,
			"request_count":   0,
			"success_count":   0,
			"error_count":     0,
			"status":          "active",
			"updated_at":      time.Now(),
		}).Error
}

func (r *ProviderAccountRepoImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if status == "disabled" {
		updates["disabled_at"] = time.Now()
	} else if status == "active" {
		updates["reactivated_at"] = time.Now()
		updates["error_count"] = 0
	}

	return r.db.WithContext(ctx).Model(&entity.ProviderAccount{}).
		Where("id = ?", id).
		Updates(updates).Error
}

	// Dedicated account queries
	func (r *ProviderAccountRepoImpl) GetDedicatedByTenant(ctx context.Context, tenantID uuid.UUID, provider string) (*entity.ProviderAccount, error) {
		var account entity.ProviderAccount
		err := r.db.WithContext(ctx).
			Where("tenant_id = ? AND provider = ? AND status = ?", tenantID, provider, "active").
			Order("priority ASC").
			First(&account).Error
		if err != nil {
			return nil, err
		}
		return &account, nil
	}

	func (r *ProviderAccountRepoImpl) GetDedicatedByUser(ctx context.Context, userID uuid.UUID, provider string) (*entity.ProviderAccount, error) {
		var account entity.ProviderAccount
		err := r.db.WithContext(ctx).
			Where("user_id = ? AND provider = ? AND status = ?", userID, provider, "active").
			Order("priority ASC").
			First(&account).Error
		if err != nil {
			return nil, err
		}
		return &account, nil
	}

	func (r *ProviderAccountRepoImpl) ListDedicatedByTenant(ctx context.Context, tenantID uuid.UUID) ([]entity.ProviderAccount, error) {
		var accounts []entity.ProviderAccount
		err := r.db.WithContext(ctx).
			Where("tenant_id = ? AND status = ?", tenantID, "active").
			Order("provider ASC, priority ASC").
			Find(&accounts).Error
		return accounts, err
	}

	func (r *ProviderAccountRepoImpl) ListDedicatedByUser(ctx context.Context, userID uuid.UUID) ([]entity.ProviderAccount, error) {
		var accounts []entity.ProviderAccount
		err := r.db.WithContext(ctx).
			Where("user_id = ? AND status = ?", userID, "active").
			Order("provider ASC, priority ASC").
			Find(&accounts).Error
		return accounts, err
	}

	func (r *ProviderAccountRepoImpl) ListPublicByProvider(ctx context.Context, provider string) ([]entity.ProviderAccount, error) {
		var accounts []entity.ProviderAccount
		err := r.db.WithContext(ctx).
			Where("tenant_id IS NULL AND user_id IS NULL AND provider = ? AND status = ?", provider, "active").
			Order("priority ASC").
			Find(&accounts).Error
		return accounts, err
	}

	func (r *ProviderAccountRepoImpl) UpdateUseDedicatedTenant(ctx context.Context, tenantID uuid.UUID, enabled bool) error {
		return r.db.WithContext(ctx).Model(&entity.Tenant{}).
			Where("id = ?", tenantID).
			Update("use_dedicated_provider", enabled).Error
	}

	func (r *ProviderAccountRepoImpl) UpdateUseDedicatedUser(ctx context.Context, userID uuid.UUID, enabled bool) error {
		return r.db.WithContext(ctx).Model(&entity.User{}).
			Where("id = ?", userID).
			Update("use_dedicated_provider", enabled).Error
	}

	// ==================== Platform Admin Repository ====================

	// PlatformAdminRepoImpl implements PlatformAdminRepository
	type PlatformAdminRepoImpl struct {
		db *gorm.DB
	}

	// NewPlatformAdminRepository creates a new platform admin repository
	func NewPlatformAdminRepository(db *gorm.DB) *PlatformAdminRepoImpl {
		return &PlatformAdminRepoImpl{db: db}
	}

	func (r *PlatformAdminRepoImpl) Create(ctx context.Context, admin *entity.PlatformAdmin) error {
		return r.db.WithContext(ctx).Create(admin).Error
	}

	func (r *PlatformAdminRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.PlatformAdmin, error) {
		var admin entity.PlatformAdmin
		err := r.db.WithContext(ctx).Where("id = ?", id).First(&admin).Error
		if err != nil {
			return nil, err
		}
		return &admin, nil
	}

	func (r *PlatformAdminRepoImpl) GetByEmail(ctx context.Context, email string) (*entity.PlatformAdmin, error) {
		var admin entity.PlatformAdmin
		err := r.db.WithContext(ctx).Where("email = ?", email).First(&admin).Error
		if err != nil {
			return nil, err
		}
		return &admin, nil
	}

	func (r *PlatformAdminRepoImpl) Update(ctx context.Context, admin *entity.PlatformAdmin) error {
		return r.db.WithContext(ctx).Save(admin).Error
	}

	func (r *PlatformAdminRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Delete(&entity.PlatformAdmin{}, id).Error
	}

	func (r *PlatformAdminRepoImpl) List(ctx context.Context, page, pageSize int) ([]entity.PlatformAdmin, int64, error) {
		var admins []entity.PlatformAdmin
		var total int64

		offset := (page - 1) * pageSize
		err := r.db.WithContext(ctx).Model(&entity.PlatformAdmin{}).Count(&total).Error
		if err != nil {
			return nil, 0, err
		}

		err = r.db.WithContext(ctx).Offset(offset).Limit(pageSize).Find(&admins).Error
		if err != nil {
			return nil, 0, err
		}

		return admins, total, nil
	}

	func (r *PlatformAdminRepoImpl) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Model(&entity.PlatformAdmin{}).
			Where("id = ?", id).
			Update("last_login_at", time.Now()).Error
	}

	func (r *PlatformAdminRepoImpl) CheckPermission(ctx context.Context, id uuid.UUID, permission string) (bool, error) {
		var admin entity.PlatformAdmin
		err := r.db.WithContext(ctx).Select("permissions").Where("id = ?", id).First(&admin).Error
		if err != nil {
			return false, err
		}

		// Parse permissions JSONB
		var permissions []string
		if admin.Permissions != "" {
			if err := json.Unmarshal([]byte(admin.Permissions), &permissions); err != nil {
				return false, err
			}
		}

		for _, p := range permissions {
			if p == permission {
				return true, nil
			}
		}
		return false, nil
	}

	func (r *PlatformAdminRepoImpl) GetPermissions(ctx context.Context, id uuid.UUID) ([]string, error) {
		var admin entity.PlatformAdmin
		err := r.db.WithContext(ctx).Select("permissions").Where("id = ?", id).First(&admin).Error
		if err != nil {
			return nil, err
		}

		var permissions []string
		if admin.Permissions != "" {
			if err := json.Unmarshal([]byte(admin.Permissions), &permissions); err != nil {
				return nil, err
			}
		}
		return permissions, nil
	}

	// ==================== Tenant Application Repository ====================

	// TenantApplicationRepoImpl implements TenantApplicationRepository
	type TenantApplicationRepoImpl struct {
		db *gorm.DB
	}

	// NewTenantApplicationRepository creates a new tenant application repository
	func NewTenantApplicationRepository(db *gorm.DB) *TenantApplicationRepoImpl {
		return &TenantApplicationRepoImpl{db: db}
	}

	func (r *TenantApplicationRepoImpl) Create(ctx context.Context, app *entity.TenantApplication) error {
		return r.db.WithContext(ctx).Create(app).Error
	}

	func (r *TenantApplicationRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.TenantApplication, error) {
		var app entity.TenantApplication
		err := r.db.WithContext(ctx).Where("id = ?", id).First(&app).Error
		if err != nil {
			return nil, err
		}
		return &app, nil
	}

	func (r *TenantApplicationRepoImpl) GetBySlug(ctx context.Context, slug string) (*entity.TenantApplication, error) {
		var app entity.TenantApplication
		err := r.db.WithContext(ctx).Where("company_slug = ?", slug).First(&app).Error
		if err != nil {
			return nil, err
		}
		return &app, nil
	}

	func (r *TenantApplicationRepoImpl) GetByEmail(ctx context.Context, email string) (*entity.TenantApplication, error) {
		var app entity.TenantApplication
		err := r.db.WithContext(ctx).Where("contact_email = ?", email).First(&app).Error
		if err != nil {
			return nil, err
		}
		return &app, nil
	}

	func (r *TenantApplicationRepoImpl) Update(ctx context.Context, app *entity.TenantApplication) error {
		return r.db.WithContext(ctx).Save(app).Error
	}

	func (r *TenantApplicationRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Delete(&entity.TenantApplication{}, id).Error
	}

	func (r *TenantApplicationRepoImpl) List(ctx context.Context, status string, page, pageSize int) ([]entity.TenantApplication, int64, error) {
		var apps []entity.TenantApplication
		var total int64

		offset := (page - 1) * pageSize
		query := r.db.WithContext(ctx).Model(&entity.TenantApplication{})
		if status != "" && status != "all" {
			query = query.Where("status = ?", status)
		}

		err := query.Count(&total).Error
		if err != nil {
			return nil, 0, err
		}

		err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&apps).Error
		if err != nil {
			return nil, 0, err
		}

		return apps, total, nil
	}

	func (r *TenantApplicationRepoImpl) ListByStatus(ctx context.Context, status string) ([]entity.TenantApplication, error) {
		var apps []entity.TenantApplication
		query := r.db.WithContext(ctx)
		if status != "" && status != "all" {
			query = query.Where("status = ?", status)
		}
		err := query.Order("created_at DESC").Find(&apps).Error
		return apps, err
	}

	func (r *TenantApplicationRepoImpl) SetStatus(ctx context.Context, id uuid.UUID, status string) error {
		return r.db.WithContext(ctx).Model(&entity.TenantApplication{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":     status,
				"updated_at": time.Now(),
			}).Error
	}

	func (r *TenantApplicationRepoImpl) Approve(ctx context.Context, id uuid.UUID, reviewerID uuid.UUID, notes string) error {
		now := time.Now()
		return r.db.WithContext(ctx).Model(&entity.TenantApplication{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":       "approved",
				"reviewed_by":  reviewerID,
				"reviewed_at":  now,
				"review_notes": notes,
				"updated_at":   now,
			}).Error
	}

	func (r *TenantApplicationRepoImpl) Reject(ctx context.Context, id uuid.UUID, reviewerID uuid.UUID, reason string) error {
		now := time.Now()
		return r.db.WithContext(ctx).Model(&entity.TenantApplication{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":           "rejected",
				"reviewed_by":      reviewerID,
				"reviewed_at":      now,
				"rejection_reason": reason,
				"updated_at":       now,
			}).Error
	}

	func (r *TenantApplicationRepoImpl) MarkTenantCreated(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
		return r.db.WithContext(ctx).Model(&entity.TenantApplication{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":     "tenant_created",
				"tenant_id":  tenantID,
				"updated_at": time.Now(),
			}).Error
	}

	// ==================== User Application Repository ====================

	// UserApplicationRepoImpl implements UserApplicationRepository
	type UserApplicationRepoImpl struct {
		db *gorm.DB
	}

	// NewUserApplicationRepository creates a new user application repository
	func NewUserApplicationRepository(db *gorm.DB) *UserApplicationRepoImpl {
		return &UserApplicationRepoImpl{db: db}
	}

	func (r *UserApplicationRepoImpl) Create(ctx context.Context, app *entity.UserApplication) error {
		return r.db.WithContext(ctx).Create(app).Error
	}

	func (r *UserApplicationRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.UserApplication, error) {
		var app entity.UserApplication
		err := r.db.WithContext(ctx).Where("id = ?", id).First(&app).Error
		if err != nil {
			return nil, err
		}
		return &app, nil
	}

	func (r *UserApplicationRepoImpl) GetByToken(ctx context.Context, token string) (*entity.UserApplication, error) {
		var app entity.UserApplication
		err := r.db.WithContext(ctx).Where("invite_token = ? AND application_type = ?", token, "invitation").First(&app).Error
		if err != nil {
			return nil, err
		}
		return &app, nil
	}

	func (r *UserApplicationRepoImpl) GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*entity.UserApplication, error) {
		var app entity.UserApplication
		err := r.db.WithContext(ctx).Where("tenant_id = ? AND email = ?", tenantID, email).First(&app).Error
		if err != nil {
			return nil, err
		}
		return &app, nil
	}

	func (r *UserApplicationRepoImpl) Update(ctx context.Context, app *entity.UserApplication) error {
		return r.db.WithContext(ctx).Save(app).Error
	}

	func (r *UserApplicationRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Delete(&entity.UserApplication{}, id).Error
	}

	func (r *UserApplicationRepoImpl) List(ctx context.Context, tenantID uuid.UUID, status string, page, pageSize int) ([]entity.UserApplication, int64, error) {
		var apps []entity.UserApplication
		var total int64

		offset := (page - 1) * pageSize
		query := r.db.WithContext(ctx).Model(&entity.UserApplication{}).Where("tenant_id = ?", tenantID)
		if status != "" && status != "all" {
			query = query.Where("status = ?", status)
		}

		err := query.Count(&total).Error
		if err != nil {
			return nil, 0, err
		}

		err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&apps).Error
		if err != nil {
			return nil, 0, err
		}

		return apps, total, nil
	}

	func (r *UserApplicationRepoImpl) ListByTenant(ctx context.Context, tenantID uuid.UUID, status string) ([]entity.UserApplication, error) {
		var apps []entity.UserApplication
		query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
		if status != "" && status != "all" {
			query = query.Where("status = ?", status)
		}
		err := query.Order("created_at DESC").Find(&apps).Error
		return apps, err
	}

	func (r *UserApplicationRepoImpl) ListAll(ctx context.Context, status string, page, pageSize int) ([]entity.UserApplication, int64, error) {
		var apps []entity.UserApplication
		var total int64

		offset := (page - 1) * pageSize
		query := r.db.WithContext(ctx).Model(&entity.UserApplication{})
		if status != "" && status != "all" {
			query = query.Where("status = ?", status)
		}

		err := query.Count(&total).Error
		if err != nil {
			return nil, 0, err
		}

		err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&apps).Error
		if err != nil {
			return nil, 0, err
		}

		return apps, total, nil
	}

	func (r *UserApplicationRepoImpl) SetStatus(ctx context.Context, id uuid.UUID, status string) error {
		return r.db.WithContext(ctx).Model(&entity.UserApplication{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":     status,
				"updated_at": time.Now(),
			}).Error
	}

	func (r *UserApplicationRepoImpl) Approve(ctx context.Context, id uuid.UUID, reviewerID uuid.UUID) error {
		now := time.Now()
		return r.db.WithContext(ctx).Model(&entity.UserApplication{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":      "approved",
				"reviewed_by": reviewerID,
				"reviewed_at": now,
				"updated_at":  now,
			}).Error
	}

	func (r *UserApplicationRepoImpl) Reject(ctx context.Context, id uuid.UUID, reviewerID uuid.UUID) error {
		now := time.Now()
		return r.db.WithContext(ctx).Model(&entity.UserApplication{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":      "rejected",
				"reviewed_by": reviewerID,
				"reviewed_at": now,
				"updated_at":  now,
			}).Error
	}

	func (r *UserApplicationRepoImpl) MarkUserCreated(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
		return r.db.WithContext(ctx).Model(&entity.UserApplication{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":          "user_created",
				"created_user_id": userID,
				"updated_at":      time.Now(),
			}).Error
	}

	func (r *UserApplicationRepoImpl) Accept(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Model(&entity.UserApplication{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":     "accepted",
				"updated_at": time.Now(),
			}).Error
	}

	func (r *UserApplicationRepoImpl) MarkExpired(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Model(&entity.UserApplication{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":     "expired",
				"updated_at": time.Now(),
			}).Error
	}

	// ==================== Budget Alert Repository ====================

	// BudgetAlertRepoImpl implements BudgetAlertRepository
	type BudgetAlertRepoImpl struct {
		db *gorm.DB
	}

	// NewBudgetAlertRepository creates a new budget alert repository
	func NewBudgetAlertRepository(db *gorm.DB) *BudgetAlertRepoImpl {
		return &BudgetAlertRepoImpl{db: db}
	}

	func (r *BudgetAlertRepoImpl) Create(ctx context.Context, alert *entity.BudgetAlert) error {
		return r.db.WithContext(ctx).Create(alert).Error
	}

	func (r *BudgetAlertRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.BudgetAlert, error) {
		var alert entity.BudgetAlert
		err := r.db.WithContext(ctx).Where("id = ?", id).First(&alert).Error
		if err != nil {
			return nil, err
		}
		return &alert, nil
	}

	func (r *BudgetAlertRepoImpl) GetByScope(ctx context.Context, scope string, scopeID uuid.UUID) ([]entity.BudgetAlert, error) {
		var alerts []entity.BudgetAlert
		err := r.db.WithContext(ctx).
			Where("scope = ? AND scope_id = ?", scope, scopeID).
			Find(&alerts).Error
		return alerts, err
	}

	func (r *BudgetAlertRepoImpl) GetByScopeAndType(ctx context.Context, scope string, scopeID uuid.UUID, alertType string) (*entity.BudgetAlert, error) {
		var alert entity.BudgetAlert
		err := r.db.WithContext(ctx).
			Where("scope = ? AND scope_id = ? AND alert_type = ?", scope, scopeID, alertType).
			First(&alert).Error
		if err != nil {
			return nil, err
		}
		return &alert, nil
	}

	func (r *BudgetAlertRepoImpl) Update(ctx context.Context, alert *entity.BudgetAlert) error {
		return r.db.WithContext(ctx).Save(alert).Error
	}

	func (r *BudgetAlertRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Delete(&entity.BudgetAlert{}, id).Error
	}

	func (r *BudgetAlertRepoImpl) List(ctx context.Context, page, pageSize int) ([]entity.BudgetAlert, int64, error) {
		var alerts []entity.BudgetAlert
		var total int64

		offset := (page - 1) * pageSize
		err := r.db.WithContext(ctx).Model(&entity.BudgetAlert{}).Count(&total).Error
		if err != nil {
			return nil, 0, err
		}

		err = r.db.WithContext(ctx).Offset(offset).Limit(pageSize).Find(&alerts).Error
		if err != nil {
			return nil, 0, err
		}

		return alerts, total, nil
	}

	func (r *BudgetAlertRepoImpl) ListEnabled(ctx context.Context) ([]entity.BudgetAlert, error) {
		var alerts []entity.BudgetAlert
		err := r.db.WithContext(ctx).Where("is_enabled = ?", true).Find(&alerts).Error
		return alerts, err
	}

	func (r *BudgetAlertRepoImpl) Enable(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Model(&entity.BudgetAlert{}).
			Where("id = ?", id).
			Update("is_enabled", true).Error
	}

	func (r *BudgetAlertRepoImpl) Disable(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Model(&entity.BudgetAlert{}).
			Where("id = ?", id).
			Update("is_enabled", false).Error
	}

	func (r *BudgetAlertRepoImpl) MarkTriggered(ctx context.Context, id uuid.UUID) error {
		now := time.Now()
		return r.db.WithContext(ctx).Model(&entity.BudgetAlert{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"last_triggered_at": now,
				"triggered_count":   gorm.Expr("triggered_count + 1"),
			}).Error
	}

	// ==================== Cost Usage Record Repository ====================

	// CostUsageRecordRepoImpl implements CostUsageRecordRepository
	type CostUsageRecordRepoImpl struct {
		db *gorm.DB
	}

	// NewCostUsageRecordRepository creates a new cost usage record repository
	func NewCostUsageRecordRepository(db *gorm.DB) *CostUsageRecordRepoImpl {
		return &CostUsageRecordRepoImpl{db: db}
	}

	func (r *CostUsageRecordRepoImpl) Create(ctx context.Context, record *entity.CostUsageRecord) error {
		return r.db.WithContext(ctx).Create(record).Error
	}

	func (r *CostUsageRecordRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.CostUsageRecord, error) {
		var record entity.CostUsageRecord
		err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error
		if err != nil {
			return nil, err
		}
		return &record, nil
	}

	func (r *CostUsageRecordRepoImpl) GetByScopeAndDate(ctx context.Context, scope string, scopeID uuid.UUID, date time.Time) (*entity.CostUsageRecord, error) {
		var record entity.CostUsageRecord
		err := r.db.WithContext(ctx).
			Where("scope = ? AND scope_id = ? AND date = ?", scope, scopeID, date).
			First(&record).Error
		if err != nil {
			return nil, err
		}
		return &record, nil
	}

	func (r *CostUsageRecordRepoImpl) Update(ctx context.Context, record *entity.CostUsageRecord) error {
		return r.db.WithContext(ctx).Save(record).Error
	}

	func (r *CostUsageRecordRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Delete(&entity.CostUsageRecord{}, id).Error
	}

	func (r *CostUsageRecordRepoImpl) List(ctx context.Context, scope string, scopeID uuid.UUID, startDate, endDate time.Time) ([]entity.CostUsageRecord, error) {
		var records []entity.CostUsageRecord
		err := r.db.WithContext(ctx).
			Where("scope = ? AND scope_id = ? AND date >= ? AND date <= ?", scope, scopeID, startDate, endDate).
			Order("date DESC").
			Find(&records).Error
		return records, err
	}

	func (r *CostUsageRecordRepoImpl) GetDailyTotal(ctx context.Context, scope string, scopeID uuid.UUID, date time.Time) (decimal.Decimal, int64, error) {
		var result struct {
			Cost     decimal.Decimal
			Tokens   int64
			Requests int
		}

		err := r.db.WithContext(ctx).Model(&entity.CostUsageRecord{}).
			Select("SUM(cost) as cost, SUM(tokens) as tokens, SUM(requests) as requests").
			Where("scope = ? AND scope_id = ? AND date = ?", scope, scopeID, date).
			Scan(&result).Error

		return result.Cost, result.Tokens, err
	}

	func (r *CostUsageRecordRepoImpl) GetMonthlyTotal(ctx context.Context, scope string, scopeID uuid.UUID, month time.Time) (decimal.Decimal, int64, error) {
		// Calculate start and end of month
		startOfMonth := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
		endOfMonth := startOfMonth.AddDate(0, 1, -1)

		var result struct {
			Cost   decimal.Decimal
			Tokens int64
		}

		err := r.db.WithContext(ctx).Model(&entity.CostUsageRecord{}).
			Select("SUM(cost) as cost, SUM(tokens) as tokens").
			Where("scope = ? AND scope_id = ? AND date >= ? AND date <= ?", scope, scopeID, startOfMonth, endOfMonth).
			Scan(&result).Error

		return result.Cost, result.Tokens, err
	}

	// ==================== User Quota Repository ====================

	// UserQuotaRepoImpl implements UserQuotaRepository
	type UserQuotaRepoImpl struct {
		db *gorm.DB
	}

	// NewUserQuotaRepository creates a new user quota repository
	func NewUserQuotaRepository(db *gorm.DB) *UserQuotaRepoImpl {
		return &UserQuotaRepoImpl{db: db}
	}

	func (r *UserQuotaRepoImpl) Create(ctx context.Context, quota *entity.UserQuota) error {
		return r.db.WithContext(ctx).Create(quota).Error
	}

	func (r *UserQuotaRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.UserQuota, error) {
		var quota entity.UserQuota
		err := r.db.WithContext(ctx).Where("id = ?", id).First(&quota).Error
		if err != nil {
			return nil, err
		}
		return &quota, nil
	}

	func (r *UserQuotaRepoImpl) GetByUserID(ctx context.Context, userID uuid.UUID) (*entity.UserQuota, error) {
		var quota entity.UserQuota
		err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&quota).Error
		if err != nil {
			return nil, err
		}
		return &quota, nil
	}

	func (r *UserQuotaRepoImpl) Update(ctx context.Context, quota *entity.UserQuota) error {
		return r.db.WithContext(ctx).Save(quota).Error
	}

	func (r *UserQuotaRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Delete(&entity.UserQuota{}, id).Error
	}

	func (r *UserQuotaRepoImpl) List(ctx context.Context, page, pageSize int) ([]entity.UserQuota, int64, error) {
		var quotas []entity.UserQuota
		var total int64

		offset := (page - 1) * pageSize
		err := r.db.WithContext(ctx).Model(&entity.UserQuota{}).Count(&total).Error
		if err != nil {
			return nil, 0, err
		}

		err = r.db.WithContext(ctx).Offset(offset).Limit(pageSize).Find(&quotas).Error
		if err != nil {
			return nil, 0, err
		}

		return quotas, total, nil
	}

	func (r *UserQuotaRepoImpl) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
		return r.db.WithContext(ctx).Model(&entity.UserQuota{}).
			Where("id = ?", id).
			Update("tokens_used", gorm.Expr("tokens_used + ?", tokens)).Error
	}

	func (r *UserQuotaRepoImpl) ResetTokensUsed(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Model(&entity.UserQuota{}).
			Where("id = ?", id).
			Update("tokens_used", 0).Error
	}

	func (r *UserQuotaRepoImpl) GetTokenUsage(ctx context.Context, id uuid.UUID) (used int64, quota int64, err error) {
		var q entity.UserQuota
		err = r.db.WithContext(ctx).Select("tokens_used, token_quota").Where("id = ?", id).First(&q).Error
		return q.TokensUsed, q.TokenQuota, err
	}

	func (r *UserQuotaRepoImpl) GetBalance(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
		var quota entity.UserQuota
		err := r.db.WithContext(ctx).Select("balance").Where("id = ?", id).First(&quota).Error
		if err != nil {
			return decimal.Zero, err
		}
		return quota.Balance, nil
	}

	func (r *UserQuotaRepoImpl) DeductBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
		result := r.db.WithContext(ctx).Model(&entity.UserQuota{}).
			Where("id = ? AND balance >= ?", id, amount).
			Update("balance", gorm.Expr("balance - ?", amount))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("insufficient balance or quota not found")
		}
		return nil
	}

	func (r *UserQuotaRepoImpl) AddBalance(ctx context.Context, id uuid.UUID, amount decimal.Decimal) error {
		return r.db.WithContext(ctx).Model(&entity.UserQuota{}).
			Where("id = ?", id).
			Update("balance", gorm.Expr("balance + ?", amount)).Error
	}

	func (r *UserQuotaRepoImpl) IncrementMonthlyCost(ctx context.Context, id uuid.UUID, cost decimal.Decimal) error {
		return r.db.WithContext(ctx).Model(&entity.UserQuota{}).
			Where("id = ?", id).
			Update("monthly_cost", gorm.Expr("monthly_cost + ?", cost)).Error
	}

	func (r *UserQuotaRepoImpl) ResetMonthlyCost(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Model(&entity.UserQuota{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"monthly_cost":     decimal.Zero,
				"monthly_reset_at": time.Now(),
			}).Error
	}

	func (r *UserQuotaRepoImpl) GetMonthlyCost(ctx context.Context, id uuid.UUID) (decimal.Decimal, error) {
		var quota entity.UserQuota
		err := r.db.WithContext(ctx).Select("monthly_cost").Where("id = ?", id).First(&quota).Error
		if err != nil {
			return decimal.Zero, err
		}
		return quota.MonthlyCost, nil
	}

	func (r *UserQuotaRepoImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
		return r.db.WithContext(ctx).Model(&entity.UserQuota{}).
			Where("id = ?", id).
			Update("status", status).Error
	}

	func (r *UserQuotaRepoImpl) GetStatus(ctx context.Context, id uuid.UUID) (string, error) {
		var quota entity.UserQuota
		err := r.db.WithContext(ctx).Select("status").Where("id = ?", id).First(&quota).Error
		if err != nil {
			return "", err
		}
		return quota.Status, nil
	}

	// ==================== Member Quota Repository ====================

	// MemberQuotaRepoImpl implements MemberQuotaRepository
	type MemberQuotaRepoImpl struct {
		db *gorm.DB
	}

	// NewMemberQuotaRepository creates a new member quota repository
	func NewMemberQuotaRepository(db *gorm.DB) *MemberQuotaRepoImpl {
		return &MemberQuotaRepoImpl{db: db}
	}

	func (r *MemberQuotaRepoImpl) Create(ctx context.Context, quota *entity.MemberQuota) error {
		return r.db.WithContext(ctx).Create(quota).Error
	}

	func (r *MemberQuotaRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.MemberQuota, error) {
		var quota entity.MemberQuota
		err := r.db.WithContext(ctx).Where("id = ?", id).First(&quota).Error
		if err != nil {
			return nil, err
		}
		return &quota, nil
	}

	func (r *MemberQuotaRepoImpl) GetByUserID(ctx context.Context, userID uuid.UUID) (*entity.MemberQuota, error) {
		var quota entity.MemberQuota
		err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&quota).Error
		if err != nil {
			return nil, err
		}
		return &quota, nil
	}

	func (r *MemberQuotaRepoImpl) GetByTenantAndUser(ctx context.Context, tenantID, userID uuid.UUID) (*entity.MemberQuota, error) {
		var quota entity.MemberQuota
		err := r.db.WithContext(ctx).Where("tenant_id = ? AND user_id = ?", tenantID, userID).First(&quota).Error
		if err != nil {
			return nil, err
		}
		return &quota, nil
	}

	func (r *MemberQuotaRepoImpl) Update(ctx context.Context, quota *entity.MemberQuota) error {
		return r.db.WithContext(ctx).Save(quota).Error
	}

	func (r *MemberQuotaRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Delete(&entity.MemberQuota{}, id).Error
	}

	func (r *MemberQuotaRepoImpl) List(ctx context.Context, page, pageSize int) ([]entity.MemberQuota, int64, error) {
		var quotas []entity.MemberQuota
		var total int64

		offset := (page - 1) * pageSize
		err := r.db.WithContext(ctx).Model(&entity.MemberQuota{}).Count(&total).Error
		if err != nil {
			return nil, 0, err
		}

		err = r.db.WithContext(ctx).Offset(offset).Limit(pageSize).Find(&quotas).Error
		if err != nil {
			return nil, 0, err
		}

		return quotas, total, nil
	}

	func (r *MemberQuotaRepoImpl) ListByTenant(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.MemberQuota, int64, error) {
		var quotas []entity.MemberQuota
		var total int64

		offset := (page - 1) * pageSize
		query := r.db.WithContext(ctx).Model(&entity.MemberQuota{}).Where("tenant_id = ?", tenantID)

		err := query.Count(&total).Error
		if err != nil {
			return nil, 0, err
		}

		err = query.Offset(offset).Limit(pageSize).Find(&quotas).Error
		if err != nil {
			return nil, 0, err
		}

		return quotas, total, nil
	}

	func (r *MemberQuotaRepoImpl) IncrementTokensUsed(ctx context.Context, id uuid.UUID, tokens int64) error {
		return r.db.WithContext(ctx).Model(&entity.MemberQuota{}).
			Where("id = ?", id).
			Update("tokens_used", gorm.Expr("tokens_used + ?", tokens)).Error
	}

	func (r *MemberQuotaRepoImpl) ResetTokensUsed(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Model(&entity.MemberQuota{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"tokens_used":     0,
				"token_reset_at": time.Now(),
			}).Error
	}

	func (r *MemberQuotaRepoImpl) GetTokenUsage(ctx context.Context, id uuid.UUID) (used int64, limit int64, err error) {
		var quota entity.MemberQuota
		err = r.db.WithContext(ctx).Select("tokens_used, token_quota_limit").Where("id = ?", id).First(&quota).Error
		if quota.TokenQuotaLimit != nil {
			return quota.TokensUsed, *quota.TokenQuotaLimit, err
		}
		return quota.TokensUsed, 0, err
	}

	func (r *MemberQuotaRepoImpl) IncrementCostUsed(ctx context.Context, id uuid.UUID, cost decimal.Decimal) error {
		return r.db.WithContext(ctx).Model(&entity.MemberQuota{}).
			Where("id = ?", id).
			Update("cost_used", gorm.Expr("cost_used + ?", cost)).Error
	}

	func (r *MemberQuotaRepoImpl) ResetCostUsed(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Model(&entity.MemberQuota{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"cost_used":     decimal.Zero,
				"cost_reset_at": time.Now(),
			}).Error
	}

	func (r *MemberQuotaRepoImpl) GetCostUsage(ctx context.Context, id uuid.UUID) (used decimal.Decimal, limit decimal.Decimal, err error) {
		var quota entity.MemberQuota
		err = r.db.WithContext(ctx).Select("cost_used, cost_limit").Where("id = ?", id).First(&quota).Error
		if quota.CostLimit != nil {
			return quota.CostUsed, *quota.CostLimit, err
		}
		return quota.CostUsed, decimal.Zero, err
	}

	func (r *MemberQuotaRepoImpl) GetTotalTokensUsedByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		var result int64
		err := r.db.WithContext(ctx).Model(&entity.MemberQuota{}).
			Where("tenant_id = ?", tenantID).
			Select("SUM(tokens_used)").
			Scan(&result).Error
		return result, err
	}

	func (r *MemberQuotaRepoImpl) GetTotalCostUsedByTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
		var result decimal.Decimal
		err := r.db.WithContext(ctx).Model(&entity.MemberQuota{}).
			Where("tenant_id = ?", tenantID).
			Select("SUM(cost_used)").
			Scan(&result).Error
		return result, err
	}

	func (r *MemberQuotaRepoImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
		return r.db.WithContext(ctx).Model(&entity.MemberQuota{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":     status,
				"updated_at": time.Now(),
			}).Error
	}

	func (r *MemberQuotaRepoImpl) SetExceeded(ctx context.Context, id uuid.UUID, reason string) error {
		now := time.Now()
		return r.db.WithContext(ctx).Model(&entity.MemberQuota{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":           "quota_exceeded",
				"exceeded_at":      now,
				"exceeded_reason":  reason,
				"updated_at":       now,
			}).Error
	}

	func (r *MemberQuotaRepoImpl) IncrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Model(&entity.MemberQuota{}).
			Where("id = ?", id).
			Update("active_api_keys", gorm.Expr("active_api_keys + 1")).Error
	}

	func (r *MemberQuotaRepoImpl) DecrementActiveAPIKeys(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Model(&entity.MemberQuota{}).
			Where("id = ? AND active_api_keys > 0", id).
			Update("active_api_keys", gorm.Expr("active_api_keys - 1")).Error
	}

	func (r *MemberQuotaRepoImpl) GetActiveAPIKeysCount(ctx context.Context, id uuid.UUID) (int, error) {
		var quota entity.MemberQuota
		err := r.db.WithContext(ctx).Select("active_api_keys").Where("id = ?", id).First(&quota).Error
		return quota.ActiveAPIKeys, err
	}

	// ==================== Credit Application Repository ====================

	// CreditApplicationRepoImpl implements CreditApplicationRepository
	type CreditApplicationRepoImpl struct {
		db *gorm.DB
	}

	// NewCreditApplicationRepository creates a new credit application repository
	func NewCreditApplicationRepository(db *gorm.DB) *CreditApplicationRepoImpl {
		return &CreditApplicationRepoImpl{db: db}
	}

	func (r *CreditApplicationRepoImpl) Create(ctx context.Context, application *entity.CreditApplication) error {
		return r.db.WithContext(ctx).Create(application).Error
	}

	func (r *CreditApplicationRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.CreditApplication, error) {
		var application entity.CreditApplication
		err := r.db.WithContext(ctx).Where("id = ?", id).First(&application).Error
		if err != nil {
			return nil, err
		}
		return &application, nil
	}

	func (r *CreditApplicationRepoImpl) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*entity.CreditApplication, error) {
		var application entity.CreditApplication
		err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&application).Error
		if err != nil {
			return nil, err
		}
		return &application, nil
	}

	func (r *CreditApplicationRepoImpl) GetLatestByTenantID(ctx context.Context, tenantID uuid.UUID) (*entity.CreditApplication, error) {
		var application entity.CreditApplication
		err := r.db.WithContext(ctx).
			Where("tenant_id = ?", tenantID).
			Order("created_at DESC").
			First(&application).Error
		if err != nil {
			return nil, err
		}
		return &application, nil
	}

	func (r *CreditApplicationRepoImpl) Update(ctx context.Context, application *entity.CreditApplication) error {
		return r.db.WithContext(ctx).Save(application).Error
	}

	func (r *CreditApplicationRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Delete(&entity.CreditApplication{}, id).Error
	}

	func (r *CreditApplicationRepoImpl) List(ctx context.Context, page, pageSize int) ([]entity.CreditApplication, int64, error) {
		var applications []entity.CreditApplication
		var total int64

		offset := (page - 1) * pageSize
		err := r.db.WithContext(ctx).Model(&entity.CreditApplication{}).Count(&total).Error
		if err != nil {
			return nil, 0, err
		}

		err = r.db.WithContext(ctx).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&applications).Error
		if err != nil {
			return nil, 0, err
		}

		return applications, total, nil
	}

	func (r *CreditApplicationRepoImpl) ListByStatus(ctx context.Context, status string, page, pageSize int) ([]entity.CreditApplication, int64, error) {
		var applications []entity.CreditApplication
		var total int64

		offset := (page - 1) * pageSize
		query := r.db.WithContext(ctx).Model(&entity.CreditApplication{}).Where("status = ?", status)

		err := query.Count(&total).Error
		if err != nil {
			return nil, 0, err
		}

		err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&applications).Error
		if err != nil {
			return nil, 0, err
		}

		return applications, total, nil
	}

	func (r *CreditApplicationRepoImpl) Approve(ctx context.Context, id uuid.UUID, approvedLimit decimal.Decimal, reviewedBy uuid.UUID, reviewNotes string) error {
		now := time.Now()
		return r.db.WithContext(ctx).Model(&entity.CreditApplication{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":         "approved",
				"approved_limit": approvedLimit,
				"reviewed_at":    now,
				"reviewed_by":    reviewedBy,
				"review_notes":   reviewNotes,
				"updated_at":     now,
			}).Error
	}

	func (r *CreditApplicationRepoImpl) Reject(ctx context.Context, id uuid.UUID, reviewedBy uuid.UUID, reviewNotes string) error {
		now := time.Now()
		return r.db.WithContext(ctx).Model(&entity.CreditApplication{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":       "rejected",
				"reviewed_at":  now,
				"reviewed_by":  reviewedBy,
				"review_notes": reviewNotes,
				"updated_at":   now,
			}).Error
	}

	func (r *CreditApplicationRepoImpl) GetPendingCount(ctx context.Context) (int64, error) {
		var count int64
		err := r.db.WithContext(ctx).Model(&entity.CreditApplication{}).
			Where("status = ?", "pending").
			Count(&count).Error
		return count, err
	}

	// ==================== Payment Order Repository ====================

	// PaymentOrderRepoImpl implements PaymentOrderRepository
	type PaymentOrderRepoImpl struct {
		db *gorm.DB
	}

	// NewPaymentOrderRepository creates a new payment order repository
	func NewPaymentOrderRepository(db *gorm.DB) *PaymentOrderRepoImpl {
		return &PaymentOrderRepoImpl{db: db}
	}

	func (r *PaymentOrderRepoImpl) Create(ctx context.Context, order *entity.PaymentOrder) error {
		return r.db.WithContext(ctx).Create(order).Error
	}

	func (r *PaymentOrderRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.PaymentOrder, error) {
		var order entity.PaymentOrder
		err := r.db.WithContext(ctx).Where("id = ?", id).First(&order).Error
		if err != nil {
			return nil, err
		}
		return &order, nil
	}

	func (r *PaymentOrderRepoImpl) GetByOrderNumber(ctx context.Context, orderNumber string) (*entity.PaymentOrder, error) {
		var order entity.PaymentOrder
		err := r.db.WithContext(ctx).Where("order_number = ?", orderNumber).First(&order).Error
		if err != nil {
			return nil, err
		}
		return &order, nil
	}

	func (r *PaymentOrderRepoImpl) Update(ctx context.Context, order *entity.PaymentOrder) error {
		return r.db.WithContext(ctx).Save(order).Error
	}

	func (r *PaymentOrderRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Delete(&entity.PaymentOrder{}, id).Error
	}

	func (r *PaymentOrderRepoImpl) List(ctx context.Context, page, pageSize int) ([]entity.PaymentOrder, int64, error) {
		var orders []entity.PaymentOrder
		var total int64

		offset := (page - 1) * pageSize
		err := r.db.WithContext(ctx).Model(&entity.PaymentOrder{}).Count(&total).Error
		if err != nil {
			return nil, 0, err
		}

		err = r.db.WithContext(ctx).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&orders).Error
		if err != nil {
			return nil, 0, err
		}

		return orders, total, nil
	}

	func (r *PaymentOrderRepoImpl) ListByUserID(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]entity.PaymentOrder, int64, error) {
		var orders []entity.PaymentOrder
		var total int64

		offset := (page - 1) * pageSize
		query := r.db.WithContext(ctx).Model(&entity.PaymentOrder{}).Where("user_id = ?", userID)

		err := query.Count(&total).Error
		if err != nil {
			return nil, 0, err
		}

		err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&orders).Error
		if err != nil {
			return nil, 0, err
		}

		return orders, total, nil
	}

	func (r *PaymentOrderRepoImpl) ListByTenantID(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.PaymentOrder, int64, error) {
		var orders []entity.PaymentOrder
		var total int64

		offset := (page - 1) * pageSize
		query := r.db.WithContext(ctx).Model(&entity.PaymentOrder{}).Where("tenant_id = ?", tenantID)

		err := query.Count(&total).Error
		if err != nil {
			return nil, 0, err
		}

		err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&orders).Error
		if err != nil {
			return nil, 0, err
		}

		return orders, total, nil
	}

	func (r *PaymentOrderRepoImpl) ListByStatus(ctx context.Context, status string, page, pageSize int) ([]entity.PaymentOrder, int64, error) {
		var orders []entity.PaymentOrder
		var total int64

		offset := (page - 1) * pageSize
		query := r.db.WithContext(ctx).Model(&entity.PaymentOrder{}).Where("status = ?", status)

		err := query.Count(&total).Error
		if err != nil {
			return nil, 0, err
		}

		err = query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&orders).Error
		if err != nil {
			return nil, 0, err
		}

		return orders, total, nil
	}

	func (r *PaymentOrderRepoImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
		return r.db.WithContext(ctx).Model(&entity.PaymentOrder{}).
			Where("id = ?", id).
			Update("status", status).Error
	}

	func (r *PaymentOrderRepoImpl) MarkPaid(ctx context.Context, id uuid.UUID, paymentID string, callbackData string) error {
		now := time.Now()
		return r.db.WithContext(ctx).Model(&entity.PaymentOrder{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":        "paid",
				"payment_id":    paymentID,
				"callback_data": callbackData,
				"callback_at":   now,
				"paid_at":       now,
				"updated_at":    now,
			}).Error
	}

	func (r *PaymentOrderRepoImpl) MarkFailed(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Model(&entity.PaymentOrder{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":     "failed",
				"updated_at": time.Now(),
			}).Error
	}

	func (r *PaymentOrderRepoImpl) MarkCancelled(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Model(&entity.PaymentOrder{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":     "cancelled",
				"updated_at": time.Now(),
			}).Error
	}

	func (r *PaymentOrderRepoImpl) GetTotalAmountByUser(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
		var result decimal.Decimal
		err := r.db.WithContext(ctx).Model(&entity.PaymentOrder{}).
			Where("user_id = ? AND status = ?", userID, "paid").
			Select("SUM(amount)").
			Scan(&result).Error
		return result, err
	}

	func (r *PaymentOrderRepoImpl) GetTotalAmountByTenant(ctx context.Context, tenantID uuid.UUID) (decimal.Decimal, error) {
		var result decimal.Decimal
		err := r.db.WithContext(ctx).Model(&entity.PaymentOrder{}).
			Where("tenant_id = ? AND status = ?", tenantID, "paid").
			Select("SUM(amount)").
			Scan(&result).Error
		return result, err
	}

	func (r *PaymentOrderRepoImpl) GenerateOrderNumber() string {
		return fmt.Sprintf("PAY-%d-%s", time.Now().UnixNano(), uuid.New().String()[:8])
	}

	func (r *PaymentOrderRepoImpl) GetByPaymentID(ctx context.Context, paymentID string) (*entity.PaymentOrder, error) {
		var order entity.PaymentOrder
		err := r.db.WithContext(ctx).Where("payment_id = ?", paymentID).First(&order).Error
		if err != nil {
			return nil, err
		}
		return &order, nil
	}

	// ListByUser alias for ListByUserID
	func (r *PaymentOrderRepoImpl) ListByUser(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]entity.PaymentOrder, int64, error) {
		return r.ListByUserID(ctx, userID, page, pageSize)
	}

	// ListByTenant alias for ListByTenantID
	func (r *PaymentOrderRepoImpl) ListByTenant(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.PaymentOrder, int64, error) {
		return r.ListByTenantID(ctx, tenantID, page, pageSize)
	}

	func (r *PaymentOrderRepoImpl) ListPendingByUser(ctx context.Context, userID uuid.UUID) ([]entity.PaymentOrder, error) {
		var orders []entity.PaymentOrder
		err := r.db.WithContext(ctx).
			Where("user_id = ? AND status = ?", userID, "pending").
			Order("created_at DESC").
			Find(&orders).Error
		return orders, err
	}

	func (r *PaymentOrderRepoImpl) ListPendingByTenant(ctx context.Context, tenantID uuid.UUID) ([]entity.PaymentOrder, error) {
		var orders []entity.PaymentOrder
		err := r.db.WithContext(ctx).
			Where("tenant_id = ? AND status = ?", tenantID, "pending").
			Order("created_at DESC").
			Find(&orders).Error
		return orders, err
	}

	func (r *PaymentOrderRepoImpl) MarkExpired(ctx context.Context) (int, error) {
		now := time.Now()
		result := r.db.WithContext(ctx).Model(&entity.PaymentOrder{}).
			Where("status = ? AND expire_at < ?", "pending", now).
			Update("status", "expired")
		return int(result.RowsAffected), result.Error
	}

	// ==================== UserTenant Repository ====================

	type UserTenantRepoImpl struct {
		db *gorm.DB
	}

	func NewUserTenantRepository(db *gorm.DB) *UserTenantRepoImpl {
		return &UserTenantRepoImpl{db: db}
	}

	func (r *UserTenantRepoImpl) Create(ctx context.Context, ut *entity.UserTenant) error {
		return r.db.WithContext(ctx).Create(ut).Error
	}

	func (r *UserTenantRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.UserTenant, error) {
		var ut entity.UserTenant
		err := r.db.WithContext(ctx).Where("id = ?", id).First(&ut).Error
		if err != nil {
			return nil, err
		}
		return &ut, nil
	}

	func (r *UserTenantRepoImpl) Update(ctx context.Context, ut *entity.UserTenant) error {
		return r.db.WithContext(ctx).Save(ut).Error
	}

	func (r *UserTenantRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Delete(&entity.UserTenant{}, id).Error
	}

	func (r *UserTenantRepoImpl) GetByUserAndTenant(ctx context.Context, userID, tenantID uuid.UUID) (*entity.UserTenant, error) {
		var ut entity.UserTenant
		err := r.db.WithContext(ctx).
			Where("user_id = ? AND tenant_id = ?", userID, tenantID).
			First(&ut).Error
		if err != nil {
			return nil, err
		}
		return &ut, nil
	}

	func (r *UserTenantRepoImpl) ListByUser(ctx context.Context, userID uuid.UUID) ([]entity.UserTenant, error) {
		var uts []entity.UserTenant
		err := r.db.WithContext(ctx).
			Where("user_id = ?", userID).
			Order("is_default DESC, joined_at DESC").
			Find(&uts).Error
		return uts, err
	}

	func (r *UserTenantRepoImpl) ListByTenant(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.UserTenant, int64, error) {
		var uts []entity.UserTenant
		var total int64

		offset := (page - 1) * pageSize
		query := r.db.WithContext(ctx).Model(&entity.UserTenant{}).Where("tenant_id = ?", tenantID)

		err := query.Count(&total).Error
		if err != nil {
			return nil, 0, err
		}

		err = query.Offset(offset).Limit(pageSize).Find(&uts).Error
		return uts, total, err
	}

	func (r *UserTenantRepoImpl) GetDefaultTenant(ctx context.Context, userID uuid.UUID) (*entity.UserTenant, error) {
		var ut entity.UserTenant
		err := r.db.WithContext(ctx).
			Where("user_id = ? AND is_default = ?", userID, true).
			First(&ut).Error
		if err != nil {
			return nil, err
		}
		return &ut, nil
	}

	func (r *UserTenantRepoImpl) SetDefaultTenant(ctx context.Context, userID, tenantID uuid.UUID) error {
		// 先清除所有默认
		err := r.db.WithContext(ctx).Model(&entity.UserTenant{}).
			Where("user_id = ?", userID).
			Update("is_default", false).Error
		if err != nil {
			return err
		}

		// 设置新的默认
		return r.db.WithContext(ctx).Model(&entity.UserTenant{}).
			Where("user_id = ? AND tenant_id = ?", userID, tenantID).
			Update("is_default", true).Error
	}

	func (r *UserTenantRepoImpl) ClearDefaultTenants(ctx context.Context, userID uuid.UUID) error {
		return r.db.WithContext(ctx).Model(&entity.UserTenant{}).
			Where("user_id = ?", userID).
			Update("is_default", false).Error
	}

	func (r *UserTenantRepoImpl) UpdateStatus(ctx context.Context, userID, tenantID uuid.UUID, status string) error {
		return r.db.WithContext(ctx).Model(&entity.UserTenant{}).
			Where("user_id = ? AND tenant_id = ?", userID, tenantID).
			Update("status", status).Error
	}

	func (r *UserTenantRepoImpl) UpdateRole(ctx context.Context, userID, tenantID uuid.UUID, role string) error {
		return r.db.WithContext(ctx).Model(&entity.UserTenant{}).
			Where("user_id = ? AND tenant_id = ?", userID, tenantID).
			Update("role", role).Error
	}

	func (r *UserTenantRepoImpl) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error) {
		var count int64
		err := r.db.WithContext(ctx).Model(&entity.UserTenant{}).
			Where("tenant_id = ? AND status = ?", tenantID, "active").
			Count(&count).Error
		return count, err
	}

	func (r *UserTenantRepoImpl) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
		var count int64
		err := r.db.WithContext(ctx).Model(&entity.UserTenant{}).
			Where("user_id = ? AND status = ?", userID, "active").
			Count(&count).Error
		return count, err
	}

	func (r *UserTenantRepoImpl) ExistsByUserAndTenant(ctx context.Context, userID, tenantID uuid.UUID) (bool, error) {
		var count int64
		err := r.db.WithContext(ctx).Model(&entity.UserTenant{}).
			Where("user_id = ? AND tenant_id = ?", userID, tenantID).
			Count(&count).Error
		return count > 0, err
	}

	// ==================== LoginAudit Repository ====================

	type LoginAuditRepoImpl struct {
		db *gorm.DB
	}

	func NewLoginAuditRepository(db *gorm.DB) *LoginAuditRepoImpl {
		return &LoginAuditRepoImpl{db: db}
	}

	func (r *LoginAuditRepoImpl) Create(ctx context.Context, audit *entity.LoginAudit) error {
		return r.db.WithContext(ctx).Create(audit).Error
	}

	func (r *LoginAuditRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.LoginAudit, error) {
		var audit entity.LoginAudit
		err := r.db.WithContext(ctx).Where("id = ?", id).First(&audit).Error
		return &audit, err
	}

	func (r *LoginAuditRepoImpl) ListByUser(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]entity.LoginAudit, int64, error) {
		var audits []entity.LoginAudit
		var total int64

		offset := (page - 1) * pageSize
		query := r.db.WithContext(ctx).Model(&entity.LoginAudit{}).Where("user_id = ?", userID)

		err := query.Count(&total).Error
		if err != nil {
			return nil, 0, err
		}

		err = query.Order("login_at DESC").Offset(offset).Limit(pageSize).Find(&audits).Error
		return audits, total, err
	}

	func (r *LoginAuditRepoImpl) ListByEmail(ctx context.Context, email string, page, pageSize int) ([]entity.LoginAudit, int64, error) {
		var audits []entity.LoginAudit
		var total int64

		offset := (page - 1) * pageSize
		query := r.db.WithContext(ctx).Model(&entity.LoginAudit{}).Where("email = ?", email)

		err := query.Count(&total).Error
		if err != nil {
			return nil, 0, err
		}

		err = query.Order("login_at DESC").Offset(offset).Limit(pageSize).Find(&audits).Error
		return audits, total, err
	}

	func (r *LoginAuditRepoImpl) ListRecent(ctx context.Context, userID uuid.UUID, limit int) ([]entity.LoginAudit, error) {
		var audits []entity.LoginAudit
		err := r.db.WithContext(ctx).
			Where("user_id = ?", userID).
			Order("login_at DESC").
			Limit(limit).
			Find(&audits).Error
		return audits, err
	}

	func (r *LoginAuditRepoImpl) ListFailed(ctx context.Context, email string, windowMinutes int) ([]entity.LoginAudit, error) {
		var audits []entity.LoginAudit
		windowStart := time.Now().Add(-time.Duration(windowMinutes) * time.Minute)
		err := r.db.WithContext(ctx).
			Where("email = ? AND success = ? AND login_at > ?", email, false, windowStart).
			Order("login_at DESC").
			Find(&audits).Error
		return audits, err
	}

	// ==================== PasswordHistory Repository ====================

	type PasswordHistoryRepoImpl struct {
		db *gorm.DB
	}

	func NewPasswordHistoryRepository(db *gorm.DB) *PasswordHistoryRepoImpl {
		return &PasswordHistoryRepoImpl{db: db}
	}

	func (r *PasswordHistoryRepoImpl) Create(ctx context.Context, history *entity.PasswordHistory) error {
		return r.db.WithContext(ctx).Create(history).Error
	}

	func (r *PasswordHistoryRepoImpl) ListRecent(ctx context.Context, userID uuid.UUID, limit int) ([]entity.PasswordHistory, error) {
		var histories []entity.PasswordHistory
		err := r.db.WithContext(ctx).
			Where("user_id = ?", userID).
			Order("created_at DESC").
			Limit(limit).
			Find(&histories).Error
		return histories, err
	}

	func (r *PasswordHistoryRepoImpl) DeleteOld(ctx context.Context, userID uuid.UUID, keepCount int) error {
		// 获取要保留的记录ID
		var keepIDs []uuid.UUID
		err := r.db.WithContext(ctx).Model(&entity.PasswordHistory{}).
			Where("user_id = ?", userID).
			Order("created_at DESC").
			Limit(keepCount).
			Pluck("id", &keepIDs).Error
		if err != nil {
			return err
		}

		// 删除其他记录
		if len(keepIDs) > 0 {
			return r.db.WithContext(ctx).
				Where("user_id = ? AND id NOT IN ?", userID, keepIDs).
				Delete(&entity.PasswordHistory{}).Error
		}
		return nil
	}

	// ==================== RefreshToken Repository ====================

	type RefreshTokenRepoImpl struct {
		db *gorm.DB
	}

	func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepoImpl {
		return &RefreshTokenRepoImpl{db: db}
	}

	func (r *RefreshTokenRepoImpl) Create(ctx context.Context, token *entity.RefreshToken) error {
		return r.db.WithContext(ctx).Create(token).Error
	}

	func (r *RefreshTokenRepoImpl) GetByTokenHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
		var token entity.RefreshToken
		err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&token).Error
		return &token, err
	}

	func (r *RefreshTokenRepoImpl) GetByUserAndDevice(ctx context.Context, userID uuid.UUID, deviceID string) (*entity.RefreshToken, error) {
		var token entity.RefreshToken
		err := r.db.WithContext(ctx).
			Where("user_id = ? AND device_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, deviceID, time.Now()).
			First(&token).Error
		return &token, err
	}

	func (r *RefreshTokenRepoImpl) ListByUser(ctx context.Context, userID uuid.UUID) ([]entity.RefreshToken, error) {
		var tokens []entity.RefreshToken
		err := r.db.WithContext(ctx).
			Where("user_id = ? AND revoked_at IS NULL", userID).
			Order("created_at DESC").
			Find(&tokens).Error
		return tokens, err
	}

	func (r *RefreshTokenRepoImpl) UpdateLastUsed(ctx context.Context, tokenHash string) error {
		return r.db.WithContext(ctx).Model(&entity.RefreshToken{}).
			Where("token_hash = ?", tokenHash).
			Update("last_used_at", time.Now()).Error
	}

	func (r *RefreshTokenRepoImpl) Revoke(ctx context.Context, tokenHash string) error {
		return r.db.WithContext(ctx).Model(&entity.RefreshToken{}).
			Where("token_hash = ?", tokenHash).
			Update("revoked_at", time.Now()).Error
	}

	func (r *RefreshTokenRepoImpl) RevokeAllByUser(ctx context.Context, userID uuid.UUID) error {
		return r.db.WithContext(ctx).Model(&entity.RefreshToken{}).
			Where("user_id = ? AND revoked_at IS NULL", userID).
			Update("revoked_at", time.Now()).Error
	}

	func (r *RefreshTokenRepoImpl) DeleteExpired(ctx context.Context) error {
		return r.db.WithContext(ctx).
			Where("expires_at < ? OR revoked_at IS NOT NULL", time.Now()).
			Delete(&entity.RefreshToken{}).Error
	}
