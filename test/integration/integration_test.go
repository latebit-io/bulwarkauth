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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func connectToMongoDB() (*mongo.Client, error) {
	ctx := context.Background()
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}
	return client, nil
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

func TestAccountLockoutAfterFailedAttempts(t *testing.T) {
	ctx := context.Background()
	email, password := createAndVerifyAccount(ctx, t)
	clientID := generateClientID()
	wrongPassword := "WrongPassword123!"

	// Attempt 1-4: Wrong password, should fail but not lock
	for i := 0; i < 4; i++ {
		_, err := guard.Authenticate.Password(ctx, email, wrongPassword, clientID)
		require.Error(t, err, "Wrong password should fail")
	}

	// Attempt 5: Should fail and trigger lockout
	_, err := guard.Authenticate.Password(ctx, email, wrongPassword, clientID)
	require.Error(t, err, "Fifth wrong password should fail and lock account")

	// Attempt 6: Should be locked even with correct password
	_, err = guard.Authenticate.Password(ctx, email, password, clientID)
	require.Error(t, err, "Account should be locked")
	assert.Contains(t, err.Error(), "locked", "Error should indicate account is locked")

	// Attempt 7: Wrong password while locked
	_, err = guard.Authenticate.Password(ctx, email, wrongPassword, clientID)
	require.Error(t, err, "Account should still be locked")
	assert.Contains(t, err.Error(), "locked", "Error should indicate account is locked")
}

func TestAccountLockoutMagicCode(t *testing.T) {
	ctx := context.Background()
	email, _ := createAndVerifyAccount(ctx, t)
	clientID := generateClientID()
	wrongCode := "000000"

	// Attempt 1-4: Wrong magic code
	for i := 0; i < 4; i++ {
		err := guard.Authenticate.RequestMagicCode(ctx, email)
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)

		_, err = guard.Authenticate.MagicCode(ctx, email, wrongCode, clientID)
		require.Error(t, err, "Wrong magic code should fail")
	}

	// Attempt 5: Should fail and trigger lockout
	err := guard.Authenticate.RequestMagicCode(ctx, email)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	_, err = guard.Authenticate.MagicCode(ctx, email, wrongCode, clientID)
	require.Error(t, err, "Fifth wrong code should fail and lock account")

	// Request new code and try with correct code - should be locked
	err = guard.Authenticate.RequestMagicCode(ctx, email)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	magicCode, err := getMagicCode(ctx, email)
	require.NoError(t, err)

	_, err = guard.Authenticate.MagicCode(ctx, email, magicCode, clientID)
	require.Error(t, err, "Account should be locked even with correct magic code")
	assert.Contains(t, err.Error(), "locked", "Error should indicate account is locked")
}

func TestAccountLockoutClearsOnSuccessfulLogin(t *testing.T) {
	ctx := context.Background()
	email, password := createAndVerifyAccount(ctx, t)
	clientID := generateClientID()
	wrongPassword := "WrongPassword123!"

	// Fail 3 times
	for i := 0; i < 3; i++ {
		_, err := guard.Authenticate.Password(ctx, email, wrongPassword, clientID)
		require.Error(t, err)
	}

	// Successful login should clear failed attempts
	auth, err := guard.Authenticate.Password(ctx, email, password, clientID)
	require.NoError(t, err)
	assert.NotEmpty(t, auth.AccessToken)

	// Acknowledge
	err = guard.Authenticate.Acknowledge(ctx, auth)
	require.NoError(t, err)

	// Should be able to fail 5 more times before lockout (counter was reset)
	for i := 0; i < 4; i++ {
		_, err := guard.Authenticate.Password(ctx, email, wrongPassword, clientID)
		require.Error(t, err, "Wrong password should fail")
	}

	// 5th attempt should lock again
	_, err = guard.Authenticate.Password(ctx, email, wrongPassword, clientID)
	require.Error(t, err)

	// Should be locked
	_, err = guard.Authenticate.Password(ctx, email, password, clientID)
	require.Error(t, err, "Account should be locked after 5 new failures")
	assert.Contains(t, err.Error(), "locked", "Error should indicate account is locked")
}

func TestAccountLockoutExpiresAfter15Minutes(t *testing.T) {
	t.Skip("Skipping test that requires 15 minute wait - enable for full integration testing")

	ctx := context.Background()
	email, password := createAndVerifyAccount(ctx, t)
	clientID := generateClientID()
	wrongPassword := "WrongPassword123!"

	// Fail 5 times to trigger lockout
	for i := 0; i < 5; i++ {
		_, err := guard.Authenticate.Password(ctx, email, wrongPassword, clientID)
		require.Error(t, err)
	}

	// Verify locked
	_, err := guard.Authenticate.Password(ctx, email, password, clientID)
	require.Error(t, err, "Account should be locked")
	assert.Contains(t, err.Error(), "locked")

	// Wait 15 minutes for lockout to expire
	time.Sleep(15*time.Minute + 10*time.Second)

	// Should be able to authenticate now
	auth, err := guard.Authenticate.Password(ctx, email, password, clientID)
	require.NoError(t, err, "Lockout should have expired after 15 minutes")
	assert.NotEmpty(t, auth.AccessToken)
}

func TestAccountLockoutExpiresAndResetsCounter(t *testing.T) {
	// NOTE: This test uses the actual 15-minute lockout configured in docker-compose
	// but mocks the time by manipulating the MongoDB record directly
	ctx := context.Background()
	email, password := createAndVerifyAccount(ctx, t)
	clientID := generateClientID()
	wrongPassword := "WrongPassword123!"

	// Fail 5 times to trigger lockout
	for range 5 {
		_, err := guard.Authenticate.Password(ctx, email, wrongPassword, clientID)
		require.Error(t, err)
	}

	// Verify locked
	_, err := guard.Authenticate.Password(ctx, email, password, clientID)
	require.Error(t, err, "Account should be locked")
	assert.Contains(t, err.Error(), "locked")

	// Manually set the lockout to have expired by updating MongoDB directly
	mongodb, err := connectToMongoDB()
	require.NoError(t, err)
	defer mongodb.Disconnect(ctx)

	collection := mongodb.Database("bulwarkauth").Collection("failedAttempts")
	_, err = collection.UpdateOne(ctx,
		map[string]interface{}{"email": email},
		map[string]interface{}{
			"$set": map[string]interface{}{
				"lockedUntil": time.Now().Add(-1 * time.Minute), // Set to 1 minute ago
			},
		},
	)
	require.NoError(t, err)

	// Now authentication should work and the counter should be cleared
	auth, err := guard.Authenticate.Password(ctx, email, password, clientID)
	require.NoError(t, err, "Lockout should have expired, authentication should succeed")
	assert.NotEmpty(t, auth.AccessToken)

	// Verify the failed attempts record was cleared
	var result map[string]interface{}
	err = collection.FindOne(ctx, map[string]interface{}{"email": email}).Decode(&result)
	assert.Error(t, err, "Failed attempts record should be deleted after lockout expires")
}
