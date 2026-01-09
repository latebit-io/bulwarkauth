# Multi-Tenant Architecture Plan: Row-Level Isolation (Phase 1)

## Executive Summary

Implement **row-level multi-tenancy** for BulwarkAuth to support multiple tenants in a single instance with complete data isolation. This approach:

- **Phase 1 (This Plan)**: Shared BulwarkAuth instance with tenant-scoped data (simple, cost-effective)
- **Phase 2 (Future)**: Optional dedicated instances for enterprise customers (premium isolation)

**Benefits:**
- ✅ Simple to implement (add `tenantId` field + middleware)
- ✅ Cost-effective (one instance serves many tenants)
- ✅ Easy tenant provisioning (create tenant record)
- ✅ Flexible upgrade path to dedicated instances
- ✅ Similar to Auth0's tiered multi-tenancy approach

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Apps                              │
│         (tenant1.yourauth.com, tenant2.yourauth.com)            │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                  BulwarkAuth (Single Instance)                   │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ Tenant Middleware                                         │  │
│  │ - Extracts tenantId from subdomain (acme.yourauth.com)    │  │
│  │ - Validates tenant exists and is active                   │  │
│  │ - Stores tenantId in Echo context                         │  │
│  └───────────────────────────────────────────────────────────┘  │
│                           ↓                                      │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ API Layer (Handlers)                                      │  │
│  │ - Extract tenantId from context                           │  │
│  │ - Pass to service layer                                   │  │
│  └───────────────────────────────────────────────────────────┘  │
│                           ↓                                      │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ Service Layer                                             │  │
│  │ - Accept tenantId parameter                               │  │
│  │ - Pass to repository layer                                │  │
│  └───────────────────────────────────────────────────────────┘  │
│                           ↓                                      │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ Repository Layer                                          │  │
│  │ - Filter ALL queries by tenantId                          │  │
│  │ - Enforce compound unique indexes                         │  │
│  └───────────────────────────────────────────────────────────┘  │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                     MongoDB (Single Database)                    │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ accounts: { tenantId, email, password, ... }             │   │
│  │ tokens: { tenantId, email, clientId, accessToken, ... }  │   │
│  │ logonCodes: { tenantId, email, code, ... }               │   │
│  │ failedAttempts: { tenantId, email, count, ... }          │   │
│  │ forgots: { tenantId, email, token, ... }                 │   │
│  │ tenants: { tenantId, name, domain, isActive, ... }       │   │
│  │ signingKeys: { keyId, privateKey, ... } (SHARED)         │   │
│  │ emails: { name, template, ... } (SHARED)                 │   │
│  └──────────────────────────────────────────────────────────┘   │
│  Indexes: { tenantId: 1, email: 1 }  (compound, unique)         │
└─────────────────────────────────────────────────────────────────┘
```

## Key Design Decisions

### 1. Tenant Identification: Subdomain-Based
**Method**: Extract tenantId from subdomain
- `acme.yourauth.com` → tenantId: `acme`
- `widgets-inc.yourauth.com` → tenantId: `widgets-inc`

**Rationale**:
- Clean, professional appearance
- Easy DNS configuration
- Works with wildcard SSL certificates
- No API contract changes required

### 2. Resource Isolation Strategy

**Tenant-Scoped Resources** (include `tenantId`):
- `accounts` - User accounts
- `tokens` - Authentication tokens  
- `logonCodes` - Magic link codes
- `failedAttempts` - Failed login tracking
- `forgots` - Password reset tokens

**Shared Resources** (no `tenantId`):
- `signingKeys` - JWT signing keys (infrastructure)
- `emails` - Email templates (infrastructure)

**Rationale**: Signing keys and templates are infrastructure concerns, not user data. Sharing reduces key management overhead.

### 3. Repository Pattern: Explicit tenantId Parameter

**Approach**: Add `tenantId string` as first parameter to all methods

```go
// Before
Read(ctx context.Context, email string) (*Account, error)

