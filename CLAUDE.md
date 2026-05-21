# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test

```bash
make build        # go build -o bin/server ./cmd/server
make run          # go run ./cmd/server -config configs/config.yaml
make test         # go test -v ./...
make lint         # golangci-lint run ./...
make fmt          # go fmt ./...
make deps         # go mod download && go mod tidy
make docker-up    # start PostgreSQL + Redis
```

Single test: `go test -v -run TestName ./path/to/package/...`

**Testing pattern**: Table-driven `t.Run()` with in-memory mock repositories (`map[uuid.UUID]*Entity`). Redis tests use `miniredis`. Error checks: `errors.Is(err, apperrors.ErrXxx)`.

## Architecture

Enterprise AI gateway (Go 1.22, Gin, GORM, PostgreSQL, Redis) that proxies LLM requests to OpenAI, Anthropic, DeepSeek, and GLM via **transparent HTTP proxying**.

```
cmd/server/main.go          — entry point, wires all dependencies
internal/
  domain/
    entity/entity.go         — GORM models (Tenant, User, APIKey, Model, UsageRecord, Bill, RechargeRecord, AuditLog, ProviderAccount, UserQuota, MemberQuota, CreditApplication, PaymentOrder)
    repository/*.go          — repository interfaces
    role/                    — tenant roles (admin/member/viewer), platform roles (super_admin/billing_admin/support)
  application/service/       — business logic (BillingService, MCPService, InitService, ProviderAccountService, QuotaService, SettlementService, PaymentService, BudgetAlertService, CostLimitService, AsyncBillingQueue, ProviderAccountManager, etc.)
  infrastructure/
    persistence/postgres/    — DB + all repository implementations in repositories/repositories.go
    persistence/redis/       — Redis + rate limit + safe service
    auth/                    — API key auth, JWT, user auth, platform admin auth, email verification
    payment/                 — payment gateways (alipay, wechat, stripe, paypal, bank transfer)
  interfaces/http/
    router.go                — route definitions
    handler/                 — HTTP handlers
    middleware/               — auth (API key + JWT + API-type-aware), rate limit, logging, recovery, safe, platform admin, tenant role guards
pkg/
  config/config.go           — viper-based config, env var substitution
  errors/errors.go           — structured error codes
  mcp/types.go               — MCP protocol types
  loadbalancer/              — provider account selection strategies
  metrics/metrics.go         — Prometheus-compatible metrics
```

## Transparent Proxy (core request path)

All LLM requests flow through `TransparentProxyHandler` via `/:api/*path` routes. The handler does **not** convert request/response formats — it forwards raw HTTP bodies and only intercepts usage data for billing.

**Flow:**
1. `APITypeAuthMiddleware` reads `:api` from URL → extracts API key from correct header (`x-api-key` for `claude`, `Authorization: Bearer` for `gpt`)
2. `RateLimitMiddleware` applies per-key + per-tenant limits
3. `TransparentProxyHandler.HandleProxy`:
   - Reads model from buffered request body
   - Strips routing prefix if present (`openai-gpt-4o` → `gpt-4o`)
   - `ResolveProvider(model)` → detects provider from model name pattern
   - Permission checks (provider/model access, balance)
   - Gets upstream key/URL: dedicated account (if UseDedicatedProvider enabled) → public pool → config.Providers fallback
   - Rewrites URL (`providerBaseURL + remainingPath`), swaps auth header
   - Proxies request body, streams/copies response
   - Extracts usage (with cache tokens) → `AsyncBillingQueue`

**API type mapping** (`apiTypeDefaultProvider`):
- `gpt` → provider `openai`
- `claude` → provider `claude`

**Model → provider resolution** (`model_resolver.go`):
| Pattern | Provider |
|---------|----------|
| `gpt-*`, `o1*`, `o3*` | openai |
| `claude-*` | claude |
| `deepseek-chat`, `deepseek-reasoner` | deepseek |
| `glm-4-plus`, `glm-4-flash` | glm |
| Other | default from API type |

## Authentication

**API Key Auth** — `AuthMiddleware` (Bearer token from `Authorization` header) and `APITypeAuthMiddleware` (checks `x-api-key` for `claude` API type, Bearer otherwise). Keys prefixed `sk-`, SHA-256 hashed. Validated against Redis cache (5min TTL) then DB. Sets context: `api_key`, `api_key_id`, `user_id`, `user`, `tenant_id`, `tenant`.

**JWT Auth** — `JWTAuthMiddleware` for `/auth/*`, `/tenant/*` endpoints. Access token 15min, refresh token 7-day with Redis blacklisting. Login security: IP rate limiting, failed attempt tracking, bcrypt cost=12, password history (last 5), new device detection.

**Context values set by auth middleware:**

| Key | AuthMiddleware | JWTAuthMiddleware |
|-----|---------------|-------------------|
| `api_key_id`, `api_key` | ✓ | |
| `user_id`, `user` | ✓ | ✓ |
| `tenant_id`, `tenant` | ✓ | ✓ |
| `user_tenant`, `role` | | ✓ |
| `token_id`, `device_id` | | ✓ |
| `platform_admin_id`, `platform_admin` | | PlatformAdminMiddleware |

## Role System

**Tenant roles**: `admin` > `member` > `viewer`
- `RequireTenantWrite()` blocks viewers from write operations

