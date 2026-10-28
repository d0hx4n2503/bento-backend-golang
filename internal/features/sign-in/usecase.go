package signin

import "context"

// Usecase is the public contract for Google sign-in.
type Usecase interface {
	SignInWithGoogle(ctx context.Context, token string) (accessToken, refreshToken string, err error)
}
