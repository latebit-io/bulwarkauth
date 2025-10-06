package email

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type EmailRepository interface {
	Create(ctx context.Context, name, template string) error
	Update(ctx context.Context, name, template string) error
	Read(ctx context.Context, name string) (string, error)
}

type EmailTemplate struct {
	Name     string
	Template string
}

type MongoDbEmailRepository struct {
	db *mongo.Database
}

func NewMongoDbEmailRepository(db *mongo.Database) *MongoDbEmailRepository {
	return &MongoDbEmailRepository{db: db}
}

func (r *MongoDbEmailRepository) Read(ctx context.Context, name string) (string, error) {
	collection := r.db.Collection("emails")
	result := collection.FindOne(ctx, bson.M{"name": name})
	if result.Err() != nil {
		return "", result.Err()
	}
	var email EmailTemplate
	err := result.Decode(&email)
	if err != nil {
		return "", err
	}
	return email.Template, nil
}

func (r *MongoDbEmailRepository) Create(ctx context.Context, name, template string) error {
	collection := r.db.Collection("emails")
	_, err := collection.InsertOne(ctx, bson.M{"name": name, "template": template})

	if err != nil {
		return err
	}

	return nil
}

func (r *MongoDbEmailRepository) Update(ctx context.Context, name, template string) error {
	collection := r.db.Collection("emails")
	_, err := collection.UpdateOne(ctx, bson.M{"name": name}, bson.M{"template": template})
	if err != nil {
		return err
	}
	return nil
}
