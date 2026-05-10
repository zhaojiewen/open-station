package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

var ErrIPBlocked = errors.New("IP is blocked")
var ErrIPRateLimitExceeded = errors.New("IP rate limit exceeded")
var ErrTooManyConcurrentConns = errors.New("too many concurrent connections")
var ErrBurstAttackBlocked = errors.New("burst attack detected, IP auto-blocked")
var ErrRateViolationBlocked = errors.New("repeated rate limit violations, IP blocked")

type ipLimiterEntry struct {
	limiter  *rate.Limiter
	lastUsed time.Time
}

type SafeService struct {
	redis         *redis.Client
	localLimiters sync.Map
	keyPrefix     string
	whitelist     map[string]bool
	blacklist     map[string]bool
	cleanupInt    time.Duration
	entryTTL      time.Duration
	stopCleanup   chan struct{}
}

func NewSafeService(redisClient *redis.Client, keyPrefix string, whitelist, blacklist []string) *SafeService {
	wl := make(map[string]bool, len(whitelist))
	for _, ip := range whitelist {
		wl[strings.TrimSpace(ip)] = true
	}
	bl := make(map[string]bool, len(blacklist))
	for _, ip := range blacklist {
		bl[strings.TrimSpace(ip)] = true
	}
	s := &SafeService{
		redis:       redisClient,
		keyPrefix:   keyPrefix,
		whitelist:   wl,
		blacklist:   bl,
		cleanupInt:  5 * time.Minute,
		entryTTL:    10 * time.Minute,
		stopCleanup: make(chan struct{}),
	}
	go s.cleanupLoop()
	return s
}

// Stop stops the cleanup goroutine
func (s *SafeService) Stop() {
	close(s.stopCleanup)
}

func (s *SafeService) cleanupLoop() {
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

func (s *SafeService) cleanupStaleEntries() {
	now := time.Now()
	var toDelete []string

	s.localLimiters.Range(func(key, value interface{}) bool {
		if entry, ok := value.(*ipLimiterEntry); ok {
			if now.Sub(entry.lastUsed) > s.entryTTL {
				toDelete = append(toDelete, key.(string))
			}
		}
		return true
	})

	for _, key := range toDelete {
		s.localLimiters.Delete(key)
	}
}

func (s *SafeService) IsWhitelisted(ip string) bool {
	return s.whitelist[ip]
}

func (s *SafeService) IsBlocked(ctx context.Context, ip string) (bool, error) {
	if s.blacklist[ip] {
		return true, nil
	}

	blockedKey := fmt.Sprintf("%sblocked:%s", s.keyPrefix, ip)
	exists, err := s.redis.Exists(ctx, blockedKey).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (s *SafeService) CheckIPRateLimit(ctx context.Context, ip string, rps, burst int) error {
	localKey := fmt.Sprintf("local:ip:%s", ip)
	now := time.Now()

	limiterI, ok := s.localLimiters.Load(localKey)
	if !ok {
		entry := &ipLimiterEntry{
			limiter:  rate.NewLimiter(rate.Limit(rps), burst),
			lastUsed: now,
		}
		s.localLimiters.Store(localKey, entry)
		limiterI = entry
	}

	entry := limiterI.(*ipLimiterEntry)
	entry.lastUsed = now

	if !entry.limiter.Allow() {
		return ErrIPRateLimitExceeded
	}

	return s.checkDistributedIPLimit(ctx, ip, rps)
}

func (s *SafeService) checkDistributedIPLimit(ctx context.Context, ip string, limit int) error {
	redisKey := fmt.Sprintf("%sip_rl:%s", s.keyPrefix, ip)

	script := `
		local current = redis.call('INCR', KEYS[1])
		if current == 1 then
			redis.call('EXPIRE', KEYS[1], ARGV[1])
		end
		return current
	`

	result, err := s.redis.Eval(ctx, script, []string{redisKey}, 1).Int()
	if err != nil {
		return nil
	}

	if result > limit {
		return ErrIPRateLimitExceeded
	}

	return nil
}

func (s *SafeService) RecordAuthFailure(ctx context.Context, ip string, windowS, maxAttempts, blockDurationS int) (bool, error) {
	counterKey := fmt.Sprintf("%sauth_fail:%s", s.keyPrefix, ip)
	blockedKey := fmt.Sprintf("%sblocked:%s", s.keyPrefix, ip)

	script := `
		local current = redis.call('INCR', KEYS[1])
		if current == 1 then
			redis.call('EXPIRE', KEYS[1], ARGV[1])
		end
		if current >= tonumber(ARGV[2]) then
			redis.call('SET', KEYS[2], '1', 'EX', ARGV[3])
			redis.call('DEL', KEYS[1])
			return 1
		end
		return 0
	`

	result, err := s.redis.Eval(ctx, script,
		[]string{counterKey, blockedKey},
		windowS, maxAttempts, blockDurationS,
	).Int()
	if err != nil {
		return false, err
	}

	return result == 1, nil
}

func (s *SafeService) ResetAuthFailures(ctx context.Context, ip string) error {
	counterKey := fmt.Sprintf("%sauth_fail:%s", s.keyPrefix, ip)
	return s.redis.Del(ctx, counterKey).Err()
}

func (s *SafeService) AcquireConnection(ctx context.Context, ip string, maxConns int) (bool, func(), error) {
	if maxConns <= 0 {
		return true, func() {}, nil
	}

	connKey := fmt.Sprintf("%sconn:%s", s.keyPrefix, ip)

	script := `
		local current = redis.call('INCR', KEYS[1])
		redis.call('EXPIRE', KEYS[1], 60)
		if current > tonumber(ARGV[1]) then
			redis.call('DECR', KEYS[1])
			return 0
		end
		return 1
	`

	result, err := s.redis.Eval(ctx, script, []string{connKey}, maxConns).Int()
	if err != nil {
		return true, func() {}, nil
	}

	release := func() {
		s.redis.Decr(ctx, connKey)
	}

	return result == 1, release, nil
}

func (s *SafeService) RecordRateViolation(ctx context.Context, ip string, windowS, maxViolations, blockDurationS int) (bool, error) {
	counterKey := fmt.Sprintf("%srate_viol:%s", s.keyPrefix, ip)
	blockedKey := fmt.Sprintf("%sblocked:%s", s.keyPrefix, ip)

	script := `
		local current = redis.call('INCR', KEYS[1])
		if current == 1 then
			redis.call('EXPIRE', KEYS[1], ARGV[1])
		end
		if current >= tonumber(ARGV[2]) then
			redis.call('SET', KEYS[2], '1', 'EX', ARGV[3])
			redis.call('DEL', KEYS[1])
			return 1
		end
		return 0
	`

	result, err := s.redis.Eval(ctx, script,
		[]string{counterKey, blockedKey},
		windowS, maxViolations, blockDurationS,
	).Int()
	if err != nil {
		return false, err
	}

	return result == 1, nil
}

func (s *SafeService) AutoBlockIP(ctx context.Context, ip string, blockDurationS int) error {
	blockedKey := fmt.Sprintf("%sblocked:%s", s.keyPrefix, ip)
	return s.redis.Set(ctx, blockedKey, "1", time.Duration(blockDurationS)*time.Second).Err()
}

func (s *SafeService) GetIPRateCount(ctx context.Context, ip string) (int, error) {
	redisKey := fmt.Sprintf("%sip_rl:%s", s.keyPrefix, ip)
	val, err := s.redis.Get(ctx, redisKey).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}
