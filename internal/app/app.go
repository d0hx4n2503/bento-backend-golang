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
func Run(ctx context.Context, cfg *config.Config) error {
	if ctx == nil {
		return fmt.Errorf("application context is required")
	}
	if cfg == nil {
		return fmt.Errorf("application configuration is required")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	log, err := logger.NewZapLogger(logger.Config{
		Development:       cfg.Logger.Development,
		DisableCaller:     cfg.Logger.DisableCaller,
		DisableStacktrace: cfg.Logger.DisableStacktrace,
		Encoding:          cfg.Logger.Encoding,
		Level:             cfg.Logger.Level,
	})
	if err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}
	defer func() { _ = log.Sync() }()

	metricRecorder, err := metrics.NewPrometheusMetrics(metricsServiceName(cfg))
	if err != nil {
		return fmt.Errorf("initialize metrics: %w", err)
	}

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
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if shutdownErr := tracerProvider.Shutdown(shutdownCtx); shutdownErr != nil {
				log.Errorf("shut down tracing provider: %v", shutdownErr)
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
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := client.Disconnect(shutdownCtx); shutdownErr != nil {
			log.Errorf("disconnect MongoDB client: %v", shutdownErr)
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
	defer func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			log.Errorf("close Redis client: %v", closeErr)
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
