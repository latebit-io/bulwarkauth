# Social Authentication Tests

This directory contains tests for social authentication providers (Google, etc.).

## Unit Tests

The standard tests use mocks and can be run without any configuration:

```bash
go test ./internal/authentication/social
```

## Integration Tests with Real Google ID Tokens

Two integration tests allow you to validate against real Google ID tokens:

- `TestGoogleValidator_WithRealToken` - Tests token validation only
- `TestGoogleValidator_WithRealToken_FullFlow` - Tests the complete authentication flow

### How to Run Integration Tests

#### Step 1: Get Your Google Client ID

You'll need a Google OAuth 2.0 Client ID. If you don't have one:

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a project or select an existing one
3. Go to "APIs & Services" > "Credentials"
4. Create an "OAuth 2.0 Client ID" (Web application type)
5. Copy the Client ID (it looks like: `xxxxx.apps.googleusercontent.com`)

#### Step 2: Generate a Test ID Token from OAuth Playground

1. Go to [Google OAuth 2.0 Playground](https://developers.google.com/oauthplayground/)
2. Click the gear icon (⚙️) in the top right
3. Check "Use your own OAuth credentials"
4. Enter your OAuth Client ID and Client Secret
5. Close the configuration dialog
6. In the left panel, select **"Google OAuth2 API v2"**
7. Check the scope: `https://www.googleapis.com/auth/userinfo.email`
8. Click **"Authorize APIs"**
9. Sign in with your Google account and grant permissions
10. Click **"Exchange authorization code for tokens"**
11. Copy the **"id_token"** value (not the access_token)

**Important**: ID tokens expire within 1 hour, so you'll need to generate a fresh token for each test run.

#### Step 3: Run the Tests

Set the environment variables and run the tests:

```bash
export GOOGLE_TEST_CLIENT_ID="your-client-id.apps.googleusercontent.com"
export GOOGLE_TEST_ID_TOKEN="eyJhbGciOiJSUzI1NiIsImtpZCI6..."  # your ID token from OAuth playground

# Run just the integration tests
go test -v -run TestGoogleValidator_WithRealToken ./internal/authentication/social
go test -v -run TestGoogleValidator_WithRealToken_FullFlow ./internal/authentication/social
```

### What the Tests Validate

**TestGoogleValidator_WithRealToken**:
- Creates a real Google validator
- Validates the ID token against Google's OIDC endpoint
- Extracts email and user ID from the token claims

**TestGoogleValidator_WithRealToken_FullFlow**:
- Everything from the first test, plus:
- Creates a MongoDB test instance
- Creates and verifies a user account
- Authenticates using the Google ID token
- Links the Google account to the user
- Generates access and refresh tokens

### Example Output

```
=== RUN   TestGoogleValidator_WithRealToken
    social-validator_test.go:469: Successfully validated Google ID token for email: user@gmail.com, Google ID: 1234567890
--- PASS: TestGoogleValidator_WithRealToken (0.52s)
=== RUN   TestGoogleValidator_WithRealToken_FullFlow
    social-validator_test.go:534: Validated token for email: user@gmail.com
    social-validator_test.go:572: Successfully authenticated and linked Google account for: user@gmail.com
--- PASS: TestGoogleValidator_WithRealToken_FullFlow (2.15s)
```

## Troubleshooting

**"Token used too early"**: Your system clock may be off. Sync your time.

**"Token expired"**: ID tokens expire quickly. Generate a fresh token from OAuth Playground.

**"Invalid audience"**: Make sure you're using the same Client ID that was used to generate the token.

**"Failed to create OIDC provider"**: Check your internet connection. The validator needs to fetch Google's public keys.