# bulwarkauth

Bulwark.Auth is an api based developer focused JWT authentication/authorization subsystem for any infrastructure.

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
- Asymmetric key signing on JWT tokens can use: 
  - RS256, 
  - TODO: RS384
  - TODO: RS512 
- Plug and play key generation and rotation for jwt signing. 
- Deep token validation on the server side checks for revocation, expiration, and more
- TODO: Client side token validation can be used to reduce round trips to the server
- Configurable refresh token and access token expiration
- bulwarkauth does not need to be deployed on internal networks, it can be public facing
- Easy to use email templating using go html/template
- Supports smtp configuration
- Sends out emails for account verification, forgot passwords, and magic links
- Supports passwordless authentication via magic links
- Supports password authentication
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
| MAGIC_LINK_URL               | The url of your application that will submit the magic code call                          | string | https://localhost:3000/magic-link     | Yes       |
| MAGIC_CODE_EXPIRE_IN_MINUTES | The number of minutes the magic code will be valid for                                    | int    | 10                                    | Yes       |
| EMAIL_SMTP                   | Whether or not to use smtp for sending emails                                             | bool   | true                                  | Yes       |
| EMAIL_SMTP_HOST              | The smtp host to use for sending emails                                                   | string | localhost                             | Yes       |
| EMAIL_SMTP_PORT              | The smtp port to use for sending emails                                                   | int    | 1025                                  | Yes       |
| EMAIL_SMTP_USER              | The smtp user to use for sending emails                                                   | string | user                                  | Yes       |
| EMAIL_SMTP_PASS              | The smtp pass to use for sending emails                                                   | string | pass                                  | Yes       |
| EMAIL_SMTP_SECURE            | Whether or not to use secure smtp for sending emails                                      | bool   | false                                 | Yes       |
| EMAIL_TEMPLATE_DIR           | The directory where the email templates are located                                       | string | src/bulwark-auth/email-templates      | Yes       |
| EMAIL_SEND_ADDRESS           | The email address to send emails from                                                     | string | admin@latebit.io                      | Yes       |
| GOOGLE_CLIENT_ID             | The google client id to use for google authentication                                     | string | secret.apps.googleusercontent.com     | No        |                                                                        |           |
| SERVICE_MODE                 | The service mode to run in only used for CI and tests                                     | string | test                                  | No        |
 
## Domain 
For domain verification you will need access to your DNS provider to add an TXT entry to verify against
This key will need to verified before using this feature until then it will be ignored