// After
Read(ctx context.Context, tenantId string, email string) (*Account, error)
```

**Rationale**:
- Type-safe and explicit
- Compile-time enforcement
- Easy to audit in code reviews
- No hidden dependencies

### 4. Database Indexes: Compound Unique Constraints

**Strategy**: Replace single-field unique indexes with compound indexes

```go
// Before: Unique index on email
{email: 1} - unique

// After: Compound unique index on (tenantId, email)
{tenantId: 1, email: 1} - unique
```

**Benefit**: Same email can exist across different tenants while maintaining uniqueness within each tenant.

## Implementation Plan

### New Files to Create (7)

1. **`internal/tenants/tenant.go`** - Tenant model
2. **`internal/tenants/tenant_repository.go`** - Tenant data access
3. **`internal/tenants/tenant_service.go`** - Tenant business logic
4. **`internal/middleware/tenant.go`** - Tenant extraction middleware
5. **`api/tenants/tenant_handlers.go`** - Tenant management API
6. **`api/tenants/tenant_routes.go`** - Tenant routes
7. **`scripts/migrate_to_multitenant.go`** - Database migration script

### Files to Modify (20+)

**Data Models** (add `tenantId` field):
- `internal/accounts/accounts.go`
- `internal/authentication/token_repository.go`
- `internal/authentication/logon_code_repository.go`
- `internal/authentication/failed_attempts.go`
- `internal/accounts/forgot_repository.go`

**Repository Interfaces** (add `tenantId` parameter):
- `internal/accounts/account_repository.go`
- `internal/authentication/token_repository.go`
- `internal/authentication/logon_code_repository.go`
- `internal/authentication/failed_attempts.go`
- `internal/accounts/forgot_repository.go`

**Service Interfaces** (add `tenantId` parameter):
- `internal/accounts/accounts.go`
- `internal/authentication/authenticate.go`
- `internal/authentication/logon_code.go`
- `internal/authentication/social/social-validator.go`
- `internal/email/email.go`

**API Handlers** (extract tenantId from context):
- `api/accounts/account_handlers.go`
- `api/authentication/authentication_handlers.go`
- `api/authentication/logon_code_handlers.go`
- `api/authentication/social_handlers.go`

**Application Bootstrap**:
- `cmd/bulwarkauth/main.go` - Wire tenant middleware
- `cmd/bulwarkauth/config.go` - Add BASE_DOMAIN config

## Implementation Steps (18 Days)

### Phase 1: Foundation (Days 1-2)
**Goal**: Create tenant management infrastructure

**Tasks**:
- [ ] Create `internal/tenants/tenant.go` model
- [ ] Create `internal/tenants/tenant_repository.go` with MongoDB implementation
- [ ] Create `internal/tenants/tenant_service.go`
- [ ] Create `internal/middleware/tenant.go` for subdomain extraction
- [ ] Create `api/tenants/tenant_handlers.go` for admin API
- [ ] Create `api/tenants/tenant_routes.go`
- [ ] Write unit tests for tenant repository
- [ ] Write unit tests for middleware

### Phase 2: Data Models (Day 3)
**Goal**: Add tenantId field to all models

**Tasks**:
- [ ] Add `TenantId string` to `Account` struct
- [ ] Add `TenantId string` to `Token` struct
- [ ] Add `TenantId string` to `LogonCode` struct
- [ ] Add `TenantId string` to `FailedAttempt` struct
- [ ] Add `TenantId string` to `Forgot` struct

### Phase 3: Repository Layer (Days 4-6)
**Goal**: Update all repositories to be tenant-aware

**Tasks**:
- [ ] Update `AccountRepository` interface (add tenantId param to all methods)
- [ ] Update `AccountRepository` implementation (filter queries by tenantId)
- [ ] Update account indexes to compound `{tenantId, email}`
- [ ] Update `TokenRepository` interface and implementation
- [ ] Update token indexes to compound `{tenantId, email, clientId}`
- [ ] Update `LogonCodeRepository` interface and implementation
- [ ] Update logon code indexes
- [ ] Update `FailedAttemptRepository` interface and implementation
- [ ] Update failed attempt indexes
- [ ] Update `ForgotRepository` interface and implementation
- [ ] Update forgot indexes
- [ ] Write multi-tenant unit tests for each repository

### Phase 4: Service Layer (Days 7-9)
**Goal**: Pass tenantId through service layer

**Tasks**:
- [ ] Update `AccountService` interface (add tenantId param)
- [ ] Update `AccountService` implementation
- [ ] Update `AuthenticationService` interface
- [ ] Update `AuthenticationService` implementation
- [ ] Update `LogonCodeService` interface and implementation
- [ ] Update `SocialService` interface and implementation
- [ ] Update `EmailService` interface (tenant context for emails)
- [ ] Write service layer unit tests

### Phase 5: API Layer (Days 10-11)
**Goal**: Extract tenantId from context in all handlers

**Tasks**:
- [ ] Update all `account_handlers.go` methods (Register, Verify, etc.)
- [ ] Update all `authentication_handlers.go` methods
- [ ] Update all `logon_code_handlers.go` methods
- [ ] Update all `social_handlers.go` methods
- [ ] Test each handler individually

### Phase 6: Application Wiring (Day 12)
**Goal**: Wire everything together

**Tasks**:
- [ ] Add `BASE_DOMAIN` to `config.go`
- [ ] Wire tenant repository/service in `main.go`
- [ ] Create tenant middleware instance
- [ ] Apply tenant middleware to all protected routes
- [ ] Add tenant admin routes (without tenant middleware)
- [ ] Test complete application startup

### Phase 7: Testing (Days 13-15)
**Goal**: Comprehensive testing

**Tasks**:
- [ ] Write integration test: Same email in different tenants
- [ ] Write integration test: Cross-tenant isolation
- [ ] Write integration test: Tenant activation/deactivation
- [ ] Write security test: Cross-tenant data access attempts
- [ ] Write security test: Token validation across tenants
- [ ] Performance test: Query performance with compound indexes
- [ ] Load test: Multiple tenants simultaneously

### Phase 8: Migration (Day 16)
**Goal**: Prepare database migration

**Tasks**:
- [ ] Create `scripts/migrate_to_multitenant.go`
- [ ] Test migration on database copy
- [ ] Create default tenant
- [ ] Verify all existing data gets `tenantId="default"`
- [ ] Document rollback procedure

### Phase 9: Deployment (Day 17)
**Goal**: Deploy to production

**Tasks**:
- [ ] Backup production database
- [ ] Run migration script
- [ ] Deploy new BulwarkAuth version
- [ ] Monitor logs for errors
- [ ] Verify default tenant works

### Phase 10: Verification (Day 18)
**Goal**: Verify everything works

**Tasks**:
- [ ] Test authentication with default tenant
- [ ] Create test tenant via admin API
- [ ] Register user in test tenant
- [ ] Verify isolation between default and test tenant
- [ ] Monitor performance
- [ ] Document tenant creation process

## Critical Code Examples

### Tenant Middleware (Subdomain Extraction)

```go
// internal/middleware/tenant.go
package middleware

