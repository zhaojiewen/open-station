# Open Station - Provider 多账户配置与自动切换功能实现总结

## ✅ 已完成的功能

### 1. 启动时可配置 Provider（可跳过）

**实现方式：**
- 启动脚本 `scripts/start-docker.sh` 询问是否配置 Provider
- 配置脚本 `scripts/setup-provider.sh` 提供交互式配置
- 完全可以跳过，后续随时配置

**代码位置：**
- `scripts/start-docker.sh` - 自动检测 Docker 并启动
- `scripts/setup-provider.sh` - Provider 配置脚本

### 2. 后续通过 API 和 MCP 配置

**实现方式：**
- MCP Service 提供完整的账户管理工具
- Claude Code CLI 可直接对话式配置
- REST API 通过 MCP endpoint 暴露

**代码位置：**
- `internal/application/service/provider_mcp_tools.go` - MCP 工具实现（8个）
- `internal/application/service/provider_account_service.go` - 账户管理服务
- `internal/application/service/mcp_service.go` - MCP 协议实现

### 3. 同一 Provider 支持多个账户配置

**数据库设计：**
- `provider_accounts` 表存储账户信息
- 支持字段：provider, name, api_key, base_url, priority, status, is_default, monthly_limit, used_this_month, error_count 等

**代码位置：**
- `internal/domain/entity/entity.go` - ProviderAccount 实体定义
- `internal/domain/repository/repository.go` - ProviderAccountRepository 接口
- `internal/infrastructure/persistence/postgres/repositories/repositories.go` - Repository 实现

### 4. 自动故障切换机制

**实现方式：**
- DynamicProxyHandler 在请求时动态选择账户
- 错误检测和自动切换逻辑
- 成本记录和状态管理

**代码位置：**
- `internal/interfaces/http/handler/dynamic_proxy_handler.go` - 动态账户切换 Handler
- `internal/infrastructure/proxy/proxy_service.go` - HTTPClientWrapper（新增）
- `internal/application/service/provider_account_service.go` - 切换逻辑

## 🎯 核心组件

### 1. ProviderAccountService（账户管理服务）

```go
// 主要方法
CreateAccount()          // 创建账户
GetActiveAccount()       // 获取活跃账户（含切换逻辑）
RecordSuccess()          // 记录成功请求和成本
RecordError()            // 记录失败请求
HandleRateLimit()        // 处理 Rate Limit 错误
HandleInsufficientQuota() // 处理余额不足
ResetMonthlyUsage()      // 重置月度用量
GetProviderStatus()      // 获取 Provider 状态
```

**文件：** `internal/application/service/provider_account_service.go`

### 2. MCP 工具（8个）

| 工具 | 功能 |
|------|------|
| `list_provider_accounts` | 查看所有账户 |
| `create_provider_account` | 创建新账户 |
| `update_provider_account` | 更新账户配置 |
| `set_default_provider_account` | 设置默认账户 |
| `enable_provider_account` | 启用账户 |
| `disable_provider_account` | 禁用账户 |
| `delete_provider_account` | 删除账户 |
| `get_provider_status` | 查看状态摘要 |

**文件：** `internal/application/service/provider_mcp_tools.go`

### 3. DynamicProxyHandler（动态代理）

```go
ExecuteRequest()           // 执行请求，支持动态切换
ExecuteStreamRequest()     // 执行流式请求
executeWithDynamicAccount() // 使用动态账户执行
buildAccountConfig()       // 构建账户配置
calculateCost()            // 计算请求成本
```

**文件：** `internal/interfaces/http/handler/dynamic_proxy_handler.go`

### 4. HTTPClientWrapper（HTTP 客户端）

```go
ChatCompletion()          // 执行 ChatCompletion 请求
StreamChatCompletion()    // 执行流式请求
parseClaudeResponse()     // 解析 Claude 响应
```

**文件：** `internal/infrastructure/proxy/proxy_service.go`

## 🔧 自动切换机制

### 触发条件

#### 1. Rate Limit（429）
```
检测关键词：rate_limit, 429, too many requests, TPM, RPM
动作：标记为 limited → 切换账户 → 5分钟后恢复
```

