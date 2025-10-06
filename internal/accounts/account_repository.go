package accounts

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/latebit-io/bulwarkauth/internal/encryption"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// AccountRepository this is not a typical repository pattern, the functions are intentionally more granular
// to restrict potential leaks of boundaries. For example the account model does not contain
// the password. This sandboxes the information to never be returned.
type AccountRepository interface {
	Create(ctx context.Context, email, password string) error
	Read(ctx context.Context, email string) (*Account, error)
	Delete(ctx context.Context, email string) error
	UpdateEmail(ctx context.Context, email, newEmail string) (*Verification, error)
	UpdatePassword(ctx context.Context, email, newPassword string) error
	PasswordMatches(ctx context.Context, email, password string) (bool, error)
	LinkSocial(ctx context.Context, email string, provider SocialProvider) error
	Verify(ctx context.Context, email string) error
}

const (
	accountCollection = "accounts"
)

// MongodbAccountRepository account repository for accounts
type MongodbAccountRepository struct {
	db         *mongo.Database
	encryption encryption.Encryption
}

// NewMongodbAccountRepository returns a MongodbAccountRepository
func NewMongodbAccountRepository(db *mongo.Database, encryption encryption.Encryption) *MongodbAccountRepository {
	collection := db.Collection(accountCollection)
	_, err := collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Fatal(err)
	}
	return &MongodbAccountRepository{
		db:         db,
		encryption: encryption,
	}
}

// Create will create a new account
func (a MongodbAccountRepository) Create(ctx context.Context, email, password string) error {
	var errorMessages []string
	if email == "" {
		errorMessages = append(errorMessages, "email is required")
	}

	if password == "" {
		errorMessages = append(errorMessages, "password is required")
	}

	if len(errorMessages) > 0 {
		return errors.New(strings.Join(errorMessages, ","))
	}

	collection := a.db.Collection(accountCollection)
	uuid, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	hashed, err := a.encryption.Encrypt(password)
	if err != nil {
		return err
	}
	_, err = collection.InsertOne(ctx,
		bson.D{
			{Key: "email", Value: email},
			{Key: "password", Value: hashed},
			{Key: "isVerified", Value: false},
			{Key: "verificationToken", Value: uuid.String()},
			{Key: "isEnabled", Value: false},
			{Key: "isDeleted", Value: false},
			{Key: "created", Value: time.Now()},
			{Key: "modified", Value: time.Now()},
		})

	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return AccountDuplicateError{
				Value: email,
			}
		}
		return err
	}

	return nil
}

// Read will retrieve an account by email
func (a MongodbAccountRepository) Read(ctx context.Context, email string) (*Account, error) {
	collection := a.db.Collection(accountCollection)
	result := collection.FindOne(ctx, bson.D{{Key: "email", Value: email}})
	var account Account
	err := result.Decode(&account)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, AccountNotFoundError{Value: email}
		}
		return nil, err
	}
	return &account, nil
}

// Delete will soft delete the account by marking it as deleted
func (a MongodbAccountRepository) Delete(ctx context.Context, email string) error {
	collection := a.db.Collection(accountCollection)
	result, err := collection.UpdateOne(ctx, bson.D{{Key: "email", Value: email}}, bson.D{{Key: "$set",
		Value: bson.D{{Key: "isDeleted", Value: true}, {Key: "modified", Value: time.Now()}}}})
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return AccountNotFoundError{Value: email}
	}
	return nil
}

// UpdateEmail will change an accounts email
func (a MongodbAccountRepository) UpdateEmail(ctx context.Context, email, newEmail string) (*Verification, error) {
	collection := a.db.Collection(accountCollection)
	verificationToken := uuid.New()
	result, err := collection.UpdateOne(ctx, bson.D{{Key: "email", Value: email}}, bson.D{{Key: "$set",
		Value: bson.D{{Key: "email", Value: newEmail}, {Key: "verificationToken", Value: verificationToken.String()}, {Key: "isVerified", Value: false},
			{Key: "modified", Value: time.Now()}}}})

	if err != nil {
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, AccountNotFoundError{Value: email}
	}

	return &Verification{Email: newEmail, Token: verificationToken.String()}, nil
}

// UpdatePassword will change an accounts password
func (a MongodbAccountRepository) UpdatePassword(ctx context.Context, email, newPassword string) error {
	collection := a.db.Collection(accountCollection)
	hashed, err := a.encryption.Encrypt(newPassword)
	if err != nil {
		return err
	}
	result, err := collection.UpdateOne(ctx, bson.D{{Key: "email", Value: email}}, bson.D{{Key: "$set",
		Value: bson.D{{Key: "password", Value: hashed}, {Key: "modified", Value: time.Now()}}}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return AccountNotFoundError{Value: email}
	}
	return nil
}

// Verify will verify account and activate it
func (a MongodbAccountRepository) Verify(ctx context.Context, email string) error {
	collection := a.db.Collection(accountCollection)
	result, err := collection.UpdateOne(ctx, bson.D{{Key: "email", Value: email}}, bson.D{{Key: "$set",
		Value: bson.D{{Key: "verificationToken", Value: ""}, {Key: "isVerified", Value: true}, {Key: "isEnabled", Value: true},
			{Key: "modified", Value: time.Now()}}}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return AccountNotFoundError{Value: email}
	}
	return nil
}

// PasswordMatches check if the password is correct
func (a MongodbAccountRepository) PasswordMatches(ctx context.Context, email, password string) (bool, error) {
	collection := a.db.Collection(accountCollection)
	result := collection.FindOne(ctx, bson.D{{Key: "email", Value: email}})

	var r bson.M
	err := result.Decode(&r)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(r["password"].(string)), []byte(password))
	if err != nil {
		return false, err
	}

	return true, nil
}

func (a MongodbAccountRepository) LinkSocial(ctx context.Context, email string, provider SocialProvider) error {
	collection := a.db.Collection(accountCollection)
	result, err := collection.UpdateOne(ctx, bson.D{{Key: "email", Value: email}}, bson.D{{Key: "$push", Value: bson.D{{Key: "socialProvider", Value: provider}}}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return AccountNotFoundError{Value: email}
	}
	return nil
}
