package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// UserRepository defines operations for user management
type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]entity.User, int64, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}