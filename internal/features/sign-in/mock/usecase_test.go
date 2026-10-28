package mock

import (
	"context"
	"errors"
	"testing"

	"github.com/d0hx4n2503/bento-backend-golang/internal/features/auth"
	"github.com/d0hx4n2503/bento-backend-golang/internal/features/sign-in"
	"github.com/d0hx4n2503/bento-backend-golang/internal/features/sign-in/domain"
	signInUsecase "github.com/d0hx4n2503/bento-backend-golang/internal/features/sign-in/usecase"
	"github.com/d0hx4n2503/bento-backend-golang/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var errTestDependency = errors.New("test dependency failed")

func TestSignInUsecaseRejectsMissingDependencies(t *testing.T) {
	t.Parallel()

	provider := &signInProviderStub{}
	users := &signInRepositoryStub{}
	tokens := &signInTokenStub{}
	tests := []struct {
		name     string
		provider signin.Provider
		users    signin.MongoRepository
		tokens   auth.Usecase
	}{
		{name: "provider", users: users, tokens: tokens},
		{name: "repository", provider: provider, tokens: tokens},
		{name: "token usecase", provider: provider, users: users},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := signInUsecase.NewService(test.provider, test.users, test.tokens); err == nil {
				t.Fatal("NewService() error = nil, want missing dependency error")
			}
		})
	}
}

func TestSignInUsecaseCreatesSessionFromVerifiedGoogleIdentity(t *testing.T) {
	t.Parallel()

	provider := &signInProviderStub{identity: models.ProviderIdentity{
		Provider: models.ProviderGoogle,
		Subject:  "  google-subject  ",
		Email:    " User@Example.com ",
	}}
	users := &signInRepositoryStub{user: models.User{ID: bson.NewObjectID(), Email: "user@example.com"}}
	tokens := &signInTokenStub{accessToken: "access-token", refreshToken: "refresh-token"}
	usecase := newSignInUsecase(t, provider, users, tokens)

	accessToken, refreshToken, err := usecase.SignInWithGoogle(context.Background(), "google-credential")
	if err != nil {
		t.Fatalf("SignInWithGoogle() error = %v", err)
	}
	if accessToken != tokens.accessToken || refreshToken != tokens.refreshToken {
		t.Fatalf("SignInWithGoogle() tokens = (%q, %q), want (%q, %q)", accessToken, refreshToken, tokens.accessToken, tokens.refreshToken)
	}
	if provider.token != "google-credential" || provider.calls != 1 {
		t.Fatalf("provider calls/token = %d/%q, want 1/%q", provider.calls, provider.token, "google-credential")
	}
	wantIdentity := models.ProviderIdentity{Provider: models.ProviderGoogle, Subject: "google-subject", Email: "user@example.com"}
	if users.identity != wantIdentity || users.calls != 1 {
		t.Fatalf("repository calls/identity = %d/%+v, want 1/%+v", users.calls, users.identity, wantIdentity)
	}
	if tokens.calls != 1 || tokens.subject != users.user.ID.Hex() || tokens.email != users.user.Email || tokens.role != "" {
		t.Fatalf("token issuer calls/claims = %d/%q/%q/%q", tokens.calls, tokens.subject, tokens.email, tokens.role)
	}
}

func TestSignInUsecaseRejectsBlankCredentialWithoutCallingDependencies(t *testing.T) {
	t.Parallel()

	provider := &signInProviderStub{}
	users := &signInRepositoryStub{}
	tokens := &signInTokenStub{}
	usecase := newSignInUsecase(t, provider, users, tokens)

	if _, _, err := usecase.SignInWithGoogle(context.Background(), " \t"); !errors.Is(err, domain.ErrInvalidIdentity) {
		t.Fatalf("SignInWithGoogle() error = %v, want %v", err, domain.ErrInvalidIdentity)
	}
	if provider.calls != 0 || users.calls != 0 || tokens.calls != 0 {
		t.Fatalf("dependencies called for blank token: provider=%d repository=%d tokens=%d", provider.calls, users.calls, tokens.calls)
	}
}

func TestSignInUsecasePropagatesProviderFailure(t *testing.T) {
	t.Parallel()

	provider := &signInProviderStub{err: errTestDependency}
	users := &signInRepositoryStub{}
	tokens := &signInTokenStub{}
	usecase := newSignInUsecase(t, provider, users, tokens)

	if _, _, err := usecase.SignInWithGoogle(context.Background(), "credential"); !errors.Is(err, errTestDependency) {
		t.Fatalf("SignInWithGoogle() error = %v, want wrapped dependency error", err)
	}
	if users.calls != 0 || tokens.calls != 0 {
		t.Fatalf("downstream dependencies called after provider failure: repository=%d tokens=%d", users.calls, tokens.calls)
	}
}

