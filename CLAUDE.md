# CLAUDE.md
You are a staff engineer who loves to write the best tests, you try not to mock 
and if possible use real services. 
This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

BulwarkAuth is an API-based, developer-focused JWT authentication/authorization subsystem written in Go with built-in multi-tenant support. It provides asymmetric key signing (RS256), passwordless authentication (magic links), password-based authentication, email verification, account lockout protection, and social sign-in capabilities (Google in development).

## Key Architecture

### Layered Architecture
The codebase follows a clean architecture pattern with three main layers:

1. **API Layer** (`api/`): HTTP handlers and routes using Echo framework
   - `api/accounts/`: Account management endpoints
   - `api/authentication/`: Authentication and logon code endpoints
   - `api/health/`: Health check endpoints
   - `api/problem/`: RFC 7807 problem details for error responses

2. **Internal Layer** (`internal/`): Business logic and domain models
   - `internal/accounts/`: Account service, repository pattern for accounts and forgot password
   - `internal/authentication/`: Authentication service, token repository, logon code (magic links), failed attempts tracking
   - `internal/tokens/`: JWT tokenizer, signing key service and repository
   - `internal/email/`: Email service with template support
   - `internal/encryption/`: Password encryption utilities
   - `internal/tenants/`: Tenant service and repository for multi-tenant support
   - `internal/utils/`: Transaction manager and test utilities

3. **Entry Point** (`cmd/bulwarkauth/`): Application bootstrap
   - `main.go`: Service initialization, dependency injection, middleware setup
   - `config.go`: Environment variable configuration

### Core Patterns

**Repository Pattern**: All data access goes through repository interfaces (`AccountRepository`, `TokenRepository`, `SigningKeyRepository`, etc.) with MongoDB implementations.

**Service Pattern**: Business logic encapsulated in service interfaces (`AccountService`, `AuthenticationService`, `LogonCodeService`, etc.) that orchestrate repositories and other services.

**Dependency Injection**: All dependencies are manually wired in `main.go` at startup.

**Transaction Management**: `TxManager` interface provides MongoDB transaction support for multi-document operations.

### Key Components

**Multi-Tenant System** (`internal/tenants/`):
- All accounts, tokens, and signing keys are isolated by tenant ID
- Each tenant has its own signing keys for JWT tokens
- Tenant service manages tenant creation and retrieval
- Use `"default"` tenant ID for single-tenant applications

**Token System** (`internal/tokens/`):
- `Tokenizer`: Creates and validates JWT access/refresh tokens
- `SigningKeyService`: Manages RSA key pairs for JWT signing with rotation support per tenant
- Tokens use RS256 signing with configurable expiration times
- Access tokens contain roles for RBAC and tenant ID, refresh tokens for renewal
- Tokens are scoped to tenants and cannot be used across tenants

**Authentication Flow**:
1. Password auth: `AuthenticationService.Authenticate()` validates credentials, checks failed attempts, returns tokens
2. Magic link auth: `LogonCodeService.Request()` generates 6-digit code, sends email; `Authenticate()` validates code, checks failed attempts
3. Token acknowledgement: `Acknowledge()` stores tokens per client ID (multi-device support)
4. Token renewal: `Renew()` uses refresh token to get new access/refresh token pair
5. Token revocation: `Revoke()` deletes tokens for a client
6. Account lockout: After 5 failed attempts (configurable), account is locked for a duration (configurable, default 300 seconds)

**Email System** (`internal/email/`):
- Template-based emails (verification, forgot password, magic link)
- Supports tenant-specific email templates via `EMAIL_TEMPLATE_DIRS`
- SMTP configuration via environment variables
- Test mode bypasses actual sending

## Coding Standards
 - always use range for loops when possible

## Development Commands

