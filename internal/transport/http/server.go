package http

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/d0hx4n2503/bento-backend-golang/config"
	"github.com/d0hx4n2503/bento-backend-golang/internal/infrastructure/observability/logger"
	"github.com/d0hx4n2503/bento-backend-golang/internal/infrastructure/observability/metrics"
	"github.com/d0hx4n2503/bento-backend-golang/internal/transport/http/middleware"
	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.opentelemetry.io/otel/trace"
)

const (
	maxHeaderBytes    = 1 << 20
	readHeaderTimeout = 5 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
)

// Dependencies contains the shared adapters required by the HTTP server.
type Dependencies struct {
	MongoClient    *mongo.Client
	Logger         logger.Logger
	Metrics        metrics.Metrics
	MetricsHandler http.Handler
	Tracer         trace.TracerProvider
	RateLimit      echo.MiddlewareFunc
}

// Server owns the Echo router and HTTP server lifecycle.
type Server struct {
	echo         *echo.Echo
	cfg          config.Config
	dependencies Dependencies
}

// NewServer constructs the Echo router and installs shared HTTP middleware.
func NewServer(cfg config.Config, dependencies Dependencies) (*Server, error) {
	if dependencies.Logger == nil || dependencies.Metrics == nil {
		return nil, fmt.Errorf("HTTP logger and metrics dependencies are required")
	}
	if dependencies.MongoClient == nil || dependencies.MetricsHandler == nil {
		return nil, fmt.Errorf("HTTP MongoDB and metrics handler dependencies are required")
	}

	e := echo.New()
	e.HTTPErrorHandler = NewErrorHandler(dependencies.Logger)
	sharedMiddleware := []echo.MiddlewareFunc{
		echoMiddleware.RequestID(),
		middleware.CORS(cfg.Server.CORSAllowedOrigins),
		echoMiddleware.Recover(),
		echoMiddleware.Secure(),
		echoMiddleware.BodyLimit(1 << 20),
		middleware.RequestLogger(dependencies.Logger),
		middleware.Metrics(dependencies.Metrics),
	}
	if dependencies.RateLimit != nil {
		sharedMiddleware = append(sharedMiddleware, dependencies.RateLimit)
	}
	e.Use(sharedMiddleware...)
	if cfg.Server.CSRF {
		e.Use(middleware.CSRF([]byte(cfg.Server.CSRFSecretKey)))
	}
	if cfg.Tracing.Enabled {
		tracingMiddleware, err := middleware.NewTracingMiddleware(cfg.Tracing.ServerAddress, dependencies.Tracer)
		if err != nil {
			return nil, err
		}
		e.Use(tracingMiddleware)
	}
	if cfg.Cookie.Secure || cfg.Server.Mode == "production" {
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
				c.Response().Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
				return next(c)
			}
		})
	}
	e.GET("/", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"service": "bento-backend-golang", "version": cfg.Server.AppVersion})
	})
	if err := mapSwaggerRoutes(e); err != nil {
		return nil, fmt.Errorf("map Swagger routes: %w", err)
	}

	return &Server{echo: e, cfg: cfg, dependencies: dependencies}, nil
}

// Run maps feature routes and serves requests until ctx is canceled.
func (s *Server) Run(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("HTTP server context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.MapHandlers(ctx); err != nil {
		return fmt.Errorf("map HTTP handlers: %w", err)
	}

	address := strings.TrimSpace(s.cfg.Server.Port)
	if !strings.Contains(address, ":") {
		address = ":" + address
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", address, err)
	}
	server := &http.Server{
		Handler:           s.echo,
		ReadTimeout:       s.cfg.Server.ReadTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      s.cfg.Server.WriteTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}

	serverErr := make(chan error, 1)
	go func() {
		s.dependencies.Logger.Infow("HTTP server listening",
			"address", address,
			"version", s.cfg.Server.AppVersion,
		)
		serverErr <- server.Serve(listener)
	}()

	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		s.dependencies.Logger.Info("HTTP server shutdown started")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shut down HTTP server: %w", err)
		}
		if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP during shutdown: %w", err)
		}
		s.dependencies.Logger.Info("HTTP server stopped")
		return nil
	}
}
