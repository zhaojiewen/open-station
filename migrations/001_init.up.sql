-- 租户表
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    plan VARCHAR(50) DEFAULT 'free',

    rate_limit_rps INTEGER DEFAULT 100,
    rate_limit_burst INTEGER DEFAULT 200,
    monthly_request_limit INTEGER DEFAULT 10000,

    billing_email VARCHAR(255),
    balance DECIMAL(10, 4) DEFAULT 0.00,
    currency VARCHAR(3) DEFAULT 'USD',

    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_tenants_slug ON tenants(slug);
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    role VARCHAR(50) DEFAULT 'member',

    rate_limit_rps INTEGER,
    rate_limit_burst INTEGER,

    status VARCHAR(20) DEFAULT 'active',
    last_login_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

-- API Key表
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    key_hash VARCHAR(64) UNIQUE NOT NULL,
    key_prefix VARCHAR(12) NOT NULL,
    name VARCHAR(255),

    permissions JSONB DEFAULT '["chat", "embeddings"]',
    allowed_models JSONB,
    allowed_providers JSONB,

    rate_limit_rps INTEGER,
    rate_limit_burst INTEGER,

    monthly_token_limit BIGINT,
    used_tokens_this_month BIGINT DEFAULT 0,

    status VARCHAR(20) DEFAULT 'active',
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);

-- 模型配置表
CREATE TABLE IF NOT EXISTS models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) NOT NULL,
    model_id VARCHAR(100) NOT NULL,
    display_name VARCHAR(255),

    prompt_price DECIMAL(10, 6) NOT NULL,
    completion_price DECIMAL(10, 6) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',

    max_tokens INTEGER,
    context_window INTEGER,

    capabilities JSONB DEFAULT '{"chat": true, "streaming": true, "embeddings": false}',
    status VARCHAR(20) DEFAULT 'active',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(provider, model_id)
);

CREATE INDEX IF NOT EXISTS idx_models_provider ON models(provider);
CREATE INDEX IF NOT EXISTS idx_models_status ON models(status);

-- 使用记录表
CREATE TABLE IF NOT EXISTS usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    user_id UUID NOT NULL REFERENCES users(id),
    api_key_id UUID REFERENCES api_keys(id),

    request_id VARCHAR(100) UNIQUE,
    provider VARCHAR(50) NOT NULL,
    model_id VARCHAR(100) NOT NULL,

    prompt_tokens INTEGER NOT NULL,
    completion_tokens INTEGER NOT NULL,
    total_tokens INTEGER NOT NULL,

    cost DECIMAL(10, 6) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',

    latency_ms INTEGER,
    status_code INTEGER,
    error_message TEXT,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_usage_records_tenant_created ON usage_records(tenant_id, created_at);
CREATE INDEX IF NOT EXISTS idx_usage_records_user_created ON usage_records(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_usage_records_request_id ON usage_records(request_id);

-- 账单表
CREATE TABLE IF NOT EXISTS bills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    bill_number VARCHAR(50) UNIQUE,

    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,

    total_tokens BIGINT NOT NULL,
    total_cost DECIMAL(10, 4) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',

    status VARCHAR(20) DEFAULT 'pending',
    paid_at TIMESTAMP WITH TIME ZONE,

    items JSONB DEFAULT '[]',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bills_tenant_id ON bills(tenant_id);
CREATE INDEX IF NOT EXISTS idx_bills_status ON bills(status);
CREATE INDEX IF NOT EXISTS idx_bills_period ON bills(period_start, period_end);

-- 充值记录表
CREATE TABLE IF NOT EXISTS recharge_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),

    amount DECIMAL(10, 4) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',

    payment_method VARCHAR(50),
    payment_id VARCHAR(255),

    status VARCHAR(20) DEFAULT 'pending',
    completed_at TIMESTAMP WITH TIME ZONE,

    notes TEXT,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_recharge_records_tenant_id ON recharge_records(tenant_id);
CREATE INDEX IF NOT EXISTS idx_recharge_records_status ON recharge_records(status);

-- 审计日志表
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id),
    user_id UUID REFERENCES users(id),

    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id UUID,

    old_values JSONB,
    new_values JSONB,

    ip_address INET,
    user_agent TEXT,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant ON audit_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at);

-- 插入默认模型定价数据
INSERT INTO models (provider, model_id, display_name, prompt_price, completion_price, max_tokens, context_window, capabilities) VALUES
('openai', 'gpt-4o', 'GPT-4o', 0.005, 0.015, 4096, 128000, '{"chat": true, "streaming": true, "vision": true}'),
('openai', 'gpt-4o-mini', 'GPT-4o Mini', 0.00015, 0.0006, 4096, 128000, '{"chat": true, "streaming": true}'),
('openai', 'gpt-4-turbo', 'GPT-4 Turbo', 0.01, 0.03, 4096, 128000, '{"chat": true, "streaming": true, "vision": true}'),
('openai', 'gpt-3.5-turbo', 'GPT-3.5 Turbo', 0.0005, 0.0015, 4096, 16385, '{"chat": true, "streaming": true}'),
('openai', 'text-embedding-3-small', 'Embedding Small', 0.00002, 0, 8191, 8191, '{"embeddings": true}'),
('openai', 'text-embedding-3-large', 'Embedding Large', 0.00013, 0, 8191, 3072, '{"embeddings": true}'),
('claude', 'claude-3-opus', 'Claude 3 Opus', 0.015, 0.075, 4096, 200000, '{"chat": true, "streaming": true, "vision": true}'),
('claude', 'claude-3-sonnet', 'Claude 3 Sonnet', 0.003, 0.015, 4096, 200000, '{"chat": true, "streaming": true, "vision": true}'),
('claude', 'claude-3-haiku', 'Claude 3 Haiku', 0.00025, 0.00125, 4096, 200000, '{"chat": true, "streaming": true, "vision": true}'),
('gemini', 'gemini-1.5-pro', 'Gemini 1.5 Pro', 0.0035, 0.0105, 8192, 1000000, '{"chat": true, "streaming": true, "vision": true}'),
('gemini', 'gemini-1.5-flash', 'Gemini 1.5 Flash', 0.000075, 0.0003, 8192, 1000000, '{"chat": true, "streaming": true, "vision": true}')
ON CONFLICT (provider, model_id) DO NOTHING;