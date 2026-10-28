package mock

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/d0hx4n2503/bento-backend-golang/internal/features/sign-in"
	"github.com/d0hx4n2503/bento-backend-golang/internal/features/sign-in/domain"
	signInRepository "github.com/d0hx4n2503/bento-backend-golang/internal/features/sign-in/repository/mongodb"
	"github.com/d0hx4n2503/bento-backend-golang/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	mongoDriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestMongoRepositoryCreatesAndUpdatesGoogleIdentity(t *testing.T) {
	t.Parallel()

	database, repository := newMongoIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := repository.EnsureIndexes(ctx); err != nil {
		t.Fatalf("EnsureIndexes() error = %v", err)
	}

	first, err := repository.FindOrCreateUser(ctx, models.ProviderIdentity{
		Provider: models.ProviderGoogle,
		Subject:  "google-subject-1",
		Email:    "first@example.com",
	})
	if err != nil {
		t.Fatalf("FindOrCreateUser() create error = %v", err)
	}
	if first.ID.IsZero() || first.GoogleID != "google-subject-1" || first.ProviderIDs[string(models.ProviderGoogle)] != "google-subject-1" {
		t.Fatalf("created user has incomplete identity data: %+v", first)
	}

	updated, err := repository.FindOrCreateUser(ctx, models.ProviderIdentity{
		Provider: models.ProviderGoogle,
		Subject:  "google-subject-1",
		Email:    "updated@example.com",
	})
	if err != nil {
		t.Fatalf("FindOrCreateUser() update error = %v", err)
	}
	if updated.ID != first.ID || updated.Email != "updated@example.com" {
		t.Fatalf("updated user = %+v, want same ID %s and updated email", updated, first.ID.Hex())
	}

	var stored models.User
	if err := database.Collection("users").FindOne(ctx, bson.D{{Key: "_id", Value: first.ID}}).Decode(&stored); err != nil {
		t.Fatalf("read stored user: %v", err)
	}
	if stored.Email != "updated@example.com" || stored.GoogleID != "google-subject-1" {
		t.Fatalf("stored user = %+v, want updated email and Google identity", stored)
	}
}

func TestMongoRepositoryDoesNotLinkAccountsByEmail(t *testing.T) {
	t.Parallel()

	database, repository := newMongoIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := repository.EnsureIndexes(ctx); err != nil {
		t.Fatalf("EnsureIndexes() error = %v", err)
	}

	existing := models.User{
		ID:          bson.NewObjectID(),
		Email:       "existing@example.com",
		GoogleID:    "existing-google-subject",
		ProviderIDs: map[string]string{string(models.ProviderGoogle): "existing-google-subject"},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if _, err := database.Collection("users").InsertOne(ctx, existing); err != nil {
		t.Fatalf("insert existing user: %v", err)
	}

	_, err := repository.FindOrCreateUser(ctx, models.ProviderIdentity{
		Provider: models.ProviderGoogle,
		Subject:  "different-google-subject",
		Email:    existing.Email,
	})
	if !errors.Is(err, domain.ErrIdentityConflict) {
		t.Fatalf("FindOrCreateUser() error = %v, want %v", err, domain.ErrIdentityConflict)
	}
}

func newMongoIntegrationRepository(t *testing.T) (*mongoDriver.Database, signin.MongoRepository) {
	t.Helper()

	uri := strings.TrimSpace(os.Getenv("MONGODB_TEST_URI"))
	if uri == "" {
		t.Skip("set MONGODB_TEST_URI to run MongoDB repository integration tests")
	}

	connectContext, cancelConnect := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelConnect()
	client, err := mongoDriver.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect MongoDB client: %v", err)
	}
	if err := client.Ping(connectContext, nil); err != nil {
		_ = client.Disconnect(context.Background())
		t.Fatalf("ping MongoDB test instance: %v", err)
	}

	database := client.Database("bento_signin_test_" + bson.NewObjectID().Hex())
	repository, err := signInRepository.NewUserRepository(database)
	if err != nil {
		_ = client.Disconnect(context.Background())
		t.Fatalf("NewUserRepository() error = %v", err)
	}
	t.Cleanup(func() {
		dropContext, cancelDrop := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelDrop()
		if err := database.Drop(dropContext); err != nil {
			t.Errorf("drop MongoDB test database: %v", err)
		}
		if err := client.Disconnect(dropContext); err != nil {
			t.Errorf("disconnect MongoDB client: %v", err)
		}
	})
	return database, repository
}
