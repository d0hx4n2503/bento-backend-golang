package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/d0hx4n2503/bento-backend-golang/internal/features/auth"
	"github.com/d0hx4n2503/bento-backend-golang/internal/features/signin"
	"github.com/d0hx4n2503/bento-backend-golang/internal/features/signin/domain"
	"github.com/d0hx4n2503/bento-backend-golang/internal/models"
)

// service coordinates provider verification, account lookup, and session issuance.
type service struct {
	google signin.Provider
	users  signin.MongoRepository
	tokens auth.Usecase
}

func NewService(google signin.Provider, users signin.MongoRepository, tokens auth.Usecase) (signin.Usecase, error) {
	if google == nil || users == nil || tokens == nil {
		return nil, fmt.Errorf("identity verifier, user repository, and token issuer are required")
	}
	return &service{google: google, users: users, tokens: tokens}, nil
}

// SignInWithGoogle resolves the submitted Google credential before creating a session.
func (s *service) SignInWithGoogle(ctx context.Context, token string) (string, string, error) {
	if strings.TrimSpace(token) == "" {
		return "", "", domain.ErrInvalidIdentity
	}

	identity, err := s.google.Verify(ctx, token)
	if err != nil {
		return "", "", fmt.Errorf("verify Google identity: %w", err)
	}
	identity, err = domain.NormalizeIdentity(identity)
	if err != nil {
		return "", "", err
	}
	if identity.Provider != models.ProviderGoogle {
		return "", "", domain.ErrInvalidIdentity
	}

	user, err := s.users.FindOrCreateUser(ctx, identity)
	if err != nil {
		return "", "", fmt.Errorf("find or create user: %w", err)
	}
	accessToken, refreshToken, err := s.tokens.Issue(user.ID.Hex(), user.Email, "")
	if err != nil {
		return "", "", fmt.Errorf("issue user session: %w", err)
	}
	return accessToken, refreshToken, nil
}

var _ signin.Usecase = (*service)(nil)
