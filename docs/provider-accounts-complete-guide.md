# Open Station - Provider 多账户配置与自动切换功能

## 功能概述

Open Station 现已完整支持 Provider 多账户配置和自动故障切换：

### ✅ 已实现的功能

1. **启动时可配置 Provider（可跳过）**
   - 一键启动脚本 `make start` 会询问是否配置 Provider
   - 可以完全跳过，系统会使用配置文件中的默认值（如果设置了环境变量）
   - 后续随时可以通过 API 或 MCP 配置

2. **后续通过 API 和 MCP 配置**
   - 8 个 MCP 工具完整支持账户管理
   - REST API 接口（通过 MCP 暴露）
   - Claude Code CLI 直接对话式管理

3. **同一 Provider 支持多个账户配置**
   - 每个账户可设置：名称、API Key、Base URL、优先级、月度限额
   - 第一个账户自动设为默认
   - 按优先级排序（0 = 最高优先）

4. **自动故障切换机制**
   - Rate Limit 错误：自动切换，5分钟后恢复
   - 余额不足（insufficient_quota）：自动切换
   - 连续失败 ≥ 5 次：标记为 limited，自动切换
   - 月度限额耗尽：标记为 exhausted，自动切换

## 使用指南

### 方式一：启动时配置（可选）

```bash
# 1. 克隆并启动
git clone https://github.com/zhaojiewen/open-station.git
cd open-station
make start

# 2. 系统会询问是否配置 Provider
# ==========================================
#    Provider API 配置
# ==========================================
# 
# 是否现在配置 Provider? [y/N]: 
# 
# 输入 N 即可跳过，稍后配置
```

### 方式二：启动后通过 MCP 配置（推荐）

```bash
# 1. 启动服务（跳过配置）
make start

# 2. 配置 MCP（如果未自动配置）
./scripts/setup-mcp.sh --claude --api-key sk-your-manager-key

# 3. 启动 Claude Code
claude

# 4. 在 Claude 中对话式配置
```

**配置示例对话：**

```
用户: "Create a provider account for openai with API key sk-xxx, name my-primary"

Claude: ✅ 已成功创建 OpenAI 账户：
        - 名称: my-primary
        - ID: uuid-xxx
        - 状态: active
        - 已设为默认账户

用户: "Add a backup account for openai with key sk-yyy, priority 1, monthly limit 50 dollars"

Claude: ✅ 已成功创建 OpenAI 备用账户：
        - 名称: openai-backup-1
        - ID: uuid-yyy
        - 优先级: 1 (故障切换顺序)
        - 月度限额: $50

用户: "Show all provider accounts"

Claude: 当前 Provider 配置状态：
        
        OpenAI:
          - my-primary (默认) - active, 优先级 0
          - openai-backup-1   - active, 优先级 1, 月限 $50
        
        Anthropic: 未配置
        
        DeepSeek: 未配置
```

### 方式三：使用配置脚本

```bash
# 交互式配置脚本
./scripts/setup-provider.sh

# 输入 Manager API Key（从启动日志获取）
# 按提示配置各 Provider
```

## MCP 工具列表（8个）

| 工具 | 功能 | 示例对话 |
|------|------|----------|
| `list_provider_accounts` | 查看所有账户 | "Show provider accounts" |
| `create_provider_account` | 创建新账户 | "Create provider account for openai with key sk-xxx" |
| `update_provider_account` | 更新账户配置 | "Update account xxx priority to 2" |
| `set_default_provider_account` | 设置默认账户 | "Set xxx as default for openai" |
| `enable_provider_account` | 启用账户 | "Enable account xxx" |
| `disable_provider_account` | 禁用账户 | "Disable account xxx" |
| `delete_provider_account` | 删除账户 | "Delete account xxx" |
| `get_provider_status` | 查看状态摘要 | "Show provider status" |

## 多账户自动切换机制

### 触发条件

#### 1. Rate Limit 错误（429）

```
请求 → Rate Limit 错误 → 标记账户为 limited → 
自动切换到下一个可用账户 → 5分钟后恢复原账户
```

**识别关键词：**
- "rate limit"
- "rate_limit"  
- "too many requests"
- "429"
- "quota exceeded"
- "TPM/RPM"

#### 2. 余额不足（insufficient_quota）

```
请求 → 余额不足错误 → 标记账户为 exhausted → 
自动切换到下一个可用账户 → 每月1日自动重置
```

**识别关键词：**
- "insufficient_quota"
- "insufficient quota"
- "billing_hard_limit_reached"
- "no remaining credits"
- "out of credits"

#### 3. 连续失败 ≥ 5 次

```
连续失败计数 ≥ 5 → 标记账户为 limited → 
自动切换到下一个可用账户 → 成功请求后清零计数
```

