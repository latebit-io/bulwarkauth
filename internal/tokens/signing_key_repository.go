package tokens

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SigningKeyRepository interface {
	Add(ctx context.Context, privateKey string, publicKey string, algorithm string) error
	GetKey(ctx context.Context, keyId string) (SigningKey, error)
	GetLatestKey(ctx context.Context) (SigningKey, error)
	GetAllKeys(ctx context.Context) ([]SigningKey, error)
}

type DefaultSigningKeyRepository struct {
	db             *mongo.Database
	collectionName string
}

func NewDefaultSigningKeyRepository(db *mongo.Database) *DefaultSigningKeyRepository {
	return &DefaultSigningKeyRepository{db: db, collectionName: "signingKeys"}
}

func (d *DefaultSigningKeyRepository) Add(ctx context.Context, privateKey string, publicKey string, algorithm string) error {
	collection := d.db.Collection(d.collectionName)
	keyId := uuid.New()
	_, err := collection.InsertOne(ctx, SigningKey{
		KeyId:      keyId.String(),
		Format:     "PKCS#1",
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Algorithm:  algorithm,
		Created:    time.Now(),
	})

	if err != nil {
		return err
	}

	return nil
}

func (d *DefaultSigningKeyRepository) GetKey(ctx context.Context, keyId string) (SigningKey, error) {
	collection := d.db.Collection(d.collectionName)
	var key SigningKey
	err := collection.FindOne(ctx, bson.D{{Key: "keyId", Value: keyId}}).Decode(&key)
	if err != nil {
		return SigningKey{}, err
	}
	return key, nil
}

func (d *DefaultSigningKeyRepository) GetLatestKey(ctx context.Context) (SigningKey, error) {
	collection := d.db.Collection(d.collectionName)
	var key SigningKey
	opt := options.FindOne().SetSort(bson.D{{"created", -1}})
	err := collection.FindOne(ctx, bson.D{{}}, opt).Decode(&key)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return SigningKey{}, nil
		}
		return SigningKey{}, err
	}

	return key, nil
}

func (d *DefaultSigningKeyRepository) GetAllKeys(ctx context.Context) ([]SigningKey, error) {
	collection := d.db.Collection(d.collectionName)
	var keys []SigningKey
	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return keys, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var key SigningKey
		err := cursor.Decode(&key)
		if err != nil {
			return keys, err
		}
		keys = append(keys, key)
	}
	if err := cursor.Err(); err != nil {
		return keys, err
	}
	return keys, nil
}
