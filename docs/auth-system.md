# User Authentication System Guide

This document provides detailed information about the user authentication system in Open Station, including multi-tenant architecture, JWT authentication, login security, and password management.

---

## Overview

Open Station provides a comprehensive authentication system that supports:

- **Multi-tenant architecture**: One user can belong to multiple tenants
- **JWT authentication**: Secure token-based authentication with access and refresh tokens
- **Login security**: Brute-force protection, anomaly detection, audit logging
- **Password security**: bcrypt hashing, complexity validation, history checking
- **Data encryption**: AES-256-GCM encryption for sensitive audit data

---

## Multi-Tenant Architecture

### Concept

The system uses a **UserTenant** association table to enable multi-tenant membership:

```
User (Email unique) ──── UserTenant ──── Tenant
         │                   │              │
         │                   │              │
    One account         Many records      Many tenants
    (global unique)     (role per tenant)  (isolated data)
```

### UserTenant Entity

| Field | Type | Description |
|-------|------|-------------|
| `UserID` | UUID | User reference |
| `TenantID` | UUID | Tenant reference |
| `Role` | string | Role in tenant: `admin`, `member`, `viewer` |
| `Status` | string | Status: `active`, `inactive` |
| `IsDefault` | bool | Is this the user's default tenant |
| `JoinedAt` | time | When user joined the tenant |

### Registration Types

#### Individual Registration

Users register individually and automatically join the **public tenant**:

```
POST /auth/register
{
    "email": "user@example.com",
    "password": "SecurePass123!",
    "name": "User Name"
}
```

**Process**:
1. Validate email format and password complexity
2. Check email uniqueness
3. Create User entity (UserMode = "individual")
4. Create UserTenant for public tenant (Role = "member", IsDefault = true)
5. Create UserQuota for personal quota
6. Generate JWT tokens
7. Return user info + tokens

#### Enterprise Registration

Users register and create a new tenant, becoming the tenant admin:

```
POST /auth/tenant/register
{
    "tenant_name": "My Company",
    "tenant_slug": "my-company",
    "email": "admin@example.com",
    "password": "SecurePass123!",
    "name": "Admin User"
}
```

**Process**:
1. Validate all inputs
2. Check tenant slug uniqueness
3. Create Tenant entity (Type = "organization")
4. Create or retrieve User entity
5. Create UserTenant (Role = "admin", IsDefault = true)
6. Generate JWT tokens
7. Return tenant + user info + tokens

---

## JWT Authentication

### Token Types

| Token | Expiry | Purpose | Storage |
|-------|--------|---------|---------|
| **Access Token** | 15 minutes | API authentication | Client memory |
| **Refresh Token** | 7 days | Refresh access token | Client storage (secure) |

### JWT Claims

```json
{
    "user_id": "uuid",
    "email": "user@example.com",
    "tenant_id": "uuid",
    "role": "admin",
    "device_id": "device-fingerprint",
    "token_id": "uuid",
    "exp": 1234567890,
    "iat": 1234567800
}
```

### Token Lifecycle

```
┌─────────────────────────────────────────────────────┐
│                    Login Flow                       │
│                                                     │
│  Login Request                                      │
│       ↓                                             │
│  Validate Credentials                               │
│       ↓                                             │
│  Generate Access Token (15min)                      │
│  Generate Refresh Token (7days)                     │
│       ↓                                             │
│  Store Refresh Token in Redis                       │
│       ↓                                             │
│  Return Tokens + User Info                          │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│                 Token Refresh Flow                  │
│                                                     │
│  Refresh Request (refresh_token + device_id)        │
│       ↓                                             │
│  Validate Refresh Token                             │
│       ↓                                             │
│  Check Device Match                                 │
│       ↓                                             │
│  Check Token Not Revoked                            │
│       ↓                                             │
│  Generate New Access Token                          │
│       ↓                                             │
│  Update Refresh Token LastUsed                      │
│       ↓                                             │
│  Return New Access Token                            │
└─────────────────────────────────────────────────────┘
```

