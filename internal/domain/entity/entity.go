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

	// 租户类型 (支付系统新增)
	Type string `gorm:"size:20;default:organization" json:"type"` // public, organization

	RateLimitRPS         int `gorm:"default:100" json:"rate_limit_rps"`
	RateLimitBurst       int `gorm:"default:200" json:"rate_limit_burst"`
	MonthlyRequestLimit  int `gorm:"default:10000" json:"monthly_request_limit"`

	BillingEmail string          `gorm:"size:255" json:"billing_email"`
	Balance      decimal.Decimal `gorm:"type:decimal(10,4);default:0" json:"balance"`
	Currency     string          `gorm:"size:3;default:USD" json:"currency"`

	// 费用限制 (原有)
	MonthlyBudgetLimit  *decimal.Decimal `gorm:"type:decimal(10,4)" json:"monthly_budget_limit,omitempty"`  // 月度费用上限
	BudgetUsedMonth     decimal.Decimal  `gorm:"type:decimal(10,4);default:0" json:"budget_used_month"`      // 当月已使用费用
	TokenLimit          *int64           `gorm:"default:0" json:"token_limit,omitempty"`                     // Token限额
	TokensUsedMonth     int64            `gorm:"default:0" json:"tokens_used_month"`                         // 当月已用Token

	// 后付费信用额度 (支付系统新增)
	CreditLimit         *decimal.Decimal `gorm:"type:decimal(10,4)" json:"credit_limit,omitempty"`           // 申请的信用额度上限
	CreditUsed          decimal.Decimal  `gorm:"type:decimal(10,4);default:0" json:"credit_used"`            // 已用信用
	CreditStatus        string           `gorm:"size:20;default:none" json:"credit_status"`                  // none, pending, approved, rejected
	CreditAppliedAt     *time.Time       `json:"credit_applied_at,omitempty"`                                // 申请时间
	CreditApprovedAt    *time.Time       `json:"credit_approved_at,omitempty"`                               // 审核通过时间
	CreditApprovedBy    *uuid.UUID       `gorm:"type:uuid" json:"credit_approved_by,omitempty"`              // 审核人ID
	CreditRejectReason  string           `gorm:"size:200" json:"credit_reject_reason,omitempty"`             // 拒绝原因

	// 结算配置 (支付系统新增)
	SettlementCycle     string           `gorm:"size:20" json:"settlement_cycle,omitempty"`                  // monthly, weekly, threshold, custom
	ThresholdAmount     *decimal.Decimal `gorm:"type:decimal(10,4)" json:"threshold_amount,omitempty"`       // 阈值付款金额
	SettlementDay       *int             `gorm:"default:1" json:"settlement_day,omitempty"`                  // 结算日

	// 套餐订阅 (支付系统新增)
	PlanID              *uuid.UUID       `gorm:"type:uuid" json:"plan_id,omitempty"`                         // 当前套餐
	SubscriptionID      *uuid.UUID       `gorm:"type:uuid" json:"subscription_id,omitempty"`                 // 订阅记录

	// 申请审批 (原有)
	ApplicationID       *uuid.UUID `gorm:"type:uuid" json:"application_id,omitempty"`           // 关联租户申请
	ApprovedBy          *uuid.UUID `gorm:"type:uuid" json:"approved_by,omitempty"`              // 审批人
	ApprovedAt          *time.Time `json:"approved_at,omitempty"`                               // 审批时间

	// 用户管理策略 (原有)
	ApprovalPolicy      string `gorm:"type:jsonb;default:'{}'" json:"approval_policy,omitempty"`      // 用户审批规则JSON
	AutoApproveNewUsers bool   `gorm:"default:false" json:"auto_approve_new_users"`                   // 自动审批新用户申请
	MaxUsers            int    `gorm:"default:100" json:"max_users"`                                   // 最大用户数
	MaxAPIKeysPerUser   int    `gorm:"default:10" json:"max_api_keys_per_user"`                        // 每用户最大Key数

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

	// 用户模式 (支付系统新增)
	UserMode      string     `gorm:"size:20;default:individual" json:"user_mode"` // individual, organization
	UserQuotaID   *uuid.UUID `gorm:"type:uuid" json:"user_quota_id,omitempty"`    // 个人配额ID (individual模式)
	UserQuota     *UserQuota `gorm:"foreignKey:UserQuotaID" json:"user_quota,omitempty"`
	MemberQuotaID *uuid.UUID `gorm:"type:uuid" json:"member_quota_id,omitempty"`  // 成员配额ID (organization模式)
	MemberQuota   *MemberQuota `gorm:"foreignKey:MemberQuotaID" json:"member_quota,omitempty"`

	RateLimitRPS   *int `gorm:"default:10" json:"rate_limit_rps,omitempty"`
	RateLimitBurst *int `gorm:"default:20" json:"rate_limit_burst,omitempty"`

	// 费用限制 (原有)
	MonthlyBudget   *decimal.Decimal `gorm:"type:decimal(10,4)" json:"monthly_budget,omitempty"`       // 月度费用上限
	BudgetUsedMonth decimal.Decimal  `gorm:"type:decimal(10,4);default:0" json:"budget_used_month"`    // 当月已使用费用
	DailyBudget     *decimal.Decimal `gorm:"type:decimal(10,4)" json:"daily_budget,omitempty"`         // 日度费用上限
	BudgetUsedToday decimal.Decimal  `gorm:"type:decimal(10,4);default:0" json:"budget_used_today"`    // 今日已使用费用
	TokenQuota      *int64           `gorm:"default:0" json:"token_quota,omitempty"`                   // Token限额
	TokensUsedMonth int64            `gorm:"default:0" json:"tokens_used_month"`                       // 当月已用Token

	// 申请审批 (原有)
	ApplicationID  *uuid.UUID `gorm:"type:uuid" json:"application_id,omitempty"`  // 关联用户申请
	ApprovedBy     *uuid.UUID `gorm:"type:uuid" json:"approved_by,omitempty"`     // 审批人
	ApprovedAt     *time.Time `json:"approved_at,omitempty"`                      // 审批时间

	// API Key限制 (原有)
	MaxAPIKeys    *int `gorm:"default:10" json:"max_api_keys,omitempty"`      // 最大Key数量限制
	ActiveAPIKeys int  `gorm:"default:0" json:"active_api_keys"`              // 当前活跃Key数量

	Status      string     `gorm:"default:active" json:"status"` // active, inactive, pending_verification
	LastLoginAt       *time.Time `json:"last_login_at,omitempty"`
	PasswordChangedAt *time.Time `json:"password_changed_at,omitempty"` // 密码修改时间

	// Email verification fields
	EmailVerified             bool       `gorm:"default:false" json:"email_verified"`
	EmailVerificationToken    string     `gorm:"size:64;index" json:"-"`
	EmailVerificationExpires  *time.Time `json:"email_verification_expires,omitempty"`
	EmailVerifiedAt           *time.Time `json:"email_verified_at,omitempty"`

	// 费用使用日期追踪 (原有)
	BudgetResetDate *time.Time `gorm:"type:date" json:"budget_reset_date,omitempty"` // 月预算重置日期
	DailyResetDate  *time.Time `gorm:"type:date" json:"daily_reset_date,omitempty"`  // 日预算重置日期

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

	// 配额关联 (支付系统新增)
	QuotaType string    `gorm:"size:20" json:"quota_type,omitempty"`     // individual, member
	QuotaID   uuid.UUID `gorm:"type:uuid;not null" json:"quota_id"`      // UserQuota.ID / MemberQuota.ID

	KeyHash   string `gorm:"unique;not null;size:64;index" json:"-"`
	KeyPrefix string `gorm:"not null;size:12" json:"key_prefix"`
	Name      string `gorm:"size:255" json:"name"`

	Permissions      string `gorm:"type:jsonb" json:"permissions"`
	AllowedModels    string `gorm:"type:jsonb" json:"allowed_models,omitempty"`
	AllowedProviders string `gorm:"type:jsonb" json:"allowed_providers,omitempty"`

	RateLimitRPS   *int `gorm:"default:10" json:"rate_limit_rps,omitempty"`
	RateLimitBurst *int `gorm:"default:20" json:"rate_limit_burst,omitempty"`

	// Token限制 (原有)
	MonthlyTokenLimit   *int64 `gorm:"default:0" json:"monthly_token_limit,omitempty"`
	UsedTokensThisMonth int64  `gorm:"default:0" json:"used_tokens_this_month"`

	// 费用限制 (原有)
	MonthlyCostLimit    *decimal.Decimal `gorm:"type:decimal(10,4)" json:"monthly_cost_limit,omitempty"`    // 月度费用上限
	MonthlyCostUsed     decimal.Decimal  `gorm:"type:decimal(10,4);default:0" json:"monthly_cost_used"`      // 当月已使用费用
	DailyCostLimit      *decimal.Decimal `gorm:"type:decimal(10,4)" json:"daily_cost_limit,omitempty"`       // 日度费用上限
	DailyCostUsed       decimal.Decimal  `gorm:"type:decimal(10,4);default:0" json:"daily_cost_used"`        // 今日已使用费用
	PerRequestCostLimit *decimal.Decimal `gorm:"type:decimal(10,4)" json:"per_request_cost_limit,omitempty"` // 单次请求费用上限

	// Token限制扩展 (原有)
	TokenLimitPerDay *int64 `gorm:"default:0" json:"token_limit_per_day,omitempty"`   // 日Token限额
	TokensUsedToday  int64  `gorm:"default:0" json:"tokens_used_today"`                // 今日已用Token

	// 预算预警阈值 (原有)
	AlertThreshold1 int `gorm:"default:80" json:"alert_threshold_1"`  // 第一级预警阈值(默认80%)
	AlertThreshold2 int `gorm:"default:90" json:"alert_threshold_2"`  // 第二级预警阈值(默认90%)
	AlertThreshold3 int `gorm:"default:100" json:"alert_threshold_3"` // 第三级预警阈值(默认100%)

	// 费用日期追踪 (原有)
	CostResetDate  *time.Time `gorm:"type:date" json:"cost_reset_date,omitempty"`  // 月费用重置日期
	DailyResetDate *time.Time `gorm:"type:date" json:"daily_reset_date,omitempty"` // 日费用重置日期

	Status     string     `gorm:"default:active" json:"status"` // active, revoked, expired
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`

	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
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

	Type        string `gorm:"size:20;default:usage" json:"type"` // usage, credit_settlement
	Description string `gorm:"type:text" json:"description,omitempty"`

	PeriodStart time.Time `gorm:"not null;index:idx_period" json:"period_start"`
	PeriodEnd   time.Time `gorm:"not null;index:idx_period" json:"period_end"`

	TotalTokens int64            `gorm:"not null" json:"total_tokens"`
	TotalCost   decimal.Decimal `gorm:"type:decimal(10,4);not null" json:"total_cost"`
	Currency    string           `gorm:"size:3;default:USD" json:"currency"`

	Status  string     `gorm:"default:pending;index" json:"status"` // pending, paid, partial_paid, overdue, cancelled
	DueDate *time.Time `json:"due_date,omitempty"`
	PaidAt  *time.Time `json:"paid_at,omitempty"`

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

// ==================== 新增实体 ====================

// PlatformAdmin 平台管理员
type PlatformAdmin struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email        string    `gorm:"unique;not null;size:255" json:"email"`
	PasswordHash string    `gorm:"not null;size:255" json:"-"`
	Name         string    `gorm:"size:255" json:"name"`
	Role         string    `gorm:"default:super_admin" json:"role"` // super_admin, support, billing_admin
	Permissions  string    `gorm:"type:jsonb;default:'[]'" json:"permissions,omitempty"`
	Status       string    `gorm:"default:active" json:"status"` // active, inactive, suspended

	LastLoginAt *time.Time `json:"last_login_at,omitempty"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TenantApplication 租户申请
