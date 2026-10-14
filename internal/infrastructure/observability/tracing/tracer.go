// Package tracing initializes OpenTelemetry tracing for an application process.
package tracing

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
)

// Config controls the OTLP trace exporter and sampling policy.
type Config struct {
	ServiceName   string
	Endpoint      string
	Insecure      bool
	SamplingRatio float64
}

// Initialize creates and installs a process-wide OpenTelemetry tracer provider.
// Call the returned provider's Shutdown method during graceful process shutdown
// so queued spans are exported.
//
// Endpoint accepts either "host:port" or a URL such as
// "https://collector.example.com:4317". If omitted, the OTLP exporter uses its
// standard environment variables and localhost default.
func Initialize(ctx context.Context, cfg Config) (*sdktrace.TracerProvider, error) {
	if ctx == nil {
		return nil, errors.New("tracing context is required")
	}
	serviceName := strings.TrimSpace(cfg.ServiceName)
	if serviceName == "" {
		return nil, errors.New("tracing service name is required")
	}
	if cfg.SamplingRatio < 0 || cfg.SamplingRatio > 1 {
		return nil, fmt.Errorf("tracing sampling ratio must be between 0 and 1, got %v", cfg.SamplingRatio)
	}

	options := make([]otlptracegrpc.Option, 0, 2)
	if endpoint := strings.TrimSpace(cfg.Endpoint); endpoint != "" {
		if strings.Contains(endpoint, "://") {
			options = append(options, otlptracegrpc.WithEndpointURL(endpoint))
		} else {
			options = append(options, otlptracegrpc.WithEndpoint(endpoint))
		}
	}
	if cfg.Insecure {
		options = append(options, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithAttributes(semconv.ServiceName(serviceName)),
	)
	if err != nil {
		_ = exporter.Shutdown(ctx)
		return nil, fmt.Errorf("create tracing resource: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SamplingRatio))),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return provider, nil
}