### Token Blacklist

When a user logs out, the access token is added to a Redis blacklist:

```
Key: jwt_blacklist:{token_hash}
Value: 1
TTL: token remaining expiry time
```

This prevents token reuse after logout.

---

## Login Security

### Brute-Force Protection

The system tracks failed login attempts and blocks IPs:

```
Redis Key: login_failed:{ip}:{email}
Value: {count, last_attempt_time}
TTL: 15 minutes (failed_window)
```

**Blocking Rules**:
- After 5 failed attempts within 15 minutes
- Block the IP for 30 minutes
- Return `ErrTooManyAttempts` error

### Login Audit Log

Every login attempt is logged in the `LoginAudit` table:

| Field | Description |
|-------|-------------|
| `UserID` | User ID (if successful) |
| `Email` | Email attempted |
| `IP` | Client IP (encrypted) |
| `UserAgent` | Browser info (encrypted) |
| `DeviceID` | Device fingerprint |
| `Success` | Whether login succeeded |
| `FailureReason` | Reason if failed |
| `LoginAt` | Timestamp |

### Anomaly Detection

The system detects unusual login patterns:

| Anomaly Type | Detection Logic |
|--------------|-----------------|
| **New Device** | DeviceID not seen in last 30 logins |
| **New IP** | IP not seen in last 30 logins |
| **Impossible Travel** | Login from distant IPs within short time |
| **Multiple IPs** | Many different IPs in short period |

When anomaly is detected:
- Flag `IsAnomaly` in login response
- Optionally send alert email to user
- Do NOT block the login (just warn)

### Device Fingerprinting

Device ID is generated from:

```go
func GenerateDeviceID(userAgent, ip string) string {
    hash := sha256.Sum256([]byte(userAgent + ip))
    return hex.EncodeToString(hash[:8])
}
```

Or client can provide a custom device ID via `X-Device-ID` header.

---

## Password Security

### bcrypt Hashing

Passwords are hashed using bcrypt with configurable cost:

```
Cost = 12 (default, ~400ms hashing time)
Cost = 14 (high security, ~1.5s)
```

**bcrypt Properties**:
- Automatically generates random salt
- Resistant to GPU/ASIC attacks
- Adjustable work factor (cost)
- Fixed output length: 60 characters

### Password Complexity Rules

| Rule | Configuration |
|------|---------------|
| Min Length | 8 characters |
| Max Length | 64 characters |
| Require Uppercase | true |
| Require Lowercase | true |
| Require Digit | true |
| Require Special | true |
| Reject Common Passwords | Yes (weak password list) |

### Password History

To prevent password reuse, the system stores password hashes in `PasswordHistory`:

```go
// Check if password was used recently
histories := historyRepo.ListRecent(userID, 5)
for _, h := range histories {
    if hasher.Verify(newPassword, h.PasswordHash) {
        return ErrPasswordInHistory
    }
}
```

### Password Hash Upgrade

When a user logs in with an old password hash (lower cost), the system automatically upgrades:

```go
if hasher.NeedsRehash(user.PasswordHash) {
    newHash, _ := hasher.Hash(password)
    userRepo.UpdatePasswordHash(ctx, user.ID, newHash)
}
```

---

## Data Encryption

### AES-256-GCM Encryption

Sensitive data in audit logs is encrypted:

| Field | Encryption Method |
|-------|-------------------|
| `LoginAudit.IP` | AES-256-GCM + SHA256 hash for query |
| `LoginAudit.UserAgent` | AES-256-GCM |
| `LoginAudit.Location` | AES-256-GCM (optional) |

### Encryption Key Management

```yaml
auth:
  encryption:
    data_key: "${AUTH_DATA_KEY}"  # 32-byte AES key from env
    key_version: 1
```

**Best Practices**:
- Read encryption key from environment variable
- Never hardcode keys in source code
- Use key management service (AWS KMS, HashiCorp Vault) in production
- Support key rotation with previous_key fallback

