# Open Station

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](deployments/docker/)
[![Claude Code](https://img.shields.io/badge/Claude_Code-Compatible-7C3AED?style=flat)](docs/claude-code-integration.md)
[![Plugin System](https://img.shields.io/badge/Plugin_System-v1.0-green?style=flat)](docs/plugin-development.md)

> Enterprise AI Gateway - Multi-model proxy, plugin architecture, MCP service in one solution

**English** | [简体中文](README.md) | [Documentation](docs/)

---

## ✨ Core Features

| Feature | Description |
|---------|-------------|
| 🔐 **User Authentication** | JWT authentication, multi-tenant support, login security, password complexity validation |
| 🔌 **Plugin Architecture** | Supports Go Native (.so) and External Adapter (HTTP/gRPC) plugins |
| 🤖 **Multi-Model Proxy** | Unified forwarding for OpenAI, Claude, Gemini, DeepSeek, GLM (44+ models) |
| 💬 **Claude Code Compatible** | Full Anthropic Messages API support, direct CLI integration |
| 📊 **MCP Service** | 26 MCP tools to manage API Keys, balance, plugins via Claude Code |
| 💰 **Enterprise Payment System** | Dual quota control, postpaid credit, multi-channel payment integration |
| 🛡️ **Multi-layer Security** | Dual rate limiting, permission control, API Key auth, Redis cache acceleration |
| 🌊 **Streaming Response** | Full SSE implementation, all providers support real-time streaming |
| 📦 **Plugin Marketplace** | Local config management, one-click install/configure/activate plugins |

---

## 🚀 Quick Start (30 seconds)

```bash
# Clone the project
git clone https://github.com/zhaojiewen/open-station.git && cd open-station

# One-click start (auto install Docker + create admin)
make start

# View API Key (auto-created on first start)
docker logs open-station-gateway 2>&1 | grep "API Key"
```

**After successful startup**:
- Gateway URL: `http://localhost:8080`
- MCP Endpoint: `http://localhost:8080/mcp`
- Health Check: `http://localhost:8080/health`

---

## 📦 Installation Guide

### Method 1: Pre-built Binary (Recommended)

Download from GitHub Release:

```bash
# One-line install (Linux/macOS)
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

### Method 2: Docker

```bash
# Docker Hub
docker pull zhaojiewen/open-station:latest
docker run -d --name open-station -p 8080:8080 zhaojiewen/open-station:latest

# GitHub Container Registry
docker pull ghcr.io/zhaojiewen/open-station:latest
docker run -d --name open-station -p 8080:8080 ghcr.io/zhaojiewen/open-station:latest

# Docker Compose (includes PostgreSQL + Redis)
git clone https://github.com/zhaojiewen/open-station.git && cd open-station
docker-compose -f deployments/docker/docker-compose.yml up -d
```

### Method 3: Build from Source

```bash
git clone https://github.com/zhaojiewen/open-station.git && cd open-station
go mod download
make build
./bin/open-station -config configs/config.yaml
```

### Method 4: Go Install

```bash
go install github.com/zhaojiewen/open-station/cmd/server@latest
open-station -config configs/config.yaml
```

---

## 📋 Table of Contents

- [Supported Models](#-supported-models)
- [Claude Code Integration](#-claude-code-integration)
- [MCP Tools List](#-mcp-tools-list)
- [Plugin System](#-plugin-system)
- [API Reference](#-api-reference)
- [Configuration](#️-configuration)
- [Architecture](#-architecture)
- [Development Guide](#-development-guide)
- [Deployment](#-deployment)
- [Roadmap](#-roadmap)
- [Contributing](#-contributing)

---

## 🎯 Supported Models

| Provider | Models | Popular Models | Highlights |
|----------|--------|----------------|------------|
| **Claude** | 9 | Opus 4.7, Sonnet 4.6, Haiku 4.5 | Strongest reasoning |
| **OpenAI** | 8 | GPT-4o, O1-mini, GPT-4o-mini | Rich ecosystem |
| **Gemini** | 7 | Gemini 3.1 Pro, Gemini 2.5 Flash | Multi-modal support |
| **DeepSeek** | 2 | V4 Pro, V4 Flash | **Best cost-performance** |
| **GLM** | 18 | GLM-5.1, GLM-4-Flash | **Free models available** |

> Access other providers via prefix: `openai-{model}`, `deepseek-{model}`, `glm-{model}`, `gemini-{model}`

---

## 💬 Claude Code Integration

### Method 1: MCP Service (Recommended)

```bash
# Configure MCP (auto writes to ~/.claude/settings.json)
./scripts/setup-mcp.sh --api-key sk-your-manager-key

# Start Claude Code
claude

# Natural language management
> "What's my balance?"                    # Check balance
> "Create API key for john@example.com"   # Create user+key
> "Install Mistral plugin"                # Install plugin
> "Add $100 to tenant abc balance"        # Recharge
```

### Method 2: API Proxy

```bash
# Set environment variables
export ANTHROPIC_BASE_URL="http://localhost:8080/v1"
export ANTHROPIC_API_KEY="sk-your-gateway-key"

# Direct usage, cross-provider access
claude --model claude-opus-4-7          # Claude (default)
claude --model openai-gpt-4o            # OpenAI
claude --model deepseek-v4-flash        # DeepSeek (best value)
claude --model glm-4-flash              # GLM (free)
```

> Detailed docs: [Claude Code Integration Guide](docs/claude-code-integration.md)

---

## 📊 MCP Tools List

### User Tools (6)

| Tool | Function | Example |
|------|----------|---------|
| `check_balance` | Check balance | "What's my balance?" |
| `get_usage_summary` | Usage summary | "Show this month usage" |
| `get_usage_details` | Usage details | "Show detailed usage" |
| `get_billing_info` | Billing info | "Get billing info" |
| `get_recharge_history` | Recharge history | "Show recharge history" |
| `get_my_api_keys` | My keys | "List my API keys" |

### Admin Tools (9)

| Tool | Function | Example |
|------|----------|---------|
| `list_all_api_keys` | All keys | "List all API keys" |
| `create_api_key` | Create key | "Create key for john@example.com" |
| `revoke_api_key` | Revoke key | "Revoke key abc-123" |
| `update_api_key` | Update permissions | "Add embedding permission to key xyz" |
| `list_users` | User list | "Show all users" |
| `adjust_balance` | Adjust balance | "Add $50 to tenant abc" |
| `get_tenant_summary` | Tenant summary | "Get tenant abc summary" |
| `list_tenants` | Tenant list | "List all tenants" |

### Plugin Tools (11)

| Tool | Function | Permission |
|------|----------|------------|
| `list_plugins` | Installed plugins | User |
| `list_available_plugins` | Available plugins | User |
| `get_plugin_status` | Plugin status | User |
| `get_plugin_providers` | Provider list | User |
| `install_plugin` | Install plugin | Admin |
| `configure_plugin` | Configure plugin | Admin |
| `activate_plugin` | Activate plugin | Admin |
| `deactivate_plugin` | Deactivate plugin | Admin |
| `uninstall_plugin` | Uninstall plugin | Admin |
| `check_plugin_health` | Health check | Admin |
| `get_all_plugin_stats` | Plugin stats | Admin |

> Detailed docs: [MCP Integration Guide](docs/mcp-integration.md)

---

## 💰 Enterprise Payment System

### Dual User Modes

| Mode | Use Case | Quota Source | Postpaid |
|------|----------|--------------|----------|
| **Individual Mode** | Public tenant users | UserQuota (independent) | Not supported |
| **Organization Mode** | Enterprise tenant members | Tenant shared + MemberQuota control | Supported (requires approval) |

### Unified Deduction Priority

```
1. Subscription Token Quota (first)
   ↓ after exhausted
2. Prepaid Balance
   ↓ when balance = 0
3. Postpaid Credit Limit (enterprise only, requires approval)
```

### Enterprise Postpaid Workflow

```
Enterprise Application → Platform Review → Approval → Set Credit Limit → Usage → Settlement
```

**Settlement Cycles**: Monthly, Weekly, Threshold-triggered, Custom

### Member Quota Control

Enterprise tenant admins can set independent quotas for members:
- **Token Quota Limit**: Limit member monthly token usage
- **Cost Limit**: Control member expense upper limit

### Payment Channels

| Channel | Region | Payment Method |
|---------|--------|----------------|
| **Alipay** | China | QR code, Web, APP |
| **WeChat Pay** | China | QR code, Web, APP |
| **Stripe** | International | Credit card, Web |
| **PayPal** | International | Account balance, Credit card |
| **Bank Transfer** | Enterprise | Offline transfer |

> Detailed docs: [Payment System Integration Guide](docs/payment-system.md)

---

## 🔐 User Authentication System

### Multi-Tenant Architecture

Users can belong to multiple tenants with complete data and billing isolation:

| Feature | Description |
|---------|-------------|
| **Individual Registration** | Automatically joins public tenant, uses UserQuota |
| **Enterprise Registration** | Creates new tenant, becomes admin automatically |
| **Tenant Switching** | Switch active tenant via API |
| **Multi-Tenant Membership** | One user can belong to multiple tenants via invitation |

### JWT Authentication

JWT Token based user authentication:

| Token Type | Expiry | Usage |
|------------|--------|-------|
| **Access Token** | 15 minutes | API access credential |
| **Refresh Token** | 7 days | Refresh Access Token |

### Login Security Protection

Multiple security measures protect the login process:

| Security Measure | Configuration |
|------------------|---------------|
| **Failed Attempts Limit** | 5 failures → IP blocked for 15 minutes |
| **Password Hashing** | bcrypt (cost=12) |
| **Password Complexity** | Min 8 chars, reject common weak passwords |
| **Password History** | Check last 5 passwords, prevent reuse |
| **Audit Logging** | Record all login attempts (IP, device, result) |
| **Anomaly Detection** | New device/IP login alerts |
| **Sensitive Data Encryption** | AES-256-GCM for IP, UserAgent, etc. |

### Authentication API Endpoints

```bash
# Public endpoints (no authentication required)
POST /auth/login              # User login
POST /auth/register           # Individual registration (join public tenant)
POST /auth/tenant/register    # Enterprise registration (create new tenant)
POST /auth/refresh            # Refresh token

# Authenticated endpoints (require JWT Token)
POST /auth/logout             # Logout current device
POST /auth/logout-all         # Logout all devices
GET  /auth/profile            # Get user info
GET  /auth/tenants            # Get user's all tenants
POST /auth/switch-tenant      # Switch current tenant
PUT  /auth/password           # Change password
```

### Authentication Configuration

```yaml
# configs/config.yaml
auth:
  jwt:
    secret_key: "${JWT_SECRET}"           # JWT signing key (required)
    access_token_expire: 15m              # Access Token expiry
    refresh_token_expire: 168h            # Refresh Token expiry (7 days)

  login_security:
    max_failed_attempts: 5                # Max failed attempts
    failed_window: 15m                    # Failed count window
    block_duration: 30m                   # Block duration
    enable_audit_log: true                # Enable audit logging
    encrypt_audit_data: true              # Encrypt audit data
    anomaly_detection: true               # Anomaly detection
    new_device_alert: true                # New device alert

  password:
    min_length: 8                         # Minimum length
    max_length: 64                        # Maximum length
    require_upper: true                   # Require uppercase
    require_lower: true                   # Require lowercase
    require_digit: true                   # Require digit
    require_special: true                 # Require special char
    history_count: 5                      # Check password history count
    bcrypt_cost: 12                       # bcrypt strength
```

### Usage Examples

```bash
# User login
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'

# Response example
{
  "user": {"id": "...", "email": "user@example.com", "name": "User"},
  "tenants": [{"tenant_id": "...", "role": "admin", "is_default": true}],
  "current_tenant_id": "...",
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2024-01-01T00:15:00Z"
}

# Individual registration
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"newuser@example.com","password":"password123","name":"New User"}'

# Enterprise registration
curl -X POST http://localhost:8080/auth/tenant/register \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_name": "My Company",
    "tenant_slug": "my-company",
    "email": "admin@example.com",
    "password": "password123",
    "name": "Admin User"
  }'

# Switch tenant
curl -X POST http://localhost:8080/auth/switch-tenant \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": "uuid-of-tenant"}'

# Change password
curl -X PUT http://localhost:8080/auth/password \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"current_password":"old123","new_password":"new123"}'
```

> Detailed docs: [User Authentication System Guide](docs/auth-system.md)

---

## 🔌 Plugin System

### Plugin Types

| Type | Description | Use Case |
|------|-------------|----------|
| **Go Native (.so)** | Compiled as dynamic library | High performance, deep integration |
| **External Adapter** | HTTP/gRPC service | Flexible deployment, independent ops |

### Built-in Plugins (5)

| Plugin | Provider | Capabilities |
|--------|----------|--------------|
| `openai` | OpenAI | Chat, Stream, Embedding |
| `anthropic` | Anthropic | Chat, Stream |
| `gemini` | Google | Chat, Stream, Embedding |
| `deepseek` | DeepSeek | Chat, Stream |
| `glm` | Zhipu AI | Chat, Stream, Embedding |

### Configure New Plugin

```yaml
# configs/config.yaml
plugins:
  enabled: true
  available_plugins:
    mistral:                              # Add Mistral
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
# Manage via MCP
claude
> "Install Mistral plugin"
> "Configure Mistral with API key sk-xxx"
> "Activate Mistral plugin"
```

> Detailed docs: [Plugin Development Guide](docs/plugin-development.md)

---

## 🔗 API Reference

### Anthropic Compatible Endpoints

```bash
POST /v1/messages          # Messages API (SSE streaming)
GET  /v1/models            # Dynamic model list

# Example: streaming request
curl -X POST http://localhost:8080/v1/messages \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-sonnet-4-6","max_tokens":1024,"messages":[{"role":"user","content":"Hello"}],"stream":true}'
```

### MCP Endpoints

```bash
POST /mcp                  # JSON-RPC endpoint
GET  /mcp                  # SSE streaming endpoint
```

### Admin Endpoints

```bash
# Billing
GET  /admin/billing/balance/:id    # Balance
POST /admin/billing/recharge       # Recharge

# API Key
GET  /admin/api-keys               # List
POST /admin/api-keys               # Create
POST /admin/api-keys/:id/revoke    # Revoke

# Plugins (14 endpoints)
GET  /admin/plugins                # Installed list
GET  /admin/plugins/available      # Available list
POST /admin/plugins/:id/install    # Install
POST /admin/plugins/:id/activate   # Activate
# ... full list in MCP Plugin Tools above

# Member Quota Management (Enterprise tenant admin)
GET  /admin/member-quotas          # Member quota list
POST /admin/member-quotas          # Create member quota
PUT  /admin/member-quotas/:id      # Update member quota
PUT  /admin/member-quotas/:id/token-limit  # Set token quota
PUT  /admin/member-quotas/:id/cost-limit   # Set cost limit
GET  /admin/member-quotas/:id/usage # Member usage stats
```

### Payment System Endpoints

```bash
# Enterprise Postpaid Application (Tenant admin)
POST /tenant/credit-application              # Apply for credit
GET  /tenant/credit-application              # View application status
PUT  /tenant/credit-application              # Update application (before review)
DELETE /tenant/credit-application            # Cancel application

# Platform Admin Review
GET  /platform/credit-applications           # Application list
GET  /platform/credit-applications/pending-count # Pending count
GET  /platform/credit-applications/:id       # Application details
POST /platform/credit-applications/:id/review # Review application (approve/reject)
PUT  /platform/tenants/:id/credit            # Adjust credit limit

# Payment Orders
POST /payment/orders                         # Create payment order
GET  /payment/orders/:id                     # Query order
POST /payment/orders/:id/cancel              # Cancel order

# Payment Callbacks
POST /payment/callback/alipay               # Alipay callback
POST /payment/callback/wechat               # WeChat callback
POST /payment/callback/stripe               # Stripe callback
POST /payment/callback/paypal               # PayPal callback
```

---

## ⚙️ Configuration

### Minimal Config

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

### Full Config Reference

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

## 🏗️ Architecture

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
│  │ • Streaming │  │ • Transform │  │ • Discovery │ │
│  │ • Auth      │  │ • Tokens    │  │ • Lifecycle │ │
│  │ • Billing   │  │ • Balance   │  │ • Market    │ │
│  └─────────────┘  └─────────────┘  └─────────────┘ │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│  │     MCP     │  │   Billing   │  │    Auth     │ │
│  │   Service   │  │   Service   │  │   Service   │ │
│  │ • 26 tools  │  │ • Real-time │  │ • Key check │ │
│  │ • Sessions  │  │ • Invoices  │  │ • Redis     │ │
│  └─────────────┘  └─────────────┘  └─────────────┘ │
│  ┌─────────────────────────────────────────────────┐│
│  │           Enterprise Payment System            ││
│  │  ┌──────────┐ ┌──────────┐ ┌─────────────────┐ ││
│  │  │  Quota   │ │ Credit   │ │  Settlement    │ ││
│  │  │ Service  │ │ App Svc  │ │   Service      │ ││
│  │  │ • Check  │ │ • Apply  │ │ • Monthly/Week │ ││
│  │  │ • Deduct │ │ • Review │ │ • Threshold    │ ││
│  │  │ • Limit  │ │ • Limit  │ │ • Invoice Gen  │ ││
│  │  └──────────┘ └──────────┘ └─────────────────┘ ││
│  │  ┌──────────┐ ┌──────────┐ ┌─────────────────┐ ││
│  │  │ Member   │ │ Payment  │ │  Notification  │ ││
│  │  │ Quota    │ │ Service  │ │    Service      │ ││
│  │  │ • Quota  │ │ • Order  │ │ • Email/SMS    │ ││
│  │  │ • Limit  │ │ • Callback│ │ • Webhook      │ ││
│  │  │ • Stats  │ │ • Channel │ │ • In-app       │ ││
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
  │(Built-in)│   │(Built-in)│   │(Built-in)│
  └─────────┘    └─────────┘    └─────────┘

  ┌─────────────────────────────────────────┐
  │           Plugin Marketplace            │
  │  ┌────────┐  ┌────────┐  ┌────────┐    │
  │  │Mistral │  │ Cohere │  │ Custom │    │
  │  │Adapter │  │Adapter │  │Provider│    │
  │  └────────┘  └────────┘  └────────┘    │
  └─────────────────────────────────────────┘
```

### Tech Stack

| Layer | Technology |
|-------|------------|
| **Language** | Go 1.22+ |
| **Web Framework** | Gin |
| **ORM** | GORM |
| **Database** | PostgreSQL 16 |
| **Cache** | Redis 7 |
| **Container** | Docker + Docker Compose |
| **Streaming** | SSE (Server-Sent Events) |

---

## 👨‍💻 Development Guide

### Local Development

```bash
# Install dependencies
go mod download

# Run tests
make test

# Test coverage
make test-coverage

# Generate HTML coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage_report.html

# Code linting
make lint

# Local run
make run

# Build
make build
```

### Test Coverage

The project has comprehensive test coverage with the core authentication system reaching 83%+:

| Module | Coverage | Key Tests |
|--------|----------|-----------|
| `infrastructure/auth` | **83.0%** | Login (94%), Register (77%), JWT validation (89%) |
| `platform_auth` | **90%+** | Platform admin login/caching/permission checks |
| `login_security` | **85%+** | Brute-force protection, anomaly detection |
| `middleware` | **26.8%** | JWT middleware, role checking |

Test patterns used:
- **Mock Repository**: Hand-written mocks using `map[uuid.UUID]*Entity` storage
- **Redis Simulation**: Using `miniredis` for Redis-related tests
- **Table-driven**: Using `t.Run()` for organized test cases
- **Error Checking**: Using `errors.Is(err, apperrors.ErrXxx)` for error type checking

### Project Structure

```
open-station/
├── cmd/server/main.go              # Entry point
├── internal/
│   ├── domain/                     # Domain layer
│   │   ├── entity/                 # Entities
│   │   └── repository/             # Interfaces
│   ├── application/service/        # Service layer
│   ├── infrastructure/             # Infrastructure
│   │   ├── persistence/            # Data access
│   │   ├── proxy/                  # Proxy clients
│   │   └── auth/                   # Authentication
│   └── interfaces/http/            # HTTP interface
│       ├── handler/                # Handlers
│       └── middleware/             # Middleware
├── pkg/                            # Public packages
│   ├── config/                     # Config
│   ├── logger/                     # Logging
│   ├── mcp/                        # MCP protocol
│   ├── plugin/                     # Plugin system
│   └── errors/                     # Error definitions
├── plugins/                        # Built-in plugins
│   ├── builtin/                    # Base framework
│   ├── openai/                     # OpenAI
│   ├── anthropic/                  # Claude
│   ├── gemini/                     # Gemini
│   ├── deepseek/                   # DeepSeek
│   └── glm/                        # GLM
├── configs/                        # Config files
├── docs/                           # Documentation
├── scripts/                        # Scripts
├── Makefile                        # Build commands
└── README.md                       # This doc
```

---

## 🚢 Deployment

### Docker (Recommended)

```bash
# Production
docker-compose -f deployments/docker/docker-compose.yml up -d

# View logs
docker logs -f open-station-gateway

# Stop
docker-compose down
```

### Kubernetes

```bash
# Using Helm (coming soon)
helm install open-station ./deployments/helm/
```

### System Service (Linux)

```bash
# Install
sudo cp bin/open-station /usr/local/bin/
sudo cp configs/config.yaml /etc/open-station/
sudo cp open-station.service /etc/systemd/system/

# Start
sudo systemctl enable open-station
sudo systemctl start open-station
```

---

## 🗺️ Roadmap

### v1.0 (Current)

- [x] Multi-model proxy (44+ models)
- [x] Claude Code MCP integration
- [x] Plugin system (5 built-in plugins)
- [x] Enterprise payment system (dual quota, postpaid)
- [x] Member quota control

### v1.1 (Planned)

- [ ] Web admin UI
- [ ] WebSocket support
- [ ] Model auto-discovery
- [ ] Alert notification system

### v1.2

- [ ] Kubernetes Helm Chart
- [ ] Multi-language support (i18n)
- [ ] GraphQL API
- [ ] More built-in plugins

### v2.0

- [ ] Multi-cluster deployment
- [ ] Real-time monitoring dashboard
- [ ] AI model selection optimization
- [ ] Enterprise features

---

## 🤝 Contributing

Welcome to contribute code, report issues, or suggest features!

### Development Process

1. Fork the project
2. Create branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push branch (`git push origin feature/amazing-feature`)
5. Submit Pull Request

### Code Standards

- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Run `make lint` for code check
- Add tests for new features
- Maintain test coverage > 80%

---

## 📄 License

[MIT License](LICENSE) © 2024-present xuhaiqing

---

## 🔗 Related Links

- [Claude Code Integration Guide](docs/claude-code-integration.md)
- [MCP Integration Guide](docs/mcp-integration.md)
- [Plugin Development Guide](docs/plugin-development.md)
- [Enterprise Payment System Guide](docs/payment-system.md)
- [User Authentication System Guide](docs/auth-system.md)
- [API Reference](docs/api-reference.md)
- [Changelog](CHANGELOG.md)

---

**Made with ❤️ by the Open Station Team**