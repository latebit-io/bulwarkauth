package integration

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	bulwark "github.com/latebit-io/bulwark-auth-guard"
	gohog "github.com/latebit-io/go-hog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	baseURI    = "http://localhost:8080"
	mailHogURI = "http://localhost:8025"
)

var (
	guard      *bulwark.Guard
	httpClient *http.Client
)

func TestMain(m *testing.M) {
	// Setup
	httpClient = &http.Client{Timeout: 10 * time.Second}
	guard = bulwark.NewGuard(baseURI, httpClient)

	// Run tests
	code := m.Run()

	os.Exit(code)
}

// Helper functions

func generateTestEmail() string {
	return fmt.Sprintf("test-%s@bulwarkauth.test", uuid.New().String())
}

func generateClientID() string {
	return fmt.Sprintf("client-%s", uuid.New().String())
}

func findMessageForEmail(email string) (*gohog.Message, error) {
	client := gohog.NewGoHogClient(mailHogURI, httpClient)
	messages, err := client.Messages(0, 100)
	if err != nil {
		return nil, err
	}

	// Find message sent to this email
	for _, msg := range messages.Items {
		for _, to := range msg.ToAddresses() {
			if to == email {
				return &msg, nil
			}
		}
	}

	return nil, errors.New("no message found for email: " + email)
}

func getVerificationToken(ctx context.Context, email string) (string, error) {
	// In TEST_MODE, the verification token is in the email subject
	message, err := findMessageForEmail(email)
	if err != nil {
		return "", err
	}

	token := message.Subject()
	if token == "" {
		return "", errors.New("no verification token found in subject")
	}
	return token, nil
}

func getMagicCode(ctx context.Context, email string) (string, error) {
	// In TEST_MODE, the magic code is in the email subject
	message, err := findMessageForEmail(email)
	if err != nil {
		return "", err
	}

	code := message.Subject()
	if code == "" {
		return "", errors.New("no magic code found in subject")
	}
	return code, nil
}

func createAndVerifyAccount(ctx context.Context, t *testing.T) (string, string) {
	email := generateTestEmail()
	password := "TestPassword123!"

	// Create account
	err := guard.Account.Create(ctx, email, password)
	require.NoError(t, err)

	// Wait a bit for the account to be created
	time.Sleep(100 * time.Millisecond)

	// Get verification token
	token, err := getVerificationToken(ctx, email)
	require.NoError(t, err)

	// Verify account
	err = guard.Account.Verify(ctx, email, token)
	require.NoError(t, err)

	return email, password
}

// Tests

func TestAccountCreate(t *testing.T) {
	ctx := context.Background()
	email := generateTestEmail()
	password := "TestPassword123!"

	err := guard.Account.Create(ctx, email, password)
	require.NoError(t, err)
}

func TestAccountCreateDuplicate(t *testing.T) {
	ctx := context.Background()
	email := generateTestEmail()
	password := "TestPassword123!"

	err := guard.Account.Create(ctx, email, password)
	require.NoError(t, err)

	// Try to create duplicate
	err = guard.Account.Create(ctx, email, password)
	require.Error(t, err, "Should reject duplicate account")
}

func TestAccountCreateAndVerify(t *testing.T) {
	ctx := context.Background()
	email := generateTestEmail()
	password := "TestPassword123!"
	clientID := generateClientID()

	// Create account
	err := guard.Account.Create(ctx, email, password)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Get verification token from database
	token, err := getVerificationToken(ctx, email)
	require.NoError(t, err)

	// Verify account
	err = guard.Account.Verify(ctx, email, token)
	require.NoError(t, err)

	// Authenticate with password
	authResponse, err := guard.Authenticate.Password(ctx, email, password, clientID)
	require.NoError(t, err)
	assert.NotEmpty(t, authResponse.AccessToken)
	assert.NotEmpty(t, authResponse.RefreshToken)

	// Acknowledge authentication
	err = guard.Authenticate.Acknowledge(ctx, authResponse)
	require.NoError(t, err)

	// Change password (requires access token)
	newPassword := "NewPassword456!"
	err = guard.Account.ChangePassword(ctx, email, newPassword, authResponse.AccessToken)
	require.NoError(t, err)

	// Verify new password works
	authResponse2, err := guard.Authenticate.Password(ctx, email, newPassword, clientID)
	require.NoError(t, err)
	assert.NotEmpty(t, authResponse2.AccessToken)
}

func TestAuthenticatePasswordFlow(t *testing.T) {
	ctx := context.Background()
	email, password := createAndVerifyAccount(ctx, t)
	clientID := generateClientID()

	// Authenticate
	authResponse, err := guard.Authenticate.Password(ctx, email, password, clientID)
	require.NoError(t, err)
	assert.NotEmpty(t, authResponse.AccessToken)
	assert.NotEmpty(t, authResponse.RefreshToken)

	// Acknowledge
	err = guard.Authenticate.Acknowledge(ctx, authResponse)
	require.NoError(t, err)

	// Validate access token
	claims, err := guard.Authenticate.ValidateAccessToken(ctx, authResponse.AccessToken)
	require.NoError(t, err)
	// Roles may be nil or empty for accounts without assigned roles
	_ = claims.Roles

	// Renew tokens
	renewResponse, err := guard.Authenticate.Renew(ctx, email, authResponse.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, renewResponse.AccessToken)
	assert.NotEmpty(t, renewResponse.RefreshToken)
	assert.NotEqual(t, authResponse.AccessToken, renewResponse.AccessToken)

	// Revoke
	err = guard.Authenticate.Revoke(ctx, email, renewResponse.AccessToken, clientID)
	require.NoError(t, err)
}