#### 2. 余额不足（insufficient_quota）
```
检测关键词：insufficient_quota, out of credits, billing_hard_limit
动作：标记为 exhausted → 切换账户 → 每月1日恢复
```

#### 3. 连续失败 ≥ 5 次
```
检测：ErrorCount >= 5
动作：标记为 limited → 切换账户 → 成功后清零
```

#### 4. 月度限额耗尽
```
检测：UsedThisMonth >= MonthlyLimit
动作：标记为 exhausted → 切换账户 → 每月1日恢复
```

### 切换流程

```
请求到达 → GetActiveAccount(provider) → 
检查默认账户 →
  ↓ 正常
使用该账户执行请求 → RecordSuccess(cost)
  ↓ 异常
检查错误类型 →
  ↓ Rate Limit
HandleRateLimit() → 更新状态为 limited → 
GetNextAvailable() → 切换到下一个 active 账户 → 重试请求
  ↓ Quota Error  
HandleInsufficientQuota() → 更新状态为 exhausted → 
GetNextAvailable() → 切换账户 → 重试请求
```

## 📊 数据库表结构

```sql
CREATE TABLE provider_accounts (
  id UUID PRIMARY KEY,
  provider VARCHAR(50) NOT NULL,
  name VARCHAR(255) NOT NULL,
  api_key VARCHAR(255) NOT NULL,
  base_url VARCHAR(255),
  priority INT DEFAULT 0,
  status VARCHAR(20) DEFAULT 'active',
  is_default BOOLEAN DEFAULT FALSE,
  
  -- 限额和用量
  monthly_limit DECIMAL(10,2),
  used_this_month DECIMAL(10,4) DEFAULT 0,
  request_count INT DEFAULT 0,
  success_count INT DEFAULT 0,
  error_count INT DEFAULT 0,
  last_error TEXT,
  last_error_at TIMESTAMP,
  
  -- 统计
  total_requests INT DEFAULT 0,
  total_success INT DEFAULT 0,
  total_errors INT DEFAULT 0,
  total_cost DECIMAL(10,4) DEFAULT 0,
  
  -- 时间戳
  created_at TIMESTAMP,
  updated_at TIMESTAMP,
  last_used_at TIMESTAMP,
  disabled_at TIMESTAMP,
  reactivated_at TIMESTAMP
);

CREATE INDEX idx_provider_status ON provider_accounts(provider, status);
CREATE INDEX idx_provider_priority ON provider_accounts(provider, priority);
```

## 🚀 使用示例

### 方式一：启动时跳过，后续 MCP 配置

```bash
# 1. 启动服务
make start

# 输出：
# ==========================================
#    Provider API 配置
# ==========================================
# 
# 是否现在配置 Provider? [y/N]: N
# 
# ✅ 跳过配置，稍后可通过 MCP/API 配置

# 2. 配置 MCP
./scripts/setup-mcp.sh --claude --api-key sk-manager-key

# 3. 在 Claude Code 中配置
claude

> "Create provider account for openai with key sk-xxx"
> "Add backup account for openai with key sk-yyy, priority 1"
> "Show provider status"
```

### 方式二：启动时配置

```bash
make start

# 是否现在配置 Provider? [y/N]: Y

# 配置 OpenAI
# OpenAI API Key: sk-xxx
# 账户名称 [openai-primary]: my-openai
# 是否配置备用账户? [y/N]: Y
# 备用账户 API Key: sk-yyy
# 月度限额 ($)(可选): 50

# ✅ 账户创建成功: my-openai (已设为默认)
# ✅ 备用账户创建成功 (优先级: 1, 月限: $50)
```

### 方式三：使用配置脚本

```bash
./scripts/setup-provider.sh

# 交互式配置各 Provider
# 支持多账户配置
# 自动设置优先级
```

## 📝 MCP 工具使用示例

### 查看账户

