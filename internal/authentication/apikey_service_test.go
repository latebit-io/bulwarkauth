package authentication

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/latebit-io/bulwarkauth/internal/accounts"
	"github.com/latebit-io/bulwarkauth/internal/encryption"
	"github.com/latebit-io/bulwarkauth/internal/tokens"
	"github.com/latebit-io/bulwarkauth/internal/utils"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type apiKeyTestEnv struct {
	db          *mongo.Database
	service     *DefaultApiKeyService
	tokenizer   tokens.Tokenizer
	accountRepo accounts.AccountRepository
	cleanup     func()
}

func setupApiKeyServiceTest(t *testing.T) *apiKeyTestEnv {
	t.Helper()
	ctx := context.Background()
	mongodb := utils.NewMongoTestUtil()
	mongoServer, err := mongodb.CreateServer()
	if err != nil {
		t.Fatal(err)
	}

	clientOptions := options.Client().ApplyURI(mongoServer.URI())
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		mongoServer.Stop()
		t.Fatal(err)
	}

	db := client.Database("testdb")
	encrypt := encryption.NewDefaultEncryption(12)

	signingRepo := tokens.NewDefaultSigningKeyRepository(db)
	signingService := tokens.NewDefaultSigningKeyService(signingRepo)
	err = signingService.Initialize(ctx)
	if err != nil {
		client.Disconnect(ctx)
		mongoServer.Stop()
		t.Fatal(err)
	}

	tokenizer := tokens.NewDefaultTokenizer("test", "test", "test", 3600, 3600, signingService)
	accountRepo := accounts.NewMongodbAccountRepository(db, encrypt)
	apiKeyRepo := NewMongoDbApiRepository(db)
	service := NewDefaultApiKeyService(apiKeyRepo, encrypt, tokenizer, accountRepo)

	return &apiKeyTestEnv{
		db:          db,
		service:     service,
		tokenizer:   tokenizer,
		accountRepo: accountRepo,
		cleanup: func() {
			client.Disconnect(ctx)
			mongoServer.Stop()
		},
	}
}

func createTestAccount(t *testing.T, env *apiKeyTestEnv, tenantID, email, password string) {
	t.Helper()
	ctx := context.Background()
	err := env.accountRepo.Create(ctx, tenantID, email, password)
	if err != nil {
		t.Fatalf("Failed to create test account: %v", err)
	}
	err = env.accountRepo.Verify(ctx, tenantID, email)
	if err != nil {
		t.Fatalf("Failed to verify test account: %v", err)
	}
}

func createTestAccessToken(t *testing.T, env *apiKeyTestEnv, tenantID, email string) string {
	t.Helper()
	token, err := env.tokenizer.CreateAccessToken(context.Background(), tenantID, email, "test-client", []string{})
	if err != nil {
		t.Fatalf("Failed to create access token: %v", err)
	}
	return token
}

// extractKeyPrefix returns the "bwa_<random>" portion from a full API key "bwa_<random>_<secret>".
func extractKeyPrefix(apiKey string) string {
	parts := strings.Split(apiKey, "_")
	if len(parts) != 3 {
		return ""
	}
	return fmt.Sprintf("%s_%s", parts[0], parts[1])
}

func TestApiKeyService_Create(t *testing.T) {
	env := setupApiKeyServiceTest(t)
	defer env.cleanup()

	tenantID := "tenant1"
	email := "test@example.com"
	createTestAccount(t, env, tenantID, email, "Password123!")

	accessToken := createTestAccessToken(t, env, tenantID, email)

	rawKey, err := env.service.Create(context.Background(), tenantID, accessToken, "my-api-key", nil)
	if err != nil {
		t.Fatalf("Failed to create api key: %v", err)
	}

	parts := strings.Split(rawKey, "_")
	if len(parts) != 3 {
		t.Fatalf("Expected key format bwa_prefix_secret, got '%s'", rawKey)
	}

	if parts[0] != apiKeyPrefix {
		t.Errorf("Expected key to start with '%s', got '%s'", apiKeyPrefix, parts[0])
	}

	if len(parts[1]) != randomLength {
		t.Errorf("Expected prefix random length %d, got %d", randomLength, len(parts[1]))
	}

	if !strings.HasPrefix(rawKey, apiKeyPrefix+"_") {
		t.Errorf("Expected key to start with '%s_', got '%s'", apiKeyPrefix, rawKey)
	}
}

