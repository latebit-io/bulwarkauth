package authentication

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/latebit-io/bulwarkauth/internal/utils"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func setupApiKeyTestDB(t *testing.T) (*mongo.Database, func()) {
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
	return db, func() {
		client.Disconnect(ctx)
		mongoServer.Stop()
	}
}

func newTestApiKey(tenantID, accountID, name, keyPrefix, keyHash string) *ApiKey {
	now := time.Now()
	return &ApiKey{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		AccountID: accountID,
		Name:      name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		IsEnabled: true,
		Created:   now,
		Modified:  now,
	}
}

func TestMongoDbApiRepository_Create(t *testing.T) {
	db, cleanup := setupApiKeyTestDB(t)
	defer cleanup()

	repo := NewMongoDbApiRepository(db)
	ctx := context.Background()

	apiKey := newTestApiKey("tenant1", "account1", "my-key", "bwa_abcd1234", "hashedSecret")

	err := repo.Create(ctx, apiKey)
	if err != nil {
		t.Fatalf("Failed to create api key: %v", err)
	}

	result, err := repo.Read(ctx, "tenant1", "account1", "bwa_abcd1234")
	if err != nil {
		t.Fatalf("Failed to read api key: %v", err)
	}

	if result.Name != "my-key" {
		t.Errorf("Expected name 'my-key', got '%s'", result.Name)
	}
	if result.AccountID != "account1" {
		t.Errorf("Expected accountID 'account1', got '%s'", result.AccountID)
	}
	if result.KeyPrefix != "bwa_abcd1234" {
		t.Errorf("Expected keyPrefix 'bwa_abcd1234', got '%s'", result.KeyPrefix)
	}
	if !result.IsEnabled {
		t.Error("Expected api key to be enabled")
	}
}

func TestMongoDbApiRepository_Create_DuplicateKeyPrefix(t *testing.T) {
	db, cleanup := setupApiKeyTestDB(t)
	defer cleanup()

	repo := NewMongoDbApiRepository(db)
	ctx := context.Background()

	apiKey1 := newTestApiKey("tenant1", "account1", "key-1", "bwa_same1234", "hash1")
	apiKey2 := newTestApiKey("tenant1", "account1", "key-2", "bwa_same1234", "hash2")

	err := repo.Create(ctx, apiKey1)
	if err != nil {
		t.Fatalf("Failed to create first api key: %v", err)
	}

	err = repo.Create(ctx, apiKey2)
	if err == nil {
		t.Fatal("Expected error creating duplicate key prefix for same account")
	}
}

func TestMongoDbApiRepository_Create_SamePrefixDifferentAccount(t *testing.T) {
	db, cleanup := setupApiKeyTestDB(t)
	defer cleanup()

	repo := NewMongoDbApiRepository(db)
	ctx := context.Background()

	apiKey1 := newTestApiKey("tenant1", "account1", "key-1", "bwa_same1234", "hash1")
	apiKey2 := newTestApiKey("tenant1", "account2", "key-1", "bwa_same1234", "hash2")

	err := repo.Create(ctx, apiKey1)
	if err != nil {
		t.Fatalf("Failed to create first api key: %v", err)
	}

	err = repo.Create(ctx, apiKey2)
	if err != nil {
		t.Fatalf("Same prefix for different accounts should be allowed: %v", err)
	}
}

func TestMongoDbApiRepository_Read_NotFound(t *testing.T) {
	db, cleanup := setupApiKeyTestDB(t)
	defer cleanup()

	repo := NewMongoDbApiRepository(db)
	ctx := context.Background()

	_, err := repo.Read(ctx, "tenant1", "account1", "bwa_nonexist")
	if err == nil {
		t.Fatal("Expected error for non-existent key")
	}

	_, ok := err.(ApiKeyNotFoundError)
	if !ok {
		t.Fatalf("Expected ApiKeyNotFoundError, got %T: %v", err, err)
	}
}

