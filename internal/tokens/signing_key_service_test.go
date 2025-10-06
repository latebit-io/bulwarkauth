package tokens

import (
	"context"
	"testing"

	"github.com/latebit-io/bulwarkauth/internal/utils"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestDefaultSigningKeyService_GenerateKey(t *testing.T) {
	tests := []struct {
		name        string
		expectedErr error
	}{
		{"generate", nil},
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

	db := client.Database("bulwark")
	signingRepo := NewDefaultSigningKeyRepository(db)
	signingService := NewDefaultSigningKeyService(signingRepo)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = signingService.Initialize(context.Background())
			assert.Equal(t, tt.expectedErr, err)
			err := signingService.GenerateKey(context.TODO())
			assert.Equal(t, tt.expectedErr, err)
			keys, err := signingService.GetAllKeys(context.TODO())
			assert.Equal(t, tt.expectedErr, err)
			assert.NotEmpty(t, keys)
		})
	}
}
