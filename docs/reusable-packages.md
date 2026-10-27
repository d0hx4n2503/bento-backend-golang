---
type: Code Reference
title: Reusable packages and ownership
description: Catalogs current reusable code and rules for shared packages, utilities, and middleware.
tags: [bento-backend, golang, packages, reuse]
sources:
  - resource: ../pkg/converter/converter.go
    title: JSON conversion helper
  - resource: ../pkg/csrf/csrf.go
    title: CSRF token primitives
  - resource: ../pkg/httpErrors/http_errors.go
    title: Current REST error helpers
  - resource: ../pkg/sanitize/sanitize.go
    title: JSON string sanitization
  - resource: ../pkg/statusModel/status.go
    title: Shared status type
  - resource: ../pkg/utils/pagination.go
    title: Pagination helpers
  - resource: ../internal/transport/http/middleware/tracing.go
    title: Echo tracing middleware adapter
---
# Reusable packages and ownership

This page is the first place to check before adding a general-purpose helper. Presence in `pkg` does not automatically mean a function is complete or appropriate for every caller; read the implementation, semantics, and limitations first.

## Where code belongs

| Put code in | When | Example |
| --- | --- | --- |
| `internal/features/<feature>/...` | One feature owns the behavior, even if the helper has a generic-looking name | Agent identity normalization used only by `agentidentity` |
| `internal/shared/<capability>` | Two or more internal features need the same capability, it has no natural feature owner, and it should remain private to this module | A shared clock abstraction, if multiple use cases genuinely require it |
| `internal/infrastructure/<vendor>` | It wraps a driver, SDK, external API, process-wide client, or technical concern | MongoDB client, Redis client, OTLP exporter |
| `internal/transport/http/middleware` | It adapts an HTTP-wide concern to Echo request/response lifecycle | Request tracing, authentication enforcement |
| `pkg/<specific-name>` | It is stable, broadly reusable, intentionally part of this module's public Go API, and has no business-feature owner | A small framework-neutral encoding package |

`internal/shared` is a place to add only when there are actual multiple consumers. Do not create it preemptively. Prefer a local function over a shared abstraction when there is one caller. `pkg/utils` should not grow into a home for unrelated functions; a package name should tell the caller what capability it provides.

## Existing package catalog

| Package / symbol | Current responsibility | Reuse guidance and known caveat |
| --- | --- | --- |
| `pkg/converter.AnyToBytesBuffer` | JSON-marshals a value into a `bytes.Buffer` without a newline | Use only when a caller specifically needs a buffer. For ordinary JSON bytes, `json.Marshal` is simpler. |
| `pkg/csrf.MakeToken`, `ValidateToken` | Creates and verifies a session-bound HMAC CSRF token | `internal/transport/http/middleware/csrf.go` should enforce the HTTP policy by calling these primitives; do not reimplement the HMAC there. Use a dedicated secret of at least 32 bytes. |
| `pkg/httpErrors` | Current REST-shaped errors, error classification, and status values | Legacy boundary helper. New feature errors should not depend on `net/http` or REST response types. `NewRestError` and `NewRestErrorWithMessage` currently construct the same value; `ErrorResponse` calls `ParseErrors` twice. |
| `pkg/sanitize.SanitizeJSON` | Sanitizes HTML in every string value in a JSON document | Only use for fields intentionally rendered as user-authored HTML. It changes stored values and is not a replacement for output escaping. Do not add a second generic sanitizer in HTTP middleware. |
| `pkg/statusModel.Status` | Status string type with Pending, Processed, Cancelled, UnderReview, and Overdue constants | Confirm these statuses are truly shared across features. If they belong to one business capability, move ownership to that feature instead of expanding this package. |
| `pkg/utils.PaginationQuery` and helpers | Parses Echo query parameters and calculates page values | Current code has edge cases: blank `page` sets `Size` to zero, and total/has-more helpers do not guard zero or negative sizes. Do not treat it as production-ready until corrected. |
| `pkg/utils.ValidateStruct` | Singleton go-playground validator wrapper | Reuse for structural validation when its default validator behavior is sufficient. Feature-specific business validation stays in the feature. |
| `pkg/utils` image functions | Checks image content types and builds a UUID-based name | `images.go` currently refers to `httpErrors` without importing/defining it. Fix and review the policy before reuse. |
| `internal/infrastructure/observability/logger.NewZapLogger` | Constructs the Zap-backed logger adapter | Create once in the composition root and pass the narrow logger interface where needed. |
| `internal/infrastructure/observability/metrics.NewPrometheusMetrics` | Creates a private Prometheus registry and request metrics | Middleware should record through the metrics interface and use normalized route templates, never raw URL paths as labels. |
| `internal/infrastructure/observability/tracing.Initialize` | Initializes the process-wide OpenTelemetry provider and propagator | Call once during process startup and shut down the provider during graceful shutdown. |
| `internal/transport/http/middleware.NewTracingMiddleware` | Adapts Echo requests to the configured OpenTelemetry provider | Keep HTTP instrumentation here; do not duplicate exporter/provider initialization in middleware. |
| `internal/infrastructure/redis.NewRedisClient` | Constructs a Redis client from the shared config | The composition root owns connection checks and closing the client. |
| `config.LoadConfig`, `ParseConfig` | Loads Viper settings, binds `APP_` environment variables, decodes typed config | Use as the single config loading path; do not parse environment variables separately in features. |

