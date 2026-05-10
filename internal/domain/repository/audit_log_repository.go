package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// AuditLogRepository defines operations for audit logging
type AuditLogRepository interface {
	Create(ctx context.Context, log *entity.AuditLog) error
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.AuditLog, int64, error)
}