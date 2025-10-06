package email

import (
	"context"
	"testing"

	"github.com/latebit-io/bulwarkauth/internal/utils"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestDefaultEmailRepository_Create(t *testing.T) {
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
	template := `
		<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"
        "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
		<html>
		</head>

		<body>
		<p>
    		Hello {{.Name}}
    		<a href="{{.URL}}">Verify email address</a>
		</p></body></html>`

	emailRepo := NewMongoDbEmailRepository(db)
	err = emailRepo.Create(context.TODO(), "test", template)

	if err != nil {
		t.Fatal(err)
	}

	readTemplate, err := emailRepo.Read(context.TODO(), "test")
	if err != nil {
		t.Fatal(err)
	}
	//assert.Equal(t, "test", readTemplate.Name)
	assert.Equal(t, template, readTemplate)
}
