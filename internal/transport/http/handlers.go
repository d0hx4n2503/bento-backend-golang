package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	authHTTP "github.com/d0hx4n2503/bento-backend-golang/internal/features/auth/delivery/http"
	authUC "github.com/d0hx4n2503/bento-backend-golang/internal/features/auth/usecase"
	signInHTTP "github.com/d0hx4n2503/bento-backend-golang/internal/features/signin/delivery/http"
	googleProvider "github.com/d0hx4n2503/bento-backend-golang/internal/features/signin/providers/google"
	signInMgRp "github.com/d0hx4n2503/bento-backend-golang/internal/features/signin/repository/mongodb"
	signInUC "github.com/d0hx4n2503/bento-backend-golang/internal/features/signin/usecase"
	"github.com/d0hx4n2503/bento-backend-golang/internal/transport/http/middleware"
	"github.com/labstack/echo/v5"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// MapHandlers initializes feature dependencies and registers API route groups.
func (s *Server) MapHandlers(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("handler context is required")
	}
	userRepository, err := signInMgRp.NewUserRepository(s.dependencies.MongoClient.Database(s.cfg.MongoDB.DatabaseName))
	if err != nil {
		return fmt.Errorf("initialize user repository: %w", err)
	}
	indexCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	err = userRepository.EnsureIndexes(indexCtx)
	cancel()
	if err != nil {
		return fmt.Errorf("initialize sign-in indexes: %w", err)
	}

	sessions, err := authUC.NewService(s.cfg.Auth)
	if err != nil {
		return fmt.Errorf("initialize authentication: %w", err)
	}
	googleVerifier, err := googleProvider.NewVerifier(s.cfg.Auth.GoogleClientID, nil)
	if err != nil {
		return fmt.Errorf("initialize Google verifier: %w", err)
	}
	signIn, err := signInUC.NewService(googleVerifier, userRepository, sessions)
	if err != nil {
		return fmt.Errorf("initialize sign-in use case: %w", err)
	}

	apiGroup := s.echo.Group("/api")
	apiGroup.GET("/health", HealthHandler(s.dependencies.MongoClient, s.cfg.MongoDB.DatabaseName, s.cfg.Server.ContextTimeout))
	s.echo.GET("/metrics", echo.WrapHandler(s.dependencies.MetricsHandler))

	v1 := s.echo.Group("/api/v1")
	authGroup := v1.Group("/auth")
	authHandlers := authHTTP.NewHandler(sessions, s.cfg)
	authHTTP.MapAuthRoutes(authGroup, authHandlers, middleware.RequireAuthentication(sessions))

	v2 := s.echo.Group("/api/v2")
	v2SignIn := v2.Group("/sign-in")
	signInHandlers := signInHTTP.NewHandler(signIn, s.cfg)
	signInHTTP.MapSignInRoutes(v2SignIn, signInHandlers)

	return nil
}

// HealthHandler reports API liveness and MongoDB readiness.
func HealthHandler(client *mongo.Client, databaseName string, timeout time.Duration) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), timeout)
		defer cancel()
		status := http.StatusOK
		databaseStatus := "up"
		if err := client.Database(databaseName).RunCommand(ctx, map[string]int{"ping": 1}).Err(); err != nil {
			status = http.StatusServiceUnavailable
			databaseStatus = "down"
		}
		healthStatus := "ok"
		if status != http.StatusOK {
			healthStatus = "unavailable"
		}
		return c.JSON(status, map[string]any{
			"status":    healthStatus,
			"timestamp": time.Now().UTC(),
			"services":  map[string]string{"api": "up", "database": databaseStatus},
		})
	}
}
