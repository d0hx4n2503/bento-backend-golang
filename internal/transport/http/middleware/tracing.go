package middleware

import (
	"fmt"

	echoOTel "github.com/labstack/echo-otel/v5"
	"github.com/labstack/echo/v5"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// NewTracingMiddleware instruments incoming Echo requests and propagates W3C
// Trace Context and Baggage headers. serverAddress should be a canonical host,
// optionally followed by a port; leave it empty when that value is not known.
func NewTracingMiddleware(serverAddress string, provider trace.TracerProvider) (echo.MiddlewareFunc, error) {
	config := echoOTel.Config{
		ServerName:     serverAddress,
		TracerProvider: provider,
		Propagators: propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	}

	tracingMiddleware, err := config.ToMiddleware()
	if err != nil {
		return nil, fmt.Errorf("configure Echo tracing middleware: %w", err)
	}
	return tracingMiddleware, nil
}
