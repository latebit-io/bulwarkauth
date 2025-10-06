package accounts

import (
	"context"
	"errors"
	"testing"

	"github.com/latebit-io/bulwarkauth/internal/encryption"
	"github.com/latebit-io/bulwarkauth/internal/tokens"
	"github.com/latebit-io/bulwarkauth/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

//go:generate mockery --name=EmailService

func TestDefaultAccountService_Create(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		password    string
		expectedErr error
	}{
		{"Valid User", "test@latebit.io", "password", nil},
		{"Empty email", "", "password", errors.New("email is required")},
		{"Empty password", "test2@latebit.io", "", errors.New("password is required")},
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

	db := client.Database("bulwark-test")
	mongodbTxManager := utils.NewMongoTxManager(client)
	accountRepo := NewMongodbAccountRepository(db, encryption.NewDefaultEncryption())
	forgotRepo := NewMongoDbForgotRepository(db)
	signingRepo := tokens.NewDefaultSigningKeyRepository(db)
	signingService := tokens.NewDefaultSigningKeyService(signingRepo)
	tokenizer := tokens.NewDefaultTokenizer("test", "test", "test", 3600,
		9600, signingService)
	mockEmailService := &MockEmailService{}
	mockEmailService.On("SendVerificationEmail", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	accountService := NewDefaultAccountService(accountRepo, forgotRepo, tokenizer, mockEmailService, mongodbTxManager)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := accountService.Create(context.TODO(), tt.email, tt.password)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestDefaultAccountService_Verification(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		password    string
		expectedErr error
	}{
		{"Valid User", "test@latebit.io", "password", nil},
		//{"Non Valid User", "", "password", errors.New("email is required")},
		//{"Empty password", "test2@latebit.io", "", errors.New("password is required")},
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

	db := client.Database("bulwark-test")
	mongodbTxManager := utils.NewMongoTxManager(client)
	accountRepo := NewMongodbAccountRepository(db, encryption.NewDefaultEncryption())
	forgotRepo := NewMongoDbForgotRepository(db)
	signingRepo := tokens.NewDefaultSigningKeyRepository(db)
	signingService := tokens.NewDefaultSigningKeyService(signingRepo)
	tokenizer := tokens.NewDefaultTokenizer("test", "test", "test", 3600,
		9600, signingService)
	mockEmailService := &MockEmailService{}
	mockEmailService.On("SendVerificationEmail", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	accountService := NewDefaultAccountService(accountRepo, forgotRepo, tokenizer, mockEmailService, mongodbTxManager)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := accountService.Create(context.TODO(), tt.email, tt.password)
			if err != nil {
				t.Fatal(err)
			}
			account, err := accountRepo.Read(context.TODO(), tt.email)
			if err != nil {
				t.Fatal(err)
			}
			err = accountService.Verify(context.TODO(), account.Email, account.VerificationToken)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}
