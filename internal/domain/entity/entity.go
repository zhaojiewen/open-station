package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Tenant struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string         `gorm:"not null" json:"name"`
	Slug      string         `gorm:"unique;not null" json:"slug"`
	Status    string         `gorm:"default:active" json:"status"` // active, suspended, deleted
	Plan      string         `gorm:"default:free" json:"plan"`     // free, basic, pro, enterprise

	RateLimitRPS         int `gorm:"default:100" json:"rate_limit_rps"`
	RateLimitBurst       int `gorm:"default:200" json:"rate_limit_burst"`
	MonthlyRequestLimit  int `gorm:"default:10000" json:"monthly_request_limit"`

	BillingEmail string          `gorm:"size:255" json:"billing_email"`
	Balance      decimal.Decimal `gorm:"type:decimal(10,4);default:0" json:"balance"`
	Currency     string          `gorm:"size:3;default:USD" json:"currency"`

	Metadata string `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID     uuid.UUID      `gorm:"not null;index" json:"tenant_id"`
	Tenant       *Tenant        `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Email        string         `gorm:"unique;not null;size:255" json:"email"`
	PasswordHash string         `gorm:"not null;size:255" json:"-"`
	Name         string         `gorm:"size:255" json:"name"`
	Role         string         `gorm:"default:member" json:"role"` // admin, member, viewer

	RateLimitRPS   *int `gorm:"default:10" json:"rate_limit_rps,omitempty"`
	RateLimitBurst *int `gorm:"default:20" json:"rate_limit_burst,omitempty"`

	Status      string    `gorm:"default:active" json:"status"` // active, inactive
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type APIKey struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID   uuid.UUID `gorm:"not null;index" json:"user_id"`
	User     *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	TenantID uuid.UUID `gorm:"not null;index" json:"tenant_id"`
	Tenant   *Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`

	KeyHash   string `gorm:"unique;not null;size:64;index" json:"-"`
	KeyPrefix string `gorm:"not null;size:12" json:"key_prefix"`
	Name      string `gorm:"size:255" json:"name"`

	Permissions      string `gorm:"type:jsonb" json:"permissions"`
	AllowedModels    string `gorm:"type:jsonb" json:"allowed_models,omitempty"`
	AllowedProviders string `gorm:"type:jsonb" json:"allowed_providers,omitempty"`

	RateLimitRPS   *int `gorm:"default:10" json:"rate_limit_rps,omitempty"`
	RateLimitBurst *int `gorm:"default:20" json:"rate_limit_burst,omitempty"`

	MonthlyTokenLimit    *int64 `gorm:"default:0" json:"monthly_token_limit,omitempty"`
	UsedTokensThisMonth  int64  `gorm:"default:0" json:"used_tokens_this_month"`

	Status     string    `gorm:"default:active" json:"status"` // active, revoked, expired
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	RevokedAt *time.Time     `json:"revoked_at,omitempty"`
}

type Model struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Provider     string    `gorm:"not null;uniqueIndex:provider_model;size:50" json:"provider"`
	ModelID      string    `gorm:"not null;uniqueIndex:provider_model;size:100" json:"model_id"`
	DisplayName  string    `gorm:"size:255" json:"display_name"`

	PromptPrice     decimal.Decimal `gorm:"type:decimal(10,6);not null" json:"prompt_price"`
	CompletionPrice decimal.Decimal `gorm:"type:decimal(10,6);not null" json:"completion_price"`
	Currency        string          `gorm:"size:3;default:USD" json:"currency"`

	MaxTokens     *int `gorm:"default:4096" json:"max_tokens,omitempty"`
	ContextWindow *int `gorm:"default:8192" json:"context_window,omitempty"`

	Capabilities string `gorm:"type:jsonb" json:"capabilities,omitempty"`

	Status string `gorm:"default:active" json:"status"` // active, deprecated

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type UsageRecord struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID  uuid.UUID `gorm:"not null;index:idx_tenant_created,priority:1" json:"tenant_id"`
	UserID    uuid.UUID `gorm:"not null;index:idx_user_created,priority:1" json:"user_id"`
	APIKeyID  uuid.UUID `gorm:"index" json:"api_key_id,omitempty"`

	RequestID string `gorm:"unique;size:100" json:"request_id"`
	Provider  string `gorm:"not null;size:50;index" json:"provider"`
	ModelID   string `gorm:"not null;size:100;index" json:"model_id"`

	PromptTokens     int `gorm:"not null" json:"prompt_tokens"`
	CompletionTokens int `gorm:"not null" json:"completion_tokens"`
	TotalTokens      int `gorm:"not null" json:"total_tokens"`

	Cost    decimal.Decimal `gorm:"type:decimal(10,6);not null" json:"cost"`
	Currency string          `gorm:"size:3;default:USD" json:"currency"`

	LatencyMs    *int    `gorm:"default:0" json:"latency_ms,omitempty"`
	StatusCode   *int    `gorm:"default:200" json:"status_code,omitempty"`
	ErrorMessage *string `gorm:"type:text" json:"error_message,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime;index:idx_tenant_created,priority:2;index:idx_user_created,priority:2" json:"created_at"`
}

type Bill struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID  uuid.UUID `gorm:"not null;index" json:"tenant_id"`
	Tenant    *Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	BillNumber string   `gorm:"unique;size:50" json:"bill_number"`

	PeriodStart time.Time `gorm:"not null;index:idx_period" json:"period_start"`
	PeriodEnd   time.Time `gorm:"not null;index:idx_period" json:"period_end"`

	TotalTokens int64            `gorm:"not null" json:"total_tokens"`
	TotalCost   decimal.Decimal `gorm:"type:decimal(10,4);not null" json:"total_cost"`
	Currency    string           `gorm:"size:3;default:USD" json:"currency"`

	Status string    `gorm:"default:pending;index" json:"status"` // pending, paid, overdue, cancelled
	PaidAt *time.Time `json:"paid_at,omitempty"`

	Items string `gorm:"type:jsonb" json:"items,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type BillItem struct {
	Provider string          `json:"provider"`
	Model    string          `json:"model"`
	Tokens   int64           `json:"tokens"`
	Cost     decimal.Decimal `json:"cost"`
}

