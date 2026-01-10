package tenants

import (
	"context"
	"testing"

	"github.com/latebit-io/bulwarkauth/internal/utils"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoDbTenantRepository_Create(t *testing.T) {
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
	tenantRepo := NewMongoDbTenantRepository(db)

	tenantID := "tenant1"
	err = tenantRepo.Create(context.TODO(), tenantID)
	assert.NoError(t, err)

	tenant, err := tenantRepo.Read(context.TODO(), tenantID)
	assert.NoError(t, err)
	assert.NotNil(t, tenant)
	assert.Equal(t, tenantID, tenant.ID)
}

func TestMongoDbTenantRepository_Read(t *testing.T) {
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
	tenantRepo := NewMongoDbTenantRepository(db)

	tenantID := "tenant1"
	err = tenantRepo.Create(context.TODO(), tenantID)
	assert.NoError(t, err)

	tenant, err := tenantRepo.Read(context.TODO(), tenantID)
	assert.NoError(t, err)
	assert.NotNil(t, tenant)
	assert.Equal(t, tenantID, tenant.ID)
}

func TestMongoDbTenantRepository_ReadAll(t *testing.T) {
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
	tenantRepo := NewMongoDbTenantRepository(db)

	tenant1ID := "tenant1"
	tenant2ID := "tenant2"

	err = tenantRepo.Create(context.TODO(), tenant1ID)
	assert.NoError(t, err)

	err = tenantRepo.Create(context.TODO(), tenant2ID)
	assert.NoError(t, err)

	tenants, err := tenantRepo.ReadAll(context.TODO())
	assert.NoError(t, err)
	assert.Len(t, tenants, 2)
}

func TestDefaultTenantService_CreateDefault(t *testing.T) {
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
	tenantRepo := NewMongoDbTenantRepository(db)
	tenantService := NewDefaultTenantService(tenantRepo)

	tenantID := "tenant1"
	err = tenantService.CreateDefault(context.TODO(), tenantID)
	assert.NoError(t, err)

	tenant, err := tenantService.GetTenant(context.TODO(), tenantID)
	assert.NoError(t, err)
	assert.NotNil(t, tenant)
	assert.Equal(t, tenantID, tenant.ID)
}

func TestDefaultTenantService_ListTenants(t *testing.T) {
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
	tenantRepo := NewMongoDbTenantRepository(db)
	tenantService := NewDefaultTenantService(tenantRepo)

	tenant1ID := "tenant1"
	tenant2ID := "tenant2"

	err = tenantService.CreateDefault(context.TODO(), tenant1ID)
	assert.NoError(t, err)

	err = tenantService.CreateDefault(context.TODO(), tenant2ID)
	assert.NoError(t, err)

	tenants, err := tenantService.ListTenants(context.TODO())
	assert.NoError(t, err)
	assert.Len(t, tenants, 2)
}
