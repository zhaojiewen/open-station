package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	"github.com/zhaojiewen/open-station/pkg/logger"
	"go.uber.org/zap"
)

// CacheWarmupService preloads active API keys into Redis cache at startup
type CacheWarmupService struct {
	apiKeyRepo repository.APIKeyRepository
	redis      *redis.Client
	cacheTTL   time.Duration
}

// NewCacheWarmupService creates a new cache warmup service
func NewCacheWarmupService(
	apiKeyRepo repository.APIKeyRepository,
	redisClient *redis.Client,
) *CacheWarmupService {
	return &CacheWarmupService{
		apiKeyRepo: apiKeyRepo,
		redis:      redisClient,
		cacheTTL:   5 * time.Minute,
	}
}

// WarmupAPIKeys loads all active API keys into Redis cache
// This reduces database hits during the initial burst of requests after startup
func (s *CacheWarmupService) WarmupAPIKeys(ctx context.Context) error {
	logger.Info("starting API key cache warmup")

	// Get all active API keys
	keys, err := s.apiKeyRepo.ListAll(ctx)
	if err != nil {
		logger.Error("failed to list API keys for warmup", zap.Error(err))
		return fmt.Errorf("failed to list API keys: %w", err)
	}

	activeCount := 0
	for _, key := range keys {
		// Only cache active keys
		if key.Status != "active" {
			continue
		}

		cacheKey := fmt.Sprintf("apikey:%s", key.KeyHash)
		err := s.redis.Set(ctx, cacheKey, key.ID.String(), s.cacheTTL).Err()
		if err != nil {
			logger.Warn("failed to cache API key",
				zap.String("key_prefix", key.KeyPrefix),
				zap.Error(err),
			)
			continue
		}

		activeCount++
	}

	logger.Info("API key cache warmup completed",
		zap.Int("total_keys", len(keys)),
		zap.Int("cached_keys", activeCount),
	)

	return nil
}

// WarmupTenantCache loads tenant balances into cache
func (s *CacheWarmupService) WarmupTenantCache(ctx context.Context, tenantRepo repository.TenantRepository) error {
	logger.Info("starting tenant cache warmup")

	// We don't need to cache all tenants - just ensure Redis connection is ready
	// Tenant data is queried per-request and cached on first access

	// Verify Redis connectivity
	if err := s.redis.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	logger.Info("tenant cache warmup completed")
	return nil
}

// StartBackgroundRefresh starts a background goroutine that periodically refreshes the cache
func (s *CacheWarmupService) StartBackgroundRefresh(ctx context.Context, interval time.Duration) chan struct{} {
	stopCh := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				logger.Info("cache background refresh stopped")
				return
			case <-ticker.C:
				// Use a fresh context for each refresh
				refreshCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if err := s.WarmupAPIKeys(refreshCtx); err != nil {
					logger.Warn("background cache refresh failed", zap.Error(err))
				}
				cancel()
			case <-ctx.Done():
				logger.Info("cache background refresh stopped (context done)")
				return
			}
		}
	}()

	logger.Info("cache background refresh started", zap.Duration("interval", interval))
	return stopCh
}

// GetCacheStats returns statistics about cached API keys
func (s *CacheWarmupService) GetCacheStats(ctx context.Context) (int64, error) {
	// Count keys with prefix "apikey:"
	keys, err := s.redis.Keys(ctx, "apikey:*").Result()
	if err != nil {
		return 0, err
	}
	return int64(len(keys)), nil
}