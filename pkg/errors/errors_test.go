package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		expected string
	}{
		{
			name: "error with code and message only",
			err: &AppError{
				Code:    "AUTH_001",
				Message: "invalid API key",
			},
			expected: "AUTH_001: invalid API key",
		},
		{
			name: "error with wrapped error",
			err: &AppError{
				Code:    "DB_001",
				Message: "database connection failed",
				Err:     errors.New("connection refused"),
			},
			expected: "DB_001: database connection failed (connection refused)",
		},
		{
			name: "error with nil wrapped error",
			err: &AppError{
				Code:    "TEST_001",
				Message: "test error",
				Err:     nil,
			},
			expected: "TEST_001: test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("Error() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewAppError(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		message string
		err     error
	}{
		{
			name:    "error without wrapped error",
			code:    "AUTH_001",
			message: "invalid credentials",
			err:     nil,
		},
		{
			name:    "error with wrapped error",
			code:    "DB_002",
			message: "query failed",
			err:     errors.New("syntax error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := NewAppError(tt.code, tt.message, tt.err)

			if appErr.Code != tt.code {
				t.Errorf("Code = %v, want %v", appErr.Code, tt.code)
			}
			if appErr.Message != tt.message {
				t.Errorf("Message = %v, want %v", appErr.Message, tt.message)
			}
			if appErr.Err != tt.err {
				t.Errorf("Err = %v, want %v", appErr.Err, tt.err)
			}
		})
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		code     string
		message  string
		hasInner bool
	}{
		{
			name:     "ErrInvalidAPIKey",
			err:      ErrInvalidAPIKey,
			code:     "AUTH_001",
			message:  "invalid API key",
			hasInner: false,
		},
		{
			name:     "ErrAPIKeyExpired",
			err:      ErrAPIKeyExpired,
			code:     "AUTH_002",
			message:  "API key has expired",
			hasInner: false,
		},
		{
			name:     "ErrAPIKeyRevoked",
			err:      ErrAPIKeyRevoked,
			code:     "AUTH_003",
			message:  "API key has been revoked",
			hasInner: false,
		},
		{
			name:     "ErrUnauthorized",
			err:      ErrUnauthorized,
			code:     "AUTH_004",
			message:  "unauthorized",
			hasInner: false,
		},
		{
			name:     "ErrForbidden",
			err:      ErrForbidden,
			code:     "AUTH_005",
			message:  "forbidden",
			hasInner: false,
		},
		{
			name:     "ErrRateLimitExceeded",
			err:      ErrRateLimitExceeded,
			code:     "RATE_001",
			message:  "rate limit exceeded",
			hasInner: false,
		},
		{
			name:     "ErrTenantLimitExceeded",
			err:      ErrTenantLimitExceeded,
			code:     "RATE_002",
			message:  "tenant rate limit exceeded",
			hasInner: false,
		},
		{
			name:     "ErrInsufficientBalance",
			err:      ErrInsufficientBalance,
			code:     "BILL_001",
			message:  "insufficient balance",
			hasInner: false,
		},
		{
			name:     "ErrInvalidAmount",
			err:      ErrInvalidAmount,
			code:     "BILL_002",
			message:  "invalid amount",
			hasInner: false,
		},
		{
			name:     "ErrInvalidRequest",
			err:      ErrInvalidRequest,
			code:     "REQ_001",
			message:  "invalid request",
			hasInner: false,
		},
		{
			name:     "ErrModelNotSupported",
			err:      ErrModelNotSupported,
			code:     "REQ_002",
			message:  "model not supported",
			hasInner: false,
		},
		{
			name:     "ErrProviderNotEnabled",
			err:      ErrProviderNotEnabled,
			code:     "REQ_003",
			message:  "provider not enabled",
			hasInner: false,
		},
		{
			name:     "ErrProviderError",
			err:      ErrProviderError,
			code:     "PROV_001",
			message:  "provider returned error",
			hasInner: false,
		},
		{
			name:     "ErrProviderTimeout",
			err:      ErrProviderTimeout,
			code:     "PROV_002",
			message:  "provider request timeout",
			hasInner: false,
		},
		{
			name:     "ErrInternal",
			err:      ErrInternal,
			code:     "INT_001",
			message:  "internal server error",
			hasInner: false,
		},
		{
			name:     "ErrDatabase",
			err:      ErrDatabase,
			code:     "INT_002",
			message:  "database error",
			hasInner: false,
		},
		{
			name:     "ErrRedis",
			err:      ErrRedis,
			code:     "INT_003",
			message:  "redis error",
			hasInner: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("Code = %v, want %v", tt.err.Code, tt.code)
			}
			if tt.err.Message != tt.message {
				t.Errorf("Message = %v, want %v", tt.err.Message, tt.message)
			}
			if (tt.err.Err != nil) != tt.hasInner {
				t.Errorf("HasInner = %v, want %v", tt.err.Err != nil, tt.hasInner)
			}
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	baseErr := errors.New("base error")
	appErr := NewAppError("TEST_001", "test error", baseErr)

	// Test error wrapping with errors.Is
	if !errors.Is(appErr.Err, baseErr) {
		t.Error("wrapped error should match base error")
	}

	// Test error message
	expected := "TEST_001: test error (base error)"
	if appErr.Error() != expected {
		t.Errorf("Error() = %v, want %v", appErr.Error(), expected)
	}
}

func TestErrorAsAppError(t *testing.T) {
	appErr := NewAppError("TEST_001", "test message", nil)

	var target *AppError
	if !errors.As(appErr, &target) {
		t.Error("errors.As should match AppError")
	}

	if target.Code != "TEST_001" {
		t.Errorf("Code = %v, want TEST_001", target.Code)
	}
}

func TestNestedErrors(t *testing.T) {
	innerErr := errors.New("inner error")
	appErr1 := NewAppError("ERR_001", "first error", innerErr)
	appErr2 := NewAppError("ERR_002", "second error", appErr1)

	// The wrapped error should be appErr1
	if appErr2.Err != appErr1 {
		t.Error("appErr2 should wrap appErr1")
	}

	// Error message should show the chain
	expected := "ERR_002: second error (ERR_001: first error (inner error))"
	if appErr2.Error() != expected {
		t.Errorf("Error() = %v, want %v", appErr2.Error(), expected)
	}
}

func TestAppErrorImplementsError(t *testing.T) {
	// Ensure AppError implements error interface
	var _ error = &AppError{}
	var _ error = NewAppError("TEST", "test", nil)
}

func BenchmarkAppError_Error(b *testing.B) {
	err := NewAppError("BENCH_001", "benchmark error", errors.New("inner"))
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

func ExampleAppError_Error() {
	err := NewAppError("AUTH_001", "invalid API key", nil)
	fmt.Println(err.Error())
	// Output: AUTH_001: invalid API key
}

func ExampleAppError_Error_withWrapped() {
	err := NewAppError("DB_001", "query failed", errors.New("connection timeout"))
	fmt.Println(err.Error())
	// Output: DB_001: query failed (connection timeout)
}