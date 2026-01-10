package authentication

import (
	"context"
	"testing"
	"time"

	"github.com/latebit-io/bulwarkauth/internal/utils"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoFailedAttemptRepository_Increment(t *testing.T) {
	ctx := context.Background()
	mongodb := utils.NewMongoTestUtil()
	mongoServer, err := mongodb.CreateServer()
	if err != nil {
		t.Fatal(err)
	}
	defer mongoServer.Stop()

	clientOptions := options.Client().ApplyURI(mongoServer.URI())
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("testdb")
	repo := NewMongoFailedAttemptRepository(db)
	tenantID := "tenant1"
	email := "test@example.com"

	// First increment
	err = repo.Increment(ctx, tenantID, email)
	if err != nil {
		t.Fatalf("Failed to increment: %v", err)
	}

	attempt, err := repo.Get(ctx, tenantID, email)
	if err != nil {
		t.Fatalf("Failed to get attempt: %v", err)
	}

	if attempt == nil {
		t.Fatal("Expected attempt to exist")
	}

	if attempt.Count != 1 {
		t.Errorf("Expected count to be 1, got %d", attempt.Count)
	}

	// Second increment
	err = repo.Increment(ctx, tenantID, email)
	if err != nil {
		t.Fatalf("Failed to increment second time: %v", err)
	}

	attempt, err = repo.Get(ctx, tenantID, email)
	if err != nil {
		t.Fatalf("Failed to get attempt: %v", err)
	}

	if attempt.Count != 2 {
		t.Errorf("Expected count to be 2, got %d", attempt.Count)
	}
}

func TestMongoFailedAttemptRepository_Lock(t *testing.T) {
	ctx := context.Background()
	mongodb := utils.NewMongoTestUtil()
	mongoServer, err := mongodb.CreateServer()
	if err != nil {
		t.Fatal(err)
	}
	defer mongoServer.Stop()

	clientOptions := options.Client().ApplyURI(mongoServer.URI())
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("testdb")
	repo := NewMongoFailedAttemptRepository(db)
	tenantID := "tenant1"
	email := "test@example.com"

	// Increment to create record
	err = repo.Increment(ctx, tenantID, email)
	if err != nil {
		t.Fatalf("Failed to increment: %v", err)
	}

	// Lock for 15 minutes
	err = repo.Lock(ctx, tenantID, email, 15*time.Minute)
	if err != nil {
		t.Fatalf("Failed to lock: %v", err)
	}

	attempt, err := repo.Get(ctx, tenantID, email)
	if err != nil {
		t.Fatalf("Failed to get attempt: %v", err)
	}

	if attempt == nil {
		t.Fatal("Expected attempt to exist")
	}

	if time.Now().After(attempt.LockedUntil) {
		t.Error("Expected LockedUntil to be in the future")
	}

	expectedLockTime := time.Now().Add(15 * time.Minute)
	timeDiff := attempt.LockedUntil.Sub(expectedLockTime).Abs()
	if timeDiff > time.Second {
		t.Errorf("Expected LockedUntil to be around %v, got %v", expectedLockTime, attempt.LockedUntil)
	}
}

func TestMongoFailedAttemptRepository_Clear(t *testing.T) {
	ctx := context.Background()
	mongodb := utils.NewMongoTestUtil()
	mongoServer, err := mongodb.CreateServer()
	if err != nil {
		t.Fatal(err)
	}
	defer mongoServer.Stop()

	clientOptions := options.Client().ApplyURI(mongoServer.URI())
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("testdb")
	repo := NewMongoFailedAttemptRepository(db)
	tenantID := "tenant1"
	email := "test@example.com"

	// Create record
	err = repo.Increment(ctx, tenantID, email)
	if err != nil {
		t.Fatalf("Failed to increment: %v", err)
	}

	// Verify it exists
	attempt, err := repo.Get(ctx, tenantID, email)
	if err != nil {
		t.Fatalf("Failed to get attempt: %v", err)
	}
	if attempt == nil {
		t.Fatal("Expected attempt to exist")
	}

	// Clear it
	err = repo.Clear(ctx, tenantID, email)
	if err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	// Verify it's gone
	attempt, err = repo.Get(ctx, tenantID, email)
	if err != nil {
		t.Fatalf("Failed to get attempt: %v", err)
	}
	if attempt != nil {
		t.Error("Expected attempt to be nil after clear")
	}
}

func TestMongoFailedAttemptRepository_Get_NonExistent(t *testing.T) {
	ctx := context.Background()
	mongodb := utils.NewMongoTestUtil()
	mongoServer, err := mongodb.CreateServer()
	if err != nil {
		t.Fatal(err)
	}
	defer mongoServer.Stop()

	clientOptions := options.Client().ApplyURI(mongoServer.URI())
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("testdb")
	repo := NewMongoFailedAttemptRepository(db)
	tenantID := "tenant1"

	attempt, err := repo.Get(ctx, tenantID, "nonexistent@example.com")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if attempt != nil {
		t.Error("Expected attempt to be nil for nonexistent email")
	}
}

func TestFailedAttempts_AccountLockout(t *testing.T) {
	ctx := context.Background()
	mongodb := utils.NewMongoTestUtil()
	mongoServer, err := mongodb.CreateServer()
	if err != nil {
		t.Fatal(err)
	}
	defer mongoServer.Stop()

	clientOptions := options.Client().ApplyURI(mongoServer.URI())
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("testdb")
	repo := NewMongoFailedAttemptRepository(db)
	tenantID := "tenant1"
	email := "test@example.com"

	// Simulate 5 failed attempts
	for i := 0; i < 5; i++ {
		err := repo.Increment(ctx, tenantID, email)
		if err != nil {
			t.Fatalf("Failed to increment attempt %d: %v", i+1, err)
		}
	}

	attempt, err := repo.Get(ctx, tenantID, email)
	if err != nil {
		t.Fatalf("Failed to get attempt: %v", err)
	}

	if attempt.Count != 5 {
		t.Errorf("Expected count to be 5, got %d", attempt.Count)
	}

	// Lock the account
	err = repo.Lock(ctx, tenantID, email, 15*time.Minute)
	if err != nil {
		t.Fatalf("Failed to lock account: %v", err)
	}

	// Verify locked
	attempt, err = repo.Get(ctx, tenantID, email)
	if err != nil {
		t.Fatalf("Failed to get attempt: %v", err)
	}

	if !time.Now().Before(attempt.LockedUntil) {
		t.Error("Expected account to be locked")
	}
}

func TestMongoFailedAttemptRepository_IncrementAndLockIfNeeded(t *testing.T) {
	ctx := context.Background()
	mongodb := utils.NewMongoTestUtil()
	mongoServer, err := mongodb.CreateServer()
	if err != nil {
		t.Fatal(err)
	}
	defer mongoServer.Stop()

	clientOptions := options.Client().ApplyURI(mongoServer.URI())
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("testdb")
	repo := NewMongoFailedAttemptRepository(db)
	tenantID := "tenant1"
	email := "test@example.com"
	maxAttempts := 5
	lockDuration := 15 * time.Minute

	// First 4 attempts should not lock
	for i := 0; i < 4; i++ {
		locked, err := repo.IncrementAndLockIfNeeded(ctx, tenantID, email, maxAttempts, lockDuration)
		if err != nil {
			t.Fatalf("Failed on attempt %d: %v", i+1, err)
		}
		if locked {
			t.Errorf("Should not be locked on attempt %d", i+1)
		}
	}

	attempt, err := repo.Get(ctx, tenantID, email)
	if err != nil {
		t.Fatalf("Failed to get attempt: %v", err)
	}
	if attempt.Count != 4 {
		t.Errorf("Expected count to be 4, got %d", attempt.Count)
	}
	if !attempt.LockedUntil.IsZero() {
		t.Error("Account should not be locked yet")
	}

	// 5th attempt should lock
	locked, err := repo.IncrementAndLockIfNeeded(ctx, tenantID, email, maxAttempts, lockDuration)
	if err != nil {
		t.Fatalf("Failed on 5th attempt: %v", err)
	}
	if !locked {
		t.Error("Account should be locked on 5th attempt")
	}

	attempt, err = repo.Get(ctx, tenantID, email)
	if err != nil {
		t.Fatalf("Failed to get attempt: %v", err)
	}
	if attempt.Count != 5 {
		t.Errorf("Expected count to be 5, got %d", attempt.Count)
	}
	if attempt.LockedUntil.IsZero() {
		t.Error("LockedUntil should be set")
	}
	if !time.Now().Before(attempt.LockedUntil) {
		t.Error("Account should be locked (LockedUntil should be in future)")
	}

	// Verify lock duration is approximately correct
	expectedLockTime := time.Now().Add(lockDuration)
	timeDiff := attempt.LockedUntil.Sub(expectedLockTime).Abs()
	if timeDiff > time.Second {
		t.Errorf("LockedUntil duration incorrect, diff: %v", timeDiff)
	}
}
