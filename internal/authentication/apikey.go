package authentication

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/latebit-io/bulwarkauth/internal/accounts"
	"github.com/latebit-io/bulwarkauth/internal/encryption"
	"github.com/latebit-io/bulwarkauth/internal/tokens"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	apiKeyPrefix     = "bwa"
	randomLength     = 8
	apiKeyCollection = "apiKeys"
)

type ApiKey struct {
	ID        string     `json:"id" bson:"id"`
	TenantID  string     `json:"tenantId" bson:"tenantId"`
	AccountID string     `json:"-" bson:"accountId"`
	Name      string     `json:"name" bson:"name"`
	KeyHash   string     `json:"-" bson:"key"`
	KeyPrefix string     `json:"keyPrefix" bson:"keyPrefix"`
	IsEnabled bool       `json:"isEnabled" bson:"isEnabled"`
	Expires   *time.Time `json:"expires" bson:"expires"`
	Created   time.Time  `json:"created" bson:"created"`
	Modified  time.Time  `json:"modified" bson:"modified"`
}

type ApiRepository interface {
	Create(ctx context.Context, apiKey *ApiKey) error
	Read(ctx context.Context, tenantID, accountID, keyPrefix string) (*ApiKey, error)
	List(ctx context.Context, tenantID, accountID string) ([]ApiKey, error)
	Delete(ctx context.Context, tenantID, accountID, keyPrefix string) error
}

type ApiKeyService interface {
	Authenticate(ctx context.Context, tenantID, email, apiKey string) (*Authenticated, error)
	Create(ctx context.Context, tenantID, accessToken, name string, expires *time.Time) (string, error)
	List(ctx context.Context, tenantID, accessToken string) ([]ApiKey, error)
	Delete(ctx context.Context, tenantID, accessToken, keyPrefix string) error
}

// MongoDbApiRepository MongoDB implementation of ApiRepository.
type MongoDbApiRepository struct {
	db *mongo.Database
}

func NewMongoDbApiRepository(db *mongo.Database) *MongoDbApiRepository {
	collection := db.Collection(apiKeyCollection)
	_, err := collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "tenantId", Value: 1}, {Key: "accountId", Value: 1}, {Key: "keyPrefix", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Fatal(err)
	}
	return &MongoDbApiRepository{db: db}
}

func (m *MongoDbApiRepository) Create(ctx context.Context, apiKey *ApiKey) error {
	collection := m.db.Collection(apiKeyCollection)
	_, err := collection.InsertOne(ctx, apiKey)
	return err
}

func (m *MongoDbApiRepository) Read(ctx context.Context, tenantID, accountID, keyPrefix string) (*ApiKey, error) {
	collection := m.db.Collection(apiKeyCollection)
	filter := bson.M{"tenantId": tenantID, "accountId": accountID, "keyPrefix": keyPrefix}
	result := collection.FindOne(ctx, filter)

	if errors.Is(result.Err(), mongo.ErrNoDocuments) {
		return nil, ApiKeyNotFoundError{Value: keyPrefix}
	}

	var apiKey ApiKey
	err := result.Decode(&apiKey)
	if err != nil {
		return nil, err
	}

	return &apiKey, nil
}

func (m *MongoDbApiRepository) List(ctx context.Context, tenantID, accountID string) ([]ApiKey, error) {
	collection := m.db.Collection(apiKeyCollection)
	filter := bson.M{"tenantId": tenantID, "accountId": accountID}
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var apiKeys []ApiKey
	for cursor.Next(ctx) {
		var apiKey ApiKey
		if err := cursor.Decode(&apiKey); err != nil {
			return nil, err
		}
		apiKeys = append(apiKeys, apiKey)
	}
	return apiKeys, nil
}

func (m *MongoDbApiRepository) Delete(ctx context.Context, tenantID, accountID, keyPrefix string) error {
	collection := m.db.Collection(apiKeyCollection)
	result, err := collection.DeleteOne(ctx, bson.M{"tenantId": tenantID, "accountId": accountID, "keyPrefix": keyPrefix})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ApiKeyNotFoundError{Value: keyPrefix}
	}
	return nil
}

// DefaultApiKeyService implementation of ApiKeyService.
type DefaultApiKeyService struct {
	apiRepo     ApiRepository
	encryption  Encryption
	tokenizer   tokens.Tokenizer
	accountRepo accounts.AccountRepository
}

