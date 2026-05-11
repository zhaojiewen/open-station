# Open Station

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](deployments/docker/)
[![Claude Code](https://img.shields.io/badge/Claude_Code-Compatible-7C3AED?style=flat)](docs/claude-code-integration.md)
[![Plugin System](https://img.shields.io/badge/Plugin_System-v1.0-green?style=flat)](docs/plugin-development.md)

> 企业级AI网关 - 多模型代理、插件化扩展、MCP服务一体化解决方案

**简体中文** | [English](README_EN.md) | [文档目录](docs/)

---

## ✨ 核心特性

| 特性 | 说明 |
|------|------|
| 🔐 **用户登录注册** | JWT认证、多租户支持、登录安全防护、密码复杂度验证 |
| 🔌 **插件化架构** | 支持Go Native (.so) 和外部适配器 (HTTP/gRPC) 两种插件类型 |
| 🤖 **多模型代理** | 统一转发 OpenAI、Claude、Gemini、DeepSeek、GLM (44+模型) |
| 💬 **Claude Code兼容** | 完整 Anthropic Messages API 支持，可直接接入 CLI |
| 📊 **MCP服务** | 26个MCP工具，通过 Claude Code 管理API Key、余额、插件 |
| 💰 **企业支付系统** | 双层配额控制、后付费信用额度、多支付渠道集成 |
| 🛡️ **多层安全** | 双层限流、权限控制、API Key认证、Redis缓存加速 |
| 🌊 **流式响应** | SSE完整实现，所有Provider支持实时流式输出 |
| 📦 **插件市场** | 本地配置管理，一键安装/配置/激活插件 |

---

## 🚀 30秒快速启动

```bash
# 克隆项目
git clone https://github.com/zhaojiewen/open-station.git && cd open-station

# 一键启动（自动安装 Docker + 创建管理员）
make start

# 查看 API Key（首次启动自动创建）
docker logs open-station-gateway 2>&1 | grep "API Key"
```

**启动成功后**:
- 网关地址: `http://localhost:8080`
- MCP端点: `http://localhost:8080/mcp`
- 健康检查: `http://localhost:8080/health`

---

## 📦 安装指南

### 方式一: 预编译二进制 (推荐)

从 GitHub Release 下载预编译版本:

```bash
# 一键安装 (Linux/macOS)
VERSION=$(curl -s https://api.github.com/repos/zhaojiewen/open-station/releases/latest | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
PLATFORM=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/;s/armv7l/armv7/')

curl -sL https://github.com/zhaojiewen/open-station/releases/download/v${VERSION}/open-station-${VERSION}-${PLATFORM}-${ARCH}.tar.gz | tar xz
./install.sh

# Windows (PowerShell)
$VERSION = (Invoke-RestMethod https://api.github.com/repos/zhaojiewen/open-station/releases/latest).tag_name -replace 'v'
$Arch = if ($ENV:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }
curl -LO https://github.com/zhaojiewen/open-station/releases/download/v$VERSION/open-station-$VERSION-windows-$Arch.zip
Expand-Archive open-station-$VERSION-windows-$Arch.zip
```

### 方式二: Docker

```bash
# Docker Hub
docker pull zhaojiewen/open-station:latest
docker run -d --name open-station -p 8080:8080 zhaojiewen/open-station:latest

# GitHub Container Registry (推荐国内用户)
docker pull ghcr.io/zhaojiewen/open-station:latest
docker run -d --name open-station -p 8080:8080 ghcr.io/zhaojiewen/open-station:latest

# Docker Compose (包含 PostgreSQL + Redis)
git clone https://github.com/zhaojiewen/open-station.git && cd open-station
docker-compose -f deployments/docker/docker-compose.yml up -d
```

### 方式三: 源码编译

```bash
git clone https://github.com/zhaojiewen/open-station.git && cd open-station
go mod download
make build
./bin/open-station -config configs/config.yaml
```

### 方式四: Go Install

```bash
go install github.com/zhaojiewen/open-station/cmd/server@latest
open-station -config configs/config.yaml
```

---

## 📋 目录

- [支持的模型](#-支持的模型)
- [Claude Code 接入](#-claude-code-接入)
- [MCP 工具列表](#-mcp-工具列表)
- [插件系统](#-插件系统)
- [API 参考](#-api-参考)
- [配置说明](#️-配置说明)
- [架构设计](#-架构设计)
- [开发指南](#-开发指南)
- [部署方案](#-部署方案)
- [路线图](#-路线图)
- [贡献指南](#-贡献指南)

---

## 🎯 支持的模型

| Provider | 模型数 | 热门模型 | 特点 |
|----------|-------|---------|------|
| **Claude** | 9 | Opus 4.7, Sonnet 4.6, Haiku 4.5 | 最强推理能力 |
| **OpenAI** | 8 | GPT-4o, O1-mini, GPT-4o-mini | 生态丰富 |
| **Gemini** | 7 | Gemini 3.1 Pro, Gemini 2.5 Flash | 多模态支持 |
| **DeepSeek** | 2 | V4 Pro, V4 Flash | **超高性价比** |
| **GLM** | 18 | GLM-5.1, GLM-4-Flash | **免费模型可用** |

> 通过前缀访问其他Provider: `openai-{model}`, `deepseek-{model}`, `glm-{model}`, `gemini-{model}`

---

## 💬 Claude Code 接入

### 方式一: MCP 服务（推荐）

```bash
# 配置 MCP（自动写入 ~/.claude/settings.json）
./scripts/setup-mcp.sh --api-key sk-your-manager-key

# 启动 Claude Code
claude

# 自然语言管理
> "What's my balance?"                    # 查询余额
> "Create API key for john@example.com"   # 创建用户+Key
> "Install Mistral plugin"                # 安装插件
> "Add $100 to tenant abc balance"        # 充值
```

### 方式二: API 代理

```bash
# 设置环境变量
export ANTHROPIC_BASE_URL="http://localhost:8080/v1"
export ANTHROPIC_API_KEY="sk-your-gateway-key"

# 直接使用，跨Provider访问
claude --model claude-opus-4-7          # Claude (默认)
claude --model openai-gpt-4o            # OpenAI
claude --model deepseek-v4-flash        # DeepSeek (性价比最高)
claude --model glm-4-flash              # GLM (免费)
```

> 详细文档: [Claude Code 集成指南](docs/claude-code-integration.md)

---

## 📊 MCP 工具列表

### 用户工具 (6个)

| 工具 | 功能 | 示例 |
|------|------|------|
| `check_balance` | 查询余额 | "What's my balance?" |
| `get_usage_summary` | 用量汇总 | "Show this month usage" |
| `get_usage_details` | 用量明细 | "Show detailed usage" |
| `get_billing_info` | 计费信息 | "Get billing info" |
| `get_recharge_history` | 充值记录 | "Show recharge history" |
| `get_my_api_keys` | 我的Keys | "List my API keys" |

### 管理工具 (9个)

| 工具 | 功能 | 示例 |
|------|------|------|
| `list_all_api_keys` | 所有Keys | "List all API keys" |
| `create_api_key` | 创建Key | "Create key for john@example.com" |
| `revoke_api_key` | 撤销Key | "Revoke key abc-123" |
| `update_api_key` | 更新权限 | "Add embedding permission to key xyz" |
| `list_users` | 用户列表 | "Show all users" |
| `adjust_balance` | 调整余额 | "Add $50 to tenant abc" |
| `get_tenant_summary` | 租户摘要 | "Get tenant abc summary" |
| `list_tenants` | 租户列表 | "List all tenants" |

### 插件工具 (11个)

| 工具 | 功能 | 权限 |
|------|------|------|
| `list_plugins` | 已安装插件 | 用户 |
| `list_available_plugins` | 可安装插件 | 用户 |
| `get_plugin_status` | 插件状态 | 用户 |
| `get_plugin_providers` | Provider列表 | 用户 |
| `install_plugin` | 安装插件 | 管理员 |
| `configure_plugin` | 配置插件 | 管理员 |
| `activate_plugin` | 激活插件 | 管理员 |
| `deactivate_plugin` | 停用插件 | 管理员 |
| `uninstall_plugin` | 卸载插件 | 管理员 |
| `check_plugin_health` | 健康检查 | 管理员 |
| `get_all_plugin_stats` | 插件统计 | 管理员 |

> 详细文档: [MCP 集成指南](docs/mcp-integration.md)

---

## 💰 企业支付系统

### 双用户模式

| 模式 | 适用场景 | 配额来源 | 后付费 |
|------|----------|----------|--------|
| **个人模式** | 公共租户用户 | UserQuota (独立配额) | 不支持 |
| **组织模式** | 企业租户成员 | Tenant共享 + MemberQuota控制 | 支持 (需审核) |

### 扣减优先级 (统一)

```
1. 套餐Token额度 (优先扣减)
   ↓ 用完后
2. 预充值余额
   ↓ 余额为0时
3. 后付费信用额度 (仅企业，需审核通过)
```

### 企业后付费流程

```
企业申请 → 平台审核 → 审核通过 → 设置信用额度 → 使用 → 结算
```

**结算周期**: 月结、周结、阈值触发、自定义

### 成员配额控制

企业租户管理员可为成员设置独立配额:
- **Token配额上限**: 限制成员月度Token使用量
- **费用限额**: 控制成员费用支出上限

### 支付渠道

| 渠道 | 适用地区 | 支付方式 |
|------|----------|----------|
| **支付宝** | 国内 | 扫码、网页、APP |
| **微信支付** | 国内 | 扫码、网页、APP |
| **Stripe** | 国际 | 信用卡、网页 |
| **PayPal** | 国际 | 账户余额、信用卡 |
| **银行转账** | 企业 | 线下转账 |

> 详细文档: [支付系统集成指南](docs/payment-system.md)

---

## 🔐 用户认证系统

### 多租户架构

用户可同时属于多个租户，各租户数据和计费完全隔离：

| 功能 | 说明 |
|------|------|
| **个人注册** | 自动加入公共租户，使用个人配额(UserQuota) |
| **企业注册** | 创建新租户，自动成为管理员 |
| **租户切换** | 通过API切换当前活跃租户 |
| **多租户归属** | 一个用户可属于多个租户，通过邀请加入 |

### JWT认证

采用JWT Token进行用户认证：

| Token类型 | 有效期 | 用途 |
|----------|--------|------|
| **Access Token** | 15分钟 | API访问凭证 |
| **Refresh Token** | 7天 | 刷新Access Token |

### 登录安全防护

多重安全措施保护登录流程：

| 安全措施 | 配置 |
|----------|------|
| **失败限制** | 5次失败后封禁IP 15分钟 |
| **密码哈希** | bcrypt (cost=12) |
| **密码复杂度** | 最少8位，禁止常见弱密码 |
| **密码历史** | 检查最近5次密码，防止循环使用 |
| **审计日志** | 记录所有登录尝试（IP、设备、结果） |
| **异常检测** | 新设备/新IP登录提醒 |
| **敏感数据加密** | AES-256-GCM加密IP、UserAgent等 |

### 认证API端点

```bash
# 公开接口（无需认证）
POST /auth/login              # 用户登录
POST /auth/register           # 个人注册（加入公共租户）
POST /auth/tenant/register    # 企业注册（创建新租户）
POST /auth/refresh            # 刷新Token

# 需认证接口（需要JWT Token）
POST /auth/logout             # 登出当前设备
POST /auth/logout-all         # 登出所有设备
GET  /auth/profile            # 获取用户信息
GET  /auth/tenants            # 获取用户所有租户
POST /auth/switch-tenant      # 切换当前租户
PUT  /auth/password           # 修改密码
```

### 认证配置

```yaml
# configs/config.yaml
auth:
  jwt:
    secret_key: "${JWT_SECRET}"           # JWT签名密钥（必填）
    access_token_expire: 15m              # Access Token有效期
    refresh_token_expire: 168h            # Refresh Token有效期（7天）
  
  login_security:
    max_failed_attempts: 5                # 最大失败次数
    failed_window: 15m                    # 失败计数窗口
    block_duration: 30m                   # 封禁时长
    enable_audit_log: true                # 记录审计日志
    encrypt_audit_data: true              # 加密审计数据
    anomaly_detection: true               # 异常登录检测
    new_device_alert: true                # 新设备提醒
    
  password:
    min_length: 8                         # 最小长度
    max_length: 64                        # 最大长度
    require_upper: true                   # 需要大写字母
    require_lower: true                   # 需要小写字母
    require_digit: true                   # 需要数字
    require_special: true                 # 需要特殊字符
    history_count: 5                      # 检查历史密码数
    bcrypt_cost: 12                       # bcrypt强度
```

### 使用示例

```bash
# 用户登录
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'

# 响应示例
{
  "user": {"id": "...", "email": "user@example.com", "name": "User"},
  "tenants": [{"tenant_id": "...", "role": "admin", "is_default": true}],
  "current_tenant_id": "...",
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2024-01-01T00:15:00Z"
}

# 个人注册
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"newuser@example.com","password":"password123","name":"New User"}'

# 企业注册
curl -X POST http://localhost:8080/auth/tenant/register \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_name": "My Company",
    "tenant_slug": "my-company",
    "email": "admin@example.com",
    "password": "password123",
    "name": "Admin User"
  }'

# 切换租户
curl -X POST http://localhost:8080/auth/switch-tenant \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": "uuid-of-tenant"}'

# 修改密码
curl -X PUT http://localhost:8080/auth/password \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"current_password":"old123","new_password":"new123"}'
```

> 详细文档: [用户认证系统指南](docs/auth-system.md)

---

## 🔌 插件系统

### 插件类型

| 类型 | 说明 | 适用场景 |
|------|------|----------|
| **Go Native (.so)** | 编译为动态库 | 高性能、深度集成 |
| **外部适配器** | HTTP/gRPC服务 | 灵活部署、独立运维 |

### 内置插件 (5个)

| 插件 | Provider | 能力 |
|------|----------|------|
| `openai` | OpenAI | Chat, Stream, Embedding |
| `anthropic` | Anthropic | Chat, Stream |
| `gemini` | Google | Chat, Stream, Embedding |
| `deepseek` | DeepSeek | Chat, Stream |
| `glm` | Zhipu AI | Chat, Stream, Embedding |

### 配置新插件

```yaml
# configs/config.yaml
plugins:
  enabled: true
  available_plugins:
    mistral:                              # 添加 Mistral
      name: "Mistral AI"
      type: "adapter"
      provider: "mistral"
      adapter_url: "http://localhost:8081"
      capabilities: [chat, stream]
      config_schema:
        type: object
        properties:
          api_key: {type: string, required: true}
```

```bash
# 通过 MCP 管理
claude
> "Install Mistral plugin"
> "Configure Mistral with API key sk-xxx"
> "Activate Mistral plugin"
```

> 详细文档: [插件开发指南](docs/plugin-development.md)

---

## 🔗 API 参考

### Anthropic 兼容接口

```bash
POST /v1/messages          # Messages API (SSE流式)
GET  /v1/models            # 动态模型列表

# 示例：流式请求
curl -X POST http://localhost:8080/v1/messages \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-sonnet-4-6","max_tokens":1024,"messages":[{"role":"user","content":"Hello"}],"stream":true}'
```

### MCP 接口

```bash
POST /mcp                  # JSON-RPC 端点
GET  /mcp                  # SSE 流式端点
```

### 管理接口

```bash
# 计费
GET  /admin/billing/balance/:id    # 余额
POST /admin/billing/recharge       # 充值

# API Key
GET  /admin/api-keys               # 列表
POST /admin/api-keys               # 创建
POST /admin/api-keys/:id/revoke    # 撤销

# 插件 (14个端点)
GET  /admin/plugins                # 已安装列表
GET  /admin/plugins/available      # 可安装列表
POST /admin/plugins/:id/install    # 安装
POST /admin/plugins/:id/activate   # 激活
# ... 完整列表见上文 MCP 插件工具

# 成员配额管理 (企业租户管理员)
GET  /admin/member-quotas          # 成员配额列表
POST /admin/member-quotas          # 创建成员配额
PUT  /admin/member-quotas/:id      # 更新成员配额
PUT  /admin/member-quotas/:id/token-limit  # 设置Token配额
PUT  /admin/member-quotas/:id/cost-limit   # 设置费用限额
GET  /admin/member-quotas/:id/usage # 成员使用统计
```

### 支付系统接口

```bash
# 企业后付费申请 (租户管理员)
POST /tenant/credit-application              # 申请后付费信用额度
GET  /tenant/credit-application              # 查看申请状态
PUT  /tenant/credit-application              # 更新申请 (审核前)
DELETE /tenant/credit-application            # 取消申请

# 平台管理员审核
GET  /platform/credit-applications           # 申请列表
GET  /platform/credit-applications/pending-count # 待审核数量
GET  /platform/credit-applications/:id       # 申请详情
POST /platform/credit-applications/:id/review # 审核申请 (通过/拒绝)
PUT  /platform/tenants/:id/credit            # 调整信用额度

# 支付订单
POST /payment/orders                         # 创建支付订单
GET  /payment/orders/:id                     # 查询订单
POST /payment/orders/:id/cancel              # 取消订单

# 支付回调
POST /payment/callback/alipay               # 支付宝回调
POST /payment/callback/wechat               # 微信回调
POST /payment/callback/stripe               # Stripe回调
POST /payment/callback/paypal               # PayPal回调
```

---

## ⚙️ 配置说明

### 最简配置

```yaml
# configs/config.yaml
server:
  port: 8080

database:
  host: localhost
  port: 5432
  dbname: ai_gateway

redis:
  host: localhost
  port: 6379

providers:
  openai:
    api_key: ${OPENAI_API_KEY}
  claude:
    api_key: ${ANTHROPIC_API_KEY}

plugins:
  enabled: true
```

### 完整配置参考

```yaml
server:
  port: 8080
  mode: release                      # debug, release, test

database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: ai_gateway
  max_open_conns: 200
  max_idle_conns: 50

redis:
  host: localhost
  port: 6379
  pool_size: 200

providers:
  openai:
    base_url: https://api.openai.com/v1
    api_key: ${OPENAI_API_KEY}
    timeout: 60s
  claude:
    base_url: https://api.anthropic.com/v1
    api_key: ${ANTHROPIC_API_KEY}
  gemini:
    api_key: ${GEMINI_API_KEY}
  deepseek:
    api_key: ${DEEPSEEK_API_KEY}
  glm:
    api_key: ${GLM_API_KEY}

billing:
  default_currency: USD
  min_balance_alert: 10.00

rate_limit:
  default_user_rps: 50
  default_tenant_rps: 500

load_balancer:
  strategy: adaptive                 # priority, round_robin, adaptive

plugins:
  enabled: true
  plugin_dir: "./plugins"
  allow_native_plugins: true
  sandbox:
    enabled: true
    max_memory_mb: 512
    timeout_seconds: 120

safe:
  enabled: true
  ip_rate_limit: {rps: 100, burst: 200}
  failed_auth:
    max_attempts: 10
    block_duration_s: 900
```

---

## 🏗️ 架构设计

```
┌─────────────────────────────────────────────────────┐
│                   Claude Code CLI                   │
│              (Anthropic Messages API)               │
└──────────────────────┬──────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────┐
│                 Open Station Gateway                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│  │  Anthropic  │  │    Proxy    │  │   Plugin    │ │
│  │   Handler   │  │   Service   │  │   Registry  │ │
│  │ • 流式转换  │  │ • 格式转换  │  │ • 发现加载  │ │
│  │ • 权限控制  │  │ • Token计数 │  │ • 生命周期  │ │
│  │ • 计费集成  │  │ • 负载均衡  │  │ • 市场管理  │ │
│  └─────────────┘  └─────────────┘  └─────────────┘ │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│  │     MCP     │  │   Billing   │  │    Auth     │ │
│  │   Service   │  │   Service   │  │   Service   │ │
│  │ • 26工具    │  │ • 实时扣费  │  │ • Key验证   │ │
│  │ • 会话管理  │  │ • 账单生成  │  │ • Redis缓存 │ │
│  └─────────────┘  └─────────────┘  └─────────────┘ │
│  ┌─────────────────────────────────────────────────┐│
│  │           Enterprise Payment System            ││
│  │  ┌──────────┐ ┌──────────┐ ┌─────────────────┐ ││
│  │  │  Quota   │ │ Credit   │ │  Settlement    │ ││
│  │  │ Service  │ │ App Svc  │ │   Service      │ ││
│  │  │ • 检查   │ │ • 申请   │ │ • 月结/周结    │ ││
│  │  │ • 扣减   │ │ • 审核   │ │ • 阈值触发    │ ││
│  │  │ • 限额   │ │ • 额度   │ │ • 账单生成    │ ││
│  │  └──────────┘ └──────────┘ └─────────────────┘ ││
│  │  ┌──────────┐ ┌──────────┐ ┌─────────────────┐ ││
│  │  │ Member   │ │ Payment  │ │   Notification  │ ││
│  │  │ Quota    │ │ Service  │ │    Service      │ ││
│  │  │ • 配额   │ │ • 订单   │ │ • 链件/短信    │ ││
│  │  │ • 限额   │ │ • 回调   │ │ • Webhook      │ ││
│  │  │ • 统计   │ │ • 渠道   │ │ • 内部通知    │ ││
│  │  └──────────┘ └──────────┘ └─────────────────┘ ││
│  └─────────────────────────────────────────────────┘│
└──────────────────────┬──────────────────────────────┘
                       │
       ┌───────────────┼───────────────┐
       │               │               │
       ▼               ▼               ▼
  ┌─────────┐    ┌─────────┐    ┌─────────┐
  │ Claude  │    │ OpenAI  │    │DeepSeek │
  │ Plugin  │    │ Plugin  │    │ Plugin  │
  │ (内置)  │    │ (内置)  │    │ (内置)  │
  └─────────┘    └─────────┘    └─────────┘

  ┌─────────────────────────────────────────┐
  │           Plugin Marketplace            │
  │  ┌────────┐  ┌────────┐  ┌────────┐    │
  │  │Mistral │  │ Cohere │  │ Custom │    │
  │  │Adapter │  │Adapter │  │Provider│    │
  │  └────────┘  └────────┘  └────────┘    │
  └─────────────────────────────────────────┘
```

### 技术栈

| 层级 | 技术 |
|------|------|
| **语言** | Go 1.22+ |
| **Web框架** | Gin |
| **ORM** | GORM |
| **数据库** | PostgreSQL 16 |
| **缓存** | Redis 7 |
| **容器化** | Docker + Docker Compose |
| **流式传输** | SSE (Server-Sent Events) |

---

## 👨‍💻 开发指南

### 本地开发

```bash
# 安装依赖
go mod download

# 运行测试
make test

# 测试覆盖率
make test-coverage

# 生成HTML覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage_report.html

# 代码检查
make lint

# 本地运行
make run

# 编译
make build
```

### 测试覆盖率

项目拥有完善的测试覆盖，核心认证系统覆盖率达到83%+：

| 模块 | 覆盖率 | 重点测试 |
|------|--------|----------|
| `infrastructure/auth` | **83.0%** | Login (94%), Register (77%), JWT验证 (89%) |
| `platform_auth` | **90%+** | 平台管理员登录/缓存/权限检查 |
| `login_security` | **85%+** | 暴力破解防护、异常检测 |
| `middleware` | **26.8%** | JWT中间件、角色检查 |

测试采用以下模式：
- **Mock Repository**: 手写Mock使用 `map[uuid.UUID]*Entity` 存储
- **Redis模拟**: 使用 `miniredis` 进行Redis相关测试
- **Table-driven**: 使用 `t.Run()` 组织测试用例
- **错误检查**: 使用 `errors.Is(err, apperrors.ErrXxx)` 检查错误类型

### 项目结构

```
open-station/
├── cmd/server/main.go              # 入口
├── internal/
│   ├── domain/                     # 领域层
│   │   ├── entity/                 # 实体
│   │   └── repository/             # 接口
│   ├── application/service/        # 服务层
│   ├── infrastructure/             # 基础设施
│   │   ├── persistence/            # 数据访问
│   │   ├── proxy/                  # 代理客户端
│   │   └── auth/                   # 认证
│   └── interfaces/http/            # HTTP接口
│       ├── handler/                # Handler
│       └── middleware/             # 中间件
├── pkg/                            # 公共包
│   ├── config/                     # 配置
│   ├── logger/                     # 日志
│   ├── mcp/                        # MCP协议
│   ├── plugin/                     # 插件系统
│   └── errors/                     # 错误定义
├── plugins/                        # 内置插件
│   ├── builtin/                    # 基础框架
│   ├── openai/                     # OpenAI
│   ├── anthropic/                  # Claude
│   ├── gemini/                     # Gemini
│   ├── deepseek/                   # DeepSeek
│   └── glm/                        # GLM
├── configs/                        # 配置文件
├── docs/                           # 文档
├── scripts/                        # 脚本
├── Makefile                        # 构建命令
└── README.md                       # 本文档
```

---

## 🚢 部署方案

### Docker Compose (生产推荐)

```bash
# 使用完整的 Docker Compose 配置（包含 PostgreSQL + Redis）
docker-compose -f deployments/docker/docker-compose.yml up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker logs -f open-station-gateway

# 停止服务
docker-compose down

# 带数据卷停止（保留数据）
docker-compose down --volumes
```

**环境变量配置**:

```bash
# 创建 .env 文件
cat > .env << EOF
OPENAI_API_KEY=sk-xxx
ANTHROPIC_API_KEY=sk-xxx
GEMINI_API_KEY=xxx
DEEPSEEK_API_KEY=sk-xxx
GLM_API_KEY=xxx
EOF

# 启动（自动加载 .env）
docker-compose up -d
```

### Docker 单容器

```bash
# 拉取镜像
docker pull zhaojiewen/open-station:latest
docker pull ghcr.io/zhaojiewen/open-station:latest

# 运行容器（需要外部 PostgreSQL + Redis）
docker run -d \
  --name open-station \
  -p 8080:8080 \
  -e DATABASE_HOST=postgres \
  -e DATABASE_PORT=5432 \
  -e DATABASE_USER=postgres \
  -e DATABASE_PASSWORD=postgres \
  -e DATABASE_DBNAME=ai_gateway \
  -e REDIS_HOST=redis \
  -e REDIS_PORT=6379 \
  -e OPENAI_API_KEY=sk-xxx \
  -v $(pwd)/configs:/etc/open-station:ro \
  --restart unless-stopped \
  zhaojiewen/open-station:latest
```

### Kubernetes (Helm)

```bash
# 添加 Helm 仓库 (待发布)
helm repo add open-station https://xuhaiqing.github.io/open-station-helm

# 安装
helm install open-station open-station/open-station \
  --set image.tag=v1.0.0 \
  --set config.openaiApiKey=sk-xxx \
  --set config.anthropicApiKey=sk-xxx \
  --set persistence.enabled=true

# 升级
helm upgrade open-station open-station/open-station --set image.tag=v1.1.0

# 删除
helm uninstall open-station
```

### Linux Systemd 服务

```bash
# 安装二进制
sudo cp bin/open-station /usr/local/bin/
sudo chmod +x /usr/local/bin/open-station

# 创建配置目录
sudo mkdir -p /etc/open-station
sudo cp configs/config.yaml /etc/open-station/
sudo cp -r plugins /etc/open-station/

# 创建 systemd 服务文件
sudo tee /etc/systemd/system/open-station.service << EOF
[Unit]
Description=Open Station AI Gateway
Documentation=https://github.com/zhaojiewen/open-station
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=open-station
Group=open-station
ExecStart=/usr/local/bin/open-station -config /etc/open-station/config.yaml
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

# 创建用户
sudo useradd -r -s /bin/false open-station

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable open-station
sudo systemctl start open-station

# 查看状态
sudo systemctl status open-station

# 查看日志
sudo journalctl -u open-station -f
```

### macOS LaunchAgent

```bash
# 安装二进制
sudo cp bin/open-station /usr/local/bin/

# 创建 LaunchAgent
cat > ~/Library/LaunchAgents/com.openstation.plist << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.openstation</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/open-station</string>
        <string>-config</string>
        <string>/etc/open-station/config.yaml</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/open-station.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/open-station.error.log</string>
</dict>
</plist>
EOF

# 加载服务
launchctl load ~/Library/LaunchAgents/com.openstation.plist

# 查看状态
launchctl list | grep openstation
```

---

## 🗺️ 路线图

### v1.0 (当前版本)

- [x] 多模型代理 (44+模型)
- [x] Claude Code MCP集成
- [x] 插件系统 (5内置插件)
- [x] 企业支付系统 (双层配额、后付费)
- [x] 成员配额控制

### v1.1 (计划中)

- [ ] Web管理界面
- [ ] WebSocket支持
- [ ] 模型自动发现
- [ ] 告警通知系统

### v1.2

- [ ] Kubernetes Helm Chart
- [ ] 多语言支持 (i18n)
- [ ] GraphQL API
- [ ] 更多内置插件

### v2.0

- [ ] 多集群部署
- [ ] 实时监控面板
- [ ] AI模型选择优化
- [ ] 企业版特性

---

## 🤝 贡献指南

欢迎贡献代码、报告问题、提出建议！

### 开发流程

1. Fork 项目
2. 创建分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送分支 (`git push origin feature/amazing-feature`)
5. 提交 Pull Request

### 代码规范

- 遵循 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- 运行 `make lint` 检查代码
- 新功能需添加测试
- 保持测试覆盖率 > 80%

---

## 📄 License

[MIT License](LICENSE) © 2024-present xuhaiqing

---

## 🔗 相关链接

- [Claude Code 集成指南](docs/claude-code-integration.md)
- [MCP 集成指南](docs/mcp-integration.md)
- [插件开发指南](docs/plugin-development.md)
- [企业支付系统指南](docs/payment-system.md)
- [API 文档](docs/api-reference.md)
- [更新日志](CHANGELOG.md)

---

**Made with ❤️ by the Open Station Team**