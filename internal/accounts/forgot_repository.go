package accounts

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Forgot struct {
	Token   string    `bson:"token"`
	Email   string    `bson:"email"`
	Created time.Time `bson:"created"`
}

type ForgotRepository interface {
	Create(ctx context.Context, email string) error
	Read(ctx context.Context, email string) (*Forgot, error)
	Delete(ctx context.Context, email string) error
}

const (
	collectionForgot = "forgots"
)

type MongoDbForgotRepository struct {
	db *mongo.Database
}

func (f *MongoDbForgotRepository) Create(ctx context.Context, email string) error {
	collection := f.db.Collection(collectionForgot)
	forgotToken := uuid.New()
	filter := bson.M{"email": email}
	opts := options.Update().SetUpsert(true)

	_, err := collection.UpdateOne(ctx, filter, bson.M{"$set": bson.M{"email": email, "created": time.Now(),
		"token": forgotToken.String()}}, opts)
	if err != nil {
		return err
	}
	return nil
}

func (f *MongoDbForgotRepository) Read(ctx context.Context, email string) (*Forgot, error) {
	collection := f.db.Collection(collectionForgot)
	result := collection.FindOne(ctx, bson.M{"email": email})
	if errors.Is(result.Err(), mongo.ErrNoDocuments) {
		return nil, AccountNotFoundError{Value: email}
	}
	var forgot Forgot
	err := result.Decode(&forgot)
	if err != nil {
		return nil, err
	}

	return &forgot, nil
}

func (f *MongoDbForgotRepository) Delete(ctx context.Context, email string) error {
	collection := f.db.Collection(collectionForgot)
	_, err := collection.DeleteOne(ctx, bson.M{"email": email})
	if err != nil {
		return err
	}
	return nil
}

func NewMongoDbForgotRepository(db *mongo.Database) *MongoDbForgotRepository {
	return &MongoDbForgotRepository{db}
}