#### 4. 月度限额耗尽

```
账户花费 ≥ 月度限额 → 标记账户为 exhausted → 
自动切换 → 每月1日重置用量并恢复
```

### 切换顺序

按优先级选择账户：
- Priority 0 = 最高优先（默认账户）
- Priority 1, 2, 3... = 备用账户

切换逻辑：
```
默认账户 (Priority 0) → 备用账户1 (Priority 1) → 备用账户2 → ...
```

### 状态说明

| 状态 | 含义 | 自动恢复 | 手动恢复 |
|------|------|----------|----------|
| `active` | 正常可用 | - | - |
| `limited` | 遇到限制（Rate Limit/连续错误） | Rate Limit: 5分钟后 | MCP enable |
| `exhausted` | 余额耗尽/月度限额用完 | 每月1日 | MCP enable |
| `disabled` | 手动禁用 | ❌ | MCP enable |

### 恢复机制

**自动恢复：**
- Rate Limit：5分钟后自动恢复为 active
- 月度限额：每月1日 00:00 自动重置所有账户

**手动恢复：**
```bash
# Claude Code CLI
> "Enable account xxx"

# 或通过 API
curl -X POST http://localhost:8080/mcp \
  -H "Authorization: Bearer $MANAGER_KEY" \
  -d '{"method":"tools/call","params":{"name":"enable_provider_account","arguments":{"account_id":"xxx"}}}'
```

## 完整配置示例

### 配置 OpenAI 双账户（主账户 + 备用）

```bash
# Claude Code CLI
> "Create openai account 'primary' with key sk-primary-key"
> "Create openai account 'backup' with key sk-backup-key, priority 1, monthly limit 100"
```

**结果：**
- 主账户 `primary`：优先级 0，无限额
- 备用账户 `backup`：优先级 1，月限 $100
- 当主账户遇到 Rate Limit → 自动切换到 backup
- 5分钟后主账户恢复，继续使用主账户

### 配置 Claude 三账户（企业级）

```bash
> "Create anthropic account 'claude-prod' with key sk-prod, priority 0"
> "Create anthropic account 'claude-staging' with key sk-staging, priority 1"
> "Create anthropic account 'claude-dev' with key sk-dev, priority 2, monthly limit 50"
```

**故障切换流程：**
```
prod → Rate Limit → staging → Rate Limit → dev
  ↓                      ↓                  ↓
 5分钟后恢复            5分钟后恢复         月限$50
```

### 按团队分配账户

```bash
# Team A 使用主账户
> "Create openai account 'team-a' with key sk-team-a"

# Team B 使用备用账户  
> "Create openai account 'team-b' with key sk-team-b, priority 1"

# Team C 使用低优先级账户
> "Create openai account 'team-c' with key sk-team-c, priority 2, monthly limit 20"
```

## 监控和告警

### 查看状态

```bash
# Claude Code CLI
> "Show provider status"

# 返回示例：
OpenAI:
  total_accounts: 3
  active: 2
  limited: 1
  exhausted: 0
  status: warning (< 50% accounts available)
  default_account:
    name: primary
    used_this_month: $45.50

Anthropic:
  total_accounts: 1
  active: 1
  status: healthy
```

### 健康状态定义

| 状态 | 条件 | 建议 |
|------|------|------|
| `healthy` | 所有账户正常 | ✅ |
| `warning` | < 50% 账户可用 | ⚠️ 添加备用账户 |
| `critical` | 无可用账户 | ❌ 立即添加账户 |
| `not_configured` | 未配置账户 | 配置至少一个账户 |

### 查看账户详情

```bash
> "Show openai accounts"

# 返回：
[
  {
    "id": "uuid-1",
    "name": "primary",
    "status": "active",
    "is_default": true,
    "priority": 0,
    "used_this_month": "45.50",
    "error_count": 0
  },
  {
    "id": "uuid-2",
    "name": "backup",
    "status": "limited",
    "priority": 1,
    "error_count": 5,
    "last_error": "rate limit exceeded"
  }
]
```

## 月度管理

### 自动重置（每月1日）

系统会自动执行：
- 重置所有账户的 `used_this_month` 为 0
- 将 `exhausted` 状态恢复为 `active`
- 清零 `error_count`
- 清零 `request_count` 和 `success_count`

### 手动重置

```bash
# 通过 MCP（需要管理员权限）
curl -X POST http://localhost:8080/admin/providers/reset-monthly \
  -H "Authorization: Bearer $MANAGER_KEY"
```

## 技术实现

### 数据库表结构

**provider_accounts 表：**

