package redis

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

var ErrRateLimitExceeded = errors.New("rate limit exceeded")

type limiterEntry struct {
	limiter  *rate.Limiter
	lastUsed time.Time
}

type RateLimitService struct {
	redis        *redis.Client
	localLimit   sync.Map
	keyPrefix    string
	cleanupInt   time.Duration
	entryTTL     time.Duration
	stopCleanup  chan struct{}
}

func NewRateLimitService(redisClient *redis.Client, keyPrefix string) *RateLimitService {
	s := &RateLimitService{
		redis:       redisClient,
		keyPrefix:   keyPrefix,
		cleanupInt:  5 * time.Minute,
		entryTTL:    10 * time.Minute,
		stopCleanup: make(chan struct{}),
	}
	go s.cleanupLoop()
	return s
}

// Stop stops the cleanup goroutine
func (s *RateLimitService) Stop() {
	close(s.stopCleanup)
}

func (s *RateLimitService) cleanupLoop() {
	ticker := time.NewTicker(s.cleanupInt)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCleanup:
			return
		case <-ticker.C:
			s.cleanupStaleEntries()
		}
	}
}

func (s *RateLimitService) cleanupStaleEntries() {
	now := time.Now()
	var toDelete []string

	s.localLimit.Range(func(key, value interface{}) bool {
		if entry, ok := value.(*limiterEntry); ok {
			if now.Sub(entry.lastUsed) > s.entryTTL {
				toDelete = append(toDelete, key.(string))
			}
		}
		return true
	})

	for _, key := range toDelete {
		s.localLimit.Delete(key)
	}
}

func (s *RateLimitService) CheckLimit(ctx context.Context, key string, rps float64, burst int) error {
	localKey := fmt.Sprintf("local:%s", key)
	now := time.Now()

	limiterI, ok := s.localLimit.Load(localKey)
	if !ok {
		entry := &limiterEntry{
			limiter:  rate.NewLimiter(rate.Limit(rps), burst),
			lastUsed: now,
		}
		s.localLimit.Store(localKey, entry)
		limiterI = entry
	}

	entry := limiterI.(*limiterEntry)
	entry.lastUsed = now

	if !entry.limiter.Allow() {
		return ErrRateLimitExceeded
	}

	return s.checkDistributedLimit(ctx, key, int(rps), time.Second)
}

func (s *RateLimitService) checkDistributedLimit(ctx context.Context, key string, limit int, window time.Duration) error {
	redisKey := fmt.Sprintf("%s%s", s.keyPrefix, key)

	script := `
		local current = redis.call('INCR', KEYS[1])
		if current == 1 then
			redis.call('EXPIRE', KEYS[1], ARGV[1])
		end
		return current
	`

	result, err := s.redis.Eval(ctx, script, []string{redisKey}, int(window.Seconds())).Int()
	if err != nil {
		return nil
	}

	if result > limit {
		return ErrRateLimitExceeded
	}

	return nil
}

func (s *RateLimitService) CheckUserLimit(ctx context.Context, userID string, rps float64, burst int) error {
	key := fmt.Sprintf("user:%s", userID)
	return s.CheckLimit(ctx, key, rps, burst)
}

func (s *RateLimitService) CheckTenantLimit(ctx context.Context, tenantID string, rps float64, burst int) error {
	key := fmt.Sprintf("tenant:%s", tenantID)
	return s.CheckLimit(ctx, key, rps, burst)
}

func (s *RateLimitService) CheckAPIKeyLimit(ctx context.Context, apiKeyID string, rps float64, burst int) error {
	key := fmt.Sprintf("apikey:%s", apiKeyID)
	return s.CheckLimit(ctx, key, rps, burst)
}

func (s *RateLimitService) GetLimitStatus(ctx context.Context, key string, limit int, window time.Duration) (int, int, error) {
	redisKey := fmt.Sprintf("%s%s", s.keyPrefix, key)

	current, err := s.redis.Get(ctx, redisKey).Int()
	if err != nil {
		if err == redis.Nil {
			return 0, limit, nil
		}
		return 0, 0, err
	}

	_, err = s.redis.TTL(ctx, redisKey).Result()
	if err != nil {
		return current, 0, err
	}

	remaining := limit - current
	if remaining < 0 {
		remaining = 0
	}

	return current, remaining, nil
}

func (s *RateLimitService) ResetLimit(ctx context.Context, key string) error {
	redisKey := fmt.Sprintf("%s%s", s.keyPrefix, key)
	return s.redis.Del(ctx, redisKey).Err()
}