### Build and Run
```bash
# Build the binary
go build -o bulwarkauth ./cmd/bulwarkauth

# Run locally (requires .env file or environment variables)
go run ./cmd/bulwarkauth

# Run with Docker Compose (includes MongoDB)
docker-compose up

# Check version
./bulwarkauth -version
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./internal/accounts
go test ./internal/tokens

# Run specific test
go test ./internal/accounts -run TestAccountService_Create

# Run tests with verbose output
go test -v ./...
```

### Code Generation
```bash
# Generate mocks (uses mockery)
mockery --all --output internal/accounts --dir internal/accounts --case underscore
```

### Integration Testing
Integration tests validate the full API using the [bulwark-auth-guard](https://github.com/latebit-io/bulwark-auth-guard) client:
```bash
# Start services with docker-compose
docker-compose up -d

# Run integration tests
go test ./test/integration/... -v

# Stop services
docker-compose down
```

### HTTP Testing
HTTP request files are in `http/` directory:
1. Copy `http/accounts.http.example` to `http/accounts.http`
2. Fill in placeholders (API_KEY, TENANT_ID, TEST_EMAIL, etc.)
3. Use REST client (VS Code REST Client, IntelliJ) to execute requests
4. Note: All requests now require `tenantId` in the request body

## Configuration

All configuration via environment variables. Required variables:
- `DB_CONNECTION`: MongoDB connection string
- `DOMAIN`: Service domain name
- `WEBSITE_NAME`: Website display name
- `EMAIL_FROM_ADDRESS`: From address for emails
- `VERIFICATION_URL`: Frontend URL for email verification
- `FORGOT_PASSWORD_URL`: Frontend URL for password reset
- `MAGIC_URL`: Frontend URL for magic link authentication

Optional but important:
- `GOOGLE_CLIENT_ID`: For Google OAuth (feature in development)
- `API_KEY_ENABLED=true` + `API_KEY`: Require API key header `X-BULWARK-API-KEY`
- `CORS_ENABLED=true` + `ALLOWED_WEB_ORIGINS`: Configure CORS
- `ACCESS_TOKEN_EXPIRE_IN_SECONDS`: Default 3600 (1 hour)
- `REFRESH_TOKEN_EXPIRE_IN_SECONDS`: Default 86400 (24 hours)
- `LOCKOUT_DURATION_IN_SEC`: Account lockout duration in seconds (default 300)
- `MAX_FAILED_ATTEMPTS`: Maximum failed auth attempts before lockout (default 5)
- `EMAIL_TEMPLATE_DIRS`: Comma-separated list of directories for tenant-specific email templates
- `TEST_MODE=true`: Bypass email sending for testing

See `cmd/bulwarkauth/config.go` for complete list.

## Testing Infrastructure

### Unit Tests
The codebase uses `memongo` for in-memory MongoDB testing:
- `internal/utils/mongo_test_util.go`: Helper for creating test MongoDB instances
- Tests use real MongoDB operations against ephemeral instance
- No need for mocking MongoDB in most tests

### Integration Tests
Located in `test/integration/`:
- Uses [bulwark-auth-guard](https://github.com/latebit-io/bulwark-auth-guard) client library
- Tests complete user flows (account creation, authentication, token management)
- Validates multi-tenant support with `tenantId: "default"`
- Requires docker-compose to run (BulwarkAuth, MongoDB, MailHog)
- 13 test cases covering all major features including account lockout

See `test/integration/README.md` for details.

## Multi-Tenant Usage

When using the bulwark-auth-guard client, all API calls require a tenant ID:

```go
const tenantID = "default"  // Or your specific tenant ID

// Create account
err := guard.Account.Create(ctx, tenantID, email, password)

// Authenticate
auth, err := guard.Authenticate.Password(ctx, tenantID, email, password, clientID)

// Validate token
claims, err := guard.Authenticate.ValidateAccessToken(ctx, tenantID, accessToken)
```

For single-tenant applications, use `"default"` as the tenant ID.

## Social Authentication (In Development)

Google social sign-in is being added:
- `internal/authentication/social/social-validator.go`: OAuth validation logic
- Requires `GOOGLE_CLIENT_ID` environment variable
