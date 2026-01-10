# Integration Tests - Quick Start

## What You Have

A complete integration test suite that validates BulwarkAuth using the [bulwark-auth-guard](https://github.com/latebit-io/bulwark-auth-guard) client library.

**13 test cases covering:**
- Account creation and verification
- Password authentication flows
- Magic link (passwordless) authentication
- Multi-device sessions
- Token renewal and revocation
- Password changes
- Account lockout protection
- Multi-tenant support

## Running the Tests

### Option 1: Using Docker Compose (Recommended)

```bash
# Start BulwarkAuth and MongoDB
docker-compose up -d

# Run all integration tests
go test -v ./test/integration/...

# Run a specific test
go test -v ./test/integration -run TestAccountCreate

# Stop services
docker-compose down
```

### Option 2: Manual Setup

```bash
# Terminal 1: Start MongoDB
docker run -d -p 27017:27017 mongo:7.0

# Terminal 2: Start BulwarkAuth
go run ./cmd/bulwarkauth

# Terminal 3: Run tests
go test -v ./test/integration/...
```

## Available Tests

| Test | What It Does |
|------|--------------|
| `TestAccountCreate` | Creates a new account |
| `TestAccountCreateDuplicate` | Verifies duplicate prevention |
| `TestAccountCreateAndVerify` | Full flow: create → verify → auth → change password |
| `TestAuthenticatePasswordFlow` | Complete auth cycle: login → acknowledge → validate → renew → revoke |
| `TestAuthenticateMagicCode` | Passwordless authentication with 6-digit codes |
| `TestAuthenticateMagicCodeFail` | Error handling for invalid requests |
| `TestMultiDeviceAuthentication` | Multiple device sessions with independent tokens |
| `TestTokenRenewal` | Multiple token renewal cycles |
| `TestPasswordChange` | Password update with access token |
| `TestAccountLockoutAfterFailedAttempts` | Account locks after 5 failed password attempts |
| `TestAccountLockoutMagicCode` | Account locks after 5 failed magic code attempts |
| `TestAccountLockoutClearsOnSuccessfulLogin` | Failed attempts counter resets on successful login |
| `TestAccountLockoutExpiresAndResetsCounter` | Lockout expires and counter resets after duration |

## Why This Matters

✅ **Real-world validation** - Uses the actual client library your users will use  
✅ **Breaking change detection** - Catches API changes that break the client  
✅ **Refactoring safety** - Confidently refactor knowing tests will catch issues  
✅ **Client compatibility** - Ensures server and client stay in sync  

## How It Works

The tests:
1. Use the `bulwark-auth-guard` client (same as end users)
2. Connect to MongoDB to retrieve verification codes (simulates email access)
3. Execute complete user journeys
4. Validate responses match expected behavior

**Example:**
```go
const tenantID = "default"  // Or your specific tenant ID

// Create account
email := generateTestEmail()
err := guard.Account.Create(ctx, tenantID, email, "Password123!")

// Get verification code from database (simulates clicking email link)
token, err := getVerificationToken(ctx, email)

// Verify account
err = guard.Account.Verify(ctx, tenantID, email, token)

// Authenticate
clientID := generateClientID()
auth, err := guard.Authenticate.Password(ctx, tenantID, email, password, clientID)

// Acknowledge tokens
err = guard.Authenticate.Acknowledge(ctx, tenantID, auth)
```

## Configuration

Default endpoints (in `integration_test.go`):
- **BulwarkAuth**: `http://localhost:8080`
- **MailHog**: `http://localhost:8025`
- **Tenant ID**: `"default"`

To change, modify the constants at the top of the file.

## Tips

- Each test creates unique accounts (no cleanup needed)
- Small delays after account creation help ensure database writes complete
- MongoDB queries retrieve verification codes/tokens (simulates email access)
- Tests run independently and can be run in any order

## See Also

- [Full README](./README.md) - Detailed documentation
- [bulwark-auth-guard](https://github.com/latebit-io/bulwark-auth-guard) - Client library
- [bulwark-auth-guard tests](https://github.com/latebit-io/bulwark-auth-guard/tree/main) - Client test examples
