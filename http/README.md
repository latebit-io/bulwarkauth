# HTTP Test Files

This directory contains HTTP request files for testing the BulwarkAuth API.

## Setup

1. Copy `accounts.http.example` to `accounts.http`
2. Replace the placeholders with your actual values:

```
{{API_KEY}} - Your BulwarkAuth API key
{{TENANT_ID}} - Tenant ID (use "default" for single-tenant setups)
{{TEST_EMAIL}} - Test email address (e.g., test@example.com)
{{TEST_PASSWORD}} - Test password
{{CLIENT_ID}} - Your client ID (unique per device/session)
{{ACCESS_TOKEN}} - JWT access token (obtained from authentication)
{{REFRESH_TOKEN}} - JWT refresh token (obtained from authentication)
{{VERIFICATION_TOKEN}} - Email verification token
{{TEST_DOMAIN}} - Domain for verification testing (e.g., example.com)
```

## Multi-Tenant Support

All API endpoints now require a `tenantId` in the request body. For single-tenant applications, use `"default"`:

```json
{
  "tenantId": "default",
  "email": "test@example.com",
  "password": "password123"
}
```

For multi-tenant applications, use your tenant identifier:

```json
{
  "tenantId": "customer-abc-123",
  "email": "test@example.com",
  "password": "password123"
}
```

## Usage

Use your favorite HTTP client (VS Code REST Client, IntelliJ HTTP Client, etc.) to execute the requests.

## Security Note

- Never commit `.http` files with real credentials to version control
- The `.gitignore` file excludes `*.http` files by default
- Use this template approach for all HTTP test files