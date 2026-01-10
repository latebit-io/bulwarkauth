package tenants

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	tenantCollection = "tenants"
)

type Tenant struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Domain      string `json:"domain"`
}

type TenantRepository interface {
	ReadAll(ctx context.Context) ([]Tenant, error)
	Read(ctx context.Context, tenantID string) (*Tenant, error)
	Create(ctx context.Context, tenantID string) error
}

type TenantService interface {
	ListTenants(ctx context.Context) ([]Tenant, error)
	GetTenant(ctx context.Context, tenantID string) (*Tenant, error)
	CreateDefault(ctx context.Context, tenantID string) error
}

type MongoDbTenantRepository struct {
	db *mongo.Database
}

func NewMongoDbTenantRepository(db *mongo.Database) TenantRepository {
	return &MongoDbTenantRepository{
		db: db,
	}
}

func (t *MongoDbTenantRepository) ReadAll(ctx context.Context) ([]Tenant, error) {
	collection := t.db.Collection(tenantCollection)
	var tenants []Tenant

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var tenant Tenant
		err := cursor.Decode(&tenant)
		if err != nil {
			return tenants, err
		}
		tenants = append(tenants, tenant)
	}
	if err := cursor.Err(); err != nil {
		return tenants, err
	}
	return tenants, nil
}

func (t *MongoDbTenantRepository) Read(ctx context.Context, tenantID string) (*Tenant, error) {
	collection := t.db.Collection(tenantCollection)
	filter := bson.M{"id": tenantID}
	var tenant Tenant
	err := collection.FindOne(ctx, filter).Decode(&tenant)
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (t *MongoDbTenantRepository) Create(ctx context.Context, tenantID string) error {
	collection := t.db.Collection(tenantCollection)
	tenant := Tenant{
		ID: tenantID,
	}
	_, err := collection.InsertOne(ctx, tenant)
	return err
}

type DefaultTenantService struct {
	repo TenantRepository
}

func NewDefaultTenantService(repo TenantRepository) TenantService {
	return &DefaultTenantService{
		repo: repo,
	}
}

func (s *DefaultTenantService) ListTenants(ctx context.Context) ([]Tenant, error) {
	return s.repo.ReadAll(ctx)
}

func (s *DefaultTenantService) GetTenant(ctx context.Context, tenantID string) (*Tenant, error) {
	return s.repo.Read(ctx, tenantID)
}

func (s *DefaultTenantService) CreateDefault(ctx context.Context, tenantID string) error {
	return s.repo.Create(ctx, tenantID)
}