import (
    "errors"
    "net/http"
    "strings"
    
    "github.com/labstack/echo/v4"
    "github.com/latebit-io/bulwarkauth/api/problem"
    "github.com/latebit-io/bulwarkauth/internal/tenants"
)

const TenantContextKey = "tenantId"

type TenantMiddleware struct {
    tenantService tenants.TenantService
    baseDomain    string
}

func NewTenantMiddleware(tenantService tenants.TenantService, baseDomain string) *TenantMiddleware {
    return &TenantMiddleware{
        tenantService: tenantService,
        baseDomain:    baseDomain,
    }
}

func (tm *TenantMiddleware) Extract() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            host := c.Request().Host
            
            // Extract subdomain: acme.yourauth.com → "acme"
            tenantId, err := extractTenantFromHost(host, tm.baseDomain)
            if err != nil {
                httpError := problem.NewBadRequest(err)
                return echo.NewHTTPError(httpError.Status, httpError)
            }
            
            // Validate tenant exists and is active
            tenant, err := tm.tenantService.Get(c.Request().Context(), tenantId)
            if err != nil {
                return echo.NewHTTPError(http.StatusNotFound, problem.Details{
                    Type:   "https://latebit.io/bulwark/errors/",
                    Title:  "Tenant Not Found",
                    Status: http.StatusNotFound,
                    Detail: "Invalid tenant: " + tenantId,
                })
            }
            
            if !tenant.IsActive {
                return echo.NewHTTPError(http.StatusForbidden, problem.Details{
                    Type:   "https://latebit.io/bulwark/errors/",
                    Title:  "Tenant Inactive",
                    Status: http.StatusForbidden,
                    Detail: "Tenant is not active",
                })
            }
            
            // Store in context for handlers
            c.Set(TenantContextKey, tenantId)
            
            return next(c)
        }
    }
}

