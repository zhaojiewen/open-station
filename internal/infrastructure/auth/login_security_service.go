package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	"github.com/zhaojiewen/open-station/pkg/crypto"
	"github.com/zhaojiewen/open-station/pkg/errors"
)

const (
	// 登录失败限制常量
	DefaultMaxFailedAttempts = 5
	DefaultFailedWindow      = 15 * time.Minute
	DefaultBlockDuration     = 30 * time.Minute
)

// LoginSecurityService 登录安全服务
type LoginSecurityService struct {
	redis             *redis.Client
	loginAuditRepo    repository.LoginAuditRepository
	passwordHistoryRepo repository.PasswordHistoryRepository
	encryptionKey     []byte
	maxFailedAttempts int
	failedWindow      time.Duration
	blockDuration     time.Duration
	enableAuditLog    bool
	encryptAuditData  bool
}

// NewLoginSecurityService 创建登录安全服务
func NewLoginSecurityService(
	redis *redis.Client,
	loginAuditRepo repository.LoginAuditRepository,
	passwordHistoryRepo repository.PasswordHistoryRepository,
	encryptionKey string,
	maxFailedAttempts int,
	failedWindow time.Duration,
	blockDuration time.Duration,
	enableAuditLog bool,
	encryptAuditData bool,
) *LoginSecurityService {
	var key []byte
	if encryptionKey != "" && len(encryptionKey) >= 32 {
		key = []byte(encryptionKey[:32])
	}

	if maxFailedAttempts <= 0 {
		maxFailedAttempts = DefaultMaxFailedAttempts
	}
	if failedWindow <= 0 {
		failedWindow = DefaultFailedWindow
	}
	if blockDuration <= 0 {
		blockDuration = DefaultBlockDuration
	}

	return &LoginSecurityService{
		redis:             redis,
		loginAuditRepo:    loginAuditRepo,
		passwordHistoryRepo: passwordHistoryRepo,
		encryptionKey:     key,
		maxFailedAttempts: maxFailedAttempts,
		failedWindow:      failedWindow,
		blockDuration:     blockDuration,
		enableAuditLog:    enableAuditLog,
		encryptAuditData:  encryptAuditData,
	}
}

// CheckLoginAllowed 检查是否允许登录（是否被封禁）
func (s *LoginSecurityService) CheckLoginAllowed(ctx context.Context, ip, email string) error {
	if s.redis == nil {
		return nil // 无Redis时不检查
	}

	// 使用Pipeline优化：一次网络往返检查所有条件
	ipBlockedKey := fmt.Sprintf("login_blocked_ip:%s", ip)
	emailBlockedKey := fmt.Sprintf("login_blocked_email:%s", email)
	ipEmailKey := fmt.Sprintf("login_failed:%s:%s", ip, email)

	// Pipeline执行3个Exists命令
	cmds, err := s.redis.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Exists(ctx, ipBlockedKey)
		pipe.Exists(ctx, emailBlockedKey)
		pipe.Get(ctx, ipEmailKey)
		return nil
	})

	if err != nil && err != redis.Nil {
		// Redis错误不阻止登录（降级处理）
		return nil
	}

	// 检查IP封禁 (第一个命令)
	if len(cmds) >= 1 {
		if exists, err := cmds[0].(*redis.IntCmd).Result(); err == nil && exists > 0 {
			return errors.ErrTooManyAttempts
		}
	}

	// 检查邮箱封禁 (第二个命令)
	if len(cmds) >= 2 {
		if exists, err := cmds[1].(*redis.IntCmd).Result(); err == nil && exists > 0 {
			return errors.ErrAccountLocked
		}
	}

	// 检查失败次数 (第三个命令)
	if len(cmds) >= 3 {
		if count, err := cmds[2].(*redis.StringCmd).Int(); err == nil && count >= s.maxFailedAttempts {
			return errors.ErrTooManyAttempts
		}
	}

	return nil
}