**Platform admin roles**: `super_admin` > `billing_admin` > `support`
- `SuperAdminMiddleware()` — super_admin only
- `PlatformPermissionMiddleware("billing:write")` — permission-gated

**API key permissions** (JSONB): `chat`, `embeddings`, `admin`, `manage`

## Route Groups

| Prefix | Auth | Purpose |
|--------|------|---------|
| `/health`, `/ready`, `/version` | none | Health checks |
| `/:api/*path` | API-type-aware API Key + rate limit | Transparent LLM proxy (`/gpt/v1/chat/completions`, `/claude/v1/messages`) |
| `/auth` (public) | none + login rate limit | Login, register, email verify, refresh token |
| `/auth` (protected) | JWT | Logout, profile, tenants, switch-tenant, change password |
| `/admin/*` | API Key + tenant admin | Billing, API keys, budget alerts, provider accounts (incl. dedicated), user/invitation management |
| `/user/*` | API Key | User profile, own API keys, member quota, dedicated provider accounts |
| `/mcp` | self-auth | MCP JSON-RPC + SSE |
| `/platform/*` | Platform Admin | Platform-level tenant/admin/credit management, dedicated provider toggle |
| `/apply/*` | none | Public tenant/user application |
| `/invite/*` | none | Public invitation acceptance |
| `/tenant/*` | JWT + tenant admin | Credit applications, settlement |
| `/payments/*` | varies | Payment orders (user), callbacks (public) |

## Key Design Decisions

**Transparent proxy**: Requests are forwarded as-is — no format conversion, no internal intermediate types. The handler only buffers the body to read the model for routing, then forwards the raw bytes. Responses are copied directly to the client with usage extracted for billing.

**Cache-aware billing**: `extractUsageFromBody/Line` captures 4 token dimensions: `prompt_tokens`, `completion_tokens`, `cache_read_input_tokens`, `cache_creation_input_tokens`. `BillingService.CalculateCost` charges:
- Uncached input = prompt − cache_read → full `PromptPrice`
- Cache reads → `CacheReadPrice` (~10% of prompt price)
- Cache writes → `CacheWritePrice`  
- Output tokens → `CompletionPrice`

Both stream and non-stream paths use shared `finalizeBilling()` which calls `CalculateEquivalentTokens` (converts actual cost to equivalent uncached prompt tokens) for API key usage tracking.

**Dedicated provider accounts**: Users and tenants can have their own API keys per provider. `ProviderAccount` has `TenantID`/`UserID` fields — nil means public account. Selection priority: user dedicated > tenant dedicated > public pool (load balanced). Controlled by `UseDedicatedProvider` flag on `User` and `Tenant` (default false). Platform admins can toggle this flag for any user/tenant via `/platform/tenants/:id/dedicated` and `/platform/users/:id/dedicated`.

**Provider account failover**: `ProviderAccountManager` manages multiple API keys per provider with `pkg/loadbalancer/`. `GetActiveAccountWithDedicated` checks dedicated accounts first, then falls back to `GetActiveAccount` (public pool). Auto-failover on rate limit / quota exhaustion with 10s cooldown.

**Streaming**: `stream_options.include_usage` injected for GPT-format streaming requests so upstream returns usage data. Anthropic usage extracted from `message_start` (input + cache tokens) and `message_delta` (output tokens).

**Atomic billing**: `RecordUsage` deducts balance → creates usage record. Rolls back balance if record creation fails.

**Async billing**: `AsyncBillingQueue` with 8 workers, 50000 queue size. `QueueBillingAsync` is non-blocking; drops events if queue is full.

## MCP (Model Context Protocol)

`POST /mcp` (JSON-RPC) and `GET /mcp` (SSE). Sessions in-memory, 30min timeout.

**User tools**: balance, usage, billing, recharge history, my API keys.

**Manager tools**: CRUD API keys, list users, adjust balance, tenant management, provider account management (7 tools: CRUD + status), budget alerts (6 tools), user applications (5 tools), tenant applications (4 tools).

## Important Conventions

- `main.go` is gitignored — changes on disk are not tracked
- All repository implementations live in a single file: `internal/infrastructure/persistence/postgres/repositories/repositories.go`
- `ProviderConfig.AuthHeaderName` controls upstream auth header (e.g. `x-api-key` for Anthropic, empty = `Authorization: Bearer`)
- SSE responses must set `X-Accel-Buffering: no`
- Config uses `${ENV_VAR}` syntax for provider API keys
- Migrations via GORM `AutoMigrate` in `main.go` at startup

## Error Codes

Structured errors in `pkg/errors/errors.go`:
- `AUTH_*` — Authentication (invalid key, expired, revoked, unauthorized, token blacklisted)
- `RATE_*` — Rate limit (exceeded, tenant limit)
- `BILL_*` — Billing (insufficient balance, invalid amount)
- `QUOTA_*` — Quota (token exceeded, credit limit, no payment source)
- `REQ_*` — Request (invalid, model/provider not supported)
- `PROV_*` — Provider (error, timeout)
- `INT_*` — Internal (server, database, redis)
- `SAF_*` — Security (IP blocked, body too large, path traversal)
- `PLATFORM_*` — Platform (permission denied, application status)

Use `errors.Is(err, apperrors.ErrXxx)` for type checking.
