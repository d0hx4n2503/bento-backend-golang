package domain

import (
	"strings"

	"github.com/d0hx4n2503/bento-backend-golang/internal/models"
)

// NormalizeIdentity trims and validates a provider identity before persistence.
func NormalizeIdentity(identity models.ProviderIdentity) (models.ProviderIdentity, error) {
	identity.Provider = models.IdentityProvider(strings.ToLower(strings.TrimSpace(string(identity.Provider))))
	identity.Subject = strings.TrimSpace(identity.Subject)
	identity.Email = strings.ToLower(strings.TrimSpace(identity.Email))
	if identity.Provider == "" || identity.Subject == "" || identity.Email == "" || !strings.Contains(identity.Email, "@") {
		return models.ProviderIdentity{}, ErrInvalidIdentity
	}
	return identity, nil
}