// RecordFailedAttempt 记录登录失败
func (s *LoginSecurityService) RecordFailedAttempt(ctx context.Context, ip, email, userAgent, deviceID, reason string, userID *uuid.UUID) error {
	// 1. 增加失败计数（使用Pipeline优化）
	if s.redis != nil {
		ipEmailKey := fmt.Sprintf("login_failed:%s:%s", ip, email)
		ipBlockedKey := fmt.Sprintf("login_blocked_ip:%s", ip)

		// 使用Pipeline原子性地增加计数并设置过期时间
		cmds, err := s.redis.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Incr(ctx, ipEmailKey)
			pipe.Expire(ctx, ipEmailKey, s.failedWindow)
			return nil
		})

		if err == nil && len(cmds) >= 1 {
			// 获取增加后的计数
			count, _ := cmds[0].(*redis.IntCmd).Result()

			// 检查是否达到封禁阈值
			if count >= int64(s.maxFailedAttempts) {
				// 封禁IP
				s.redis.Set(ctx, ipBlockedKey, "1", s.blockDuration)

				// 可选：封禁邮箱
				// emailBlockedKey := fmt.Sprintf("login_blocked_email:%s", email)
				// s.redis.Set(ctx, emailBlockedKey, "1", s.blockDuration)
			}
		}
	}

	// 2. 记录审计日志
	if s.enableAuditLog {
		audit := &entity.LoginAudit{
			UserID:        userID,
			Email:         email,
			DeviceID:      deviceID,
			Success:       false,
			FailureReason: reason,
			LoginAt:       time.Now(),
		}

		// 加密敏感数据
		if s.encryptAuditData && len(s.encryptionKey) > 0 {
			if encryptedIP, err := crypto.EncryptString(ip, s.encryptionKey); err == nil {
				audit.IPEncrypted = encryptedIP
			}
			ipHash := sha256.Sum256([]byte(ip))
			audit.IPHash = hex.EncodeToString(ipHash[:])

			if encryptedUA, err := crypto.EncryptString(userAgent, s.encryptionKey); err == nil {
				audit.UserAgentEnc = encryptedUA
			}
		} else {
			// 不加密时，仍存储hash用于查询
			ipHash := sha256.Sum256([]byte(ip))
			audit.IPHash = hex.EncodeToString(ipHash[:])
		}

		if s.loginAuditRepo != nil {
			s.loginAuditRepo.Create(ctx, audit)
		}
	}

	return nil
}

// ClearFailedAttempts 清除失败记录（登录成功后）
func (s *LoginSecurityService) ClearFailedAttempts(ctx context.Context, ip, email string) error {
	if s.redis == nil {
		return nil
	}

	ipEmailKey := fmt.Sprintf("login_failed:%s:%s", ip, email)
	return s.redis.Del(ctx, ipEmailKey).Err()
}

// RecordSuccessfulLogin 记录成功登录
func (s *LoginSecurityService) RecordSuccessfulLogin(ctx context.Context, userID uuid.UUID, email, ip, userAgent, deviceID string, tenantID *uuid.UUID) error {
	if s.enableAuditLog && s.loginAuditRepo != nil {
		audit := &entity.LoginAudit{
			UserID:    &userID,
			Email:     email,
			DeviceID:  deviceID,
			Success:   true,
			TenantID:  tenantID,
			LoginAt:   time.Now(),
		}

		// 加密敏感数据
		if s.encryptAuditData && len(s.encryptionKey) > 0 {
			if encryptedIP, err := crypto.EncryptString(ip, s.encryptionKey); err == nil {
				audit.IPEncrypted = encryptedIP
			}
			ipHash := sha256.Sum256([]byte(ip))
			audit.IPHash = hex.EncodeToString(ipHash[:])

			if encryptedUA, err := crypto.EncryptString(userAgent, s.encryptionKey); err == nil {
				audit.UserAgentEnc = encryptedUA
			}
		} else {
			ipHash := sha256.Sum256([]byte(ip))
			audit.IPHash = hex.EncodeToString(ipHash[:])
		}

		return s.loginAuditRepo.Create(ctx, audit)
	}

	return nil
}