func TestApiKeyService_Create_WithExpiry(t *testing.T) {
	env := setupApiKeyServiceTest(t)
	defer env.cleanup()

	tenantID := "tenant1"
	email := "test@example.com"
	createTestAccount(t, env, tenantID, email, "Password123!")

	accessToken := createTestAccessToken(t, env, tenantID, email)
	expires := time.Now().Add(24 * time.Hour)

	rawKey, err := env.service.Create(context.Background(), tenantID, accessToken, "expiring-key", &expires)
	if err != nil {
		t.Fatalf("Failed to create api key: %v", err)
	}

	// Verify the key was created by listing and checking the expiry
	keys, err := env.service.List(context.Background(), tenantID, accessToken)
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}

	keyPrefix := extractKeyPrefix(rawKey)
	var found bool
	for _, k := range keys {
		if k.KeyPrefix == keyPrefix {
			found = true
			if k.Expires == nil {
				t.Fatal("Expected expires to be set")
			}
		}
	}
	if !found {
		t.Fatalf("Created key with prefix '%s' not found in list", keyPrefix)
	}
}

func TestApiKeyService_Create_InvalidToken(t *testing.T) {
	env := setupApiKeyServiceTest(t)
	defer env.cleanup()

	_, err := env.service.Create(context.Background(), "tenant1", "invalid-token", "my-key", nil)
	if err == nil {
		t.Fatal("Expected error with invalid token")
	}
}

func TestApiKeyService_Create_TenantMismatch(t *testing.T) {
	env := setupApiKeyServiceTest(t)
	defer env.cleanup()

	tenantID := "tenant1"
	email := "test@example.com"
	createTestAccount(t, env, tenantID, email, "Password123!")

	accessToken := createTestAccessToken(t, env, tenantID, email)

	_, err := env.service.Create(context.Background(), "tenant2", accessToken, "my-key", nil)
	if err == nil {
		t.Fatal("Expected error with tenant mismatch")
	}
}

func TestApiKeyService_List(t *testing.T) {
	env := setupApiKeyServiceTest(t)
	defer env.cleanup()

	tenantID := "tenant1"
	email := "test@example.com"
	createTestAccount(t, env, tenantID, email, "Password123!")

	accessToken := createTestAccessToken(t, env, tenantID, email)
	ctx := context.Background()

	_, err := env.service.Create(ctx, tenantID, accessToken, "key-1", nil)
	if err != nil {
		t.Fatalf("Failed to create first key: %v", err)
	}

	_, err = env.service.Create(ctx, tenantID, accessToken, "key-2", nil)
	if err != nil {
		t.Fatalf("Failed to create second key: %v", err)
	}

	keys, err := env.service.List(ctx, tenantID, accessToken)
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("Expected 2 keys, got %d", len(keys))
	}
}

func TestApiKeyService_List_IsolatedByAccount(t *testing.T) {
	env := setupApiKeyServiceTest(t)
	defer env.cleanup()

	tenantID := "tenant1"
	ctx := context.Background()

	createTestAccount(t, env, tenantID, "user1@example.com", "Password123!")
	createTestAccount(t, env, tenantID, "user2@example.com", "Password123!")

	token1 := createTestAccessToken(t, env, tenantID, "user1@example.com")
	token2 := createTestAccessToken(t, env, tenantID, "user2@example.com")

	_, err := env.service.Create(ctx, tenantID, token1, "user1-key", nil)
	if err != nil {
		t.Fatalf("Failed to create key for user1: %v", err)
	}

	_, err = env.service.Create(ctx, tenantID, token2, "user2-key", nil)
	if err != nil {
		t.Fatalf("Failed to create key for user2: %v", err)
	}

	keys1, err := env.service.List(ctx, tenantID, token1)
	if err != nil {
		t.Fatalf("Failed to list keys for user1: %v", err)
	}

	if len(keys1) != 1 {
		t.Fatalf("Expected 1 key for user1, got %d", len(keys1))
	}

	if keys1[0].Name != "user1-key" {
		t.Errorf("Expected key name 'user1-key', got '%s'", keys1[0].Name)
	}
}

func TestApiKeyService_Delete(t *testing.T) {
	env := setupApiKeyServiceTest(t)
	defer env.cleanup()

	tenantID := "tenant1"
	email := "test@example.com"
	createTestAccount(t, env, tenantID, email, "Password123!")

	accessToken := createTestAccessToken(t, env, tenantID, email)
	ctx := context.Background()

	rawKey, err := env.service.Create(ctx, tenantID, accessToken, "to-delete", nil)
	if err != nil {
		t.Fatalf("Failed to create key: %v", err)
	}

	keyPrefix := extractKeyPrefix(rawKey)
	err = env.service.Delete(ctx, tenantID, accessToken, keyPrefix)
	if err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	keys, err := env.service.List(ctx, tenantID, accessToken)
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("Expected 0 keys after delete, got %d", len(keys))
	}
}

func TestApiKeyService_Delete_NotFound(t *testing.T) {
	env := setupApiKeyServiceTest(t)
	defer env.cleanup()

	tenantID := "tenant1"
	email := "test@example.com"
	createTestAccount(t, env, tenantID, email, "Password123!")

	accessToken := createTestAccessToken(t, env, tenantID, email)

	err := env.service.Delete(context.Background(), tenantID, accessToken, "bwa_nonexist")
	if err == nil {
		t.Fatal("Expected error deleting non-existent key")
	}

	_, ok := err.(ApiKeyNotFoundError)
	if !ok {
		t.Fatalf("Expected ApiKeyNotFoundError, got %T: %v", err, err)
	}
}

