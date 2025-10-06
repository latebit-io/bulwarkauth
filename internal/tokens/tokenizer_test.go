package tokens

import (
	"context"
	"strings"
	"testing"

	"github.com/latebit-io/bulwarkauth/internal/utils"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestDefaultTokenizer_CreateAccessToken(t *testing.T) {
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

	err = signingService.Initialize(context.TODO())
	if err != nil {
		t.Fatal(err)
	}

	tokenizer := NewDefaultTokenizer("test", "test", "test", 3600, 9600, signingService)
	a, err := tokenizer.CreateAccessToken(context.TODO(), "test@latebit.io", []string{"test:read-write"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 3, len(strings.Split(a, ".")))
	valid, err := tokenizer.ValidateAccessToken(context.TODO(), "test@latebit.io", a)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotEmpty(t, valid)
}

func TestDefaultTokenizer_RefreshAccessToken(t *testing.T) {
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

	err = signingService.Initialize(context.TODO())
	if err != nil {
		t.Fatal(err)
	}

	tokenizer := NewDefaultTokenizer("test", "test", "test", 3600, 9600, signingService)
	r, err := tokenizer.CreateRefreshToken(context.TODO(), "test@latebit.io")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 3, len(strings.Split(r, ".")))

	valid, err := tokenizer.ValidateRefreshToken(context.TODO(), "test@latebit.io", r)

	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, valid)
}
