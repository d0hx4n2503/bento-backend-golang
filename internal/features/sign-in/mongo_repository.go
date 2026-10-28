package signin

import (
	"context"

	"github.com/d0hx4n2503/bento-backend-golang/internal/models"
)

// MongoRepository is the persistence contract required by sign-in.
type MongoRepository interface {
	EnsureIndexes(ctx context.Context) error
	FindOrCreateUser(ctx context.Context, identity models.ProviderIdentity) (models.User, error)
}
