# bulwarkauth

Bulwark.Auth is an api based developer focused JWT authentication/authorization subsystem for any infrastructure with built-in multi-tenant support.

# Releases and contributing guidelines

### Releases
Releases will be rolling as features are developed and tested, then merged into main a release will be cut. 

### Contributions

- Each contribution will need a issue/ticket created to track the feature or bug fix before it will be considered
- The PR must pass all tests and be reviewed by an official maintainer
- Each PR must be linked to an issue/ticket, once the PR is merged the ticket will be auto closed
- Each feature/bugfix needs to have unit tests
- Each feature must have the code documented inline


# Key Features:
- **Multi-tenant support** - Isolate users, accounts, and authentication data by tenant ID
- Asymmetric key signing on JWT tokens can use: 
  - RS256, 
  - TODO: RS384
  - TODO: RS512 
- Plug and play key generation and rotation for jwt signing per tenant
- Deep token validation on the server side checks for revocation, expiration, and more
- TODO: Client side token validation can be used to reduce round trips to the server
- Configurable refresh token and access token expiration
- bulwarkauth does not need to be deployed on internal networks, it can be public facing
- Easy to use email templating using go html/template with tenant-specific customization
- Supports smtp configuration
- Sends out emails for account verification, forgot passwords, and magic links
- Supports passwordless authentication via magic links
- Supports password authentication
- Account lockout protection after failed authentication attempts (configurable)
- TODO: Supports third party authentication via Google (more to come)
- Uses token acknowledgement to prevent replay attacks and supports multiple devices
- TODO: Account management and administration via admin service

# Configuring and Running bulwarkauth (BA)

bulwarkauth (BA) is best run using the official docker container found here:
https://github.com/latebit-io/bulwarkauth/pkgs/container/bulwarkauth

It can also be installed via executable binary for all major architectures found here:
https://github.com/latebit-io/bulwarkauth/releases

For a k8s deployment coming soon. 

## Configuration
Configuration is done via environment variables. The following is a list of all available environment variables, any
confidential values should use proper secrets management.

| Name                         | Description                                                                               | Value  | Example                               | Mandatory |
|------------------------------|-------------------------------------------------------------------------------------------|--------|---------------------------------------|-----------|
| DB_CONNECTION                | The connection string to the mongo database                                               | string | mongodb://localhost:27017             | Yes       |
| DB_NAME_SEED                 | will append the seed onto the db name, this is needed if running many different instances | string | BulwarkAuth-{seed}                    | No        |
| DOMAIN                       | The domain name the service will be used for                                              | string | latebit.io                            | Yes       |
| WEBSITE_NAME                 | The name of the website the service will be used for                                      | string | Latebit                               | Yes       |
| VERIFICATION_URL             | The url of your application that will make the token verification call                    | string | https://localhost:3000/verify         | Yes       |
| FORGOT_PASSWORD_URL          | The url of your application that will use the forgot password call                        | string | https://localhost:3000/reset-password | Yes       |
| MAGIC_URL                    | The url of your application that will submit the magic code call                          | string | https://localhost:3000/magic-link     | Yes       |
| MAGIC_CODE_EXPIRE_IN_MINUTES | The number of minutes the magic code will be valid for                                    | int    | 10                                    | Yes       |
| EMAIL_SMTP                   | Whether or not to use smtp for sending emails                                             | bool   | true                                  | Yes       |
| EMAIL_SMTP_HOST              | The smtp host to use for sending emails                                                   | string | localhost                             | Yes       |
| EMAIL_SMTP_PORT              | The smtp port to use for sending emails                                                   | int    | 1025                                  | Yes       |
| EMAIL_SMTP_USER              | The smtp user to use for sending emails                                                   | string | user                                  | Yes       |
| EMAIL_SMTP_PASS              | The smtp pass to use for sending emails                                                   | string | pass                                  | Yes       |
| EMAIL_SMTP_SECURE            | Whether or not to use secure smtp for sending emails                                      | bool   | false                                 | Yes       |
| EMAIL_TEMPLATE_DIRS          | The directory where the email templates are located                                       | string | src/bulwark-auth/email-templates      | No        |
| EMAIL_FROM_ADDRESS           | The email address to send emails from                                                     | string | admin@latebit.io                      | Yes       |
| GOOGLE_CLIENT_ID             | The google client id to use for google authentication                                     | string | secret.apps.googleusercontent.com     | No        |
| LOCKOUT_DURATION_IN_SEC      | Duration in seconds for account lockout after failed attempts                             | int    | 300                                   | No        |
| MAX_FAILED_ATTEMPTS          | Maximum failed authentication attempts before lockout                                     | int    | 5                                     | No        |
| ACCESS_TOKEN_EXPIRE_IN_SECONDS  | Access token expiration time in seconds                                                | int    | 3600                                  | No        |
| REFRESH_TOKEN_EXPIRE_IN_SECONDS | Refresh token expiration time in seconds                                               | int    | 86400                                 | No        |
| TEST_MODE                    | Run in test mode (bypasses email sending)                                                 | bool   | false                                 | No        |
| API_KEY                      | API key for securing endpoints (when API_KEY_ENABLE=true)                                 | string | your-secure-api-key                   | No        |
| API_KEY_ENABLE               | Enable API key authentication                                                             | bool   | false                                 | No        |
| CORS_ENABLED                 | Enable CORS support                                                                       | bool   | false                                 | No        |
| ALLOWED_WEB_ORIGINS          | Comma-separated list of allowed CORS origins                                              | string | http://localhost:3000                 | No        |
 
## Multi-Tenant Support

BulwarkAuth supports multi-tenancy out of the box, allowing you to isolate users, accounts, and authentication data by tenant ID.

### How It Works

- Each API request includes a `tenantId` parameter (or uses `"default"` for single-tenant setups)
- Accounts, tokens, signing keys, and authentication data are isolated per tenant
- Each tenant can have its own:
  - JWT signing keys (automatically generated and rotated)
  - Email templates
  - Account lockout settings
  - Token expiration policies

### Using Multi-Tenancy

When using the [bulwark-auth-guard](https://github.com/latebit-io/bulwark-auth-guard) client library, simply pass the tenant ID to all API calls:

```go
// Create account for a specific tenant
err := guard.Account.Create(ctx, "tenant-123", email, password)

// Authenticate user within their tenant
auth, err := guard.Authenticate.Password(ctx, "tenant-123", email, password, clientID)

// Validate token (automatically scoped to tenant)
claims, err := guard.Authenticate.ValidateAccessToken(ctx, "tenant-123", accessToken)
```

### Single-Tenant Setup

If you're running a single-tenant application, use `"default"` as your tenant ID:

```go
const tenantID = "default"

err := guard.Account.Create(ctx, tenantID, email, password)
```

### Benefits

- **Data Isolation**: Complete separation of user data between tenants
- **Security**: Tokens from one tenant cannot be used in another
- **Scalability**: Run multiple applications/customers on the same BulwarkAuth instance
- **Per-Tenant Configuration**: Customize settings for each tenant via email templates and future admin features

## Domain 
For domain verification you will need access to your DNS provider to add an TXT entry to verify against
This key will need to verified before using this feature until then it will be ignored

