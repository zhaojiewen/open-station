# Provider 多账户配置实现指南

## 已完成部分

### 1. 数据模型 ✅

**文件: `internal/domain/entity/entity.go`**
- 新增 `ProviderAccount` 实体，支持：
  - 多账户配置
  - 优先级排序
  - 状态管理（active, limited, exhausted, disabled）
  - 月度限额和用量统计
  - 错误计数和自动切换

### 2. Repository ✅

**文件: `internal/domain/repository/repository.go`**
- 新增 `ProviderAccountRepository` 接口

**文件: `internal/infrastructure/persistence/postgres/repositories/repositories.go`**
- 实现 `ProviderAccountRepoImpl`
- 包含所有必要的方法：
  - `GetDefaultByProvider` - 获取默认账户
  - `GetNextAvailable` - 获取下一个可用账户
  - `IncrementUsage` - 记录使用量
  - `RecordError` - 记录错误
  - `UpdateStatus` - 更新状态
  - `ResetMonthlyUsage` - 重置月度用量

### 3. Service ✅

**文件: `internal/application/service/provider_account_service.go`**
- 完整的 `ProviderAccountService` 实现
- 包含账户管理、切换逻辑、健康检查

**文件: `internal/application/service/provider_mcp_tools.go`**
- MCP 工具实现（9个工具）
  - `list_provider_accounts`
  - `create_provider_account`
  - `update_provider_account`
  - `set_default_provider_account`
  - `enable_provider_account`
  - `disable_provider_account`
  - `delete_provider_account`
  - `get_provider_status`

## 需要完成的步骤

### 1. 修改 main.go

```go
// 在 cmd/server/main.go 中添加
providerAccountRepo := repositories.NewProviderAccountRepository(db)
providerAccountService := service.NewProviderAccountService(providerAccountRepo)
mcpService := service.NewMCPService(authService, billingService, providerAccountService)

// 在 runMigrations 中添加
&entity.ProviderAccount{},
```

### 2. 合并 Provider MCP 工具到 mcp_service.go

将 `provider_mcp_tools.go` 中的工具函数复制到 `mcp_service.go`，并：
1. 在 `getManagerTools()` 中添加工具定义
2. 在 `executeTool()` switch 中添加 case 语句

工具定义（添加到 getManagerTools）:
```go
{Name: "list_provider_accounts", Title: "List Provider Accounts", Description: "List all provider accounts with status",
    InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
        "provider": map[string]interface{}{"type": "string", "description": "Provider name (optional)"},
        "status":   map[string]interface{}{"type": "string", "description": "Filter by status"},
    }}},
{Name: "create_provider_account", Title: "Create Provider Account", Description: "Add a new provider API account",
    InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
        "provider":     map[string]interface{}{"type": "string", "description": "Provider name"},
        "name":         map[string]interface{}{"type": "string", "description": "Account name"},
        "api_key":      map[string]interface{}{"type": "string", "description": "API key"},
        "base_url":     map[string]interface{}{"type": "string", "description": "Base URL (optional)"},
        "priority":     map[string]interface{}{"type": "integer", "description": "Priority"},
        "monthly_limit": map[string]interface{}{"type": "number", "description": "Monthly limit"},
    }, "required": []string{"provider", "name", "api_key"}}},
```

### 3. 修改 ProxyService 支持动态账户选择

```go
// 在 internal/infrastructure/proxy/proxy_service.go 中
// 替换静态配置为动态获取

type ProxyService struct {
    providerAccountService *service.ProviderAccountService
    // ...
}

func (s *ProxyService) getProviderAccount(ctx context.Context, provider string) (*entity.ProviderAccount, error) {
    return s.providerAccountService.GetActiveAccount(ctx, provider)
}

// 在请求失败时处理切换
func (s *ProxyService) handleRequestError(ctx context.Context, account *entity.ProviderAccount, errMsg string) {
    if s.providerAccountService.IsRateLimitError(errMsg) {
        s.providerAccountService.HandleRateLimit(ctx, account.ID)
    } else if s.providerAccountService.IsQuotaError(errMsg) {
        s.providerAccountService.HandleInsufficientQuota(ctx, account.ID)
    } else {
        s.providerAccountService.RecordError(ctx, account.ID, errMsg)
    }
}
```