```sql
CREATE TABLE provider_accounts (
  id UUID PRIMARY KEY,
  provider VARCHAR(50) NOT NULL,      -- openai, anthropic, gemini, deepseek, glm
  name VARCHAR(255) NOT NULL,         -- 账户名称
  api_key VARCHAR(255) NOT NULL,      -- API Key
  base_url VARCHAR(255),              -- API Base URL（可选）
  priority INT DEFAULT 0,             -- 优先级（0 最高）
  status VARCHAR(20) DEFAULT 'active', -- active, limited, exhausted, disabled
  is_default BOOLEAN DEFAULT FALSE,   -- 是否默认账户
  
  monthly_limit DECIMAL(10,2),        -- 月度限额（可选）
  used_this_month DECIMAL(10,4),      -- 本月已用
  request_count INT,                  -- 本月请求次数
  success_count INT,                  -- 本月成功次数
  error_count INT,                    -- 连续错误次数
  last_error TEXT,                    -- 最后错误信息
  last_error_at TIMESTAMP,            -- 最后错误时间
  
  total_requests INT,                 -- 总请求次数
  total_success INT,                  -- 总成功次数
  total_errors INT,                   -- 总错误次数
  total_cost DECIMAL(10,4),           -- 总花费
  
  created_at TIMESTAMP,
  updated_at TIMESTAMP,
  last_used_at TIMESTAMP,
  disabled_at TIMESTAMP,
  reactivated_at TIMESTAMP
);
```

### 核心服务

**ProviderAccountService：**
- `CreateAccount()` - 创建账户
- `GetActiveAccount()` - 获取当前活跃账户
- `RecordSuccess()` - 记录成功请求
- `RecordError()` - 记录失败请求
- `HandleRateLimit()` - 处理 Rate Limit
- `HandleInsufficientQuota()` - 处理余额不足
- `ResetMonthlyUsage()` - 重置月度用量

**DynamicProxyHandler：**
- 在请求时动态选择账户
- 自动切换和重试
- 成本计算和记录

### 请求流程

```
用户请求 → DynamicProxyHandler → 
  ↓
GetActiveAccount(provider) → 
  ↓
使用账户 API Key 发送请求 → 
  ↓
成功？ → RecordSuccess(cost) → 返回响应
  ↓
失败？ → 检查错误类型 →
  ↓ Rate Limit
HandleRateLimit() → 切换账户 → 重试请求
  ↓ Quota Error  
HandleInsufficientQuota() → 切换账户 → 重试请求
  ↓ 其他错误
RecordError() → 返回错误
```

## API 接口

### MCP 端点

```
POST /mcp
```

**创建账户：**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "create_provider_account",
    "arguments": {
      "provider": "openai",
      "name": "my-account",
      "api_key": "sk-xxx",
      "priority": 0,
      "monthly_limit": 100
    }
  }
}
```

### 管理 API（未来实现）

```
GET  /admin/providers                  # 查看所有 Provider
GET  /admin/providers/:provider        # 查看特定 Provider
POST /admin/providers/accounts         # 创建账户
PUT  /admin/providers/accounts/:id     # 更新账户
DELETE /admin/providers/accounts/:id   # 删除账户
POST /admin/providers/reset-monthly    # 重置月度用量
```

## 最佳实践

### 1. 至少配置 2 个账户

```bash
# 每个 Provider 至少配置一个主账户和一个备用账户
> "Create openai account 'main' with key sk-main"
> "Create openai account 'backup' with key sk-backup, priority 1"
```

### 2. 设置合理的月度限额

```bash
# 为备用账户设置限额，防止意外超支
> "Create openai account 'backup' with key sk-xxx, monthly limit 50"
```

### 3. 定期监控状态

```bash
# 每周检查一次
> "Show provider status"

# 关注 warning/critical 状态
```

### 4. 及时清理不用的账户

```bash
> "Delete account old-account-id"
```

## 故障排查

### 账户无法切换

**原因：** 只有一个账户或所有账户都被标记为 limited/exhausted

**解决：** 添加备用账户或手动启用账户

```bash
> "Enable account xxx"
```

### Rate Limit 持续触发

**原因：** 所有账户都遇到 Rate Limit

**解决：** 
1. 增加账户数量
2. 降低请求频率
3. 使用不同 Provider

### 月度限额提前耗尽

**原因：** 请求量超出预期

**解决：**
1. 提高月度限额
2. 添加更多备用账户
3. 使用成本更低的模型

## 总结

Open Station 的多账户管理功能提供了：

✅ **灵活配置** - 启动时可跳过，随时可配置  
✅ **自动切换** - Rate Limit、余额不足自动切换  
✅ **多账户支持** - 同一 Provider 支持多个账户  
✅ **优先级管理** - 按优先级自动选择和切换  
✅ **成本控制** - 月度限额防止超支  
✅ **完整监控** - MCP 工具实时查看状态  

通过这些功能，确保 AI 服务的高可用性和成本可控性。