### Query Encrypted Fields

For searchable encrypted fields (like IP), store both encrypted value and hash:

```go
type LoginAudit struct {
    IPEncrypted  string  // AES encrypted original IP
    IPHash       string  // SHA256 hash for query
}

// Query by IP
ipHash := sha256Hash(ip)
db.Where("ip_hash = ?", ipHash).First(&audit)

// Decrypt to get original IP
originalIP := encryptionService.Decrypt(audit.IPEncrypted)
```

---

## API Endpoints Reference

### Public Endpoints

#### Login

```bash
POST /auth/login
Content-Type: application/json

{
    "email": "user@example.com",
    "password": "SecurePass123!"
}
```

**Response**:
```json
{
    "user": {
        "id": "uuid",
        "email": "user@example.com",
        "name": "User Name",
        "role": "admin"
    },
    "tenants": [
        {
            "tenant_id": "uuid",
            "role": "admin",
            "status": "active",
            "is_default": true,
            "joined_at": "2024-01-01T00:00:00Z"
        }
    ],
    "current_tenant_id": "uuid",
    "access_token": "eyJhbG...",
    "refresh_token": "eyJhbG...",
    "expires_at": "2024-01-01T00:15:00Z",
    "is_anomaly": false,
    "anomaly_type": ""
}
```

#### Individual Registration

```bash
POST /auth/register
Content-Type: application/json

{
    "email": "newuser@example.com",
    "password": "SecurePass123!",
    "name": "New User"
}
```

**Response**:
```json
{
    "user": {
        "id": "uuid",
        "email": "newuser@example.com",
        "name": "New User"
    },
    "tenant_id": "public-tenant-uuid",
    "access_token": "eyJhbG...",
    "refresh_token": "eyJhbG...",
    "expires_at": "2024-01-01T00:15:00Z"
}
```

#### Enterprise Registration

```bash
POST /auth/tenant/register
Content-Type: application/json

{
    "tenant_name": "My Company",
    "tenant_slug": "my-company",
    "email": "admin@example.com",
    "password": "SecurePass123!",
    "name": "Admin User"
}
```

**Response**:
```json
{
    "tenant": {
        "id": "uuid",
        "name": "My Company",
        "slug": "my-company",
        "status": "active",
        "plan": "free"
    },
    "user": {
        "id": "uuid",
        "email": "admin@example.com",
        "name": "Admin User",
        "role": "admin"
    },
    "access_token": "eyJhbG...",
    "refresh_token": "eyJhbG...",
    "expires_at": "2024-01-01T00:15:00Z"
}
```

#### Refresh Token

```bash
POST /auth/refresh
Content-Type: application/json

{
    "refresh_token": "eyJhbG...",
    "device_id": "optional-device-id"
}
```

**Response**:
```json
{
    "access_token": "eyJhbG...",
    "expires_at": "2024-01-01T00:15:00Z"
}
```

### Authenticated Endpoints

All authenticated endpoints require the `Authorization: Bearer <access_token>` header.

#### Get Profile

```bash
GET /auth/profile
Authorization: Bearer eyJhbG...
```

**Response**:
```json
{
    "user": {
        "id": "uuid",
        "email": "user@example.com",
        "name": "User Name",
        "role": "admin",
        "status": "active",
        "user_mode": "organization"
    },
    "tenants": [
        {
            "tenant_id": "uuid",
            "role": "admin",
            "status": "active",
            "is_default": true,
            "joined_at": "2024-01-01T00:00:00Z"
        }
    ],
    "current_tenant_id": "uuid"
}
```

#### Get Tenants

```bash
GET /auth/tenants
Authorization: Bearer eyJhbG...
```

**Response**:
```json
{
    "tenants": [
        {
            "tenant_id": "uuid",
            "role": "admin",
            "status": "active",
            "is_default": true,
            "joined_at": "2024-01-01T00:00:00Z"
        },
        {
            "tenant_id": "uuid2",
            "role": "member",
            "status": "active",
            "is_default": false,
            "joined_at": "2024-02-01T00:00:00Z"
        }
    ]
}
```

