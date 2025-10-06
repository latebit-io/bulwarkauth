package authentication

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LogonCode struct {
	Email   string    `bson:"email"`
	Code    string    `bson:"code"`
	Expires time.Time `bson:"expires"`
	Created time.Time `bson:"created"`
}

type LogonCodeRepository interface {
	Create(ctx context.Context, email string, code string, expires time.Time) error
	Delete(ctx context.Context, email string, code string) error
	Read(ctx context.Context, email string) (*LogonCode, error)
}

const (
	logonCodeCollectionName = "logonCodes"
)

type DefaultLogonCodeRepository struct {
	db *mongo.Database
}

func NewDefaultLogonCodeRepository(db *mongo.Database) *DefaultLogonCodeRepository {
	return &DefaultLogonCodeRepository{db}
}

func (c *DefaultLogonCodeRepository) Create(ctx context.Context, email, code string, expires time.Time) error {
	collection := c.db.Collection(logonCodeCollectionName)
	filter := bson.D{{"email", email}}
	update := bson.D{
		{"$set", bson.D{
			{"code", code},
			{"expires", expires},
			{"created", time.Now()},
		}},
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}
	return nil
}

func (c *DefaultLogonCodeRepository) Delete(ctx context.Context, email string, code string) error {
	collection := c.db.Collection(logonCodeCollectionName)
	_, err := collection.DeleteOne(ctx, bson.D{{"email", email}, {"code", code}})
	if err != nil {
		return err
	}
	return nil
}

func (c *DefaultLogonCodeRepository) Read(ctx context.Context, email string) (*LogonCode, error) {
	collection := c.db.Collection(logonCodeCollectionName)
	var logonCode LogonCode
	err := collection.FindOne(ctx, bson.D{{"email", email}}).Decode(&logonCode)
	if err != nil {
		return nil, err
	}
	return &logonCode, nil
}
