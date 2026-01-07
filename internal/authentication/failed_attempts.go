package authentication

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	failedAttemptsCollectionName = "failedAttempts"
)

type FailedAttempt struct {
	Email       string    `bson:"email"`
	Count       int       `bson:"count"`
	LockedUntil time.Time `bson:"lockedUntil"`
	UpdatedAt   time.Time `bson:"updatedAt"`
}

type FailedAttemptRepository interface {
	Get(ctx context.Context, email string) (*FailedAttempt, error)
	Increment(ctx context.Context, email string) error
	Lock(ctx context.Context, email string, duration time.Duration) error
	Clear(ctx context.Context, email string) error
	IncrementAndLockIfNeeded(ctx context.Context, email string, maxAttempts int, lockDuration time.Duration) (bool, error)
}

type MongoFailedAttemptRepository struct {
	collection *mongo.Collection
}

func NewMongoFailedAttemptRepository(db *mongo.Database) FailedAttemptRepository {
	collection := db.Collection(failedAttemptsCollectionName)

	// Create index on email
	_, _ = collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	// Create TTL index to auto-delete old records after 24 hours
	_, _ = collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "updatedAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(86400), // 24 hours
	})

	return &MongoFailedAttemptRepository{
		collection: collection,
	}
}

func (r *MongoFailedAttemptRepository) Get(ctx context.Context, email string) (*FailedAttempt, error) {
	var attempt FailedAttempt
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&attempt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &attempt, nil
}

func (r *MongoFailedAttemptRepository) Increment(ctx context.Context, email string) error {
	update := bson.M{
		"$inc": bson.M{"count": 1},
		"$set": bson.M{"updatedAt": time.Now()},
		"$setOnInsert": bson.M{
			"email":       email,
			"lockedUntil": time.Time{},
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.collection.UpdateOne(ctx, bson.M{"email": email}, update, opts)
	return err
}

func (r *MongoFailedAttemptRepository) Lock(ctx context.Context, email string, duration time.Duration) error {
	lockedUntil := time.Now().Add(duration)
	update := bson.M{
		"$set": bson.M{
			"lockedUntil": lockedUntil,
			"updatedAt":   time.Now(),
		},
	}

	_, err := r.collection.UpdateOne(ctx, bson.M{"email": email}, update)
	return err
}

func (r *MongoFailedAttemptRepository) Clear(ctx context.Context, email string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"email": email})
	return err
}

// IncrementAndLockIfNeeded atomically increments the failed attempt count and locks the account if the threshold is reached.
// Returns true if the account was locked, false otherwise.
func (r *MongoFailedAttemptRepository) IncrementAndLockIfNeeded(ctx context.Context, email string, maxAttempts int, lockDuration time.Duration) (bool, error) {
	now := time.Now()
	lockedUntil := now.Add(lockDuration)

	// First, increment the counter
	update := bson.M{
		"$inc": bson.M{"count": 1},
		"$set": bson.M{"updatedAt": now},
		"$setOnInsert": bson.M{
			"email":       email,
			"lockedUntil": time.Time{},
		},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var result FailedAttempt
	err := r.collection.FindOneAndUpdate(ctx, bson.M{"email": email}, update, opts).Decode(&result)
	if err != nil {
		return false, err
	}

	// If we've reached the threshold, lock the account
	if result.Count >= maxAttempts {
		lockUpdate := bson.M{
			"$set": bson.M{
				"lockedUntil": lockedUntil,
				"updatedAt":   now,
			},
		}

		_, err := r.collection.UpdateOne(ctx, bson.M{"email": email}, lockUpdate)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}
