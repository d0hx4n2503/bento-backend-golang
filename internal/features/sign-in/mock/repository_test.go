package mock

import (
	"context"
	"errors"
	"testing"

	"github.com/d0hx4n2503/bento-backend-golang/internal/features/sign-in"
	"github.com/d0hx4n2503/bento-backend-golang/internal/features/sign-in/domain"
	signInRepository "github.com/d0hx4n2503/bento-backend-golang/internal/features/sign-in/repository/mongodb"
	"github.com/d0hx4n2503/bento-backend-golang/internal/models"
	mongoDriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestNewUserRepositoryRequiresDatabase(t *testing.T) {
	t.Parallel()

	if _, err := signInRepository.NewUserRepository(nil); err == nil {
		t.Fatal("NewUserRepository(nil) error = nil, want validation error")
	}
}

func TestFindOrCreateUserRejectsInvalidAndUnsupportedIdentity(t *testing.T) {
	t.Parallel()

	repository := newSignInRepository(t)
	tests := []struct {
		name     string
		identity models.ProviderIdentity
	}{
		{name: "missing subject", identity: models.ProviderIdentity{Provider: models.ProviderGoogle, Email: "user@example.com"}},
		{name: "invalid email", identity: models.ProviderIdentity{Provider: models.ProviderGoogle, Subject: "google-subject", Email: "invalid"}},
		{name: "unsupported provider", identity: models.ProviderIdentity{Provider: models.IdentityProvider("facebook"), Subject: "facebook-subject", Email: "user@example.com"}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := repository.FindOrCreateUser(context.Background(), test.identity); !errors.Is(err, domain.ErrInvalidIdentity) {
				t.Fatalf("FindOrCreateUser() error = %v, want %v", err, domain.ErrInvalidIdentity)
			}
		})
	}
}

func TestFindOrCreateUserHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	repository := newSignInRepository(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := repository.FindOrCreateUser(ctx, models.ProviderIdentity{
		Provider: models.ProviderGoogle,
		Subject:  "google-subject",
		Email:    "user@example.com",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("FindOrCreateUser() error = %v, want context.Canceled", err)
	}
}

func TestEnsureIndexesHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	repository := newSignInRepository(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repository.EnsureIndexes(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("EnsureIndexes() error = %v, want context.Canceled", err)
	}
}

func newSignInRepository(t *testing.T) signin.MongoRepository {
	t.Helper()

	client, err := mongoDriver.Connect(options.Client().ApplyURI("mongodb://127.0.0.1:27017"))
	if err != nil {
		t.Fatalf("connect MongoDB client: %v", err)
	}
	repository, err := signInRepository.NewUserRepository(client.Database("signin_repository_test"))
	if err != nil {
		t.Fatalf("NewUserRepository() error = %v", err)
	}
	t.Cleanup(func() {
		if err := client.Disconnect(context.Background()); err != nil {
			t.Errorf("disconnect MongoDB client: %v", err)
		}
	})
	return repository
}
