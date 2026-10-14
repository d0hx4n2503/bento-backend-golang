---
type: Operations Reference
title: Configuration and observability
description: Documents the current Go configuration loader and observability components and their ownership.
tags: [bento-backend, golang, configuration, observability]
sources:
  - resource: ../config/config.go
    title: Configuration schema and Viper loader
  - resource: ../internal/infrastructure/observability/logger/zap_logger.go
    title: Zap logger
  - resource: ../internal/infrastructure/observability/metrics/metric.go
    title: Prometheus metrics
  - resource: ../internal/infrastructure/observability/tracing/tracer.go
    title: OpenTelemetry tracing
  - resource: ../internal/transport/http/middleware/tracing.go
    title: Echo tracing middleware
---
# Configuration and observability

## Configuration loading

`config.LoadConfig(filename)` reads an explicit path, or searches the current directory and `config/` for a bare filename. It binds environment variables with the `APP_` prefix and maps nested keys to uppercase underscore-separated names. For example, `server.port` maps to `APP_SERVER_PORT`. `config.ParseConfig` decodes the Viper instance into `config.Config`.

The current config schema declares these sections:

| Section | Declared fields / purpose |
| --- | --- |
| `server` | Version, ports, mode, JWT/CSRF secrets, cookie name, timeouts, SSL, debug flags |
| `postgres` | Host, port, user, password, database, SSL mode, driver |
| `redis` | Address, password, DB, idle connections, pool size, pool timeout |
| `mongodb` | URI |
| `cookie` | Name, max age, secure, HTTP-only flags |
| `store` | Image storage folder |
| `session` | Prefix, name, expiry |
| `metrics` | Metrics address and service name |
| `tracing` | Enabled, exporter endpoint, service name/address, insecure transport flag, sampling ratio |
| `logger` | Development mode, caller/stacktrace settings, encoding, level |
| `aws` | Endpoint, access keys, TLS flag, MinIO endpoint |

The YAML files and `.env.example` are empty at present. The loader supplies tracing defaults (`service_name=bento-backend-golang`, `sampling_ratio=0.1`), but most fields do not have defaults or validation in this package. Add validation for required values and invalid ranges before starting clients. Keep secrets out of committed configuration files and logs.

## Logging

`internal/infrastructure/observability/logger.NewZapLogger` creates a Zap-backed logger that writes to stderr. It accepts development mode, encoding (`json` or `console`), level, caller, and stacktrace settings. The composition root should create it once, pass a narrow interface to consumers, and call `Sync` during shutdown. Avoid logging the same error in every layer; log once where the error is handled or where useful operational context is known.

## Metrics

`NewPrometheusMetrics(serviceName)` creates a private registry with request count and duration metrics plus Go/process collectors. `Handler()` exposes the metrics endpoint handler. HTTP middleware should report status, method, and a normalized route template. Never use raw URL paths, user IDs, wallet addresses, or other high-cardinality values as metric labels.

Metrics server address and route registration are not wired yet. Keep the listener lifecycle in the application composition root and make its shutdown explicit.

## Tracing

`internal/infrastructure/observability/tracing.Initialize` creates the OTLP gRPC exporter, a service resource, parent-based trace-ID sampling, and W3C Trace Context/Baggage propagation. The function installs a process-wide OpenTelemetry provider, so call it once at startup. Shut down the returned provider with a bounded context during graceful shutdown.

`internal/transport/http/middleware.NewTracingMiddleware` adapts the configured provider to Echo v5. Provider/exporter setup belongs to infrastructure; incoming HTTP instrumentation belongs to transport. The tracing config schema has `enabled`, endpoint, service name, server address, insecure, and sampling-ratio fields, but the application does not yet wire those fields into startup.

## Startup and shutdown ownership

`internal/app` should own the order of configuration parsing, logger and client creation, tracing initialization, handler registration, server start, signal handling, and cleanup. Each constructed client/provider must have one clear shutdown owner. Avoid independent setup paths in `cmd/api`, handlers, or middleware.
