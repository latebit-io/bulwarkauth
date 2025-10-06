package accounts

import (
	"context"
	"errors"
	"testing"

	"github.com/latebit-io/bulwarkauth/internal/encryption"
	"github.com/latebit-io/bulwarkauth/internal/utils"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestUserRepository_CreateAccount(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		password    string
		expectedErr error
	}{
		{"Valid User", "test@latebit.io", "password", nil},
		{"Empty email", "", "password", errors.New("email is required")},
		{"Empty password", "test@latebit.io", "", errors.New("password is required")},
		{"Duplicate email", "test@latebit.io", "password", AccountDuplicateError{Value: "test@latebit.io"}},
	}
	mongodb := utils.NewMongoTestUtil()
	mongoServer, err := mongodb.CreateServer()
	if err != nil {
		t.Fatal(err)
	}
	defer mongoServer.Stop()

	// Connect to the in-memory MongoDB server
	clientOptions := options.Client().ApplyURI(mongoServer.URI())
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := client.Disconnect(context.TODO())
		if err != nil {
			t.Fatal(err)
		}
	}()

	// Create a test database and collection
	db := client.Database("bulwark")
	accountsRepo := NewMongodbAccountRepository(db, encryption.NewDefaultEncryption())
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			err := accountsRepo.Create(context.TODO(), tt.email, tt.password)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestUserRepository_ReadAccount(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		password      string
		expectedEmail string
		expectedErr   error
	}{
		{"Valid User", "test@latebit.io", "password", "test@latebit.io", nil},
		{"Not Found", "tes1t@latebit.io", "password", "test2@latebit.io", AccountNotFoundError{Value: "test2@latebit.io"}},
	}
	mongodb := utils.NewMongoTestUtil()
	mongoServer, err := mongodb.CreateServer()
	if err != nil {
		t.Fatal(err)
	}
	defer mongoServer.Stop()

	// Connect to the in-memory MongoDB server
	clientOptions := options.Client().ApplyURI(mongoServer.URI())
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := client.Disconnect(context.TODO())
		if err != nil {
			t.Fatal(err)
		}
	}()

	// Create a test database and collection
	db := client.Database("bulwark")
	accountsRepo := NewMongodbAccountRepository(db, encryption.NewDefaultEncryption())
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			err := accountsRepo.Create(context.TODO(), tt.email, tt.password)
			assert.Equal(t, nil, err)
			account, err := accountsRepo.Read(context.TODO(), tt.expectedEmail)
			assert.Equal(t, tt.expectedErr, err)
			if err == nil {
				assert.Equal(t, tt.expectedEmail, account.Email)
			}
		})
	}
}

func TestUserRepository_Delete(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		password      string
		expectedEmail string
		expectedErr   error
	}{
		{"Valid User", "test@latebit.io", "password", "test@latebit.io", nil},
		{"Not Found", "tes1t@latebit.io", "password", "test2@latebit.io", AccountNotFoundError{Value: "test2@latebit.io"}},
	}
	mongodb := utils.NewMongoTestUtil()
	mongoServer, err := mongodb.CreateServer()
	if err != nil {
		t.Fatal(err)
	}
	defer mongoServer.Stop()

	// Connect to the in-memory MongoDB server
	clientOptions := options.Client().ApplyURI(mongoServer.URI())
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := client.Disconnect(context.TODO())
		if err != nil {
			t.Fatal(err)
		}
	}()

	// Create a test database and collection
	db := client.Database("bulwark")
	accountsRepo := NewMongodbAccountRepository(db, encryption.NewDefaultEncryption())
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			err := accountsRepo.Create(context.TODO(), tt.email, tt.password)
			assert.Equal(t, nil, err)
			err = accountsRepo.Delete(context.TODO(), tt.expectedEmail)
			assert.Equal(t, tt.expectedErr, err)
			if err == nil {
				account, err := accountsRepo.Read(context.TODO(), tt.expectedEmail)
				assert.Equal(t, nil, err)
				assert.Equal(t, true, account.IsDeleted)
			}
		})
	}
}

func TestUserRepository_UpdateEmail(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		newEmail    string
		expectedErr error
	}{
		{"Valid User", "test@latebit.io", "test1@latebit.io", nil},
		{"Not Found", "tes2t@latebit.io", "test2@latebit.io", AccountNotFoundError{Value: "test2@latebit.io"}},
	}
	mongodb := utils.NewMongoTestUtil()
	mongoServer, err := mongodb.CreateServer()
	if err != nil {
		t.Fatal(err)
	}
	defer mongoServer.Stop()

	// Connect to the in-memory MongoDB server
	clientOptions := options.Client().ApplyURI(mongoServer.URI())
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := client.Disconnect(context.TODO())
		if err != nil {
			t.Fatal(err)
		}
	}()

	// Create a test database and collection
	db := client.Database("bulwark")
	accountsRepo := NewMongodbAccountRepository(db, encryption.NewDefaultEncryption())
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			err := accountsRepo.Create(context.TODO(), tt.email, "password")
			assert.Equal(t, nil, err)
			v, err := accountsRepo.UpdateEmail(context.TODO(), tt.email, tt.newEmail)
			assert.Equal(t, nil, err)
			if err == nil {
				assert.Equal(t, v.Email, tt.newEmail)
				assert.NotEmpty(t, v.Token)
				account, err := accountsRepo.Read(context.TODO(), tt.newEmail)
				assert.Equal(t, nil, err)
				assert.Equal(t, tt.newEmail, account.Email)
			}
		})
	}
}

func TestUserRepository_UpdatePassword(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		password    string
		newPassword string
		match       bool
		expectedErr error
	}{
		{"Valid User", "test@latebit.io", "password", "password1", true, nil},
		//{"Does not match", "test1@latebit.io", "password", "password1", false, internal.NotFoundError{Value: "test2@latebit.io"}},
	}
	mongodb := utils.NewMongoTestUtil()
	mongoServer, err := mongodb.CreateServer()
	if err != nil {
		t.Fatal(err)
	}
	defer mongoServer.Stop()

	// Connect to the in-memory MongoDB server
	clientOptions := options.Client().ApplyURI(mongoServer.URI())
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := client.Disconnect(context.TODO())
		if err != nil {
			t.Fatal(err)
		}
	}()

	// Create a test database and collection
	db := client.Database("bulwark")
	accountsRepo := NewMongodbAccountRepository(db, encryption.NewDefaultEncryption())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := accountsRepo.Create(context.TODO(), tt.email, tt.password)
			assert.Equal(t, nil, err)
			err = accountsRepo.UpdatePassword(context.TODO(), tt.email, tt.newPassword)
			assert.Equal(t, nil, err)
			if err == nil {
				match, err := accountsRepo.PasswordMatches(context.TODO(), tt.email, tt.newPassword)
				assert.Equal(t, nil, err)
				assert.Equal(t, tt.match, match)
			}
		})
	}
}
