package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// PlatformAdminRepository defines operations for platform admin management
type PlatformAdminRepository interface {
	Create(ctx context.Context, admin *entity.PlatformAdmin) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.PlatformAdmin, error)
	GetByEmail(ctx context.Context, email string) (*entity.PlatformAdmin, error)
	Update(ctx context.Context, admin *entity.PlatformAdmin) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, page, pageSize int) ([]entity.PlatformAdmin, int64, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error

	// 权限检查
	CheckPermission(ctx context.Context, id uuid.UUID, permission string) (bool, error)
	GetPermissions(ctx context.Context, id uuid.UUID) ([]string, error)
}

// TenantApplicationRepository defines operations for tenant application management
type TenantApplicationRepository interface {
	Create(ctx context.Context, app *entity.TenantApplication) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.TenantApplication, error)
	GetBySlug(ctx context.Context, slug string) (*entity.TenantApplication, error)
	GetByEmail(ctx context.Context, email string) (*entity.TenantApplication, error)
	Update(ctx context.Context, app *entity.TenantApplication) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, status string, page, pageSize int) ([]entity.TenantApplication, int64, error)
	ListByStatus(ctx context.Context, status string) ([]entity.TenantApplication, error)

	// 状态操作
	SetStatus(ctx context.Context, id uuid.UUID, status string) error
	Approve(ctx context.Context, id uuid.UUID, reviewerID uuid.UUID, notes string) error
	Reject(ctx context.Context, id uuid.UUID, reviewerID uuid.UUID, reason string) error
	MarkTenantCreated(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error
}

// UserApplicationRepository defines operations for user application management
type UserApplicationRepository interface {
	Create(ctx context.Context, app *entity.UserApplication) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.UserApplication, error)
	GetByToken(ctx context.Context, token string) (*entity.UserApplication, error)
	GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*entity.UserApplication, error)
	Update(ctx context.Context, app *entity.UserApplication) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, status string, page, pageSize int) ([]entity.UserApplication, int64, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, status string) ([]entity.UserApplication, error)
	ListAll(ctx context.Context, status string, page, pageSize int) ([]entity.UserApplication, int64, error)

	// 状态操作
	SetStatus(ctx context.Context, id uuid.UUID, status string) error
	Approve(ctx context.Context, id uuid.UUID, reviewerID uuid.UUID) error
	Reject(ctx context.Context, id uuid.UUID, reviewerID uuid.UUID) error
	MarkUserCreated(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	Accept(ctx context.Context, id uuid.UUID) error
	MarkExpired(ctx context.Context, id uuid.UUID) error
}

// BudgetAlertRepository defines operations for budget alert management
type BudgetAlertRepository interface {
	Create(ctx context.Context, alert *entity.BudgetAlert) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.BudgetAlert, error)
	GetByScope(ctx context.Context, scope string, scopeID uuid.UUID) ([]entity.BudgetAlert, error)
	GetByScopeAndType(ctx context.Context, scope string, scopeID uuid.UUID, alertType string) (*entity.BudgetAlert, error)
	Update(ctx context.Context, alert *entity.BudgetAlert) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, page, pageSize int) ([]entity.BudgetAlert, int64, error)
	ListEnabled(ctx context.Context) ([]entity.BudgetAlert, error)

	// 状态操作
	Enable(ctx context.Context, id uuid.UUID) error
	Disable(ctx context.Context, id uuid.UUID) error
	MarkTriggered(ctx context.Context, id uuid.UUID) error
}

// CostUsageRecordRepository defines operations for cost usage record management
type CostUsageRecordRepository interface {
	Create(ctx context.Context, record *entity.CostUsageRecord) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.CostUsageRecord, error)
	GetByScopeAndDate(ctx context.Context, scope string, scopeID uuid.UUID, date time.Time) (*entity.CostUsageRecord, error)
	Update(ctx context.Context, record *entity.CostUsageRecord) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, scope string, scopeID uuid.UUID, startDate, endDate time.Time) ([]entity.CostUsageRecord, error)

	// 聚合操作
	GetDailyTotal(ctx context.Context, scope string, scopeID uuid.UUID, date time.Time) (decimal.Decimal, int64, error)
	GetMonthlyTotal(ctx context.Context, scope string, scopeID uuid.UUID, month time.Time) (decimal.Decimal, int64, error)
}