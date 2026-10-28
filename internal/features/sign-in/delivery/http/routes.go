package http

import (
	"github.com/d0hx4n2503/bento-backend-golang/internal/features/sign-in"
	"github.com/labstack/echo/v5"
)

// MapSignInRoutes registers Google sign-in endpoints on the provided group.
func MapSignInRoutes(signInGroup *echo.Group, handlers signin.Handlers) {
	signInGroup.POST("/google", handlers.GoogleSignIn())
}
