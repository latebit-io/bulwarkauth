package social

import (
	"context"
	"os"
	"testing"

	"github.com/latebit-io/bulwarkauth/internal/accounts"
	"github.com/latebit-io/bulwarkauth/internal/encryption"
	"github.com/latebit-io/bulwarkauth/internal/tokens"
	"github.com/latebit-io/bulwarkauth/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestGoogleValidator_WithRealToken is an integration test that validates a real Google ID token.
// This test is skipped unless you provide credentials via environment variables.
//
// To run this test:
//  1. Go to https://developers.google.com/oauthplayground/
//  2. Select "Google OAuth2 API v2" -> "https://www.googleapis.com/auth/userinfo.email"
//  3. Click "Authorize APIs" and sign in with your Google account
//  4. Click "Exchange authorization code for tokens"
//  5. Copy the "id_token" value
//  6. Set environment variables and run:
//     export GOOGLE_TEST_CLIENT_ID="your-client-id.apps.googleusercontent.com"
//     export GOOGLE_TEST_ID_TOKEN="your-id-token-from-playground"
//     go test -v -run TestGoogleValidator_WithRealToken ./internal/authentication/social
//
// Note: ID tokens expire quickly (usually within an hour), so you'll need to generate
// a fresh token each time you run this test.
func TestGoogleValidator_WithRealToken(t *testing.T) {
	clientID := os.Getenv("GOOGLE_TEST_CLIENT_ID")
	idToken := os.Getenv("GOOGLE_TEST_ID_TOKEN")

	if clientID == "" || idToken == "" {
		t.Skip("Skipping integration test: set GOOGLE_TEST_CLIENT_ID and GOOGLE_TEST_ID_TOKEN to run")
	}

	// Create real Google validator
	validator, err := NewGoogleValidator(clientID)
	if err != nil {
		t.Fatalf("Failed to create Google validator: %v", err)
	}

	// Validate the real token
	social, err := validator.ValidateToken(context.Background(), idToken)

	// Assertions
	assert.NoError(t, err, "Token validation should succeed with valid Google ID token")
	assert.NotNil(t, social, "Social object should not be nil")
	assert.Equal(t, "google", social.Provider)
	assert.NotEmpty(t, social.Email, "Email should be present in token claims")
	assert.NotEmpty(t, social.ID, "Google user ID (sub) should be present in token claims")

	t.Logf("Successfully validated Google ID token for email: %s, Google ID: %s", social.Email, social.ID)
}

// TestGoogleValidator_WithRealToken_FullFlow is an integration test that validates
// the complete authentication flow with a real Google ID token and real services.
//
// To run this test, follow the same steps as TestGoogleValidator_WithRealToken above.
func TestGoogleValidator_WithRealToken_FullFlow(t *testing.T) {
	clientID := os.Getenv("GOOGLE_TEST_CLIENT_ID")
	idToken := os.Getenv("GOOGLE_TEST_ID_TOKEN")

	if clientID == "" || idToken == "" {
		t.Skip("Skipping integration test: set GOOGLE_TEST_CLIENT_ID and GOOGLE_TEST_ID_TOKEN to run")
	}

	// Setup MongoDB
	mongodb := utils.NewMongoTestUtil()
	mongoServer, err := mongodb.CreateServer()
	if err != nil {
		t.Fatal(err)
	}
	defer mongoServer.Stop()

	clientOptions := options.Client().ApplyURI(mongoServer.URI())
	mongoClient, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := mongoClient.Disconnect(context.TODO())
		if err != nil {
			t.Fatal(err)
		}
	}()

	db := mongoClient.Database("bulwark-test")
	mongodbTxManager := utils.NewMongoTxManager(mongoClient)
	encrypt := encryption.NewDefaultEncryption()
	accountRepo := accounts.NewMongodbAccountRepository(db, encrypt)
	forgotRepo := accounts.NewMongoDbForgotRepository(db)
	signingRepo := tokens.NewDefaultSigningKeyRepository(db)
	signingService := tokens.NewDefaultSigningKeyService(signingRepo)
	err = signingService.Initialize(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	tokenizer := tokens.NewDefaultTokenizer("test", "test", "test", 3600, 9600, signingService)

	mockEmailService := &accounts.MockEmailService{}
	mockEmailService.On("SendVerificationEmail", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	accountService := accounts.NewDefaultAccountService(accountRepo, forgotRepo, tokenizer, mockEmailService, mongodbTxManager)

	// Create real Google validator
	googleValidator, err := NewGoogleValidator(clientID)
	if err != nil {
		t.Fatalf("Failed to create Google validator: %v", err)
	}

	// First, validate the token to get the email
	social, err := googleValidator.ValidateToken(context.Background(), idToken)
	if err != nil {
		t.Fatalf("Failed to validate Google token: %v", err)
	}

	t.Logf("Validated token for email: %s", social.Email)

	// Create and verify account with the email from the token
	err = accountService.Create(context.Background(), social.Email, "test-password")
	if err != nil {
		t.Fatal(err)
	}

	account, err := accountRepo.Read(context.Background(), social.Email)
	if err != nil {
		t.Fatal(err)
	}

	err = accountService.Verify(context.Background(), social.Email, account.VerificationToken)
	if err != nil {
		t.Fatal(err)
	}

	// Setup social service with real Google validator
	socialService := NewDefaultSocialService(accountRepo, accountService, encrypt, tokenizer)
	socialService.AddValidator(googleValidator)

	// Authenticate with the real Google ID token
	authenticated, err := socialService.Authenticate(context.Background(), idToken, "google")

	// Assertions
	assert.NoError(t, err, "Authentication should succeed")
	assert.NotNil(t, authenticated, "Authenticated object should not be nil")
	assert.NotEmpty(t, authenticated.AccessToken, "Access token should be generated")
	assert.NotEmpty(t, authenticated.RefreshToken, "Refresh token should be generated")

	// Verify social provider was linked
	updatedAccount, err := accountRepo.Read(context.Background(), social.Email)
	assert.NoError(t, err)
	assert.Len(t, updatedAccount.SocialProviders, 1, "Should have one social provider linked")
	assert.Equal(t, "google", updatedAccount.SocialProviders[0].Name)
	assert.Equal(t, social.ID, updatedAccount.SocialProviders[0].SocialId)

	t.Logf("Successfully authenticated and linked Google account for: %s", social.Email)
}