func NewDefaultApiKeyService(repo ApiRepository, encryption Encryption, tokenizer tokens.Tokenizer, accountRepo accounts.AccountRepository) *DefaultApiKeyService {
	return &DefaultApiKeyService{
		apiRepo:     repo,
		encryption:  encryption,
		tokenizer:   tokenizer,
		accountRepo: accountRepo,
	}
}

func (s *DefaultApiKeyService) Authenticate(ctx context.Context, tenantID, email, apiKey string) (*Authenticated, error) {
	splitKey := strings.Split(apiKey, "_")
	if len(splitKey) != 3 {
		return nil, ApiKeyInvalidError{Value: email}
	}

	account, err := s.accountRepo.Read(ctx, tenantID, email)
	if err != nil {
		return nil, err
	}

	keyPrefix := fmt.Sprintf("%s_%s", splitKey[0], splitKey[1])
	apiKeyResult, err := s.apiRepo.Read(ctx, tenantID, account.ID, keyPrefix)
	if err != nil {
		return nil, err
	}

	if !apiKeyResult.IsEnabled {
		return nil, ApiKeyDisabledError{Value: email}
	}

	if apiKeyResult.Expires != nil && apiKeyResult.Expires.Before(time.Now()) {
		return nil, ApiKeyExpiredError{Value: email}
	}

	verified, err := s.encryption.Verify(apiKeyResult.KeyHash, splitKey[2])
	if err != nil {
		return nil, err
	}
	if !verified {
		return nil, ApiKeyInvalidError{Value: email}
	}

	accessToken, err := s.tokenizer.CreateAccessToken(ctx, tenantID, email, "apikey", account.Roles)
	if err != nil {
		return nil, err
	}
	refreshToken, err := s.tokenizer.CreateRefreshToken(ctx, tenantID, email, "apikey")
	if err != nil {
		return nil, err
	}

	return &Authenticated{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *DefaultApiKeyService) Create(ctx context.Context, tenantID, accessToken, name string, expires *time.Time) (string, error) {
	claims, err := s.tokenizer.ValidateAccessToken(ctx, accessToken)
	if err != nil {
		return "", err
	}

	if claims.TenantID != tenantID {
		return "", errors.New("token invalid")
	}

	accountID, err := s.resolveAccountID(ctx, tenantID, claims.Subject)
	if err != nil {
		return "", err
	}

	return s.generate(ctx, tenantID, accountID, name, expires)
}

func (s *DefaultApiKeyService) List(ctx context.Context, tenantID, accessToken string) ([]ApiKey, error) {
	claims, err := s.tokenizer.ValidateAccessToken(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	if claims.TenantID != tenantID {
		return nil, errors.New("token invalid")
	}

	accountID, err := s.resolveAccountID(ctx, tenantID, claims.Subject)
	if err != nil {
		return nil, err
	}

	return s.apiRepo.List(ctx, tenantID, accountID)
}

func (s *DefaultApiKeyService) Delete(ctx context.Context, tenantID, accessToken, keyPrefix string) error {
	claims, err := s.tokenizer.ValidateAccessToken(ctx, accessToken)
	if err != nil {
		return err
	}

	if claims.TenantID != tenantID {
		return errors.New("token invalid")
	}

	accountID, err := s.resolveAccountID(ctx, tenantID, claims.Subject)
	if err != nil {
		return err
	}

	return s.apiRepo.Delete(ctx, tenantID, accountID, keyPrefix)
}

// resolveAccountID looks up the account by email and returns the internal account ID.
func (s *DefaultApiKeyService) resolveAccountID(ctx context.Context, tenantID, email string) (string, error) {
	account, err := s.accountRepo.Read(ctx, tenantID, email)
	if err != nil {
		return "", err
	}
	return account.ID, nil
}

func (s *DefaultApiKeyService) generate(ctx context.Context, tenantID, accountID, name string, expire *time.Time) (string, error) {
	id := uuid.New().String()
	randomStr := encryption.GenerateRandomString(randomLength)
	keyPrefix := fmt.Sprintf("%s_%s", apiKeyPrefix, randomStr)
	key := uuid.New().String()
	hashKey, err := s.encryption.Encrypt(key)
	if err != nil {
		return "", err
	}
	apiKey := &ApiKey{
		ID:        id,
		TenantID:  tenantID,
		AccountID: accountID,
		Name:      name,
		KeyHash:   hashKey,
		KeyPrefix: keyPrefix,
		IsEnabled: true,
		Expires:   expire,
		Created:   time.Now(),
		Modified:  time.Now(),
	}

	if err := s.apiRepo.Create(ctx, apiKey); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s_%s", apiKey.KeyPrefix, key), nil
}