#### Switch Tenant

```bash
POST /auth/switch-tenant
Authorization: Bearer eyJhbG...
Content-Type: application/json

{
    "tenant_id": "uuid-to-switch-to"
}
```

**Response**:
```json
{
    "access_token": "eyJhbG...",
    "current_tenant_id": "uuid-to-switch-to",
    "expires_at": "2024-01-01T00:15:00Z"
}
```

#### Change Password

```bash
PUT /auth/password
Authorization: Bearer eyJhbG...
Content-Type: application/json

{
    "current_password": "OldPass123!",
    "new_password": "NewPass456!"
}
```

**Response**:
```json
{
    "message": "password changed successfully, please login again"
}
```

#### Logout

```bash
POST /auth/logout
Authorization: Bearer eyJhbG...
```

**Response**:
```json
{
    "message": "logged out successfully"
}
```

#### Logout All Devices

```bash
POST /auth/logout-all
Authorization: Bearer eyJhbG...
```

**Response**:
```json
{
    "message": "logged out from all devices"
}
```

---

## Error Codes

Authentication-related error codes:

| Code | Error | HTTP Status |
|------|-------|-------------|
| `LOGIN_001` | `ErrInvalidCredentials` | 401 |
| `LOGIN_002` | `ErrUserNotFound` | 401 |
| `LOGIN_003` | `ErrUserInactive` | 401 |
| `LOGIN_004` | `ErrTenantSuspended` | 403 |
| `LOGIN_005` | `ErrTooManyAttempts` | 429 |
| `LOGIN_006` | `ErrSessionExpired` | 401 |
| `LOGIN_007` | `ErrTokenRevoked` | 401 |
| `LOGIN_008` | `ErrDeviceMismatch` | 401 |
| `LOGIN_009` | `ErrRefreshTokenInvalid` | 401 |
| `REGISTER_001` | `ErrEmailExists` | 400 |
| `REGISTER_002` | `ErrInvalidEmailFormat` | 400 |
| `REGISTER_003` | `ErrPasswordTooShort` | 400 |
| `REGISTER_004` | `ErrPasswordNoUpper` | 400 |
| `REGISTER_005` | `ErrPasswordNoLower` | 400 |
| `REGISTER_006` | `ErrPasswordNoDigit` | 400 |
| `REGISTER_007` | `ErrPasswordNoSpecial` | 400 |
| `REGISTER_008` | `ErrPasswordInHistory` | 400 |
| `TENANT_006` | `ErrTenantSlugExists` | 400 |

---

## Configuration Reference

### JWT Configuration

```yaml
auth:
  jwt:
    secret_key: "${JWT_SECRET}"       # Required: 256-bit secret
    access_token_expire: 15m          # Access token expiry
    refresh_token_expire: 168h        # Refresh token expiry (7 days)
```

### Login Security Configuration

```yaml
auth:
  login_security:
    max_failed_attempts: 5            # Max failed attempts before block
    failed_window: 15m                # Failed attempt counting window
    block_duration: 30m               # Block duration after max failures
    enable_audit_log: true            # Enable login audit logging
    encrypt_audit_data: true          # Encrypt sensitive audit data
    anomaly_detection: true           # Enable anomaly detection
    new_device_alert: true            # Send alert for new device login
```

### Password Configuration

```yaml
auth:
  password:
    min_length: 8                     # Minimum password length
    max_length: 64                    # Maximum password length
    require_upper: true               # Require uppercase letter
    require_lower: true               # Require lowercase letter
    require_digit: true               # Require digit
    require_special: true             # Require special character
    history_count: 5                  # Password history count to check
    bcrypt_cost: 12                   # bcrypt hashing cost (10-14)
```

### Encryption Configuration

