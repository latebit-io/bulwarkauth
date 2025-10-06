package domain

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tryvium-travels/memongo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestDefaultDomainService_GenerateVerification(t *testing.T) {
	// Setup database
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create repository and service
	domainRepo := NewDefaultDomainRepository(db)
	domainService := NewDefaultDomainService(domainRepo, "latebit.io-test")

	// Test cases
	tests := []struct {
		name        string
		domain      string
		setupDB     func(context.Context) error // Optional setup for specific test case
		expectedErr error
		validate    func(t *testing.T, verification *DomainVerification)
	}{
		{
			name:        "Happy path - valid domain",
			domain:      "latebit.io",
			expectedErr: nil,
			validate: func(t *testing.T, v *DomainVerification) {
				assert.Equal(t, "latebit.io", v.Domain)
				assert.NotEmpty(t, v.VerifyToken)
				assert.Contains(t, v.DNSRecord, "bulwark-verify=")
				assert.False(t, v.Verified)
			},
		},
		{
			name:        "Error path - empty domain",
			domain:      "",
			expectedErr: errors.New("domain cannot be empty"),
			validate:    nil, // No validation for error cases
		},
		{
			name:   "Error path - duplicate domain",
			domain: "duplicate.com",
			setupDB: func(ctx context.Context) error {
				// Create a pre-existing record
				existing := &DomainVerification{
					Domain:      "duplicate.com",
					VerifyToken: "existing-token",
					DNSRecord:   "bulwark-verify=existing-token",
					Verified:    false,
					CreatedAt:   time.Now(),
					ModifiedAt:  time.Now(),
				}
				return domainRepo.Create(ctx, existing)
			},
			expectedErr: DuplicateVerificationError{
				Value: "duplicate.com",
			},
			validate: nil,
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Optional test-specific setup
			if tt.setupDB != nil {
				err := tt.setupDB(ctx)
				if err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			// Run the function being tested
			verification, err := domainService.GenerateVerification(ctx, tt.domain, "bulwark-salt")
			//assert.Equal(t, tt.expectedErr, err)
			//assert.Equal(t, tt.expectedErr, verification)

			// Assert on errors
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
				assert.Nil(t, verification)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, verification)

				// Run validation function if provided
				if tt.validate != nil {
					tt.validate(t, verification)
				}

				// Verify it was stored in the database
				stored, err := domainRepo.Read(ctx, tt.domain)
				assert.NoError(t, err)
				assert.Equal(t, verification.VerifyToken, stored.VerifyToken)
			}
		})
	}
}

func TestDefaultDomainService_GetAll(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create repository and service
	domainRepo := NewDefaultDomainRepository(db)
	domainService := NewDefaultDomainService(domainRepo, "latebit.io-test")

	tests := []struct {
		name        string
		domain      string
		setupDB     func(context.Context) error // Optional setup for specific test case
		expectedErr error
		validate    func(t *testing.T, domains []DomainVerification)
	}{
		{
			name:        "Happy path - valid domain",
			domain:      "latebit.io",
			expectedErr: nil,
			validate: func(t *testing.T, d []DomainVerification) {
				assert.Len(t, d, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err := domainService.GenerateVerification(ctx, tt.domain, "bulwark-salt")
			if err != nil {
				t.Fatalf("Failed to generate verification: %v", err)
			}

			domains, err := domainService.GetAll(ctx)
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
				assert.Nil(t, domains)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, domains)
				if tt.validate != nil {
					tt.validate(t, domains)
				}
			}
			defer cancel()
		})
	}
}

func TestDefaultDomainService_Verify(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create repository and service
	domainRepo := NewDefaultDomainRepository(db)
	domainService := NewDefaultDomainService(domainRepo, "latebit.io-test")

	tests := []struct {
		name        string
		domain      string
		setupDB     func(context.Context) error // Optional setup for specific test case
		expectedErr error
		validate    func(t *testing.T, service DomainService)
	}{
		{
			name:        "Error path - validate domain",
			domain:      "latebit.io",
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		verification, err := domainService.GenerateVerification(ctx, tt.domain, "bulwark-salt")
		if err != nil {
			t.Fatalf("Failed to generate verification: %v", err)
		}
		err = domainService.Verify(ctx, verification.Domain)
		if tt.expectedErr != nil {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr.Error())
		} else {
			assert.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, domainService)
			}
		}
		defer cancel()
	}
}

// setupMongoServer creates and starts an in-memory MongoDB server with appropriate options
func setupMongoServer() (*memongo.Server, error) {
	opts := &memongo.Options{
		MongoVersion: "7.0.0",
	}

	if runtime.GOARCH == "arm64" && runtime.GOOS == "darwin" {
		opts.DownloadURL = "https://fastdl.mongodb.org/osx/mongodb-macos-x86_64-7.0.0.tgz"
	}

	return memongo.StartWithOptions(opts)
}

// connectToMongo connects to a MongoDB server and returns the client and test database
func connectToMongo(uri string) (*mongo.Client, *mongo.Database, error) {
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return nil, nil, err
	}

	db := client.Database("bulwark_test")
	return client, db, nil
}

// setupTestDB creates a test database and returns a cleanup function
func setupTestDB(t *testing.T) (*mongo.Database, func()) {
	server, err := setupMongoServer()
	if err != nil {
		t.Fatalf("Failed to start MongoDB server: %v", err)
	}

	client, db, err := connectToMongo(server.URI())
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	cleanup := func() {
		if err := client.Disconnect(context.Background()); err != nil {
			t.Errorf("Failed to disconnect MongoDB client: %v", err)
		}
		server.Stop()
	}

	return db, cleanup
}
