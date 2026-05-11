package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// UserTenantRepository 用户-租户关联Repository接口
type UserTenantRepository interface {
	// CRUD操作
	Create(ctx context.Context, ut *entity.UserTenant) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.UserTenant, error)
	Update(ctx context.Context, ut *entity.UserTenant) error
	Delete(ctx context.Context, id uuid.UUID) error

	// 查询操作
	GetByUserAndTenant(ctx context.Context, userID, tenantID uuid.UUID) (*entity.UserTenant, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]entity.UserTenant, error)
	ListByTenant(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.UserTenant, int64, error)
	GetDefaultTenant(ctx context.Context, userID uuid.UUID) (*entity.UserTenant, error)

	// 状态操作
	SetDefaultTenant(ctx context.Context, userID, tenantID uuid.UUID) error
	ClearDefaultTenants(ctx context.Context, userID uuid.UUID) error
	UpdateStatus(ctx context.Context, userID, tenantID uuid.UUID, status string) error
	UpdateRole(ctx context.Context, userID, tenantID uuid.UUID, role string) error

	// 统计
	CountByTenant(ctx context.Context, tenantID uuid.UUID) (int64, error)
	CountByUser(ctx context.Context, userID uuid.UUID) (int64, error)
	ExistsByUserAndTenant(ctx context.Context, userID, tenantID uuid.UUID) (bool, error)
}

// LoginAuditRepository 登录审计日志Repository接口
type LoginAuditRepository interface {
	Create(ctx context.Context, audit *entity.LoginAudit) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.LoginAudit, error)
	ListByUser(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]entity.LoginAudit, int64, error)
	ListByEmail(ctx context.Context, email string, page, pageSize int) ([]entity.LoginAudit, int64, error)
	ListRecent(ctx context.Context, userID uuid.UUID, limit int) ([]entity.LoginAudit, error)
	ListFailed(ctx context.Context, email string, windowMinutes int) ([]entity.LoginAudit, error)
}

// PasswordHistoryRepository 密码历史Repository接口
type PasswordHistoryRepository interface {
	Create(ctx context.Context, history *entity.PasswordHistory) error
	ListRecent(ctx context.Context, userID uuid.UUID, limit int) ([]entity.PasswordHistory, error)
	DeleteOld(ctx context.Context, userID uuid.UUID, keepCount int) error
}

// RefreshTokenRepository Refresh Token Repository接口
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *entity.RefreshToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)
	GetByUserAndDevice(ctx context.Context, userID uuid.UUID, deviceID string) (*entity.RefreshToken, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]entity.RefreshToken, error)
	UpdateLastUsed(ctx context.Context, tokenHash string) error
	Revoke(ctx context.Context, tokenHash string) error
	RevokeAllByUser(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}