func extractTenantFromHost(host, baseDomain string) (string, error) {
    // Remove port: acme.yourauth.com:8080 → acme.yourauth.com
    if idx := strings.Index(host, ":"); idx != -1 {
        host = host[:idx]
    }
    
    // Check if host ends with baseDomain
    if !strings.HasSuffix(host, baseDomain) {
        return "", errors.New("invalid domain")
    }
    
    // Extract subdomain
    subdomain := strings.TrimSuffix(host, "."+baseDomain)
    
    // Validate (no www, no empty)
    if subdomain == "" || subdomain == baseDomain || subdomain == "www" {
        return "", errors.New("tenant subdomain required")
    }
    
    // Validate format: alphanumeric + hyphens, 3-63 chars
    if !isValidTenantId(subdomain) {
        return "", errors.New("invalid tenant id format")
    }
    
    return subdomain, nil
}

func isValidTenantId(tenantId string) bool {
    if len(tenantId) < 3 || len(tenantId) > 63 {
        return false
    }
    for _, ch := range tenantId {
        if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
            return false
        }
    }
    return true
}

// Helper to get tenantId from Echo context
func GetTenantId(c echo.Context) (string, error) {
    tenantId := c.Get(TenantContextKey)
    if tenantId == nil {
        return "", errors.New("tenant context not found")
    }
    
    tenantIdStr, ok := tenantId.(string)
    if !ok {
        return "", errors.New("invalid tenant context type")
    }
    
    return tenantIdStr, nil
}
```

### Repository Pattern (Example: AccountRepository)

```go
// Before
func (r MongodbAccountRepository) Read(ctx context.Context, email string) (*Account, error) {
    collection := r.db.Collection("accounts")
    result := collection.FindOne(ctx, bson.M{"email": email})
    // ...
}

// After
func (r MongodbAccountRepository) Read(ctx context.Context, tenantId, email string) (*Account, error) {
    collection := r.db.Collection("accounts")
    result := collection.FindOne(ctx, bson.M{
        "tenantId": tenantId,
        "email":    email,
    })
    // ...
}

// Index creation in constructor
func NewMongodbAccountRepository(db *mongo.Database, encryption encryption.Encryption) AccountRepository {
    collection := db.Collection("accounts")
    
    // Compound unique index
    _, _ = collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
        Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "email", Value: 1}},
        Options: options.Index().SetUnique(true),
    })
    
    return &MongodbAccountRepository{db: db, encryption: encryption}
}
```

### Handler Pattern (Extract tenantId from context)

```go
// api/accounts/account_handlers.go
func (ah AccountHandler) Register(c echo.Context) error {
    // Extract tenantId from context
    tenantId, err := middleware.GetTenantId(c)
    if err != nil {
        httpError := problem.NewServerError(err)
        return echo.NewHTTPError(httpError.Status, httpError)
    }
    
    newAccountRequest := new(NewAccountRequest)
    err = c.Bind(newAccountRequest)
    if err != nil {
        httpError := problem.NewBadRequest(err)
        return echo.NewHTTPError(httpError.Status, httpError)
    }

    ctx := c.Request().Context()
    
    // Pass tenantId to service
    err = ah.accounts.Register(ctx, tenantId, newAccountRequest.Email, newAccountRequest.Password)
    if err != nil {
        var dupErr accounts.AccountDuplicateError
        if errors.As(err, &dupErr) {
            httpError := problem.NewConflict(err)
            return echo.NewHTTPError(httpError.Status, httpError)
        }
        httpError := problem.NewServerError(err)
        return echo.NewHTTPError(httpError.Status, httpError)
    }
    
    return c.NoContent(http.StatusCreated)
}
```

### Database Migration Script

```go
// scripts/migrate_to_multitenant.go
package main

