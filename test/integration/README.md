# Integration Tests

Integration tests for BulwarkAuth using the [bulwark-auth-guard](https://github.com/latebit-io/bulwark-auth-guard) Go client library.

## Overview

These tests validate that the BulwarkAuth API works correctly with the actual Go client that users will use. This ensures:

- **API compatibility** - The server and client stay in sync
- **Real-world validation** - Tests use the actual client library
- **Breaking change detection** - Catches API changes that would break the client
- **Client refactoring safety** - Ensures client changes don't break functionality

## Prerequisites

The tests require:

1. **BulwarkAuth server** running at `http://localhost:8080`
2. **MongoDB** running at `mongodb://localhost:27017`

### Quick Start with Docker Compose

The easiest way to run the tests is using docker-compose:

```bash
# Start BulwarkAuth and MongoDB
docker-compose up -d

# Run integration tests
go test ./test/integration/...

# Stop services
docker-compose down
```

### Manual Setup

If you prefer to run services manually:

```bash
# Terminal 1: Start MongoDB
docker run -d -p 27017:27017 mongo:7.0

# Terminal 2: Start BulwarkAuth (from project root)
go run ./cmd/bulwarkauth

# Terminal 3: Run tests
go test ./test/integration/...
```

## Running the Tests

### Run all integration tests
```bash
go test ./test/integration/...
```

### Run with verbose output
```bash
go test -v ./test/integration/...
```

### Run specific test
```bash
go test ./test/integration -run TestAccountCreate
go test ./test/integration -run TestAuthenticatePasswordFlow
go test ./test/integration -run TestMagicCode
```

### Run with coverage
```bash
go test -cover ./test/integration/...
```

## Test Coverage

### Account Management
- `TestAccountCreate` - Basic account creation
- `TestAccountCreateDuplicate` - Duplicate prevention
- `TestAccountCreateAndVerify` - Complete account lifecycle (create → verify → auth → change password)

### Password Authentication
- `TestAuthenticatePasswordFlow` - Full password auth flow (authenticate → acknowledge → validate → renew → revoke)
- `TestMultiDeviceAuthentication` - Multiple device sessions with independent tokens
- `TestTokenRenewal` - Multiple token renewal cycles

### Magic Link Authentication
- `TestAuthenticateMagicCode` - Passwordless auth with 6-digit codes
- `TestAuthenticateMagicCodeFail` - Error handling for non-existent accounts

## How It Works

The tests:

1. **Use the bulwark-auth-guard client** - Same library your users will use
2. **Connect directly to MongoDB** - Retrieve verification codes and tokens (simulates email access)
3. **Test real flows** - Complete user journeys from signup to authentication
4. **Verify client compatibility** - Ensures API changes don't break the client

### Example Flow

```go
// Create account
email := generateTestEmail()
password := "TestPassword123!"
err := client.Account.Create(ctx, email, password)

// Get verification token from database (simulates clicking email link)
token, err := getVerificationToken(ctx, email)

// Verify account
err = client.Account.Verify(ctx, email, token)

// Authenticate
auth, err := client.Authenticate.Password(ctx, email, clientID, password)

// Use tokens
claims, err := client.Authenticate.ValidateAccessToken(ctx, auth.AccessToken)
```

## Helper Functions

The test file provides helper functions to simulate email access:

- `getVerificationToken(ctx, email)` - Retrieves email verification token from database
- `getMagicCode(ctx, email)` - Retrieves 6-digit magic code from database
- `getForgotPasswordToken(ctx, email)` - Retrieves forgot password token from database
- `createAndVerifyAccount(ctx, t)` - Creates and verifies an account (common setup)
- `generateTestEmail()` - Generates unique test email addresses
- `generateClientID()` - Generates unique client/device IDs

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration-test:
    runs-on: ubuntu-latest
    
    services:
      mongodb:
        image: mongo:7.0
        ports:
          - 27017:27017
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      
      - name: Start BulwarkAuth
        run: |
          go build -o bulwarkauth ./cmd/bulwarkauth
          ./bulwarkauth &
          sleep 5  # Wait for server to start
      
      - name: Run integration tests
        run: go test -v ./test/integration/...
```

## Configuration

Tests use these constants (defined in `integration_test.go`):

```go
const (
    baseURI = "http://localhost:8080"  // BulwarkAuth API endpoint
    dbURI   = "mongodb://localhost:27017"  // MongoDB connection
)
```

To test against different endpoints, modify these constants.

## Troubleshooting

### Connection refused
- Ensure BulwarkAuth is running on port 8080
- Check server logs for startup errors
- Verify `.env` file has correct configuration

### MongoDB connection error
- Ensure MongoDB is running on port 27017
- Check MongoDB is accepting connections: `mongosh mongodb://localhost:27017`

### Verification token not found
- Add a small delay after account creation: `time.Sleep(100 * time.Millisecond)`
- Check MongoDB `accounts` collection for the account
- Ensure email service is configured (even in test mode)

### Tests are slow
- Reduce timeouts if not needed
- Run specific tests instead of full suite
- Check for network issues or slow MongoDB responses

## Comparison with Unit Tests

| Aspect | Unit Tests | Integration Tests |
|--------|-----------|-------------------|
| **Scope** | Individual functions/methods | Complete user flows |
| **Speed** | Very fast (<1ms) | Slower (~100ms-1s) |
| **Dependencies** | Mocked/stubbed | Real services |
| **Purpose** | Test internal logic | Test API compatibility |
| **When to use** | During development | Before releases, CI/CD |

## Best Practices

1. **Use unique emails** - Always call `generateTestEmail()` for isolation
2. **Clean state** - Each test creates its own accounts
3. **Add delays** - Small sleeps after async operations (email, code generation)
4. **Test real flows** - Complete user journeys, not just individual endpoints
5. **Check client behavior** - Verify the client works as users expect
6. **Document assumptions** - Add comments explaining expected API behavior

## Benefits

✅ **Catches breaking changes** - Know immediately if API changes break the client  
✅ **Real-world validation** - Tests actual usage patterns  
✅ **Client verification** - Ensures the client library works correctly  
✅ **Refactoring confidence** - Safe to refactor knowing tests will catch issues  
✅ **Documentation** - Tests serve as usage examples  

## Related Links

- [BulwarkAuth Documentation](../../README.md)
- [bulwark-auth-guard Client](https://github.com/latebit-io/bulwark-auth-guard)
- [bulwark-auth-guard Tests](https://github.com/latebit-io/bulwark-auth-guard/tree/main) - Inspiration for these tests
