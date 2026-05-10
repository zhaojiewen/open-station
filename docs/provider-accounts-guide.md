# Open Station - Provider 多账户配置方案

## 功能概述

Open Station 现支持 Provider 多账户配置，实现：
- ✅ 启动时可配置 Provider（可跳过）
- ✅ 后续通过 API 和 MCP 配置
- ✅ 同一 Provider 支持多个账户
- ✅ 自动故障切换

## 使用流程

### 1. 快速启动

```bash
# 克隆项目
git clone https://github.com/zhaojiewen/open-station.git
cd open-station

# 一键启动（自动安装 Docker，创建管理员）
make start

# 提示配置 Provider，可跳过
```

### 2. Provider 配置（启动后）

```bash
# 方式一：交互式配置脚本
./scripts/setup-provider.sh

# 方式二：Claude Code CLI MCP 工具
claude

# 在 Claude 中输入
> "Create provider account for openai with API key sk-xxx, name my-openai"
> "Add backup account for openai with key sk-yyy, priority 1"
> "Show provider status"
```

### 3. MCP Provider 工具

| 工具 | 功能 | 示例 |
|------|------|------|
| `list_provider_accounts` | 查看所有账户 | "Show provider accounts" |
| `create_provider_account` | 创建新账户 | "Create provider account for openai" |
| `update_provider_account` | 更新账户 | "Update account xxx priority to 2" |
| `set_default_provider_account` | 设置默认 | "Set xxx as default for openai" |
| `enable_provider_account` | 启用账户 | "Enable account xxx" |
| `disable_provider_account` | 禁用账户 | "Disable account xxx" |
| `delete_provider_account` | 删除账户 | "Delete account xxx" |
| `get_provider_status` | 查看状态 | "Show provider status" |

## 多账户切换机制

### 自动切换触发条件

1. **Rate Limit 错误**
   - 检测到 rate limit 错误
   - 标记账户为 `limited`
   - 切换到下一个可用账户
   - 5分钟后尝试恢复

2. **余额不足**
   - 检测到 insufficient_quota 错误
   - 标记账户为 `exhausted`
   - 切换到下一个可用账户

3. **连续错误**
   - 连续失败 ≥ 5 次
   - 标记账户为 `limited`
   - 自动切换

4. **月度限额**
   - 账户花费 ≥ 月度限额
   - 标记账户为 `exhausted`
   - 自动切换

### 切换顺序

按优先级选择账户：
- Priority 0 = 最高优先（默认账户）
- Priority 1, 2, 3... = 备用账户

切换逻辑：
```
默认账户 → 备用账户1 → 备用账户2 → ...
```

### 恢复机制

- Rate Limit：5分钟后自动恢复
- 月度限额：每月 1 日重置用量
- 手动恢复：通过 MCP/API 启用账户

## 配置示例

### 配置 OpenAI 双账户

```bash
# Claude Code CLI
> "Create provider account for openai with key sk-primary, name primary, priority 0"
> "Create provider account for openai with key sk-backup, name backup, priority 1, monthly_limit 50"

# 或使用脚本
./scripts/setup-provider.sh
```

结果：
```
OpenAI:
  - primary (默认) - 优先级 0, 无限额
  - backup (备用)  - 优先级 1, 月限 $50
```

### 配置 Claude 三账户

```bash
> "Add anthropic account 'claude-main' with key sk-xxx, priority 0"
> "Add anthropic account 'claude-backup-1' with key sk-yyy, priority 1"
> "Add anthropic account 'claude-backup-2' with key sk-zzz, priority 2, monthly_limit 100"
```

## 状态说明

| 状态 | 含义 | 自动恢复 |
|------|------|----------|
| `active` | 正常可用 | - |
| `limited` | 遇到限制 | Rate Limit: 5分钟 |
| `exhausted` | 余额耗尽 | 每月 1 日 |
| `disabled` | 手动禁用 | 需手动启用 |

## 月度管理

### 自动重置

每月 1 日 00:00 自动执行：
- 重置所有账户的月度用量
- 将 `exhausted` 状态恢复为 `active`
- 清零错误计数

### 手动管理

```bash
# 重置月度用量
curl -X POST http://localhost:8080/admin/providers/reset-monthly \
  -H "Authorization: Bearer $MANAGER_KEY"
```

## 监控和告警

### 查看状态

```bash
# Claude Code CLI
> "Show provider status"

# API
curl http://localhost:8080/admin/providers/status \
  -H "Authorization: Bearer $MANAGER_KEY"
```

返回示例：
```json
{
  "openai": {
    "total_accounts": 2,
    "active": 1,
    "limited": 1,
    "status": "warning",
    "default_account": {
      "name": "primary",
      "used_this_month": "45.50"
    }
  }
}
```

### 健康状态

| 状态 | 条件 |
|------|------|
| `healthy` | 所有账户正常 |
| `warning` | < 50% 账户可用 |
| `critical` | 无可用账户 |
| `not_configured` | 未配置账户 |

## 常见场景

### 场景1：配置多个账户避免中断

```bash
# 主账户遇到限制 → 自动切换备用账户 → 无中断
> "Create openai account 'main' with key sk-main"
> "Create openai account 'backup' with key sk-backup"
```

### 场景2：控制成本

```bash
# 设置月度限额，防止超支
> "Create openai account 'budget' with key sk-xxx, monthly_limit 50"
```

### 场景3：按团队分配

```bash
# 不同团队使用不同账户
> "Create anthropic account 'team-a' with key sk-team-a"
> "Create anthropic account 'team-b' with key sk-team-b"
```

## 技术实现

详见: [docs/provider-accounts-implementation.md](provider-accounts-implementation.md)

核心组件：
- `ProviderAccount` 实体 - 存储账户配置
- `ProviderAccountService` - 管理账户和切换
- `ProxyService` - 动态选择账户
- MCP 工具 - 用户管理接口

## 快速配置命令

```bash
# 配置单个账户
./scripts/configure-provider.sh openai sk-xxx my-key

# 查看状态
make status

# Claude Code CLI
claude --model openai-gpt-4o
```