import (
    "context"
    "flag"
    "log"
    "time"
    
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
    dbUri := flag.String("db", "", "MongoDB URI")
    dbName := flag.String("dbname", "bulwarkauth", "Database name")
    defaultTenant := flag.String("tenant", "default", "Default tenant ID for existing data")
    flag.Parse()
    
    if *dbUri == "" {
        log.Fatal("Database URI required")
    }
    
    ctx := context.Background()
    client, err := mongo.Connect(ctx, options.Client().ApplyURI(*dbUri))
    if err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)
    
    db := client.Database(*dbName)
    
    log.Println("Starting multi-tenant migration...")
    
    // Migrate each collection
    collections := []string{"accounts", "tokens", "logonCodes", "failedAttempts", "forgots"}
    
    for _, collName := range collections {
        log.Printf("Migrating collection: %s\n", collName)
        coll := db.Collection(collName)
        
        // Add tenantId field to all documents
        filter := bson.M{"tenantId": bson.M{"$exists": false}}
        update := bson.M{"$set": bson.M{"tenantId": *defaultTenant}}
        
        result, err := coll.UpdateMany(ctx, filter, update)
        if err != nil {
            log.Fatalf("Failed to migrate %s: %v", collName, err)
        }
        
        log.Printf("  Updated %d documents\n", result.ModifiedCount)
    }
    
    // Create default tenant
    log.Println("Creating default tenant...")
    tenantColl := db.Collection("tenants")
    _, err = tenantColl.InsertOne(ctx, bson.M{
        "tenantId": *defaultTenant,
        "name":     "Default Tenant",
        "domain":   "default.yourauth.com",
        "isActive": true,
        "created":  time.Now(),
        "modified": time.Now(),
    })
    if err != nil {
        log.Printf("Warning: Could not create default tenant: %v\n", err)
    }
    
    log.Println("Migration completed successfully!")
}
```

## Security Considerations

### Critical Security Checks

**Code Review Checklist**:
- [ ] All MongoDB queries include `tenantId` filter
- [ ] All unique indexes include `tenantId` as first field
- [ ] No repository accepts wildcard `tenantId` (empty, "*", etc.)
- [ ] Middleware validates tenant exists and is active
- [ ] Admin APIs (tenant management) are NOT protected by tenant middleware
- [ ] Token validation includes tenant check

### Testing Security Boundaries

**Multi-Tenant Integration Tests**:
```go
func TestMultiTenant_DataIsolation(t *testing.T) {
    ctx := context.Background()
    
    // Create account in tenant1
    guard1 := bulwark.NewGuard("http://tenant1.localhost:8080", httpClient)
    err := guard1.Account.Create(ctx, "user@example.com", "password123")
    require.NoError(t, err)
    
    // Switch to tenant2
    guard2 := bulwark.NewGuard("http://tenant2.localhost:8080", httpClient)
    
    // Attempt to authenticate same email in tenant2 - should fail
    _, err = guard2.Authenticate.Password(ctx, "user@example.com", "password123", "client1")
    require.Error(t, err, "Should not find user in tenant2")
    
    // Create same email in tenant2 - should succeed (different tenant)
    err = guard2.Account.Create(ctx, "user@example.com", "password123")
    require.NoError(t, err, "Same email should work in different tenant")
    
    // Both should be able to authenticate independently
    auth1, err := guard1.Authenticate.Password(ctx, "user@example.com", "password123", "client1")
    require.NoError(t, err)
    
    auth2, err := guard2.Authenticate.Password(ctx, "user@example.com", "password123", "client2")
    require.NoError(t, err)
    
    // Tokens should be different
    assert.NotEqual(t, auth1.AccessToken, auth2.AccessToken)
}
```

### Potential Security Issues & Mitigations

| Issue | Risk | Mitigation |
|-------|------|------------|
| Missing tenantId in query | Cross-tenant data leak | Code reviews + integration tests |
| Subdomain spoofing | Bypass tenant isolation | Middleware validates tenant in DB |
| Cross-tenant token use | Token from tenant1 used in tenant2 | Add tenantId to JWT claims (future) |
| Index race condition | Duplicates during migration | Run migration offline |

## Configuration Changes

### New Environment Variables

```bash
# .env file additions
BASE_DOMAIN=yourauth.com
```

### Updated Docker Compose

```yaml
# docker-compose.yaml
services:
  bulwarkauth:
    image: ghcr.io/latebit-io/bulwarkauth:latest
    environment:
      - BASE_DOMAIN=yourauth.com
      - DB_CONNECTION=mongodb://mongodb:27017
      # ... other existing config
    ports:
      - "8080:8080"
