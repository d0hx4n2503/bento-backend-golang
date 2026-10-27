package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/d0hx4n2503/bento-backend-golang/config"
	"github.com/d0hx4n2503/bento-backend-golang/internal/infrastructure/mongodb"
	"github.com/d0hx4n2503/bento-backend-golang/internal/infrastructure/observability/logger"
	"github.com/d0hx4n2503/bento-backend-golang/internal/infrastructure/observability/metrics"
	"github.com/d0hx4n2503/bento-backend-golang/internal/infrastructure/observability/tracing"
	redisInfra "github.com/d0hx4n2503/bento-backend-golang/internal/infrastructure/redis"
	transportHTTP "github.com/d0hx4n2503/bento-backend-golang/internal/transport/http"
	"github.com/d0hx4n2503/bento-backend-golang/internal/transport/http/middleware"
	"go.opentelemetry.io/otel"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
)

// Run wires the API process and owns the lifecycle of its shared dependencies.
func Run(ctx context.Context, cfg *config.Config, log logger.Logger) error {
	if ctx == nil {
		return fmt.Errorf("application context is required")
	}
	if cfg == nil {
		return fmt.Errorf("application configuration is required")
	}
	if log == nil {
		return fmt.Errorf("application logger is required")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	log.Infow("starting API application",
		"version", cfg.Server.AppVersion,
		"mode", cfg.Server.Mode,
	)

	metricRecorder, err := metrics.NewPrometheusMetrics(metricsServiceName(cfg))
	if err != nil {
		return fmt.Errorf("initialize metrics: %w", err)
	}
	log.Infow("initialized Prometheus metrics", "service_name", metricsServiceName(cfg))

	var tracerProvider *sdkTrace.TracerProvider
	if cfg.Tracing.Enabled {
		tracingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		tracerProvider, err = tracing.Initialize(tracingCtx, tracing.Config{
			ServiceName:   cfg.Tracing.ServiceName,
			Endpoint:      cfg.Tracing.Endpoint,
			Insecure:      cfg.Tracing.Insecure,
			SamplingRatio: cfg.Tracing.SamplingRatio,
		})
		cancel()
		if err != nil {
			return fmt.Errorf("initialize tracing: %w", err)
		}
		log.Infow("initialized OpenTelemetry tracing", "service_name", cfg.Tracing.ServiceName)
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if shutdownErr := tracerProvider.Shutdown(shutdownCtx); shutdownErr != nil {
				log.Errorw("failed to shut down tracing provider", "error", shutdownErr)
			}
		}()
	}
	tracer := otel.GetTracerProvider()
	if tracerProvider != nil {
		tracer = tracerProvider
	}

	databaseCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	client, err := mongodb.NewClient(databaseCtx, cfg.MongoDB)
	cancel()
	if err != nil {
		return fmt.Errorf("initialize MongoDB: %w", err)
	}
	log.Infow("connected to MongoDB", "database", cfg.MongoDB.DatabaseName)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := client.Disconnect(shutdownCtx); shutdownErr != nil {
			log.Errorw("failed to disconnect MongoDB client", "error", shutdownErr)
		}
	}()

	redisClient := redisInfra.NewRedisClient(cfg)
	redisCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = redisClient.Ping(redisCtx).Err()
	cancel()
	if err != nil {
		_ = redisClient.Close()
		return fmt.Errorf("connect to Redis: %w", err)
	}
	log.Info("connected to Redis")
	defer func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			log.Errorw("failed to close Redis client", "error", closeErr)
		}
	}()

	httpServer, err := transportHTTP.NewServer(*cfg, transportHTTP.Dependencies{
		MongoClient:    client,
		Logger:         log,
		Metrics:        metricRecorder,
		MetricsHandler: metricRecorder.Handler(),
		Tracer:         tracer,
		RateLimit:      middleware.RateLimit(redisClient),
	})
	if err != nil {
		return fmt.Errorf("initialize HTTP server: %w", err)
	}
	return httpServer.Run(ctx)
}

func metricsServiceName(cfg *config.Config) string {
	name := strings.TrimSpace(cfg.Metrics.ServiceName)
	if name != "" {
		return strings.ReplaceAll(name, "-", "_")
	}
	return "bento_backend_golang"
}