type TenantApplication struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CompanyName    string    `gorm:"not null;size:255" json:"company_name"`
	CompanySlug    string    `gorm:"unique;not null;size:100" json:"company_slug"`
	ContactEmail   string    `gorm:"not null;size:255" json:"contact_email"`
	ContactPhone   string    `gorm:"size:50" json:"contact_phone,omitempty"`
	ContactName    string    `gorm:"not null;size:255" json:"contact_name"`
	RequestedPlan  string    `gorm:"default:free" json:"requested_plan"` // free, basic, pro, enterprise
	RequestedFeatures string `gorm:"type:jsonb;default:'{}'" json:"requested_features,omitempty"` // 申请的功能特性

	Status          string    `gorm:"default:pending;index" json:"status"` // pending, reviewing, approved, rejected, tenant_created
	ReviewedBy      *uuid.UUID `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	ReviewNotes     string     `gorm:"type:text" json:"review_notes,omitempty"`
	RejectionReason string     `gorm:"type:text" json:"rejection_reason,omitempty"`

	TenantID *uuid.UUID `gorm:"type:uuid" json:"tenant_id,omitempty"` // 创建后关联的租户

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// UserApplication 用户申请/邀请
type UserApplication struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID      uuid.UUID `gorm:"not null;index" json:"tenant_id"`
	Tenant        *Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Email         string    `gorm:"not null;size:255" json:"email"`
	Name          string    `gorm:"size:255" json:"name,omitempty"`
RequestedRole string    `gorm:"default:member" json:"requested_role"` // member, viewer

	ApplicationType string    `gorm:"default:request;index" json:"application_type"` // request, invitation, direct_create
	InvitedBy       *uuid.UUID `gorm:"type:uuid" json:"invited_by,omitempty"`    // 邀请人 (invitation类型)
	CreatedBy       *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`    // 创建人 (direct_create类型)

	Status          string     `gorm:"default:pending;index" json:"status"` // pending, approved, rejected, user_created, expired, accepted
	ReviewedBy      *uuid.UUID `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	CreatedUserID   *uuid.UUID `gorm:"type:uuid" json:"created_user_id,omitempty"` // 创建后的用户ID

	// 邀请链接相关
	InviteToken string     `gorm:"unique;size:64" json:"invite_token,omitempty"` // 邀请码 (invitation类型)
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`                         // 邀请过期时间

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// BudgetAlert 预算预警配置
type BudgetAlert struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Scope            string    `gorm:"not null;size:20;index:idx_scope" json:"scope"` // tenant, user, api_key
	ScopeID          uuid.UUID `gorm:"not null;index:idx_scope" json:"scope_id"`
	AlertType        string    `gorm:"not null;size:30" json:"alert_type"` // budget_80, budget_90, budget_100, usage_limit
	ThresholdPercent int       `gorm:"not null" json:"threshold_percent"`  // 80, 90, 100

	// 通知配置
	NotifyEmails  string `gorm:"type:jsonb;default:'[]'" json:"notify_emails,omitempty"`  // JSONB: ["admin@example.com"]
	NotifySlack   string `gorm:"size:255" json:"notify_slack,omitempty"`                   // Slack webhook URL
	NotifyWebhook string `gorm:"size:255" json:"notify_webhook,omitempty"`                 // Custom webhook URL

	// 状态
	LastTriggeredAt *time.Time `json:"last_triggered_at,omitempty"`
	TriggeredCount  int        `gorm:"default:0" json:"triggered_count"`
	IsEnabled       bool       `gorm:"default:true" json:"is_enabled"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// CostUsageRecord 费用使用快照（用于统计和预警）
type CostUsageRecord struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Scope     string    `gorm:"not null;size:20;index:idx_scope_date" json:"scope"` // tenant, user, api_key
	ScopeID   uuid.UUID `gorm:"not null;index:idx_scope_date" json:"scope_id"`

	Date      time.Time `gorm:"type:date;not null;index:idx_scope_date" json:"date"` // 统计日期
	Cost      decimal.Decimal `gorm:"type:decimal(10,6);default:0" json:"cost"`       // 当日费用
	Tokens    int64           `gorm:"default:0" json:"tokens"`                        // 当日Token
	Requests  int             `gorm:"default:0" json:"requests"`                      // 当日请求数

	LimitAmount   *decimal.Decimal `gorm:"type:decimal(10,6)" json:"limit_amount,omitempty"` // 当日限额
	PercentUsed   int              `gorm:"default:0" json:"percent_used"`                     // 使用百分比

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// ==================== 支付系统实体 ====================

// UserQuota 个人配额 - 公共租户用户专用
type UserQuota struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"not null;uniqueIndex" json:"user_id"` // 用户ID (唯一)
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	TenantID  uuid.UUID `gorm:"not null" json:"tenant_id"` // 公共租户ID
	Tenant    *Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`

	// 个人余额 (完全独立)
	Balance  decimal.Decimal `gorm:"type:decimal(10,4);default:0" json:"balance"`
	Currency string          `gorm:"size:3;default:USD" json:"currency"`

	// 个人套餐订阅 (完全独立)
	PlanID         *uuid.UUID `gorm:"type:uuid" json:"plan_id,omitempty"`         // 当前套餐
	SubscriptionID *uuid.UUID `gorm:"type:uuid" json:"subscription_id,omitempty"` // 订阅记录

	// Token配额 (套餐提供或自定义)
	TokenQuota   int64      `gorm:"default:0" json:"token_quota"`     // Token配额上限
	TokensUsed   int64      `gorm:"default:0" json:"tokens_used"`     // 已用Token
	TokenResetAt *time.Time `json:"token_reset_at,omitempty"`         // 配额重置时间 (月度)

	// 月度统计
	MonthlyCost    decimal.Decimal `gorm:"type:decimal(10,4);default:0" json:"monthly_cost"`       // 月度费用累计
	MonthlyResetAt *time.Time      `json:"monthly_reset_at,omitempty"`                             // 月度重置时间

	// 状态控制 (个人无后付费)
	Status        string     `gorm:"size:20;default:pending_payment" json:"status"` // pending_payment, active, suspended
	LastPaymentAt *time.Time `json:"last_payment_at,omitempty"`                     // 最后支付时间

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// MemberQuota 成员配额 - 企业租户专用
type MemberQuota struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID uuid.UUID `gorm:"not null;index" json:"tenant_id"` // 关联租户
	Tenant   *Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	UserID   uuid.UUID `gorm:"not null;uniqueIndex:tenant_user" json:"user_id"` // 关联用户
	User     *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`

	// 成员Token配额控制 (管理员分配)
	TokenQuotaLimit *int64      `gorm:"type:bigint" json:"token_quota_limit,omitempty"` // Token配额上限
	TokensUsed      int64       `gorm:"default:0" json:"tokens_used"`                   // 已用Token
	TokenResetAt    *time.Time  `json:"token_reset_at,omitempty"`                       // 配额重置时间 (月度)

	// 成员费用限额控制
	CostLimit     *decimal.Decimal `gorm:"type:decimal(10,4)" json:"cost_limit,omitempty"`   // 费用限额
	CostLimitType string           `gorm:"size:20" json:"cost_limit_type,omitempty"`         // 限额类型: monthly, daily
	CostUsed      decimal.Decimal  `gorm:"type:decimal(10,4);default:0" json:"cost_used"`    // 已用费用
	CostResetAt   *time.Time       `json:"cost_reset_at,omitempty"`                          // 费用重置时间

	// API Key数量限制
	MaxAPIKeys    *int `gorm:"default:10" json:"max_api_keys,omitempty"`    // 最大API Key数
	ActiveAPIKeys int  `gorm:"default:0" json:"active_api_keys"`             // 活跃API Key数

	// 状态控制
	Status         string     `gorm:"size:20;default:active" json:"status"` // active, quota_exceeded, suspended
	ExceededAt     *time.Time `json:"exceeded_at,omitempty"`                 // 超限时间
	ExceededReason string     `gorm:"size:200" json:"exceeded_reason,omitempty"` // 超限原因

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// CreditApplication 后付费申请记录
type CreditApplication struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID uuid.UUID `gorm:"not null;index" json:"tenant_id"` // 关联租户
	Tenant   *Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`

	// 申请信息
	RequestedLimit decimal.Decimal `gorm:"type:decimal(10,4);not null" json:"requested_limit"` // 申请的信用额度
	Reason         string          `gorm:"type:text" json:"reason,omitempty"`                   // 申请原因

	// 审核信息
	Status        string     `gorm:"size:20;default:pending" json:"status"` // pending, approved, rejected
	ApprovedLimit *decimal.Decimal `gorm:"type:decimal(10,4)" json:"approved_limit,omitempty"` // 审核通过的额度
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`                 // 审核时间
	ReviewedBy    *uuid.UUID `gorm:"type:uuid" json:"reviewed_by,omitempty"` // 审核人ID
	ReviewNotes   string     `gorm:"type:text" json:"review_notes,omitempty"` // 审核备注

	// 结算配置 (审核时可设置)
	SettlementCycle string           `gorm:"size:20" json:"settlement_cycle,omitempty"`     // monthly, weekly, threshold, custom
	ThresholdAmount *decimal.Decimal `gorm:"type:decimal(10,4)" json:"threshold_amount,omitempty"` // 阈值付款金额
	SettlementDay   *int             `gorm:"default:1" json:"settlement_day,omitempty"`    // 结算日

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// PaymentOrder 支付订单 (支持双模式)
type PaymentOrder struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// 双模式支持
	PaymentMode string    `gorm:"size:20;not null" json:"payment_mode"`         // individual, organization
	UserID      uuid.UUID `gorm:"not null;index" json:"user_id"`                // 支付用户
	User        *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`

	// 个人模式
	UserQuotaID *uuid.UUID `gorm:"type:uuid" json:"user_quota_id,omitempty"` // 关联个人配额
	UserQuota   *UserQuota `gorm:"foreignKey:UserQuotaID" json:"user_quota,omitempty"`

	// 组织模式
	TenantID *uuid.UUID `gorm:"type:uuid" json:"tenant_id,omitempty"` // 关联租户
	Tenant   *Tenant    `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`

	OrderNumber string `gorm:"unique;not null;size:50" json:"order_number"` // PAY-{timestamp}-{random}
	OrderType   string `gorm:"size:20;not null" json:"order_type"`          // recharge, subscription, credit_settlement

	Amount   decimal.Decimal `gorm:"type:decimal(10,4);not null" json:"amount"`
	Currency string          `gorm:"size:3;default:USD" json:"currency"`

	PaymentProvider string `gorm:"size:20" json:"payment_provider,omitempty"` // alipay, wechat, stripe, paypal, bank
	PaymentMethod   string `gorm:"size:20" json:"payment_method,omitempty"`   // qr_code, web, app, bank_transfer
	PaymentID       string `gorm:"size:100" json:"payment_id,omitempty"`      // 外部订单ID

	Status    string     `gorm:"size:20;default:pending" json:"status"` // pending, paid, failed, cancelled, expired
	ExpireAt  *time.Time `json:"expire_at,omitempty"`
	PaidAt    *time.Time `json:"paid_at,omitempty"`

	CallbackData string     `gorm:"type:text" json:"callback_data,omitempty"` // 回调原始数据 (JSON)
	CallbackAt   *time.Time `json:"callback_at,omitempty"`

	// 关联
	SubscriptionID *uuid.UUID `gorm:"type:uuid" json:"subscription_id,omitempty"`
	BillID         *uuid.UUID `gorm:"type:uuid" json:"bill_id,omitempty"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// ==================== 登录认证实体 ====================