func TestApiKeyService_Authenticate(t *testing.T) {
	env := setupApiKeyServiceTest(t)
	defer env.cleanup()

	tenantID := "tenant1"
	email := "test@example.com"
	createTestAccount(t, env, tenantID, email, "Password123!")

	accessToken := createTestAccessToken(t, env, tenantID, email)
	ctx := context.Background()

	rawKey, err := env.service.Create(ctx, tenantID, accessToken, "auth-key", nil)
	if err != nil {
		t.Fatalf("Failed to create key: %v", err)
	}

	authenticated, err := env.service.Authenticate(ctx, tenantID, email, rawKey)
	if err != nil {
		t.Fatalf("Failed to authenticate with api key: %v", err)
	}

	if authenticated.AccessToken == "" {
		t.Error("Expected access token to be set")
	}

	if authenticated.RefreshToken == "" {
		t.Error("Expected refresh token to be set")
	}
}

func TestApiKeyService_Authenticate_InvalidKey(t *testing.T) {
	env := setupApiKeyServiceTest(t)
	defer env.cleanup()

	tenantID := "tenant1"
	email := "test@example.com"
	createTestAccount(t, env, tenantID, email, "Password123!")

	accessToken := createTestAccessToken(t, env, tenantID, email)
	ctx := context.Background()

	rawKey, err := env.service.Create(ctx, tenantID, accessToken, "auth-key", nil)
	if err != nil {
		t.Fatalf("Failed to create key: %v", err)
	}

	keyPrefix := extractKeyPrefix(rawKey)
	wrongKey := keyPrefix + "_wrongsecret"
	_, err = env.service.Authenticate(ctx, tenantID, email, wrongKey)
	if err == nil {
		t.Fatal("Expected error with invalid key")
	}
}

func TestApiKeyService_Authenticate_BadFormat(t *testing.T) {
	env := setupApiKeyServiceTest(t)
	defer env.cleanup()

	_, err := env.service.Authenticate(context.Background(), "tenant1", "test@example.com", "not-a-valid-key")
	if err == nil {
		t.Fatal("Expected error with bad key format")
	}

	_, ok := err.(ApiKeyInvalidError)
	if !ok {
		t.Fatalf("Expected ApiKeyInvalidError, got %T: %v", err, err)
	}
}

func TestApiKeyService_Authenticate_DisabledKey(t *testing.T) {
	env := setupApiKeyServiceTest(t)
	defer env.cleanup()

	tenantID := "tenant1"
	email := "test@example.com"
	createTestAccount(t, env, tenantID, email, "Password123!")

	accessToken := createTestAccessToken(t, env, tenantID, email)
	ctx := context.Background()

	rawKey, err := env.service.Create(ctx, tenantID, accessToken, "disabled-key", nil)
	if err != nil {
		t.Fatalf("Failed to create key: %v", err)
	}

	// Disable the key directly in the database
	keyPrefix := extractKeyPrefix(rawKey)
	collection := env.db.Collection(apiKeyCollection)
	_, err = collection.UpdateOne(ctx,
		map[string]string{"keyPrefix": keyPrefix},
		map[string]interface{}{"$set": map[string]bool{"isEnabled": false}},
	)
	if err != nil {
		t.Fatalf("Failed to disable key: %v", err)
	}

	_, err = env.service.Authenticate(ctx, tenantID, email, rawKey)
	if err == nil {
		t.Fatal("Expected error with disabled key")
	}

	_, ok := err.(ApiKeyDisabledError)
	if !ok {
		t.Fatalf("Expected ApiKeyDisabledError, got %T: %v", err, err)
	}
}

func TestApiKeyService_Authenticate_ExpiredKey(t *testing.T) {
	env := setupApiKeyServiceTest(t)
	defer env.cleanup()

	tenantID := "tenant1"
	email := "test@example.com"
	createTestAccount(t, env, tenantID, email, "Password123!")

	accessToken := createTestAccessToken(t, env, tenantID, email)
	ctx := context.Background()

	expired := time.Now().Add(-1 * time.Hour)
	rawKey, err := env.service.Create(ctx, tenantID, accessToken, "expired-key", &expired)
	if err != nil {
		t.Fatalf("Failed to create key: %v", err)
	}

	_, err = env.service.Authenticate(ctx, tenantID, email, rawKey)
	if err == nil {
		t.Fatal("Expected error with expired key")
	}

	_, ok := err.(ApiKeyExpiredError)
	if !ok {
		t.Fatalf("Expected ApiKeyExpiredError, got %T: %v", err, err)
	}
}