```yaml
auth:
  encryption:
    data_key: "${AUTH_DATA_KEY}"      # Required: 32-byte AES key
    key_version: 1                    # Key version for rotation
```

---

## Middleware Usage

### JWT Auth Middleware

Apply JWT authentication to protected routes:

```go
// In router setup
authGroup := router.Group("/auth")
authGroup.Use(middleware.JWTAuthMiddleware(
    container.Services.JWTService,
    container.Services.UserAuthService,
))

// Protected endpoints
authGroup.POST("/logout", authHandler.Logout)
authGroup.GET("/profile", authHandler.GetProfile)
```

### Role-based Access Control

Restrict access by role:

```go
// Require admin role
adminGroup := router.Group("/admin")
adminGroup.Use(middleware.JWTAuthMiddleware(...))
adminGroup.Use(middleware.RequireAdmin())

// Require specific roles
apiGroup.Use(middleware.RequireRole("admin", "member"))
```

### Context Helpers

Extract user info from context in handlers:

```go
func (h *Handler) SomeEndpoint(c *gin.Context) {
    userID := GetUserIDFromContext(c)
    tenantID := GetTenantIDFromContext(c)
    userTenant := middleware.GetUserTenant(c)
    role := middleware.GetRole(c)
    
    // Use the extracted info
}
```

---

## Best Practices

### Client Implementation

1. **Store refresh token securely**: Use secure storage (not localStorage in browsers)
2. **Refresh token before expiry**: Check expiry and refresh proactively
3. **Handle token expiry gracefully**: Redirect to login when refresh fails
4. **Include device ID**: Send consistent device ID for anomaly detection
5. **Handle anomaly warnings**: Show user notification when `is_anomaly` is true

### Server Deployment

1. **Use strong JWT secret**: Generate 256-bit random secret
2. **Configure encryption keys**: Set AUTH_DATA_KEY environment variable
3. **Enable audit logging**: Keep login history for security analysis
4. **Monitor anomaly alerts**: Review and act on unusual login patterns
5. **Regular key rotation**: Rotate encryption keys periodically

### Security Considerations

1. **HTTPS is mandatory**: Never transmit tokens over HTTP
2. **Short token expiry**: 15-minute access tokens minimize exposure
3. **Device binding**: Refresh tokens tied to device prevent theft
4. **Rate limiting**: Already protected by brute-force limits
5. **No password in logs**: Never log passwords or tokens

---

## Troubleshooting

### Common Issues

**Issue**: "too many failed attempts"
- **Cause**: IP blocked due to failed login attempts
- **Solution**: Wait for block duration (30 min) or contact admin

**Issue**: "token expired"
- **Cause**: Access token expired after 15 minutes
- **Solution**: Use refresh token to get new access token

**Issue**: "device mismatch"
- **Cause**: Refresh token used from different device
- **Solution**: Use correct device ID or re-login

**Issue**: "password in history"
- **Cause**: New password matches recent password
- **Solution**: Use a different password not in history

### Debug Commands

```bash
# Check failed login attempts in Redis
redis-cli GET "login_failed:192.168.1.1:user@example.com"

# Check if token is blacklisted
redis-cli GET "jwt_blacklist:<token_hash>"

# View login audit logs (admin)
curl -X GET http://localhost:8080/admin/login-audits \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

---

## Migration Guide

### From API Key Auth to User Auth

If you're migrating from API key-only authentication:

1. **Create public tenant**: Ensure public tenant exists
2. **Register users**: Use `/auth/register` endpoint
3. **Update API keys**: Link existing API keys to users
4. **Enable both auth**: Support both API key and JWT for transition
5. **Gradual rollout**: Migrate users gradually with clear communication

### Database Migration

The new entities are auto-migrated by GORM:

```go
// In main.go or container
db.AutoMigrate(
    &entity.UserTenant{},
    &entity.LoginAudit{},
    &entity.PasswordHistory{},
    &entity.RefreshToken{},
)
```

---

## Platform Admin Authentication

The platform authentication system provides separate authentication for platform administrators (super admins, support staff, billing admins).

### Platform Admin Entity

| Field | Type | Description |
|-------|------|-------------|
| `ID` | UUID | Unique identifier |
| `Email` | string | Admin email (unique) |
| `PasswordHash` | string | bcrypt hashed password |
| `Name` | string | Display name |
| `Role` | string | Role: `super_admin`, `support`, `billing_admin` |
| `Permissions` | JSON | Array of permissions (e.g., `["read", "write", "*"]`) |
| `Status` | string | Status: `active`, `inactive`, `suspended` |
| `LastLoginAt` | time | Last login timestamp |

### Platform Admin Service

```go
// Login for platform admin
func (s *PlatformAuthService) Login(ctx context.Context, email, password string) (*entity.PlatformAdmin, string, error)