```

## Tenant Management API

### Admin Endpoints (No Tenant Middleware)

```
POST   /api/admin/tenants              # Create new tenant
GET    /api/admin/tenants              # List all tenants
GET    /api/admin/tenants/:tenantId    # Get tenant details
PUT    /api/admin/tenants/:tenantId    # Update tenant
DELETE /api/admin/tenants/:tenantId    # Deactivate tenant
```

### Creating a Tenant

```bash
curl -X POST http://localhost:8080/api/admin/tenants \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "acme",
    "name": "Acme Corporation",
    "domain": "acme.yourauth.com"
  }'
```

### Tenant Record Structure

```go
type Tenant struct {
    TenantId   string    `bson:"tenantId"`    // e.g., "acme"
    Name       string    `bson:"name"`        // e.g., "Acme Corporation"
    Domain     string    `bson:"domain"`      // e.g., "acme.yourauth.com"
    IsActive   bool      `bson:"isActive"`    // true = active, false = deactivated
    Created    time.Time `bson:"created"`
    Modified   time.Time `bson:"modified"`
}
```

## Future Enhancements (Phase 2)

### Dedicated Instances for Enterprise Customers

When a customer requires complete infrastructure isolation:

1. **Deploy dedicated BulwarkAuth instance** for that tenant
2. **Migrate tenant data** to dedicated MongoDB database
3. **Add tenant router service** (or update DNS) to route requests
4. **No code changes** required (already tenant-aware)

**Benefits**:
- Same codebase supports both shared and dedicated
- Easy upgrade path (tenant graduates to dedicated instance)
- Flexible pricing tiers (shared = standard, dedicated = enterprise)

### Tenant-Specific Signing Keys

Add `tenantId` to signing keys for complete cryptographic isolation:

```go
type SigningKey struct {
    TenantId   string `bson:"tenantId"`  // ADD THIS (default "shared")
    KeyId      string `bson:"keyId"`
    // ... existing fields
}
```

### Per-Tenant Rate Limiting

Make rate limiter tenant-aware:

```go
rateLimiter := middleware.RateLimiter(
    middleware.NewRateLimiterMemoryStoreWithConfig(
        middleware.RateLimiterMemoryStoreConfig{
            KeyGenerator: func(c echo.Context) (string, error) {
                tenantId, _ := middleware.GetTenantId(c)
                return tenantId + ":" + c.RealIP(), nil
            },
        },
    ),
)
```

## Rollback Plan

If issues occur after deployment:

1. **Stop application**
2. **Restore database** from backup: `mongorestore --drop <backup-dir>`
3. **Revert to previous version** of BulwarkAuth
4. **Investigate** issue in staging environment
5. **Fix and redeploy** when ready

## Summary

This plan implements **row-level multi-tenancy** for BulwarkAuth with:

- ✅ **Complete data isolation** between tenants
- ✅ **Subdomain-based** tenant identification
- ✅ **Minimal code changes** (add tenantId parameter throughout stack)
- ✅ **Flexible architecture** (easy upgrade to dedicated instances)
- ✅ **Security-first approach** (compound indexes, middleware validation, comprehensive testing)
- ✅ **18-day implementation timeline** with clear phases

**Next Steps**: 
1. Review this plan
2. Mark which phases you want to implement yourself
3. Mark which phases to delegate to an agent
4. Begin implementation!
