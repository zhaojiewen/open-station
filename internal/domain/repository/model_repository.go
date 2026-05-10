package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// ModelRepository defines operations for model pricing management
type ModelRepository interface {
	Create(ctx context.Context, model *entity.Model) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Model, error)
	GetByProviderModel(ctx context.Context, provider, modelID string) (*entity.Model, error)
	Update(ctx context.Context, model *entity.Model) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, provider string) ([]entity.Model, error)
	ListActive(ctx context.Context) ([]entity.Model, error)
	GetPricing(ctx context.Context, provider, modelID string) (*entity.Model, error)
}