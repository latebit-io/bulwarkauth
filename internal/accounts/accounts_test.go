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

func TestDefaultAccountService_Register(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		password    string
		expectedErr error
	}{
		{"Valid User", "test@latebit.io", "password", nil},
		{"Empty email", "", "password", errors.New("invalid email format")},
		{"Empty password", "test2@latebit.io", "", errors.New("password must be at least 8 characters")},
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
	accountRepo := NewMongodbAccountRepository(db, encryption.NewDefaultEncryption(12))
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
			err := accountService.Register(context.TODO(), tt.email, tt.password)
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
		//{"Non Valid User", "", "password", errors.New("invalid email format")},
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
	accountRepo := NewMongodbAccountRepository(db, encryption.NewDefaultEncryption(12))
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
			err := accountService.Register(context.TODO(), tt.email, tt.password)
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

func TestDefaultAccountService_UpdatePassword_WithMismatchedToken(t *testing.T) {
	mongodb := utils.NewMongoTestUtil()
	mongoServer, err := mongodb.CreateServer()
	if err != nil {
		t.Fatal(err)
	}
	defer mongoServer.Stop()

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
	accountRepo := NewMongodbAccountRepository(db, encryption.NewDefaultEncryption(12))
	forgotRepo := NewMongoDbForgotRepository(db)
	signingRepo := tokens.NewDefaultSigningKeyRepository(db)
	signingService := tokens.NewDefaultSigningKeyService(signingRepo)
	err = signingService.Initialize(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	tokenizer := tokens.NewDefaultTokenizer("test", "test", "test", 3600,
		9600, signingService)
	mockEmailService := &MockEmailService{}
	mockEmailService.On("SendVerificationEmail", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	accountService := NewDefaultAccountService(accountRepo, forgotRepo, tokenizer, mockEmailService, mongodbTxManager)

	// Create two users
	user1Email := "user1@example.com"
	user2Email := "user2@example.com"

	err = accountService.Register(context.TODO(), user1Email, "password123")
	if err != nil {
		t.Fatal(err)
	}

	err = accountService.Register(context.TODO(), user2Email, "password456")
	if err != nil {
		t.Fatal(err)
	}

	// Verify both accounts
	account1, err := accountRepo.Read(context.TODO(), user1Email)
	if err != nil {
		t.Fatal(err)
	}
	err = accountService.Verify(context.TODO(), account1.Email, account1.VerificationToken)
	if err != nil {
		t.Fatal(err)
	}

	account2, err := accountRepo.Read(context.TODO(), user2Email)
	if err != nil {
		t.Fatal(err)
	}
	err = accountService.Verify(context.TODO(), account2.Email, account2.VerificationToken)
	if err != nil {
		t.Fatal(err)
	}

	// Create access token for user1
	user1Token, err := tokenizer.CreateAccessToken(context.TODO(), user1Email, "client1", []string{"user"})
	if err != nil {
		t.Fatal(err)
	}

	// Test 1: Valid scenario - user1 updates their own password
	t.Run("User updates own password - should succeed", func(t *testing.T) {
		err := accountService.UpdatePassword(context.TODO(), user1Email, "newpassword123", user1Token)
		assert.NoError(t, err)
	})

	// Test 2: Security vulnerability - user1 tries to update user2's password using user1's token
	t.Run("User tries to update another user's password - should fail", func(t *testing.T) {
		err := accountService.UpdatePassword(context.TODO(), user2Email, "hacked1234", user1Token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token invalid")

		// Verify user2's password was NOT changed - original password should still work
		matched, err := accountRepo.PasswordMatches(context.TODO(), user2Email, "password456")
		assert.NoError(t, err)
		assert.True(t, matched, "user2's original password should still be valid")
	})

	// Test 3: Invalid token
	t.Run("Invalid token - should fail", func(t *testing.T) {
		err := accountService.UpdatePassword(context.TODO(), user1Email, "newpassword", "invalid-token")
		assert.Error(t, err)
	})
}

func TestDefaultAccountService_Delete_WithMismatchedToken(t *testing.T) {
	mongodb := utils.NewMongoTestUtil()
	mongoServer, err := mongodb.CreateServer()
	if err != nil {
		t.Fatal(err)
	}
	defer mongoServer.Stop()

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
	accountRepo := NewMongodbAccountRepository(db, encryption.NewDefaultEncryption(12))
	forgotRepo := NewMongoDbForgotRepository(db)
	signingRepo := tokens.NewDefaultSigningKeyRepository(db)
	signingService := tokens.NewDefaultSigningKeyService(signingRepo)
	err = signingService.Initialize(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	tokenizer := tokens.NewDefaultTokenizer("test", "test", "test", 3600,
		9600, signingService)
	mockEmailService := &MockEmailService{}
	mockEmailService.On("SendVerificationEmail", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	accountService := NewDefaultAccountService(accountRepo, forgotRepo, tokenizer, mockEmailService, mongodbTxManager)

	// Create two users
	user1Email := "user1@example.com"
	user2Email := "user2@example.com"

	err = accountService.Register(context.TODO(), user1Email, "password123")
	if err != nil {
		t.Fatal(err)
	}

	err = accountService.Register(context.TODO(), user2Email, "password456")
	if err != nil {
		t.Fatal(err)
	}

	// Verify both accounts
	account1, err := accountRepo.Read(context.TODO(), user1Email)
	if err != nil {
		t.Fatal(err)
	}
	err = accountService.Verify(context.TODO(), account1.Email, account1.VerificationToken)
	if err != nil {
		t.Fatal(err)
	}

	account2, err := accountRepo.Read(context.TODO(), user2Email)
	if err != nil {
		t.Fatal(err)
	}
	err = accountService.Verify(context.TODO(), account2.Email, account2.VerificationToken)
	if err != nil {
		t.Fatal(err)
	}

	// Create access token for user1
	user1Token, err := tokenizer.CreateAccessToken(context.TODO(), user1Email, "client1", []string{"user"})
	if err != nil {
		t.Fatal(err)
	}

	// Test 1: Security vulnerability - user1 tries to delete user2's account using user1's token
	t.Run("User tries to delete another user's account - should fail", func(t *testing.T) {
		err := accountService.Delete(context.TODO(), user2Email, user1Token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token invalid")

		// Verify user2's account still exists and is not deleted
		account2After, err := accountRepo.Read(context.TODO(), user2Email)
		assert.NoError(t, err)
		assert.NotNil(t, account2After)
		assert.False(t, account2After.IsDeleted, "user2's account should not be deleted")
	})

	// Test 2: Valid scenario - user1 deletes their own account
	t.Run("User deletes own account - should succeed", func(t *testing.T) {
		err := accountService.Delete(context.TODO(), user1Email, user1Token)
		assert.NoError(t, err)

		// Verify user1's account is marked as deleted
		account1After, err := accountRepo.Read(context.TODO(), user1Email)
		assert.NoError(t, err)
		assert.True(t, account1After.IsDeleted)
	})

	// Test 3: Invalid token
	t.Run("Invalid token - should fail", func(t *testing.T) {
		err := accountService.Delete(context.TODO(), user2Email, "invalid-token")
		assert.Error(t, err)
	})
}