func TestMongoDbApiRepository_List(t *testing.T) {
	db, cleanup := setupApiKeyTestDB(t)
	defer cleanup()

	repo := NewMongoDbApiRepository(db)
	ctx := context.Background()

	apiKey1 := newTestApiKey("tenant1", "account1", "key-1", "bwa_prefix01", "hash1")
	apiKey2 := newTestApiKey("tenant1", "account1", "key-2", "bwa_prefix02", "hash2")
	apiKey3 := newTestApiKey("tenant1", "account2", "key-3", "bwa_prefix03", "hash3")

	for _, key := range []*ApiKey{apiKey1, apiKey2, apiKey3} {
		if err := repo.Create(ctx, key); err != nil {
			t.Fatalf("Failed to create api key: %v", err)
		}
	}

	keys, err := repo.List(ctx, "tenant1", "account1")
	if err != nil {
		t.Fatalf("Failed to list api keys: %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("Expected 2 keys for account1, got %d", len(keys))
	}
}

func TestMongoDbApiRepository_List_Empty(t *testing.T) {
	db, cleanup := setupApiKeyTestDB(t)
	defer cleanup()

	repo := NewMongoDbApiRepository(db)
	ctx := context.Background()

	keys, err := repo.List(ctx, "tenant1", "nonexistent")
	if err != nil {
		t.Fatalf("Failed to list api keys: %v", err)
	}

	if keys != nil {
		t.Errorf("Expected nil for empty list, got %v", keys)
	}
}

func TestMongoDbApiRepository_Delete(t *testing.T) {
	db, cleanup := setupApiKeyTestDB(t)
	defer cleanup()

	repo := NewMongoDbApiRepository(db)
	ctx := context.Background()

	apiKey := newTestApiKey("tenant1", "account1", "my-key", "bwa_delete01", "hash1")
	err := repo.Create(ctx, apiKey)
	if err != nil {
		t.Fatalf("Failed to create api key: %v", err)
	}

	err = repo.Delete(ctx, "tenant1", "account1", "bwa_delete01")
	if err != nil {
		t.Fatalf("Failed to delete api key: %v", err)
	}

	_, err = repo.Read(ctx, "tenant1", "account1", "bwa_delete01")
	if err == nil {
		t.Fatal("Expected error reading deleted key")
	}
}

func TestMongoDbApiRepository_Delete_NotFound(t *testing.T) {
	db, cleanup := setupApiKeyTestDB(t)
	defer cleanup()

	repo := NewMongoDbApiRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, "tenant1", "account1", "bwa_nonexist")
	if err == nil {
		t.Fatal("Expected error deleting non-existent key")
	}

	_, ok := err.(ApiKeyNotFoundError)
	if !ok {
		t.Fatalf("Expected ApiKeyNotFoundError, got %T: %v", err, err)
	}
}

func TestMongoDbApiRepository_List_IsolatedByTenant(t *testing.T) {
	db, cleanup := setupApiKeyTestDB(t)
	defer cleanup()

	repo := NewMongoDbApiRepository(db)
	ctx := context.Background()

	apiKey1 := newTestApiKey("tenant1", "account1", "key-1", "bwa_prefix01", "hash1")
	apiKey2 := newTestApiKey("tenant2", "account1", "key-2", "bwa_prefix01", "hash2")

	for _, key := range []*ApiKey{apiKey1, apiKey2} {
		if err := repo.Create(ctx, key); err != nil {
			t.Fatalf("Failed to create api key: %v", err)
		}
	}

	keys, err := repo.List(ctx, "tenant1", "account1")
	if err != nil {
		t.Fatalf("Failed to list api keys: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("Expected 1 key for tenant1/account1, got %d", len(keys))
	}

	if keys[0].TenantID != "tenant1" {
		t.Errorf("Expected tenantId 'tenant1', got '%s'", keys[0].TenantID)
	}
}
