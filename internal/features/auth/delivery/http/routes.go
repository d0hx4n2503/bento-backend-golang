package http

import (
	"github.com/d0hx4n2503/bento-backend-golang/internal/features/auth"
	"github.com/labstack/echo/v5"
)

// MapAuthRoutes registers authentication endpoints on the provided auth group.
func MapAuthRoutes(authGroup *echo.Group, handlers auth.Handlers, authenticate echo.MiddlewareFunc) {
	authGroup.POST("/admin-login", handlers.AdminLogin())
	authGroup.POST("/refresh", handlers.Refresh())
	authGroup.POST("/logout", handlers.Logout())
	authGroup.GET("/check", handlers.Check(), authenticate)
}