type RechargeRecord struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID uuid.UUID `gorm:"not null;index" json:"tenant_id"`
	Tenant   *Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`

	Amount        decimal.Decimal `gorm:"type:decimal(10,4);not null" json:"amount"`
	Currency      string          `gorm:"size:3;default:USD" json:"currency"`
	PaymentMethod string          `gorm:"size:50" json:"payment_method,omitempty"`
	PaymentID     string          `gorm:"size:255" json:"payment_id,omitempty"`

	Status     string    `gorm:"default:pending;index" json:"status"` // pending, completed, failed
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	Notes string `gorm:"type:text" json:"notes,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type AuditLog struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID  uuid.UUID `gorm:"index:idx_tenant" json:"tenant_id,omitempty"`
	UserID    uuid.UUID `gorm:"index:idx_user" json:"user_id,omitempty"`

	Action       string    `gorm:"not null;size:100" json:"action"`
	ResourceType string    `gorm:"size:50" json:"resource_type,omitempty"`
	ResourceID   uuid.UUID `json:"resource_id,omitempty"`

	OldValues string `gorm:"type:jsonb" json:"old_values,omitempty"`
	NewValues string `gorm:"type:jsonb" json:"new_values,omitempty"`

	IPAddress string `gorm:"type:inet" json:"ip_address,omitempty"`
	UserAgent string `gorm:"type:text" json:"user_agent,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

// ProviderAccount 存储多个 Provider API 账户配置
// 支持同一 Provider 配置多个账号，用于故障切换
type ProviderAccount struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Provider   string    `gorm:"not null;index:idx_provider_status;size:50" json:"provider"` // openai, anthropic, gemini, deepseek, glm
	Name       string    `gorm:"not null;size:255" json:"name"`                             // 账户名称，便于识别
	APIKey     string    `gorm:"not null;size:255" json:"api_key"`                          // API Key
	BaseURL    string    `gorm:"size:255" json:"base_url"`                                  // API Base URL（可选）
	Priority   int       `gorm:"default:0;index:idx_provider_priority" json:"priority"`     // 优先级，0 最高
	Status     string    `gorm:"default:active;index:idx_provider_status" json:"status"`    // active, disabled, limited, exhausted
	IsDefault  bool      `gorm:"default:false" json:"is_default"`                           // 是否默认账户

	// 限额和用量
	MonthlyLimit   *decimal.Decimal `gorm:"type:decimal(10,2)" json:"monthly_limit,omitempty"`    // 月度限额（可选）
	UsedThisMonth  decimal.Decimal  `gorm:"type:decimal(10,4);default:0" json:"used_this_month"` // 本月已用金额
	RequestCount   int64            `gorm:"default:0" json:"request_count"`                      // 本月请求次数
	SuccessCount   int64            `gorm:"default:0" json:"success_count"`                      // 本月成功次数
	ErrorCount     int               `gorm:"default:0" json:"error_count"`                        // 连续错误次数
	LastError      *string           `gorm:"type:text" json:"last_error,omitempty"`               // 最后错误信息
	LastErrorAt    *time.Time        `json:"last_error_at,omitempty"`                             // 最后错误时间

	// 统计
	TotalRequests  int64 `gorm:"default:0" json:"total_requests"`  // 总请求次数
	TotalSuccess   int64 `gorm:"default:0" json:"total_success"`   // 总成功次数
	TotalErrors    int64 `gorm:"default:0" json:"total_errors"`    // 总错误次数
	TotalCost      decimal.Decimal `gorm:"type:decimal(10,4);default:0" json:"total_cost"` // 总花费

	// 配置
	Timeout       int    `gorm:"default:120" json:"timeout"`            // 请求超时（秒）
	RetryCount    int    `gorm:"default:3" json:"retry_count"`          // 重试次数
	RateLimitRPS  int    `gorm:"default:100" json:"rate_limit_rps"`     // 每秒请求限制
	EnabledModels string `gorm:"type:jsonb" json:"enabled_models,omitempty"` // 启用的模型列表（可选）

	// 时间
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`    // 最后使用时间
	DisabledAt   *time.Time `json:"disabled_at,omitempty"`    // 禁用时间
	ReactivatedAt *time.Time `json:"reactivated_at,omitempty"` // 重新激活时间
}