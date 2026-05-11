package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
)

// JWTClaims JWT自定义Claims
type JWTClaims struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	TenantID  uuid.UUID `json:"tenant_id,omitempty"`
	Role      string    `json:"role"`
	DeviceID  string    `json:"device_id,omitempty"`
	TokenID   uuid.UUID `json:"token_id"` // 用于黑名单
	jwt.RegisteredClaims
}

// JWTService JWT服务
type JWTService struct {
	secretKey         []byte
	accessTokenExpiry time.Duration
	refreshTokenExpiry time.Duration
	redis             *redis.Client
}

// NewJWTService 创建JWT服务
func NewJWTService(secretKey string, accessTokenExpiry, refreshTokenExpiry time.Duration, redisClient *redis.Client) *JWTService {
	if secretKey == "" {
		panic("JWT secret key is required")
	}
	return &JWTService{
		secretKey:         []byte(secretKey),
		accessTokenExpiry: accessTokenExpiry,
		refreshTokenExpiry: refreshTokenExpiry,
		redis:             redisClient,
	}
}

// GenerateToken 生成access token和refresh token
func (s *JWTService) GenerateToken(userID, tenantID uuid.UUID, email, role, deviceID string) (accessToken, refreshToken string, tokenID uuid.UUID, err error) {
	tokenID = uuid.New()
	now := time.Now()

	// Access token
	accessClaims := JWTClaims{
		UserID:    userID,
		Email:     email,
		TenantID:  tenantID,
		Role:      role,
		DeviceID:  deviceID,
		TokenID:   tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        tokenID.String(),
		},
	}

	accessToken, err = s.generateTokenWithClaims(accessClaims)
	if err != nil {
		return "", "", uuid.Nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Refresh token
	refreshClaims := JWTClaims{
		UserID:    userID,
		Email:     email,
		DeviceID:  deviceID,
		TokenID:   tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        tokenID.String(),
		},
	}

	refreshToken, err = s.generateTokenWithClaims(refreshClaims)
	if err != nil {
		return "", "", uuid.Nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, tokenID, nil
}

// generateTokenWithClaims 根据claims生成token
func (s *JWTService) generateTokenWithClaims(claims JWTClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// ValidateToken 验证access token
func (s *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	// 解析token
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, apperrors.ErrSessionExpired
		}
		return nil, apperrors.ErrTokenInvalid
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, apperrors.ErrTokenInvalid
	}

	// 检查黑名单
	if s.redis != nil {
		tokenHash := s.hashToken(tokenString)
		key := fmt.Sprintf("jwt_blacklist:%s", tokenHash)
		exists, err := s.redis.Exists(context.Background(), key).Result()
		if err == nil && exists > 0 {
			return nil, apperrors.ErrTokenRevoked
		}
	}

	return claims, nil
}

// InvalidateToken 将token加入黑名单
func (s *JWTService) InvalidateToken(tokenString string) error {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		// 即使token无效，也尝试加入黑名单（防止已过期的token被复用）
		if !errors.Is(err, apperrors.ErrSessionExpired) {
			return err
		}
		// 对于过期token，使用默认TTL
		if s.redis != nil {
			tokenHash := s.hashToken(tokenString)
			key := fmt.Sprintf("jwt_blacklist:%s", tokenHash)
			return s.redis.Set(context.Background(), key, "1", 24*time.Hour).Err()
		}
		return nil
	}

	if s.redis == nil {
		return nil
	}

	// 计算剩余有效期作为TTL
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		ttl = 5 * time.Minute // 最小TTL
	}

	tokenHash := s.hashToken(tokenString)
	key := fmt.Sprintf("jwt_blacklist:%s", tokenHash)

	return s.redis.Set(context.Background(), key, "1", ttl).Err()
}

// InvalidateTokenByID 通过TokenID将所有相关token加入黑名单
func (s *JWTService) InvalidateTokenByID(tokenID uuid.UUID, ttl time.Duration) error {
	if s.redis == nil {
		return nil
	}

	key := fmt.Sprintf("jwt_blacklist_id:%s", tokenID.String())
	return s.redis.Set(context.Background(), key, "1", ttl).Err()
}

// IsTokenIDBlacklisted 检查TokenID是否在黑名单中
func (s *JWTService) IsTokenIDBlacklisted(tokenID uuid.UUID) bool {
	if s.redis == nil {
		return false
	}

	key := fmt.Sprintf("jwt_blacklist_id:%s", tokenID.String())
	exists, err := s.redis.Exists(context.Background(), key).Result()
	return err == nil && exists > 0
}

// RefreshToken 使用refresh token刷新access token
func (s *JWTService) RefreshToken(refreshTokenString string, tenantID uuid.UUID, role string) (newAccessToken string, err error) {
	// 验证refresh token
	claims, err := s.ValidateToken(refreshTokenString)
	if err != nil {
		return "", apperrors.ErrRefreshTokenInvalid
	}

	// 检查TokenID黑名单
	if s.IsTokenIDBlacklisted(claims.TokenID) {
		return "", apperrors.ErrTokenRevoked
	}

	// 生成新的access token（使用相同的TokenID，便于批量撤销）
	now := time.Now()
	newClaims := JWTClaims{
		UserID:    claims.UserID,
		Email:     claims.Email,
		TenantID:  tenantID,
		Role:      role,
		DeviceID:  claims.DeviceID,
		TokenID:   claims.TokenID, // 保持相同的TokenID
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        claims.TokenID.String(),
		},
	}

	return s.generateTokenWithClaims(newClaims)
}

// hashToken 对token进行SHA256哈希
func (s *JWTService) hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// GetAccessTokenExpiry 获取access token有效期
func (s *JWTService) GetAccessTokenExpiry() time.Duration {
	return s.accessTokenExpiry
}

// GetRefreshTokenExpiry 获取refresh token有效期
func (s *JWTService) GetRefreshTokenExpiry() time.Duration {
	return s.refreshTokenExpiry
}