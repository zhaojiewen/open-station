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