package signin

import (
	"context"

	"github.com/d0hx4n2503/bento-backend-golang/internal/models"
)

// Provider resolves a provider credential into a verified identity.
type Provider interface {
	Verify(ctx context.Context, credential string) (models.ProviderIdentity, error)
}