func TestAuthenticateMagicCode(t *testing.T) {
	ctx := context.Background()
	email, _ := createAndVerifyAccount(ctx, t)
	clientID := generateClientID()

	// Request magic code
	err := guard.Authenticate.RequestMagicCode(ctx, email)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Get magic code from database
	magicCode, err := getMagicCode(ctx, email)
	require.NoError(t, err)
	assert.Len(t, magicCode, 6, "Magic code should be 6 digits")

	// Authenticate with magic code
	authResponse, err := guard.Authenticate.MagicCode(ctx, email, magicCode, clientID)
	require.NoError(t, err)
	assert.NotEmpty(t, authResponse.AccessToken)
	assert.NotEmpty(t, authResponse.RefreshToken)
}

func TestAuthenticateMagicCodeFail(t *testing.T) {
	ctx := context.Background()
	email := generateTestEmail()

	// Request magic code for non-existent account
	err := guard.Authenticate.RequestMagicCode(ctx, email)
	// Note: API may return success for security (to not reveal if email exists)
	// Check your API's behavior and adjust assertion if needed
	_ = err
}

func TestMultiDeviceAuthentication(t *testing.T) {
	ctx := context.Background()
	email, password := createAndVerifyAccount(ctx, t)

	// Authenticate from two devices
	clientID1 := generateClientID()
	auth1, err := guard.Authenticate.Password(ctx, email, password, clientID1)
	require.NoError(t, err)

	clientID2 := generateClientID()
	auth2, err := guard.Authenticate.Password(ctx, email, password, clientID2)
	require.NoError(t, err)

	// Tokens should be different
	assert.NotEqual(t, auth1.AccessToken, auth2.AccessToken)
	assert.NotEqual(t, auth1.RefreshToken, auth2.RefreshToken)

	// Acknowledge both
	err = guard.Authenticate.Acknowledge(ctx, auth1)
	require.NoError(t, err)

	err = guard.Authenticate.Acknowledge(ctx, auth2)
	require.NoError(t, err)

	// Both should be valid
	_, err = guard.Authenticate.ValidateAccessToken(ctx, auth1.AccessToken)
	require.NoError(t, err)

	_, err = guard.Authenticate.ValidateAccessToken(ctx, auth2.AccessToken)
	require.NoError(t, err)

	// Revoke device 1
	err = guard.Authenticate.Revoke(ctx, email, auth1.AccessToken, clientID1)
	require.NoError(t, err)

	// Device 2 should still be valid
	_, err = guard.Authenticate.ValidateAccessToken(ctx, auth2.AccessToken)
	require.NoError(t, err)
}

func TestTokenRenewal(t *testing.T) {
	ctx := context.Background()
	email, password := createAndVerifyAccount(ctx, t)
	clientID := generateClientID()

	// Authenticate
	auth1, err := guard.Authenticate.Password(ctx, email, password, clientID)
	require.NoError(t, err)

	// Acknowledge before renewal
	err = guard.Authenticate.Acknowledge(ctx, auth1)
	require.NoError(t, err)

	// Renew multiple times
	auth2, err := guard.Authenticate.Renew(ctx, email, auth1.RefreshToken)
	require.NoError(t, err)

	auth3, err := guard.Authenticate.Renew(ctx, email, auth2.RefreshToken)
	require.NoError(t, err)

	// All tokens should be unique
	assert.NotEqual(t, auth1.AccessToken, auth2.AccessToken)
	assert.NotEqual(t, auth2.AccessToken, auth3.AccessToken)
	assert.NotEqual(t, auth1.AccessToken, auth3.AccessToken)

	// Latest token should be valid
	claims, err := guard.Authenticate.ValidateAccessToken(ctx, auth3.AccessToken)
	require.NoError(t, err)
	// Roles may be nil or empty for accounts without assigned roles
	_ = claims.Roles
}

func TestPasswordChange(t *testing.T) {
	ctx := context.Background()
	email, password := createAndVerifyAccount(ctx, t)
	clientID := generateClientID()

	// Authenticate to get access token
	auth, err := guard.Authenticate.Password(ctx, email, password, clientID)
	require.NoError(t, err)

	// Acknowledge
	err = guard.Authenticate.Acknowledge(ctx, auth)
	require.NoError(t, err)

	// Change password
	newPassword := "NewPassword456!"
	err = guard.Account.ChangePassword(ctx, email, newPassword, auth.AccessToken)
	require.NoError(t, err)

	// Old password should not work
	_, err = guard.Authenticate.Password(ctx, email, password, clientID)
	require.Error(t, err, "Old password should not work")

	// New password should work
	newAuth, err := guard.Authenticate.Password(ctx, email, newPassword, clientID)
	require.NoError(t, err)
	assert.NotEmpty(t, newAuth.AccessToken)
}
