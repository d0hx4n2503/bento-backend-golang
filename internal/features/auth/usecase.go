package auth

import "github.com/d0hx4n2503/bento-backend-golang/internal/models"

// Usecase is the public contract for session and admin authentication flows.
type Usecase interface {
	Issue(subject, email, role string) (accessToken, refreshToken string, err error)
	Refresh(token string) (accessToken, refreshToken string, err error)
	Authenticate(token string) (models.Principal, error)
	AdminLogin(username, password string) (accessToken, refreshToken string, err error)
}