```
用户: "Show all provider accounts"

Claude: 
{
  "openai": {
    "total_accounts": 2,
    "active": 2,
    "limited": 0,
    "exhausted": 0,
    "status": "healthy",
    "accounts": [
      {
        "id": "uuid-1",
        "name": "primary",
        "status": "active",
        "is_default": true,
        "priority": 0,
        "used_this_month": "45.50"
      },
      {
        "id": "uuid-2",
        "name": "backup",
        "status": "active",
        "priority": 1,
        "monthly_limit": "100"
      }
    ]
  }
}
```

### 创建账户

```
用户: "Create openai account 'team-a' with key sk-team-a, priority 0"

Claude: ✅ 已成功创建账户：
{
  "id": "uuid-new",
  "provider": "openai",
  "name": "team-a",
  "status": "active",
  "is_default": true,
  "priority": 0,
  "message": "team-a is now available for openai"
}
```

### 禁用账户

```
用户: "Disable account uuid-2"

Claude: ✅ Provider account uuid-2 disabled. 
System will switch to next available account.
```

### 查看状态

```
用户: "Show provider status"

Claude:
{
  "openai": {
    "status": "healthy",
    "total_accounts": 2,
    "active": 2,
    "default_account": {
      "name": "primary",
      "used_this_month": "45.50",
      "error_count": 0
    }
  },
  "anthropic": {
    "status": "warning",
    "total_accounts": 1,
    "active": 1,
    "limited": 0,
    "exhausted": 0
  },
  "deepseek": {
    "status": "not_configured"
  }
}
```

## 🔄 状态管理

### 状态定义

| 状态 | 含义 | 触发条件 | 自动恢复 |
|------|------|----------|----------|
| `active` | 正常可用 | 默认状态 | - |
| `limited` | 临时受限 | Rate Limit 或连续错误≥5 | 5分钟（Rate Limit） |
| `exhausted` | 耗尽 | 余额不足或月度限额用完 | 每月1日 |
| `disabled` | 手动禁用 | MCP/API 调用 | ❌ 需手动启用 |

### 恢复机制

**自动恢复：**
- Rate Limit：5分钟后 goroutine 自动恢复
- 月度用量：每月1日 00:00 ResetMonthlyUsage()

**手动恢复：**
```bash
> "Enable account uuid-xxx"
```

## 🎨 最佳实践

### 1. 至少配置 2 个账户

每个 Provider 建议配置：
- 主账户（priority 0）：无限制
- 备用账户（priority 1）：月度限额控制成本

### 2. 监控账户状态

定期检查：
```bash
> "Show provider status"
```

关注：
- `warning` 状态：< 50% 账户可用
- `critical` 状态：无可用账户

### 3. 设置合理的月度限额

```bash
# 备用账户设置限额防止超支
> "Create openai account 'backup' with key sk-xxx, monthly limit 50"
```

### 4. 不同团队使用不同账户

```bash
# Team A
> "Create openai account 'team-a' with key sk-team-a, priority 0"

# Team B  
> "Create openai account 'team-b' with key sk-team-b, priority 1"

# Team C (低优先级)
> "Create openai account 'team-c' with key sk-team-c, priority 2, monthly limit 20"
```

## 📚 文档

完整文档已创建：
- `docs/provider-accounts-complete-guide.md` - 完整使用指南
- `docs/provider-accounts-implementation.md` - 技术实现细节
- `docs/provider-accounts-guide.md` - 快速开始指南

## ✅ 总结

Open Station 现已完整实现：

1. **✅ 启动时可配置 Provider（可跳过）**
   - make start 询问配置，可完全跳过
   - 后续随时通过 MCP/API 配置

2. **✅ 后续通过 API 和 MCP 配置**
   - 8 个 MCP 工具完整支持
   - Claude Code CLI 对话式配置
   - REST API endpoint

3. **✅ 同一 Provider 支持多个账户配置**
   - 数据库表完整设计
   - 优先级、限额、状态管理
   - 动态切换 Handler 实现

4. **✅ 自动故障切换**
   - Rate Limit：自动切换，5分钟恢复
   - 余额不足：自动切换，每月恢复
   - 连续失败：自动切换
   - 月度限额：自动切换

通过这些功能，Open Station 提供了高可用性、成本可控的 AI 网关服务！