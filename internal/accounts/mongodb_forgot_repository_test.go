package accounts

import (
	"context"
	"testing"

	"github.com/latebit-io/bulwarkauth/internal/utils"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoDbForgotRepository_Create(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{"Valid User", "test@latebit.io"},
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

	forgotRepo := NewMongoDbForgotRepository(db)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := forgotRepo.Create(context.Background(), tt.email); err != nil {
				t.Fatal(err)
			}
			token, err := forgotRepo.Read(context.Background(), tt.email)
			if err != nil {
				t.Fatal(err)
			}
			old := token.Token
			assert.Equal(t, tt.email, token.Email)

			err = forgotRepo.Create(context.Background(), tt.email)
			if err != nil {
				t.Fatal(err)
			}

			token, err = forgotRepo.Read(context.Background(), tt.email)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tt.email, token.Email)
			assert.NotEqual(t, old, token.Email)
			err = forgotRepo.Delete(context.Background(), tt.email)
			if err != nil {
				t.Fatal(err)
			}
			token, err = forgotRepo.Read(context.Background(), tt.email)
			assert.Equal(t, AccountNotFoundError{Value: tt.email}, err)
		})
	}
}