// Validate session with caching
func (s *PlatformAuthService) ValidateSession(ctx context.Context, adminID uuid.UUID) (*entity.PlatformAdmin, error)

// Check specific permission
func (s *PlatformAuthService) CheckPermission(ctx context.Context, adminID uuid.UUID, permission string) (bool, error)

// Role checking
func (s *PlatformAuthService) HasRole(ctx context.Context, adminID uuid.UUID, role string) (bool, error)
func (s *PlatformAuthService) IsSuperAdmin(ctx context.Context, adminID uuid.UUID) (bool, error)
```

### In-Memory Caching

Platform admins are cached in memory for 5 minutes to reduce database queries:

```go
// Cache structure
type cachedPlatformAdmin struct {
    admin      *entity.PlatformAdmin
    permissions []string
    expiry     time.Time
}

// Cache invalidation on update/delete
func (s *PlatformAuthService) invalidateCache(id uuid.UUID)
```

---

## Test Coverage

The authentication system has comprehensive test coverage across all components:

### Coverage Summary

| Package | Coverage | Key Functions |
|---------|----------|---------------|
| `internal/infrastructure/auth` | **83.0%** | Login (94%), Register (77%), ValidateSession (100%) |
| `platform_auth.go` | **90%+** | Login (92%), CreateAdmin (85%), UpdateAdmin (95%) |
| `jwt_service.go` | **85%+** | GenerateToken (82%), ValidateToken (89%), InvalidateToken (94%) |
| `login_security_service.go` | **85%+** | CheckLoginAllowed (100%), RecordFailedAttempt (74%) |

### Test Patterns

Tests follow these patterns:

1. **Mock Repositories**: Hand-written mocks using `map[uuid.UUID]*Entity` storage
2. **Redis Simulation**: Using `miniredis` for Redis-based tests (blacklist, rate limiting)
3. **Table-driven Tests**: Using `t.Run()` for organized test cases
4. **Error Checking**: Using `errors.Is(err, apperrors.ErrXxx)` for error type checking

### Running Tests

```bash
# Run all auth tests
go test -v ./internal/infrastructure/auth/...

# Run with coverage
go test -coverprofile=coverage_auth.out ./internal/infrastructure/auth/...
go tool cover -html=coverage_auth.out -o coverage_report.html

# Check per-function coverage
go tool cover -func=coverage_auth.out | grep "Login"
go tool cover -func=coverage_auth.out | grep "Register"
```

### Key Test Files

| File | Tests | Focus Areas |
|------|-------|-------------|
| `user_auth_service_test.go` | 40+ | Login/Register/RefreshToken edge cases |
| `jwt_service_test.go` | 25+ | Token generation/validation/blacklist |
| `login_security_service_test.go` | 30+ | Brute-force protection/anomaly detection |
| `platform_auth_test.go` | 20+ | Platform admin CRUD/caching/permissions |
| `jwt_auth_test.go` | 20+ | Middleware context extraction/role checking |

---

## Related Documentation

- [API Reference](api-reference.md)
- [Enterprise Payment System](payment-system.md)
- [MCP Integration](mcp-integration.md)
- [Claude Code Integration](claude-code-integration.md)