// UserTenant 用户-租户关联（多租户支持）
type UserTenant struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID      `gorm:"not null;index:idx_user_tenant,priority:1" json:"user_id"`
	TenantID  uuid.UUID      `gorm:"not null;index:idx_user_tenant,priority:2" json:"tenant_id"`
	User      *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Tenant    *Tenant        `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Role      string         `gorm:"default:member;size:20" json:"role"`    // admin, member, viewer
	Status    string         `gorm:"default:active;size:20" json:"status"` // active, inactive
	IsDefault bool           `gorm:"default:false" json:"is_default"`      // 是否为用户默认租户
	JoinedAt  time.Time      `gorm:"autoCreateTime" json:"joined_at"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// LoginAudit 登录审计日志
type LoginAudit struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID         *uuid.UUID     `gorm:"type:uuid;index" json:"user_id,omitempty"` // 登录成功时填充
	Email          string         `gorm:"size:255;index" json:"email"`              // 登录邮箱
	IPEncrypted    string         `gorm:"size:256" json:"-"`                        // 加密后的IP
	IPHash         string         `gorm:"size:64;index" json:"-"`                   // IP的hash（用于查询）
	UserAgentEnc   string         `gorm:"size:512" json:"-"`                        // 加密后的UserAgent
	DeviceID       string         `gorm:"size:64;index" json:"device_id"`           // 设备指纹（hash）
	DeviceInfo     string         `gorm:"type:jsonb" json:"device_info,omitempty"`  // 设备信息JSON
	Success        bool           `gorm:"default:false;index" json:"success"`       // 登录是否成功
	FailureReason  string         `gorm:"size:200" json:"failure_reason,omitempty"` // 失败原因
	LoginAt        time.Time      `gorm:"autoCreateTime;index" json:"login_at"`
	Location       string         `gorm:"size:100" json:"location,omitempty"`       // IP地理位置
	TenantID       *uuid.UUID     `gorm:"type:uuid" json:"tenant_id,omitempty"`     // 登录的租户

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
}

// PasswordHistory 密码历史记录
type PasswordHistory struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID       uuid.UUID      `gorm:"not null;index" json:"user_id"`
	PasswordHash string         `gorm:"size:60;not null" json:"-"` // bcrypt hash，不暴露
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`

	// 不存储原始密码！只保留hash用于比对验证
}

// RefreshToken Refresh Token记录
type RefreshToken struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID       uuid.UUID      `gorm:"not null;index" json:"user_id"`
	TokenHash    string         `gorm:"size:64;unique;not null" json:"-"` // Token hash
	DeviceID     string         `gorm:"size:64;not null" json:"device_id"` // 设备ID
	DeviceInfo   string         `gorm:"type:jsonb" json:"device_info,omitempty"` // 设备信息
	ExpiresAt    time.Time      `gorm:"not null;index" json:"expires_at"`
	LastUsedAt   *time.Time     `json:"last_used_at,omitempty"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	RevokedAt    *time.Time     `json:"revoked_at,omitempty"` // 撤销时间
}