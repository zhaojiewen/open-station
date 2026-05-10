# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test

```bash
make build        # go build -o bin/server ./cmd/server
make run          # go run ./cmd/server -config configs/config.yaml
make test         # go test -v ./...
make test-coverage # targeted coverage on pkg, domain, auth, service, middleware
make lint         # golangci-lint run ./...
make fmt          # go fmt ./...
make deps         # go mod download && go mod tidy
make docker-up    # start PostgreSQL + Redis + Gateway
make start        # full Docker deployment (auto-installs Docker if needed)
```

Single test: `go test -v -run TestName ./path/to/package/...`

**Testing pattern**: Tests use in-memory mock repositories (see `billing_service_test.go` for examples). Mocks implement repository interfaces with `map[uuid.UUID]*Entity` storage. Use `errors.Is(err, apperrors.ErrXxx)` to check error types.

## Architecture

This is an enterprise AI gateway (Go 1.22, Gin, GORM, PostgreSQL, Redis) that proxies LLM requests across OpenAI, Claude, Gemini, DeepSeek, and GLM. It follows a DDD-inspired layered architecture:

```
cmd/server/main.go          — entry point, wires all dependencies
internal/
  domain/
    entity/entity.go         — GORM entity models (Tenant, User, APIKey, Model, UsageRecord, Bill, RechargeRecord, AuditLog, ProviderAccount)
    repository/repository.go — interface definitions for all repositories
  application/service/       — business logic (BillingService, MCPService, InitService, ProviderAccountService)
  infrastructure/
    persistence/postgres/    — DB connection + repository implementations
    persistence/redis/       — Redis connection + rate limit service
    proxy/proxy_service.go   — multi-provider HTTP clients (OpenAI, Claude, DeepSeek, GLM)
    auth/auth_service.go     — API key validation, creation, permission checking
  interfaces/http/
    router.go                — route definitions
    handler/                 — HTTP handlers (AnthropicHandler for Claude Code, ProxyHandler, MCPHandler, BillingHandler)
    middleware/               — auth, rate limit, logging, recovery
pkg/
  config/config.go           — viper-based config loading, env var substitution
  logger/logger.go           — zap logger wrapper
  errors/errors.go           — structured error codes (AUTH_*, RATE_*, BILL_*, REQ_*, PROV_*, INT_*)
  mcp/types.go               — MCP protocol type definitions
```

## Key Design Decisions

**Model routing**: Model IDs with a provider prefix (e.g. `openai-gpt-4o`, `deepseek-v4-flash`) route to the corresponding provider. Claude model IDs without a prefix (`claude-sonnet-4-6`) default to the claude provider via a hardcoded mapping in `AnthropicHandler.modelMapping`.

**Anthropic compatibility layer**: `POST /v1/messages` accepts Anthropic Messages API format with SSE streaming. The AnthropicHandler converts Anthropic request/response formats to/from the internal `ProxyRequest`/`ProxyResponse` format. The Anthropic stream events (message_start, content_block_start, content_block_delta, content_block_stop, message_stop) are generated from OpenAI-format stream chunks.

**API Key design**: Keys are prefixed `sk-`, stored as SHA-256 hashes. Validation checks an in-memory Redis cache first (TTL 5min), then falls back to the database. Keys carry JSONB-encoded permissions, allowed models, and allowed providers.

**Multi-tenancy**: Users belong to Tenants. Each API key is tied to both a User and a Tenant. Billing is at the tenant level (balance, usage records, bills). Rate limiting is dual-level: per-API-key and per-tenant.

**Provider account failover**: `DynamicProxyHandler` supports multiple API accounts per provider with priority ordering. On failure (rate limit, quota exhaustion), it automatically fails over to the next available account and records the error for the failed account.

**HTTP connection pooling**: `ProxyService` uses a shared `http.Transport` with optimized pooling (`MaxIdleConns: 100`, `MaxIdleConnsPerHost: 20`, `IdleConnTimeout: 90s`) for high-throughput proxy requests.

**Migrations**: Run via GORM `AutoMigrate` in `cmd/server/main.go` at startup — no separate migration tool needed. SQL migration files in `migrations/` are supplementary.

**No manual DB migration runner**: The Makefile's `migrate` target references `./cmd/migrate` but no such file exists; migrations happen via AutoMigrate on server start.

## MCP (Model Context Protocol)

The gateway exposes `POST /mcp` (JSON-RPC) and `GET /mcp` (SSE) for Claude Code integration. The MCP flow:
1. Client sends `initialize` with API key → server creates a session, returns capabilities
2. Client calls `tools/list` → server returns tools based on session role (user vs manager)
3. Client calls `tools/call` → server executes the named tool

Sessions are in-memory (30min timeout). User tools (6): balance, usage, billing, recharge history, my API keys. Manager tools (15): list/create/revoke/update API keys, list users, adjust balance, tenant management, provider account management.

## Important Conventions

- Always set `X-Accel-Buffering: no` on SSE responses to prevent nginx buffering
- The `AnthropicHandler.Messages` method performs its own auth extraction from headers rather than relying solely on `AuthMiddleware` (dual-path auth)
- `BillingService.RecordUsage` deducts balance atomically and rolls back if usage record creation fails
- Repository implementations live in `internal/infrastructure/persistence/postgres/repositories/` and export constructors like `NewTenantRepository(db *gorm.DB) *TenantRepoImpl`
- Config uses `${ENV_VAR}` syntax for provider API keys, resolved by viper's `AutomaticEnv()`
- **Async billing**: `AsyncBillingQueue` handles background usage recording with 4 workers, batching 100 records per flush. Started in `main.go` and stopped on shutdown.
- **Security middleware**: `SafeMiddleware` provides IP rate limiting, blacklist/whitelist, failed auth tracking, path traversal detection, and burst attack auto-blocking. Configured via `safe` section in config.yaml.

## Error Codes

Structured errors in `pkg/errors/errors.go` use prefixed codes:
- `AUTH_*` — Authentication errors (invalid key, expired, revoked, unauthorized)
- `RATE_*` — Rate limit errors (exceeded, tenant limit)
- `BILL_*` — Billing errors (insufficient balance, invalid amount)
- `REQ_*` — Request errors (invalid, model/provider not supported)
- `PROV_*` — Provider errors (provider error, timeout)
- `INT_*` — Internal errors (server, database, redis)
- `SAF_*` — Security errors (IP blocked, body too large, path traversal, suspicious header)

Use `errors.Is(err, apperrors.ErrXxx)` for type checking and `apperrors.IsAuthError(err)` etc. for category checks.