### 4. 修改启动脚本

在 `scripts/start-docker.sh` 中添加 Provider 配置提示：

```bash
configure_providers() {
    echo ""
    echo "=========================================="
    echo "   Provider 配置"
    echo "=========================================="
    echo ""
    echo "可配置以下 Provider（可跳过，稍后通过 MCP/API 配置）："
    echo "  - openai"
    echo "  - anthropic (Claude)"
    echo "  - gemini"
    echo "  - deepseek"
    echo "  - glm"
    echo ""
    read -p "是否现在配置 Provider? [y/N]: " CONFIGURE_PROVIDERS

    if [[ "$CONFIGURE_PROVIDERS" =~ ^[Yy]$ ]]; then
        configure_provider_accounts
    fi
}

configure_provider_accounts() {
    echo ""
    for provider in openai anthropic gemini deepseek glm; do
        echo "配置 $provider:"
        read -p "API Key (可跳过): " API_KEY

        if [ -n "$API_KEY" ]; then
            read -p "账户名称 [$provider-default]: " ACCOUNT_NAME
            ACCOUNT_NAME=${ACCOUNT_NAME:-"$provider-default"}

            # 创建账户
            curl -X POST http://localhost:8080/mcp \
                -H "Authorization: Bearer $MANAGER_KEY" \
                -H "Content-Type: application/json" \
                -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"create_provider_account\",\"arguments\":{\"provider\":\"$provider\",\"name\":\"$ACCOUNT_NAME\",\"api_key\":\"$API_KEY\"}}}"

            echo "✅ $provider 已配置"
        fi
        echo ""
    done
}
```

### 5. 添加 HTTP API 接口

创建 `internal/interfaces/http/handler/provider_handler.go`:

```go
package handler

type ProviderHandler struct {
    providerAccountService *service.ProviderAccountService
}

// GET /admin/providers - 列出所有 Provider 账户
// POST /admin/providers - 创建新账户
// PUT /admin/providers/:id - 更新账户
// DELETE /admin/providers/:id - 删除账户
// POST /admin/providers/:id/set-default - 设置默认
// POST /admin/providers/:id/enable - 启用账户
// POST /admin/providers/:id/disable - 禁用账户
```

### 6. 添加定时任务

月度用量重置：

```go
// 每月 1 日 00:00 执行
func scheduleMonthlyReset() {
    for {
        now := time.Now()
        nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
        time.Sleep(nextMonth.Sub(now))

        providerAccountService.ResetMonthlyUsage(context.Background())
    }
}
```

## 使用示例

### MCP 工具使用

在 Claude Code CLI 中：

```bash
# 查看所有 Provider 状态
> "Show me provider status"

# 配置 OpenAI 账户
> "Add OpenAI account 'my-openai-key' with API key sk-xxx"

# 配置备用账户
> "Add another OpenAI account 'openai-backup' with API key sk-yyy, priority 1"

# 查看账户列表
> "List all OpenAI accounts"

# 设置默认账户
> "Set 'my-openai-key' as default for openai"

# 禁用账户
> "Disable OpenAI account 'old-key'"
```

### 账户切换逻辑

当请求失败时，系统自动：
1. 检测错误类型（rate limit / quota exceeded）
2. 记录错误并增加错误计数
3. 如果连续错误 ≥ 5 次，标记为 `limited`
4. 自动切换到下一个可用账户
5. 5分钟后尝试恢复原账户

### 月度限额

配置月度限额后：
- 账户花费接近限额（80%）→ 告警
- 账户超过限额 → 标记为 `exhausted`，自动切换
- 每月 1 日自动重置用量

## 快速配置脚本

创建 `scripts/configure-provider.sh`:

```bash
#!/bin/bash
# Provider 快速配置脚本

PROVIDER=$1
API_KEY=$2
NAME=$3

curl -X POST http://localhost:8080/mcp \
    -H "Authorization: Bearer $MANAGER_KEY" \
    -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"create_provider_account\",\"arguments\":{\"provider\":\"$PROVIDER\",\"name\":\"$NAME\",\"api_key\":\"$API_KEY\"}}}"
```

使用：
```bash
./scripts/configure-provider.sh openai sk-xxx my-openai-key
```