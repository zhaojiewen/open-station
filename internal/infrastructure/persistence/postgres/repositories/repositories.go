package repositories

import (
	"context"
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
	// PluginRepoImpl implements PluginRepository
	type PluginRepoImpl struct {
		db *gorm.DB
	}

	// NewPluginRepository creates a new plugin repository
	func NewPluginRepository(db *gorm.DB) *PluginRepoImpl {
		return &PluginRepoImpl{db: db}
	}

	// Create creates a new plugin
	func (r *PluginRepoImpl) Create(ctx context.Context, plugin *entity.Plugin) error {
		return r.db.WithContext(ctx).Create(plugin).Error
	}

	// GetByID retrieves a plugin by its UUID
	func (r *PluginRepoImpl) GetByID(ctx context.Context, id uuid.UUID) (*entity.Plugin, error) {
		var plugin entity.Plugin
		err := r.db.WithContext(ctx).Where("id = ?", id).First(&plugin).Error
		if err != nil {
			return nil, err
		}
		return &plugin, nil
	}

	// GetByPluginID retrieves a plugin by its string ID
	func (r *PluginRepoImpl) GetByPluginID(ctx context.Context, pluginID string) (*entity.Plugin, error) {
		var plugin entity.Plugin
		err := r.db.WithContext(ctx).Where("plugin_id = ?", pluginID).First(&plugin).Error
		if err != nil {
			return nil, err
		}
		return &plugin, nil
	}

	// GetByProvider retrieves a plugin by provider name
	func (r *PluginRepoImpl) GetByProvider(ctx context.Context, provider string) (*entity.Plugin, error) {
		var plugin entity.Plugin
		err := r.db.WithContext(ctx).Where("provider = ?", provider).First(&plugin).Error
		if err != nil {
			return nil, err
		}
		return &plugin, nil
	}

	// Update updates a plugin
	func (r *PluginRepoImpl) Update(ctx context.Context, plugin *entity.Plugin) error {
		return r.db.WithContext(ctx).Save(plugin).Error
	}

	// Delete removes a plugin
	func (r *PluginRepoImpl) Delete(ctx context.Context, id uuid.UUID) error {
		return r.db.WithContext(ctx).Delete(&entity.Plugin{}, id).Error
	}

	// List returns all plugins with pagination
	func (r *PluginRepoImpl) List(ctx context.Context, page, pageSize int) ([]entity.Plugin, int64, error) {
		var plugins []entity.Plugin
		var total int64

		offset := (page - 1) * pageSize

		err := r.db.WithContext(ctx).Model(&entity.Plugin{}).Count(&total).Error
		if err != nil {
			return nil, 0, err
		}

		err = r.db.WithContext(ctx).
			Order("created_at DESC").
			Offset(offset).
			Limit(pageSize).
			Find(&plugins).Error
		if err != nil {
			return nil, 0, err
		}

		return plugins, total, nil
	}

	// ListByStatus returns plugins with a specific status
	func (r *PluginRepoImpl) ListByStatus(ctx context.Context, status string) ([]entity.Plugin, error) {
		var plugins []entity.Plugin
		err := r.db.WithContext(ctx).Where("status = ?", status).Find(&plugins).Error
		return plugins, err
	}

	// ListActive returns only active plugins
	func (r *PluginRepoImpl) ListActive(ctx context.Context) ([]entity.Plugin, error) {
		var plugins []entity.Plugin
		err := r.db.WithContext(ctx).Where("status = ?", "active").Find(&plugins).Error
		return plugins, err
	}

	// SetStatus updates plugin status
	func (r *PluginRepoImpl) SetStatus(ctx context.Context, pluginID string, status string) error {
		updates := map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}

		if status == "inactive" {
			updates["disabled_at"] = time.Now()
		} else if status == "active" {
			updates["disabled_at"] = nil
		}

		return r.db.WithContext(ctx).Model(&entity.Plugin{}).
			Where("plugin_id = ?", pluginID).
			Updates(updates).Error
	}

	// SetConfig updates plugin configuration
	func (r *PluginRepoImpl) SetConfig(ctx context.Context, pluginID string, config string) error {
		return r.db.WithContext(ctx).Model(&entity.Plugin{}).
			Where("plugin_id = ?", pluginID).
			Updates(map[string]interface{}{
				"config":     config,
				"updated_at": time.Now(),
			}).Error
	}

	// RecordRequest increments request count
	func (r *PluginRepoImpl) RecordRequest(ctx context.Context, pluginID string) error {
		return r.db.WithContext(ctx).Model(&entity.Plugin{}).
			Where("plugin_id = ?", pluginID).
			Updates(map[string]interface{}{
				"request_count": gorm.Expr("request_count + 1"),
				"updated_at":    time.Now(),
			}).Error
	}

	// RecordSuccess increments success count and updates latency
	func (r *PluginRepoImpl) RecordSuccess(ctx context.Context, pluginID string, latencyMs int64) error {
		return r.db.WithContext(ctx).Model(&entity.Plugin{}).
			Where("plugin_id = ?", pluginID).
			Updates(map[string]interface{}{
				"success_count": gorm.Expr("success_count + 1"),
				"updated_at":    time.Now(),
				"health_score":  gorm.Expr("LEAST(100, health_score + 1)"),
			}).Error
	}

	// RecordError increments error count
	func (r *PluginRepoImpl) RecordError(ctx context.Context, pluginID string, errMsg string) error {
		return r.db.WithContext(ctx).Model(&entity.Plugin{}).
			Where("plugin_id = ?", pluginID).
			Updates(map[string]interface{}{
				"error_count":   gorm.Expr("error_count + 1"),
				"last_error":    errMsg,
				"last_error_at": time.Now(),
				"updated_at":    time.Now(),
				"health_score":  gorm.Expr("GREATEST(0, health_score - 5)"),
			}).Error
	}

	// IncrementCost adds to total cost
	func (r *PluginRepoImpl) IncrementCost(ctx context.Context, pluginID string, cost decimal.Decimal) error {
		return r.db.WithContext(ctx).Model(&entity.Plugin{}).
			Where("plugin_id = ?", pluginID).
			Update("total_cost", gorm.Expr("total_cost + ?", cost)).Error
	}

	// GetStats returns usage statistics for a plugin
	func (r *PluginRepoImpl) GetStats(ctx context.Context, pluginID string) (*entity.PluginStatus, error) {
		plugin, err := r.GetByPluginID(ctx, pluginID)
		if err != nil {
			return nil, err
		}
		status := plugin.ToPluginStatus()
		return &status, nil
	}

	// GetAllStats returns statistics for all plugins
	func (r *PluginRepoImpl) GetAllStats(ctx context.Context) ([]entity.PluginStatus, error) {
		plugins, _, err := r.List(ctx, 1, 1000)
		if err != nil {
			return nil, err
		}

		stats := make([]entity.PluginStatus, len(plugins))
		for i, p := range plugins {
			stats[i] = p.ToPluginStatus()
		}

		return stats, nil
	}

	// Exists checks if a plugin exists
	func (r *PluginRepoImpl) Exists(ctx context.Context, pluginID string) bool {
		var count int64
		r.db.WithContext(ctx).Model(&entity.Plugin{}).
			Where("plugin_id = ?", pluginID).
			Count(&count)
		return count > 0
	}

	// ProviderExists checks if a provider has a plugin
	func (r *PluginRepoImpl) ProviderExists(ctx context.Context, provider string) bool {
		var count int64
		r.db.WithContext(ctx).Model(&entity.Plugin{}).
			Where("provider = ?", provider).
			Count(&count)
		return count > 0
	}

	// GetProviders returns list of all provider names with plugins
	func (r *PluginRepoImpl) GetProviders(ctx context.Context) ([]string, error) {
		var providers []string
		err := r.db.WithContext(ctx).Model(&entity.Plugin{}).
			Distinct("provider").
			Pluck("provider", &providers).Error
		return providers, err
	}

	// ResetStats resets statistics for a plugin
	func (r *PluginRepoImpl) ResetStats(ctx context.Context, pluginID string) error {
		return r.db.WithContext(ctx).Model(&entity.Plugin{}).
			Where("plugin_id = ?", pluginID).
			Updates(map[string]interface{}{
				"request_count":  0,
				"success_count":  0,
				"error_count":    0,
				"total_cost":     0,
				"last_error":     nil,
				"last_error_at":  nil,
				"health_score":   100,
				"updated_at":     time.Now(),
			}).Error
	}

	// UpdateHealthScore updates health score
	func (r *PluginRepoImpl) UpdateHealthScore(ctx context.Context, pluginID string, score int) error {
		return r.db.WithContext(ctx).Model(&entity.Plugin{}).
			Where("plugin_id = ?", pluginID).
			Update("health_score", score).Error
	}

	// HealthCheckAll returns health status for all active plugins
	func (r *PluginRepoImpl) HealthCheckAll(ctx context.Context) (map[string]int, error) {
		plugins, err := r.ListActive(ctx)
		if err != nil {
			return nil, err
		}

		healthScores := make(map[string]int)
		for _, p := range plugins {
			healthScores[p.PluginID] = p.HealthScore
		}

		return healthScores, nil
	}