`pkg/utils/auth.go`, `http.go`, and `jwt.go` remain placeholders. The auth feature implements Bento session JWT issuance and verification in `internal/features/auth/usecase`; its HTTP adapter is under `internal/features/auth/delivery/http`. Transport auth middleware delegates token verification to that use case. OAuth client tokens remain owned by the separate `oauth` feature.

## Middleware and package responsibilities

Middleware is an adapter, not a second implementation of a capability. Keep the reusable protocol/algorithm in one package, then let middleware call it and apply request-specific policy.

| Concern | Shared implementation / owner | HTTP adapter |
| --- | --- | --- |
| CSRF token algorithm | `pkg/csrf` today | `internal/transport/http/middleware/csrf.go` |
| Request authentication | Feature use case or auth policy, to be defined | `internal/transport/http/middleware/auth.go` |
| Request sanitization | Explicit input boundary for fields requiring HTML sanitization | No generic sanitization middleware; do not sanitize all JSON globally by default |
| Metrics | `internal/infrastructure/observability/metrics` | `middleware/metrics.go` records HTTP observations |
| Tracing | `internal/infrastructure/observability/tracing` initializes provider | `middleware/tracing.go` instruments Echo |
| HTTP error serialization | `internal/transport/http` | `http_error.go` should own status/body mapping for the transport |

Authentication, CSRF, CORS, metrics, rate limiting, request logging, and tracing middleware are implemented. `debug.go` and `middlewares.go` are empty placeholders.

## Avoiding duplicate code

Before adding a helper, search for the capability name and behavior, then ask:

1. Does a package already implement the same operation? Reuse it if its semantics fit; do not wrap it in a second copy merely to rename it.
2. Is the existing code incomplete or buggy? Fix the owning package and add a focused regression test with the implementation.
3. Does the helper contain business language or invariants? Keep it in that feature.
4. Does it depend on Echo, MongoDB, Redis, or another vendor? Keep it at that adapter boundary.
5. Will more than one feature consume it, and is there no single feature owner? Consider `internal/shared/<capability>`.
6. Is it intentionally supported as a reusable API outside `internal`? Only then consider `pkg/<capability>`.

There is no separate copy of these shared middleware behaviors in `pkg`. The main overlap risk is architectural: `pkg/httpErrors` and `internal/transport/http/http_error.go` can both map errors to HTTP responses. Keep feature errors transport-neutral and assign response serialization to `internal/transport/http`.
