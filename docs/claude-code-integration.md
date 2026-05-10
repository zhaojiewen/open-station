# Claude Code CLI 接入指南

本指南说明如何让 Claude Code CLI 通过 open-station 网关访问 AI 模型。

## 方案架构

```
Claude Code CLI  →  open-station网关  →  多Provider转发
   (Anthropic格式)    (/v1/messages)      (格式转换)
                      ↓
                Claude/OpenAI/DeepSeek/GLM/Gemini
```

## 快速配置

### 方式一：环境变量配置

```bash
# 设置网关端点
export ANTHROPIC_BASE_URL="http://localhost:8080/v1"

# 设置API Key（从open-station网关获取）
export ANTHROPIC_API_KEY="sk-your-gateway-api-key"

# 启动Claude Code
claude
```

### 方式二：settings.json 配置

创建或编辑 `~/.claude/settings.json`：

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "http://localhost:8080/v1",
    "ANTHROPIC_API_KEY": "sk-your-gateway-api-key"
  }
}
```

### 方式三：项目级配置

在项目目录创建 `.claude/settings.local.json`（推荐，可加入 .gitignore）：

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "http://localhost:8080/v1",
    "ANTHROPIC_API_KEY": "sk-your-gateway-api-key"
  }
}
```

## 认证方式

open-station 网关支持两种认证方式：

| 方式 | Header | 配置变量 |
|------|--------|----------|
| API Key | `X-Api-Key` | `ANTHROPIC_API_KEY` |
| Bearer Token | `Authorization: Bearer` | `ANTHROPIC_AUTH_TOKEN` |

两种方式等效，推荐使用 `ANTHROPIC_API_KEY`。

## 模型访问

### Claude 模型（默认）

Claude Code CLI 默认使用 Claude 模型，网关会直接转发：

```bash
# Claude Code 自动选择 claude-sonnet-4-6
claude

# 指定模型
claude --model claude-opus-4-7
```

### 其他 Provider 模型

通过模型名称前缀访问其他 Provider：

| 格式 | Provider | 示例 |
|------|----------|------|
| `openai-{model}` | OpenAI | `openai-gpt-4o`, `openai-gpt-4o-mini` |
| `deepseek-{model}` | DeepSeek | `deepseek-v4-flash`, `deepseek-v4-pro` |
| `glm-{model}` | GLM智谱 | `glm-4.7`, `glm-4.5-air` |
| `gemini-{model}` | Google Gemini | `gemini-2.5-flash`, `gemini-3-flash-preview` |

示例：
```bash
# 使用 GPT-4o
claude --model openai-gpt-4o

# 使用 DeepSeek V4 Flash（性价比高）
claude --model deepseek-v4-flash

# 使用免费的 GLM-4 Flash
claude --model glm-4-flash

# 使用 Gemini 2.5 Flash
claude --model gemini-2.5-flash
```

## API 端点

open-station 网关提供的端点：

| 端点 | 用途 | 格式 |
|------|------|------|
| `/v1/messages` | Claude Code 主端点 | Anthropic Messages API |
| `/v1/models` | 模型列表 | Anthropic 格式 |
| `/v1/proxy/chat/completions` | 统一代理端点 | OpenAI 格式 |
| `/v1/{provider}/chat/completions` | Provider 直连 | OpenAI 格式 |

## 验证配置

### 方法一：Claude Code 内验证

```bash
claude
> /status
```

查看输出中的 API endpoint 是否为 `http://localhost:8080/v1`

### 方法二：直接测试网关

```bash
curl -X POST http://localhost:8080/v1/messages \
  -H "Content-Type: application/json" \
  -H "X-Api-Key: sk-your-api-key" \
  -d '{
    "model": "claude-sonnet-4-6",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

### 方法三：查看模型列表

```bash
curl http://localhost:8080/v1/models
```

## 高级配置

### 代理设置（如需要）

```bash
export HTTPS_PROXY="http://proxy.example.com:8080"
export HTTP_PROXY="http://proxy.example.com:8080"
```

### 自定义 Headers

```bash
export ANTHROPIC_CUSTOM_HEADERS="X-Custom-Header: value"
```

### 禁用 Beta Headers

如果网关不支持某些 beta 功能：

```bash
export CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS=1
```

## 获取 API Key

### 通过管理 API 创建

```bash
# 创建租户（管理员）
curl -X POST http://localhost:8080/admin/tenants \
  -H "Authorization: Bearer admin-key" \
  -H "Content-Type: application/json" \
  -d '{"name": "My Team", "slug": "my-team"}'

# 创建用户
curl -X POST http://localhost:8080/admin/users \
  -H "Authorization: Bearer admin-key" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": "tenant-id", "email": "user@example.com", "name": "User"}'

# 创建 API Key
curl -X POST http://localhost:8080/admin/api-keys \
  -H "Authorization: Bearer admin-key" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "user-id", "name": "Claude Code Key"}'
```

### 通过用户自助 API

```bash
# 登录后获取个人 API Key
curl -X POST http://localhost:8080/user/api-keys \
  -H "Authorization: Bearer your-existing-key" \
  -H "Content-Type: application/json" \
  -d '{"name": "My CLI Key"}'
```

## 计费说明

- 所有请求通过网关计费
- 按 Token 使用量扣费
- 支持余额充值
- 可查看使用明细和账单

```bash
# 查看余额
curl http://localhost:8080/admin/billing/balance/{tenant_id} \
  -H "Authorization: Bearer admin-key"

# 充值
curl -X POST http://localhost:8080/admin/billing/recharge \
  -H "Authorization: Bearer admin-key" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": "tenant-id", "amount": 100.00}'
```

## 故障排查

### 问题：连接失败

检查：
1. 网关是否运行：`curl http://localhost:8080/health`
2. 端点配置是否正确
3. API Key 是否有效

### 问题：认证失败

检查：
1. API Key 格式是否正确（应以 `sk-` 开头）
2. API Key 是否已激活
3. 是否有足够的余额

### 问题：模型不支持

检查：
1. 模型名称是否正确
2. 模型是否在数据库中配置
3. Provider API Key 是否配置

## 环境变量完整参考

| 变量 | 说明 | 示例 |
|------|------|------|
| `ANTHROPIC_BASE_URL` | 网关端点 | `http://localhost:8080/v1` |
| `ANTHROPIC_API_KEY` | API Key | `sk-xxx` |
| `ANTHROPIC_AUTH_TOKEN` | Bearer Token | `xxx` |
| `CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS` | 禁用 beta | `1` |
| `HTTPS_PROXY` | HTTP代理 | `http://proxy:8080` |

## 相关文档

- [Claude Code 官方文档](https://code.claude.com/docs)
- [open-station 项目 README](../README.md)
- [API 接口文档](./api.md)