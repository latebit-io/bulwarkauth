package authentication

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	collectionTokens = "tokens"
)

type Token struct {
	Id           string    `json:"id"`
	TenantID     string    `json:"tenantId"`
	Email        string    `json:"email"`
	ClientId     string    `json:"clientId"`
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	CreatedAt    time.Time `json:"createdAt"`
	ModifiedAt   time.Time `json:"modifiedAt"`
}

type TokenRepository interface {
	Create(ctx context.Context, tenantID, email, clientId, accessToken, refreshToken string) error
	Delete(ctx context.Context, tenantID, email, clientId string) error
	DeleteByEmail(ctx context.Context, tenantID, email string) error
	Read(ctx context.Context, tenantID, email, clientId string) (*Token, error)
}

type DefaultTokenRepository struct {
	db *mongo.Database
}

func NewDefaultTokenRepository(db *mongo.Database) *DefaultTokenRepository {
	return &DefaultTokenRepository{db}
}

func (t *DefaultTokenRepository) Create(ctx context.Context, tenantID, email, clientId, accessToken, refreshToken string) error {
	collection := t.db.Collection(collectionTokens)

	filter := bson.M{"tenantId": tenantID, "email": email, "clientId": clientId}
	update := bson.D{
		{"$set", bson.D{
			{"tenantId", tenantID},
			{"accessToken", accessToken},
			{"refreshToken", refreshToken},
			{"modifiedAt", time.Now()},
		}},
		{"$setOnInsert", bson.D{
			{"createdAt", time.Now()},
		}},
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {
		return err
	}

	return nil
}

func (t *DefaultTokenRepository) DeleteByEmail(ctx context.Context, tenantID, email string) error {
	collection := t.db.Collection(collectionTokens)
	_, err := collection.DeleteOne(ctx, bson.M{"tenantId": tenantID, "email": email})
	if err != nil {
		return err
	}
	return nil
}

func (t *DefaultTokenRepository) Delete(ctx context.Context, tenantID, email, clientId string) error {
	collection := t.db.Collection(collectionTokens)
	_, err := collection.DeleteOne(ctx, bson.M{"tenantId": tenantID, "email": email, "clientId": clientId})
	if err != nil {
		return err
	}
	return nil
}

func (t *DefaultTokenRepository) Read(ctx context.Context, tenantID, email, clientId string) (*Token, error) {
	collection := t.db.Collection(collectionTokens)
	var token Token
	err := collection.FindOne(ctx, bson.M{"tenantId": tenantID, "email": email}).Decode(&token)
	if err != nil {
		return nil, err
	}
	return &token, nil
}
