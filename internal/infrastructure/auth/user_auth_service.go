package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
	"github.com/zhaojiewen/open-station/internal/domain/repository"
	apperrors "github.com/zhaojiewen/open-station/pkg/errors"
	"github.com/zhaojiewen/open-station/pkg/password"
	"gorm.io/gorm"
)

// UserAuthService 用户认证服务
type UserAuthService struct {
	userRepo           repository.UserRepository
	tenantRepo         repository.TenantRepository
	userTenantRepo     repository.UserTenantRepository
	refreshTokenRepo   repository.RefreshTokenRepository
	jwtService         *JWTService
	loginSecurity      *LoginSecurityService
	passwordHasher     *password.PasswordHasher
	redis              *redis.Client
	publicTenantSlug   string
	db                 *gorm.DB // 添加DB用于事务
	emailVerification  *EmailVerificationService
	requireEmailVerify bool // 是否需要邮箱验证
}

// NewUserAuthService 创建用户认证服务
func NewUserAuthService(
	userRepo repository.UserRepository,
	tenantRepo repository.TenantRepository,
	userTenantRepo repository.UserTenantRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtService *JWTService,
	loginSecurity *LoginSecurityService,
	passwordHasher *password.PasswordHasher,
	redis *redis.Client,
	publicTenantSlug string,
	db *gorm.DB,
	emailVerification *EmailVerificationService,
	requireEmailVerify bool,
) *UserAuthService {
	return &UserAuthService{
		userRepo:           userRepo,
		tenantRepo:         tenantRepo,
		userTenantRepo:     userTenantRepo,
		refreshTokenRepo:   refreshTokenRepo,
		jwtService:         jwtService,
		loginSecurity:      loginSecurity,
		passwordHasher:     passwordHasher,
		redis:              redis,
		publicTenantSlug:   publicTenantSlug,
		db:                 db,
		emailVerification:  emailVerification,
		requireEmailVerify: requireEmailVerify,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string
	Password string
	IP       string
	UserAgent string
	DeviceID string
}

// LoginResponse 登录响应
type LoginResponse struct {
	User            *entity.User
	UserTenants     []entity.UserTenant
	DefaultTenantID uuid.UUID
	AccessToken     string
	RefreshToken    string
	ExpiresAt       time.Time
	IsAnomaly       bool
	AnomalyType     string
}

// Login 用户登录
func (s *UserAuthService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// 1. 检查是否允许登录
	if err := s.loginSecurity.CheckLoginAllowed(ctx, req.IP, req.Email); err != nil {
		return nil, err
	}

	// 2. 查找用户
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		// 用户不存在也记录失败（防止枚举攻击）
		s.loginSecurity.RecordFailedAttempt(ctx, req.IP, req.Email, req.UserAgent, req.DeviceID, "invalid_credentials", nil)
		// 执行一次假的密码哈希验证，防止通过响应时间推断用户是否存在（时序攻击防护）
		s.passwordHasher.Verify("dummy_password", "dummy_hash_for_timing")
		return nil, apperrors.ErrInvalidCredentials
	}

	// 3. 检查用户状态
	if user.Status != "active" {
		// 如果需要邮箱验证且用户未验证
		if s.requireEmailVerify && user.Status == "pending_verification" {
			s.loginSecurity.RecordFailedAttempt(ctx, req.IP, req.Email, req.UserAgent, req.DeviceID, "email_not_verified", &user.ID)
			return nil, apperrors.ErrEmailNotVerified
		}
		s.loginSecurity.RecordFailedAttempt(ctx, req.IP, req.Email, req.UserAgent, req.DeviceID, "user_inactive", &user.ID)
		return nil, apperrors.ErrUserInactive
	}

	// 3.5. 检查邮箱验证状态（如果需要）
	if s.requireEmailVerify && !user.EmailVerified {
		s.loginSecurity.RecordFailedAttempt(ctx, req.IP, req.Email, req.UserAgent, req.DeviceID, "email_not_verified", &user.ID)
		return nil, apperrors.ErrEmailNotVerified
	}

	// 4. 验证密码
	if !s.passwordHasher.Verify(req.Password, user.PasswordHash) {
		s.loginSecurity.RecordFailedAttempt(ctx, req.IP, req.Email, req.UserAgent, req.DeviceID, "invalid_password", &user.ID)
		return nil, apperrors.ErrInvalidCredentials
	}

	// 5. 检查密码是否需要升级（异步执行，不阻塞登录）
	if s.passwordHasher.NeedsRehash(user.PasswordHash) {
		go func() {
			// 使用新的context避免被取消
			asyncCtx := context.Background()
			newHash, rehashErr := s.passwordHasher.Hash(req.Password)
			if rehashErr == nil {
				userCopy := *user
				userCopy.PasswordHash = newHash
				if updateErr := s.userRepo.Update(asyncCtx, &userCopy); updateErr != nil {
					// 密码升级失败不影响登录，但应该记录日志
					// 在生产环境中应该使用日志记录此警告
				}
			}
		}()
	}

	// 6. 清除失败记录
	s.loginSecurity.ClearFailedAttempts(ctx, req.IP, req.Email)

	// 7. 获取用户的租户列表
	userTenants, err := s.userTenantRepo.ListByUser(ctx, user.ID)
	if err != nil || len(userTenants) == 0 {
		s.loginSecurity.RecordFailedAttempt(ctx, req.IP, req.Email, req.UserAgent, req.DeviceID, "no_tenant", &user.ID)
		return nil, fmt.Errorf("user has no tenant")
	}

	// 8. 获取默认租户
	defaultTenant, err := s.userTenantRepo.GetDefaultTenant(ctx, user.ID)
	if err != nil || defaultTenant == nil {
		// 如果没有默认租户，使用第一个
		defaultTenant = &userTenants[0]
	}

	// 9. 检查租户状态
	tenant, err := s.tenantRepo.GetByID(ctx, defaultTenant.TenantID)
	if err != nil || tenant.Status != "active" {
		return nil, apperrors.ErrTenantSuspended
	}

	// 10. 生成设备指纹（如果没有提供）
	deviceID := req.DeviceID
	if deviceID == "" {
		deviceID = GenerateDeviceID(req.UserAgent, req.IP)
	}

	// 11. 异常检测
	isAnomaly, anomalyType := s.loginSecurity.DetectAnomaly(ctx, user.ID, req.IP, deviceID)

	// 12. 生成JWT token
	accessToken, refreshToken, _, err := s.jwtService.GenerateToken(
		user.ID,
		defaultTenant.TenantID,
		user.Email,
		defaultTenant.Role,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// 13. 存储refresh token
	refreshTokenHash := hashToken(refreshToken)
	deviceInfo := map[string]string{
		"user_agent": req.UserAgent,
		"ip":         req.IP,
	}
	deviceInfoJSON, _ := json.Marshal(deviceInfo)

	refreshTokenRecord := &entity.RefreshToken{
		UserID:     user.ID,
		TokenHash:  refreshTokenHash,
		DeviceID:   deviceID,
		DeviceInfo: string(deviceInfoJSON),
		ExpiresAt:  time.Now().Add(s.jwtService.GetRefreshTokenExpiry()),
	}
	if err := s.refreshTokenRepo.Create(ctx, refreshTokenRecord); err != nil {
		// Refresh token存储失败不应该阻止登录，但应该记录
		// 后续刷新token操作可能会失败
	}

	// 14. 更新最后登录时间
	now := time.Now()
	user.LastLoginAt = &now
	if err := s.userRepo.Update(ctx, user); err != nil {
		// 最后登录时间更新失败不影响登录流程
	}

	// 15. 记录成功登录
	s.loginSecurity.RecordSuccessfulLogin(ctx, user.ID, user.Email, req.IP, req.UserAgent, deviceID, &defaultTenant.TenantID)

	return &LoginResponse{
		User:            user,
		UserTenants:     userTenants,
		DefaultTenantID: defaultTenant.TenantID,
		AccessToken:     accessToken,
		RefreshToken:    refreshToken,
		ExpiresAt:       time.Now().Add(s.jwtService.GetAccessTokenExpiry()),
		IsAnomaly:       isAnomaly,
		AnomalyType:     anomalyType,
	}, nil
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email     string
	Password  string
	Name      string
	IP        string
	UserAgent string
}

// RegisterResponse 注册响应
type RegisterResponse struct {
	User        *entity.User
	UserTenant  *entity.UserTenant
	AccessToken string
	RefreshToken string
	ExpiresAt   time.Time
}

// Register 个人注册（加入公共租户）
func (s *UserAuthService) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	// 1. 验证邮箱格式
	if !isValidEmail(req.Email) {
		return nil, apperrors.ErrInvalidEmailFormat
	}

	// 2. 验证密码复杂度
	if err := password.ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	// 3. 检查邮箱是否已注册
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		// 如果用户已存在但未验证，允许重新发送验证邮件
		if !existingUser.EmailVerified && s.requireEmailVerify {
			if err := s.emailVerification.SendVerificationEmail(ctx, existingUser); err != nil {
				return nil, fmt.Errorf("failed to send verification email: %w", err)
			}
			return &RegisterResponse{
				User:        existingUser,
				UserTenant:  nil,
				AccessToken: "",
				RefreshToken: "",
				ExpiresAt:   time.Time{},
			}, nil
		}
		return nil, apperrors.ErrEmailExists
	}

	// 4. 获取公共租户
	publicTenant, err := s.tenantRepo.GetBySlug(ctx, s.publicTenantSlug)
	if err != nil {
		return nil, fmt.Errorf("public tenant not found: %w", err)
	}

	// 5. 创建用户
	userID := uuid.New()
	passwordHash, err := s.passwordHasher.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	user := &entity.User{
		ID:           userID,
		TenantID:     publicTenant.ID, // 公共租户作为主租户
		Email:        req.Email,
		PasswordHash: passwordHash,
		Name:         req.Name,
		Role:         "member",
		Status:       "pending_verification", // 需要邮箱验证
		UserMode:     "individual",
		EmailVerified: false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	userTenant := &entity.UserTenant{
		UserID:    userID,
		TenantID:  publicTenant.ID,
		Role:      "member",
		Status:    "active",
		IsDefault: true,
		JoinedAt:  now,
	}

	// 使用事务确保原子性
	if s.db != nil {
		err = s.db.Transaction(func(tx *gorm.DB) error {
			// 创建用户
			if err := tx.Create(user).Error; err != nil {
				return fmt.Errorf("failed to create user: %w", err)
			}

			// 创建UserTenant关联
			if err := tx.Create(userTenant).Error; err != nil {
				return fmt.Errorf("failed to create user tenant: %w", err)
			}

			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		// 无事务时的fallback（保持向后兼容）
		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		if err := s.userTenantRepo.Create(ctx, userTenant); err != nil {
			return nil, fmt.Errorf("failed to create user tenant: %w", err)
		}
	}

	// 6. 发送验证邮件（如果需要）
	if s.requireEmailVerify && s.emailVerification != nil {
		if err := s.emailVerification.SendVerificationEmail(ctx, user); err != nil {
			// 发送失败不阻止注册，但返回提示
			return &RegisterResponse{
				User:         user,
				UserTenant:   userTenant,
				AccessToken:  "",
				RefreshToken: "",
				ExpiresAt:    time.Time{},
			}, nil
		}
	}

	// 7. 如果不需要邮箱验证，直接生成token
	if !s.requireEmailVerify {
		deviceID := GenerateDeviceID(req.UserAgent, req.IP)
		accessToken, refreshToken, _, err := s.jwtService.GenerateToken(
			user.ID,
			publicTenant.ID,
			user.Email,
			"member",
			deviceID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to generate token: %w", err)
		}

		refreshTokenHash := hashToken(refreshToken)
		refreshTokenRecord := &entity.RefreshToken{
			UserID:     user.ID,
			TokenHash:  refreshTokenHash,
			DeviceID:   deviceID,
			ExpiresAt:  time.Now().Add(s.jwtService.GetRefreshTokenExpiry()),
		}
		if err := s.refreshTokenRepo.Create(ctx, refreshTokenRecord); err != nil {
			// 记录错误但不阻止
		}

		s.loginSecurity.RecordSuccessfulLogin(ctx, user.ID, user.Email, req.IP, req.UserAgent, deviceID, &publicTenant.ID)

		return &RegisterResponse{
			User:         user,
			UserTenant:   userTenant,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresAt:    time.Now().Add(s.jwtService.GetAccessTokenExpiry()),
		}, nil
	}

	// 需要验证，返回无token的响应
	return &RegisterResponse{
		User:         user,
		UserTenant:   userTenant,
		AccessToken:  "",
		RefreshToken: "",
		ExpiresAt:    time.Time{},
	}, nil
}

// RegisterTenantRequest 企业注册请求
type RegisterTenantRequest struct {
	TenantName string
	TenantSlug string
	Email      string
	Password   string
	Name       string
	IP         string
	UserAgent  string
}

// RegisterTenantResponse 企业注册响应
type RegisterTenantResponse struct {
	Tenant      *entity.Tenant
	User        *entity.User
	UserTenant  *entity.UserTenant
	AccessToken string
	RefreshToken string
	ExpiresAt   time.Time
}

// RegisterTenant 企业注册（创建新租户）
func (s *UserAuthService) RegisterTenant(ctx context.Context, req *RegisterTenantRequest) (*RegisterTenantResponse, error) {
	// 1. 验证邮箱格式
	if !isValidEmail(req.Email) {
		return nil, apperrors.ErrInvalidEmailFormat
	}

	// 2. 验证密码复杂度
	if err := password.ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	// 3. 验证租户slug
	if !isValidSlug(req.TenantSlug) {
		return nil, fmt.Errorf("invalid tenant slug format")
	}

	// 4. 检查邮箱是否已注册
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		// 邮箱已存在，检查是否可以加入新租户
		// 这里允许一个用户拥有多个租户
	}

	// 5. 检查租户slug是否已存在
	existingTenant, err := s.tenantRepo.GetBySlug(ctx, req.TenantSlug)
	if err == nil && existingTenant != nil {
		return nil, apperrors.ErrTenantSlugExists
	}

	// 6. 创建租户
	tenantID := uuid.New()
	tenant := &entity.Tenant{
		ID:               tenantID,
		Name:             req.TenantName,
		Slug:             req.TenantSlug,
		Status:           "active",
		Plan:             "free",
		Type:             "organization",
		Balance:          decimal.Zero,
		Currency:         "USD",
		MaxUsers:         10,
		MaxAPIKeysPerUser: 5,
		RateLimitRPS:     100,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// 7. 创建或获取用户
	var user *entity.User
	if existingUser != nil {
		user = existingUser
	} else {
		userID := uuid.New()
		passwordHash, err := s.passwordHasher.Hash(req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}

		user = &entity.User{
			ID:           userID,
			TenantID:     tenantID,
			Email:        req.Email,
			PasswordHash: passwordHash,
			Name:         req.Name,
			Role:         "admin",
			Status:       "active",
			UserMode:     "organization",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	// 8. 创建UserTenant关联（管理员角色）
	userTenant := &entity.UserTenant{
		UserID:    user.ID,
		TenantID:  tenantID,
		Role:      "admin",
		Status:    "active",
		IsDefault: true,
		JoinedAt:  time.Now(),
	}
	if err := s.userTenantRepo.Create(ctx, userTenant); err != nil {
		return nil, fmt.Errorf("failed to create user tenant: %w", err)
	}

	// 9. 清除用户的其他默认租户标记
	s.userTenantRepo.ClearDefaultTenants(ctx, user.ID)
	s.userTenantRepo.SetDefaultTenant(ctx, user.ID, tenantID)

	// 10. 生成设备指纹
	deviceID := GenerateDeviceID(req.UserAgent, req.IP)

	// 11. 生成JWT token
	accessToken, refreshToken, _, err := s.jwtService.GenerateToken(
		user.ID,
		tenantID,
		user.Email,
		"admin",
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// 12. 存储refresh token
	refreshTokenHash := hashToken(refreshToken)
	refreshTokenRecord := &entity.RefreshToken{
		UserID:     user.ID,
		TokenHash:  refreshTokenHash,
		DeviceID:   deviceID,
		ExpiresAt:  time.Now().Add(s.jwtService.GetRefreshTokenExpiry()),
	}
	s.refreshTokenRepo.Create(ctx, refreshTokenRecord)

	// 13. 记录成功登录
	s.loginSecurity.RecordSuccessfulLogin(ctx, user.ID, user.Email, req.IP, req.UserAgent, deviceID, &tenantID)

	return &RegisterTenantResponse{
		Tenant:       tenant,
		User:         user,
		UserTenant:   userTenant,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(s.jwtService.GetAccessTokenExpiry()),
	}, nil
}

// ValidateToken 验证JWT token并返回用户信息
func (s *UserAuthService) ValidateToken(ctx context.Context, tokenString string) (*entity.User, *entity.UserTenant, *JWTClaims, error) {
	// 1. 验证JWT
	claims, err := s.jwtService.ValidateToken(tokenString)
	if err != nil {
		return nil, nil, nil, err
	}

	// 2. 获取用户
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, nil, nil, apperrors.ErrUserNotFound
	}

	// 3. 检查用户状态
	if user.Status != "active" {
		return nil, nil, nil, apperrors.ErrUserInactive
	}

	// 4. 获取UserTenant
	userTenant, err := s.userTenantRepo.GetByUserAndTenant(ctx, claims.UserID, claims.TenantID)
	if err != nil {
		return nil, nil, nil, apperrors.ErrUserNotInTenant
	}

	// 5. 检查UserTenant状态
	if userTenant.Status != "active" {
		return nil, nil, nil, apperrors.ErrUserInactive
	}

	return user, userTenant, claims, nil
}

// SwitchTenant 切换当前租户
func (s *UserAuthService) SwitchTenant(ctx context.Context, userID, tenantID uuid.UUID, currentToken string) (newAccessToken string, err error) {
	// 1. 验证用户有权限访问该租户
	userTenant, err := s.userTenantRepo.GetByUserAndTenant(ctx, userID, tenantID)
	if err != nil {
		return "", apperrors.ErrUserNotInTenant
	}

	if userTenant.Status != "active" {
		return "", apperrors.ErrUserInactive
	}

	// 2. 验证当前token获取设备信息
	_, err = s.jwtService.ValidateToken(currentToken)
	if err != nil {
		return "", err
	}

	// 3. 设置为默认租户
	s.userTenantRepo.SetDefaultTenant(ctx, userID, tenantID)

	// 4. 生成新的access token
	newAccessToken, err = s.jwtService.RefreshToken(currentToken, tenantID, userTenant.Role)
	if err != nil {
		return "", fmt.Errorf("failed to generate new token: %w", err)
	}

	return newAccessToken, nil
}

// Logout 用户登出
func (s *UserAuthService) Logout(ctx context.Context, tokenString string) error {
	// 1. 将access token加入黑名单
	return s.jwtService.InvalidateToken(tokenString)
}

// LogoutAll 登出所有设备
func (s *UserAuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	// 1. 撤销所有refresh token
	return s.refreshTokenRepo.RevokeAllByUser(ctx, userID)
}

// RefreshToken 使用refresh token获取新的access token
func (s *UserAuthService) RefreshToken(ctx context.Context, refreshTokenString string, deviceID string) (newAccessToken string, err error) {
	// 1. 验证refresh token
	claims, err := s.jwtService.ValidateToken(refreshTokenString)
	if err != nil {
		return "", apperrors.ErrRefreshTokenInvalid
	}

	// 2. 检查设备匹配
	if deviceID != "" && claims.DeviceID != deviceID {
		return "", apperrors.ErrDeviceMismatch
	}

	// 3. 检查refresh token是否在数据库中且未被撤销
	refreshTokenHash := hashToken(refreshTokenString)
	rt, err := s.refreshTokenRepo.GetByTokenHash(ctx, refreshTokenHash)
	if err != nil || rt == nil || rt.RevokedAt != nil {
		return "", apperrors.ErrRefreshTokenInvalid
	}

	// 4. 获取用户的默认租户
	userTenant, err := s.userTenantRepo.GetDefaultTenant(ctx, claims.UserID)
	if err != nil {
		return "", fmt.Errorf("no default tenant")
	}

	// 5. 刷新access token
	newAccessToken, err = s.jwtService.RefreshToken(refreshTokenString, userTenant.TenantID, userTenant.Role)
	if err != nil {
		return "", err
	}

	// 6. 更新refresh token最后使用时间
	s.refreshTokenRepo.UpdateLastUsed(ctx, refreshTokenHash)

	return newAccessToken, nil
}

// ChangePassword 修改密码
func (s *UserAuthService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	// 1. 获取用户
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return apperrors.ErrUserNotFound
	}

	// 2. 验证当前密码
	if !s.passwordHasher.Verify(currentPassword, user.PasswordHash) {
		return apperrors.ErrInvalidCredentials
	}

	// 3. 验证新密码复杂度
	if err := password.ValidatePassword(newPassword); err != nil {
		return err
	}

	// 4. 检查新密码是否与当前密码相同
	if s.passwordHasher.Verify(newPassword, user.PasswordHash) {
		return fmt.Errorf("new password cannot be same as current")
	}

	// 5. 检查密码历史
	history, err := s.loginSecurity.passwordHistoryRepo.ListRecent(ctx, userID, 5)
	if err == nil {
		for _, h := range history {
			if s.passwordHasher.Verify(newPassword, h.PasswordHash) {
				return apperrors.ErrPasswordInHistory
			}
		}
	}

	// 6. 生成新密码hash
	newHash, err := s.passwordHasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// 7. 保存旧密码到历史
	s.loginSecurity.SavePasswordHistory(ctx, userID, user.PasswordHash, 5)

	// 8. 更新密码
	user.PasswordHash = newHash
	now := time.Now()
	user.PasswordChangedAt = &now
	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// 9. 撤销所有token（强制重新登录）
	s.refreshTokenRepo.RevokeAllByUser(ctx, userID)

	return nil
}

// GetUserTenants 获取用户所有租户
func (s *UserAuthService) GetUserTenants(ctx context.Context, userID uuid.UUID) ([]entity.UserTenant, error) {
	return s.userTenantRepo.ListByUser(ctx, userID)
}

// VerifyEmail 验证邮箱
func (s *UserAuthService) VerifyEmail(ctx context.Context, token string) (*entity.User, error) {
	if s.emailVerification == nil {
		return nil, fmt.Errorf("email verification service not configured")
	}
	return s.emailVerification.VerifyEmail(ctx, token)
}

// ResendVerification 重发验证邮件
func (s *UserAuthService) ResendVerification(ctx context.Context, email string) error {
	if s.emailVerification == nil {
		return fmt.Errorf("email verification service not configured")
	}
	return s.emailVerification.ResendVerification(ctx, email)
}

// helper functions

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func isValidEmail(email string) bool {
	// 简单的邮箱格式验证
	pattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return pattern.MatchString(email)
}

func isValidSlug(slug string) bool {
	// slug格式：小写字母、数字、连字符，3-50字符
	if len(slug) < 3 || len(slug) > 50 {
		return false
	}
	pattern := regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)
	return pattern.MatchString(slug)
}

// GenerateInviteToken 生成邀请token
func GenerateInviteToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}