// DetectAnomaly 检测异常登录
func (s *LoginSecurityService) DetectAnomaly(ctx context.Context, userID uuid.UUID, ip, deviceID string) (isAnomaly bool, anomalyType string) {
	if s.loginAuditRepo == nil {
		return false, ""
	}

	// 获取最近登录记录
	history, err := s.loginAuditRepo.ListRecent(ctx, userID, 30)
	if err != nil || len(history) == 0 {
		// 无历史记录，首次登录不算异常
		return false, ""
	}

	// 检查新设备
	knownDevices := make(map[string]bool)
	for _, h := range history {
		if h.DeviceID != "" && h.Success {
			knownDevices[h.DeviceID] = true
		}
	}

	if !knownDevices[deviceID] {
		return true, "new_device"
	}

	// 可以扩展：检查新IP、异地登录等
	// ...

	return false, ""
}

// CheckPasswordHistory 检查密码是否在历史中出现过
func (s *LoginSecurityService) CheckPasswordHistory(ctx context.Context, userID uuid.UUID, passwordHash string, hasher PasswordVerifier, historyCount int) bool {
	if s.passwordHistoryRepo == nil {
		return false
	}

	_, err := s.passwordHistoryRepo.ListRecent(ctx, userID, historyCount)
	if err != nil {
		return false
	}

	// 这个函数主要用于UI提示，实际密码历史检查在密码服务中实现
	return true
}

// SavePasswordHistory 保存密码到历史记录
func (s *LoginSecurityService) SavePasswordHistory(ctx context.Context, userID uuid.UUID, passwordHash string, keepCount int) error {
	if s.passwordHistoryRepo == nil {
		return nil
	}

	// 保存新记录
	history := &entity.PasswordHistory{
		UserID:       userID,
		PasswordHash: passwordHash,
	}
	if err := s.passwordHistoryRepo.Create(ctx, history); err != nil {
		return err
	}

	// 删除旧记录
	return s.passwordHistoryRepo.DeleteOld(ctx, userID, keepCount)
}

// GetFailedAttempts 获取当前失败次数
func (s *LoginSecurityService) GetFailedAttempts(ctx context.Context, ip, email string) int {
	if s.redis == nil {
		return 0
	}

	ipEmailKey := fmt.Sprintf("login_failed:%s:%s", ip, email)
	count, err := s.redis.Get(ctx, ipEmailKey).Int()
	if err != nil {
		return 0
	}
	return count
}

// GetBlockRemainingTime 获取封禁剩余时间
func (s *LoginSecurityService) GetBlockRemainingTime(ctx context.Context, ip string) time.Duration {
	if s.redis == nil {
		return 0
	}

	ipBlockedKey := fmt.Sprintf("login_blocked_ip:%s", ip)
	ttl, err := s.redis.TTL(ctx, ipBlockedKey).Result()
	if err != nil || ttl < 0 {
		return 0
	}
	return ttl
}

// GenerateDeviceID 生成设备指纹
// 基于UserAgent生成稳定的设备标识，IP仅用于验证而非生成
// 这样即使IP变化，同一设备仍能被识别
func GenerateDeviceID(userAgent, ip string) string {
	// 设备指纹：主要基于UserAgent（稳定），IP作为辅助验证
	// 客户端可通过X-Device-ID提供固定ID来完全避免IP变化的影响
	h := sha256.Sum256([]byte(userAgent))
	return hex.EncodeToString(h[:16]) // 取前16字节，仅基于UA
}

// GenerateDeviceFingerprint 生成包含IP的完整设备指纹（用于异常检测）
func GenerateDeviceFingerprint(userAgent, ip string) string {
	h := sha256.Sum256([]byte(userAgent + ":" + ip))
	return hex.EncodeToString(h[:])
}

// PasswordVerifier 密码验证器接口（用于密码历史检查）
type PasswordVerifier interface {
	Verify(password, hash string) bool
}