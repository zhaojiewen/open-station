package errors

import "fmt"

type AppError struct {
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewAppError(code, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

var (
	ErrInvalidAPIKey      = NewAppError("AUTH_001", "invalid API key", nil)
	ErrAPIKeyExpired      = NewAppError("AUTH_002", "API key has expired", nil)
	ErrAPIKeyRevoked      = NewAppError("AUTH_003", "API key has been revoked", nil)
	ErrUnauthorized       = NewAppError("AUTH_004", "unauthorized", nil)
	ErrForbidden          = NewAppError("AUTH_005", "forbidden", nil)
	ErrTokenLimitExceeded = NewAppError("AUTH_006", "monthly token limit exceeded", nil)

	ErrRateLimitExceeded  = NewAppError("RATE_001", "rate limit exceeded", nil)
	ErrTenantLimitExceeded = NewAppError("RATE_002", "tenant rate limit exceeded", nil)

	ErrInsufficientBalance = NewAppError("BILL_001", "insufficient balance", nil)
	ErrInvalidAmount       = NewAppError("BILL_002", "invalid amount", nil)

	ErrInvalidRequest      = NewAppError("REQ_001", "invalid request", nil)
	ErrModelNotSupported   = NewAppError("REQ_002", "model not supported", nil)
	ErrProviderNotEnabled  = NewAppError("REQ_003", "provider not enabled", nil)

	ErrProviderError       = NewAppError("PROV_001", "provider returned error", nil)
	ErrProviderTimeout     = NewAppError("PROV_002", "provider request timeout", nil)

	ErrInternal            = NewAppError("INT_001", "internal server error", nil)
	ErrDatabase            = NewAppError("INT_002", "database error", nil)
	ErrRedis               = NewAppError("INT_003", "redis error", nil)

	ErrIPBlocked              = NewAppError("SAF_001", "IP address is blocked", nil)
	ErrIPRateLimitExceeded    = NewAppError("SAF_002", "IP rate limit exceeded", nil)
	ErrRequestBodyTooLarge    = NewAppError("SAF_003", "request body too large", nil)
	ErrTooManyConcurrentConns  = NewAppError("SAF_004", "too many concurrent connections", nil)
	ErrMethodNotAllowed       = NewAppError("SAF_005", "request method not allowed", nil)
	ErrPathTraversal          = NewAppError("SAF_006", "path traversal detected", nil)
	ErrBadUserAgent           = NewAppError("SAF_007", "bad or missing user agent", nil)
	ErrRequestHeadersTooLarge  = NewAppError("SAF_008", "request headers too large", nil)
	ErrBurstAttackAutoBlocked = NewAppError("SAF_009", "burst attack detected, IP auto-blocked", nil)
	ErrInvalidContentType     = NewAppError("SAF_010", "invalid content type", nil)
	ErrURLTooLong             = NewAppError("SAF_011", "request URL too long", nil)
	ErrSuspiciousHeader       = NewAppError("SAF_012", "suspicious header detected", nil)
	ErrRateViolationBlocked   = NewAppError("SAF_013", "repeated rate limit violations, IP blocked", nil)

	// ===== 费用限制错误 (QUOTA) =====
	ErrQuotaExceeded               = NewAppError("QUOTA_001", "quota exceeded", nil)
	ErrBudgetExceeded              = NewAppError("QUOTA_002", "budget exceeded", nil)
	ErrTenantBudgetExceeded        = NewAppError("QUOTA_003", "tenant budget exceeded", nil)
	ErrTenantMonthlyBudgetExceeded = NewAppError("QUOTA_004", "tenant monthly budget exceeded", nil)
	ErrUserBudgetExceeded          = NewAppError("QUOTA_005", "user budget exceeded", nil)
	ErrUserMonthlyBudgetExceeded   = NewAppError("QUOTA_006", "user monthly budget exceeded", nil)
	ErrUserDailyBudgetExceeded     = NewAppError("QUOTA_007", "user daily budget exceeded", nil)
	ErrAPIKeyCostExceeded          = NewAppError("QUOTA_008", "api key cost exceeded", nil)
	ErrAPIKeyMonthlyCostExceeded   = NewAppError("QUOTA_009", "api key monthly cost exceeded", nil)
	ErrAPIKeyDailyCostExceeded     = NewAppError("QUOTA_010", "api key daily cost exceeded", nil)
	ErrAPIKeyPerRequestCostExceeded = NewAppError("QUOTA_011", "api key per-request cost exceeded", nil)
	ErrAPIKeyTokenLimitExceeded    = NewAppError("QUOTA_012", "api key token limit exceeded", nil)
	ErrAPIKeyDailyTokenExceeded    = NewAppError("QUOTA_013", "api key daily token exceeded", nil)
	// 支付系统配额错误
	ErrTokenQuotaExceeded          = NewAppError("QUOTA_014", "token quota exceeded", nil)
	ErrTenantTokenQuotaExceeded    = NewAppError("QUOTA_015", "tenant token quota exceeded", nil)
	ErrTenantPlanQuotaExceeded     = NewAppError("QUOTA_016", "tenant plan quota exceeded", nil)
	ErrMemberTokenQuotaExceeded    = NewAppError("QUOTA_017", "member token quota exceeded", nil)
	ErrMemberCostLimitExceeded     = NewAppError("QUOTA_018", "member cost limit exceeded", nil)
	ErrMemberSuspended             = NewAppError("QUOTA_019", "member suspended", nil)
	ErrUserSuspended               = NewAppError("QUOTA_020", "user suspended", nil)
	ErrCreditLimitExceeded         = NewAppError("QUOTA_021", "credit limit exceeded", nil)
	ErrCreditNotApproved           = NewAppError("QUOTA_022", "credit not approved", nil)
	ErrInvalidQuotaType            = NewAppError("QUOTA_023", "invalid quota type", nil)
	ErrNoPaymentSource             = NewAppError("QUOTA_024", "no available payment source", nil)

	// ===== 申请审批错误 (APP) =====
	ErrApplicationNotFound        = NewAppError("APP_001", "application not found", nil)
	ErrApplicationAlreadyProcessed = NewAppError("APP_002", "application already processed", nil)
	ErrApplicationRejected        = NewAppError("APP_003", "application rejected", nil)
	ErrApplicationPending         = NewAppError("APP_004", "application still pending", nil)
	ErrApplicationExpired         = NewAppError("APP_005", "application expired", nil)
	ErrApplicationInvalidStatus   = NewAppError("APP_006", "invalid application status for this operation", nil)

	// ===== 审批操作错误 (APPR) =====
	ErrApprovalNotAuthorized     = NewAppError("APPR_001", "not authorized to approve", nil)
	ErrApprovalAlreadyApproved   = NewAppError("APPR_002", "already approved", nil)
	ErrApprovalAlreadyRejected   = NewAppError("APPR_003", "already rejected", nil)
	ErrApprovalInvalidTransition = NewAppError("APPR_004", "invalid status transition", nil)

	// ===== 租户错误 (TENANT) =====
	ErrTenantNotFound           = NewAppError("TENANT_001", "tenant not found", nil)
	ErrTenantSuspended          = NewAppError("TENANT_002", "tenant suspended", nil)
	ErrTenantDeleted            = NewAppError("TENANT_003", "tenant deleted", nil)
	ErrTenantMaxUsersReached    = NewAppError("TENANT_004", "maximum users reached", nil)
	ErrTenantMaxAPIKeysReached  = NewAppError("TENANT_005", "maximum api keys reached", nil)
	ErrTenantSlugExists         = NewAppError("TENANT_006", "tenant slug already exists", nil)

	// ===== 平台管理员错误 (PLAT) =====
	ErrPlatformAdminNotFound    = NewAppError("PLAT_001", "platform admin not found", nil)
	ErrPlatformAdminInactive    = NewAppError("PLAT_002", "platform admin inactive", nil)
	ErrPlatformPermissionDenied = NewAppError("PLAT_003", "platform permission denied", nil)
	ErrPlatformAdminExists      = NewAppError("PLAT_004", "platform admin email already exists", nil)

	// ===== 用户错误 (USER) =====
	ErrUserNotFound             = NewAppError("USER_001", "user not found", nil)
	ErrUserInactive             = NewAppError("USER_002", "user inactive", nil)
	ErrUserMaxAPIKeysReached    = NewAppError("USER_003", "user maximum api keys reached", nil)
	ErrUserEmailExists          = NewAppError("USER_004", "user email already exists", nil)
	ErrUserNotInTenant          = NewAppError("USER_005", "user not in this tenant", nil)

	// ===== 邀请错误 (INVITE) =====
	ErrInviteNotFound           = NewAppError("INVITE_001", "invitation not found", nil)
	ErrInviteExpired            = NewAppError("INVITE_002", "invitation expired", nil)
	ErrInviteAlreadyAccepted    = NewAppError("INVITE_003", "invitation already accepted", nil)
	ErrInviteInvalidToken       = NewAppError("INVITE_004", "invalid invitation token", nil)

	// ===== 登录错误 (LOGIN) =====
	ErrInvalidCredentials     = NewAppError("LOGIN_001", "invalid email or password", nil)
	ErrTooManyAttempts        = NewAppError("LOGIN_002", "too many failed login attempts", nil)
	ErrAccountLocked          = NewAppError("LOGIN_003", "account is locked", nil)
	ErrSessionExpired         = NewAppError("LOGIN_004", "session expired", nil)
	ErrTokenInvalid           = NewAppError("LOGIN_005", "invalid token", nil)
	ErrTokenRevoked           = NewAppError("LOGIN_006", "token has been revoked", nil)
	ErrRefreshTokenInvalid    = NewAppError("LOGIN_007", "invalid refresh token", nil)
	ErrDeviceMismatch         = NewAppError("LOGIN_008", "device mismatch", nil)

	// ===== 注册错误 (REGISTER) =====
	ErrEmailExists            = NewAppError("REGISTER_001", "email already registered", nil)
	// ErrTenantSlugExists 使用 TENANT_006
	ErrPasswordTooShort       = NewAppError("REGISTER_003", "password must be at least 8 characters", nil)
	ErrPasswordTooLong        = NewAppError("REGISTER_004", "password must be at most 64 characters", nil)
	ErrPasswordTooWeak        = NewAppError("REGISTER_005", "password does not meet complexity requirements", nil)
	ErrPasswordInHistory      = NewAppError("REGISTER_006", "password was used recently", nil)
	ErrInvalidEmailFormat     = NewAppError("REGISTER_007", "invalid email format", nil)
	ErrTenantRequired         = NewAppError("REGISTER_008", "tenant_id is required for user registration", nil)

	// ===== Email验证错误 (VERIFY) =====
	ErrInvalidVerificationToken = NewAppError("VERIFY_001", "invalid verification token", nil)
	ErrVerificationTokenExpired = NewAppError("VERIFY_002", "verification token has expired", nil)
	ErrEmailAlreadyVerified     = NewAppError("VERIFY_003", "email already verified", nil)
	ErrEmailNotVerified         = NewAppError("VERIFY_004", "email not verified, please verify first", nil)

	// ===== 加密错误 (CRYPTO) =====
	ErrEncryptionFailed       = NewAppError("CRYPTO_001", "encryption failed", nil)
	ErrDecryptionFailed      = NewAppError("CRYPTO_002", "decryption failed", nil)
	ErrInvalidKey            = NewAppError("CRYPTO_003", "invalid encryption key", nil)
	ErrKeyNotFound           = NewAppError("CRYPTO_004", "encryption key not found", nil)
)

// IsAuthError checks if the error is an authentication related error
func IsAuthError(err error) bool {
	var appErr *AppError
	if As(err, &appErr) {
		return appErr.Code[:4] == "AUTH"
	}
	return false
}

// IsRateLimitError checks if the error is a rate limit related error
func IsRateLimitError(err error) bool {
	var appErr *AppError
	if As(err, &appErr) {
		return appErr.Code[:4] == "RATE"
	}
	return false
}

// IsBillingError checks if the error is a billing related error
func IsBillingError(err error) bool {
	var appErr *AppError
	if As(err, &appErr) {
		return appErr.Code[:4] == "BILL"
	}
	return false
}

// IsProviderError checks if the error is a provider related error
func IsProviderError(err error) bool {
	var appErr *AppError
	if As(err, &appErr) {
		return appErr.Code[:4] == "PROV"
	}
	return false
}

// IsQuotaError checks if the error is a quota/cost limit related error
func IsQuotaError(err error) bool {
	var appErr *AppError
	if As(err, &appErr) {
		return appErr.Code[:5] == "QUOTA"
	}
	return false
}

// IsApplicationError checks if the error is an application related error
func IsApplicationError(err error) bool {
	var appErr *AppError
	if As(err, &appErr) {
		return appErr.Code[:3] == "APP"
	}
	return false
}

// IsTenantError checks if the error is a tenant related error
func IsTenantError(err error) bool {
	var appErr *AppError
	if As(err, &appErr) {
		return appErr.Code[:6] == "TENANT"
	}
	return false
}

// IsUserError checks if the error is a user related error
func IsUserError(err error) bool {
	var appErr *AppError
	if As(err, &appErr) {
		return appErr.Code[:4] == "USER"
	}
	return false
}

// IsInviteError checks if the error is an invitation related error
func IsInviteError(err error) bool {
	var appErr *AppError
	if As(err, &appErr) {
		return appErr.Code[:6] == "INVITE"
	}
	return false
}

// IsLoginError checks if the error is a login related error
func IsLoginError(err error) bool {
	var appErr *AppError
	if As(err, &appErr) {
		return appErr.Code[:5] == "LOGIN"
	}
	return false
}

// IsRegisterError checks if the error is a registration related error
func IsRegisterError(err error) bool {
	var appErr *AppError
	if As(err, &appErr) {
		return appErr.Code[:8] == "REGISTER"
	}
	return false
}

// IsCryptoError checks if the error is a crypto related error
func IsCryptoError(err error) bool {
	var appErr *AppError
	if As(err, &appErr) {
		return appErr.Code[:6] == "CRYPTO"
	}
	return false
}

// IsVerifyError checks if the error is an email verification related error
func IsVerifyError(err error) bool {
	var appErr *AppError
	if As(err, &appErr) {
		return appErr.Code[:6] == "VERIFY"
	}
	return false
}

// Is checks if the error matches the target error code
func Is(err error, target *AppError) bool {
	var appErr *AppError
	if As(err, &appErr) {
		return appErr.Code == target.Code
	}
	return false
}

// As extracts the AppError from an error chain
func As(err error, target **AppError) bool {
	if err == nil {
		return false
	}
	if appErr, ok := err.(*AppError); ok {
		*target = appErr
		return true
	}
	return false
}