func TestSignInUsecaseRejectsNonGoogleIdentity(t *testing.T) {
	t.Parallel()

	provider := &signInProviderStub{identity: models.ProviderIdentity{
		Provider: models.IdentityProvider("facebook"),
		Subject:  "provider-subject",
		Email:    "user@example.com",
	}}
	users := &signInRepositoryStub{}
	tokens := &signInTokenStub{}
	usecase := newSignInUsecase(t, provider, users, tokens)

	if _, _, err := usecase.SignInWithGoogle(context.Background(), "credential"); !errors.Is(err, domain.ErrInvalidIdentity) {
		t.Fatalf("SignInWithGoogle() error = %v, want %v", err, domain.ErrInvalidIdentity)
	}
	if users.calls != 0 || tokens.calls != 0 {
		t.Fatalf("downstream dependencies called for non-Google identity: repository=%d tokens=%d", users.calls, tokens.calls)
	}
}

func TestSignInUsecasePropagatesRepositoryAndTokenFailures(t *testing.T) {
	t.Parallel()

	t.Run("repository failure", func(t *testing.T) {
		users := &signInRepositoryStub{err: errTestDependency}
		tokens := &signInTokenStub{}
		usecase := newSignInUsecase(t, &signInProviderStub{identity: validGoogleIdentity()}, users, tokens)

		if _, _, err := usecase.SignInWithGoogle(context.Background(), "credential"); !errors.Is(err, errTestDependency) {
			t.Fatalf("SignInWithGoogle() error = %v, want wrapped repository error", err)
		}
		if tokens.calls != 0 {
			t.Fatalf("token issuer calls = %d, want 0", tokens.calls)
		}
	})

	t.Run("token issuer failure", func(t *testing.T) {
		tokens := &signInTokenStub{err: errTestDependency}
		usecase := newSignInUsecase(t, &signInProviderStub{identity: validGoogleIdentity()}, &signInRepositoryStub{user: models.User{ID: bson.NewObjectID(), Email: "user@example.com"}}, tokens)

		if _, _, err := usecase.SignInWithGoogle(context.Background(), "credential"); !errors.Is(err, errTestDependency) {
			t.Fatalf("SignInWithGoogle() error = %v, want wrapped token error", err)
		}
	})
}

func newSignInUsecase(t *testing.T, provider signin.Provider, users signin.MongoRepository, tokens auth.Usecase) signin.Usecase {
	t.Helper()
	usecase, err := signInUsecase.NewService(provider, users, tokens)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return usecase
}

func validGoogleIdentity() models.ProviderIdentity {
	return models.ProviderIdentity{Provider: models.ProviderGoogle, Subject: "google-subject", Email: "user@example.com"}
}

type signInProviderStub struct {
	identity models.ProviderIdentity
	err      error
	calls    int
	token    string
}

func (s *signInProviderStub) Verify(_ context.Context, token string) (models.ProviderIdentity, error) {
	s.calls++
	s.token = token
	return s.identity, s.err
}

type signInRepositoryStub struct {
	user     models.User
	err      error
	calls    int
	identity models.ProviderIdentity
}

func (s *signInRepositoryStub) EnsureIndexes(context.Context) error {
	return nil
}

func (s *signInRepositoryStub) FindOrCreateUser(_ context.Context, identity models.ProviderIdentity) (models.User, error) {
	s.calls++
	s.identity = identity
	return s.user, s.err
}

type signInTokenStub struct {
	accessToken  string
	refreshToken string
	err          error
	calls        int
	subject      string
	email        string
	role         string
}

func (s *signInTokenStub) Issue(subject, email, role string) (string, string, error) {
	s.calls++
	s.subject = subject
	s.email = email
	s.role = role
	return s.accessToken, s.refreshToken, s.err
}

func (s *signInTokenStub) Refresh(string) (string, string, error) {
	return "", "", nil
}

func (s *signInTokenStub) Authenticate(string) (models.Principal, error) {
	return models.Principal{}, nil
}

func (s *signInTokenStub) AdminLogin(string, string) (string, string, error) {
	return "", "", nil
}

var _ auth.Usecase = (*signInTokenStub)(nil)
var _ signin.MongoRepository = (*signInRepositoryStub)(nil)
var _ signin.Provider = (*signInProviderStub)(nil)
