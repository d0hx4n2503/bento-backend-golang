---
type: Project Reference
title: Go backend project map
description: Maps the Go repository areas and feature names to the existing NestJS backend.
tags: [bento-backend, golang, project-map]
sources:
  - resource: ../go.mod
    title: Go module and dependencies
  - resource: ../../backend/src/app.module.ts
    title: NestJS module registration
---
# Project map

## Repository areas

| Path | Role | Current state |
| --- | --- | --- |
| `cmd/api` | HTTP API process entry point | Loads config, handles signals, and delegates lifecycle to `internal/app` |
| `cmd/worker` | Background worker process, if required by a feature | Placeholder only |
| `cmd/migrate` | Explicit database migration process, if adopted | Placeholder only |
| `config` | Typed configuration and Viper loading | Loads and validates local/Docker settings for API, MongoDB, and auth |
| `internal/app` | Process composition and shared dependency lifecycle | Receives the configured logger; initializes MongoDB, Redis, metrics, and optional tracing; delegates HTTP setup and shutdown |
| `internal/features` | Business capabilities and their use-case boundaries | Auth and sign-in are implemented; most other directories are empty scaffolds |
| `internal/models` | Shared feature data and token models, named `<feature>_model.go` | User, sign-in identity, and session principal models are defined here |
| `internal/infrastructure` | Shared vendor clients and technical adapters | MongoDB, logging, metrics, and tracing are implemented; other adapters remain placeholders |
| `internal/transport/http` | HTTP server, feature wiring, route groups, shared middleware, and response mapping | Server lifecycle, auth/sign-in wiring, health/metrics, error mapping, auth/CSRF/logging/metrics middleware, and tracing adapter are implemented |
| `migrations` | Versioned database schema/index changes | MongoDB and PostgreSQL directories are placeholders |
| `pkg` | Reusable packages that are intentionally exported from this module | A few utilities exist; see [Reusable packages](reusable-packages.md) |
| `docs` | Architecture, implementation, and API documentation | OpenAPI reference and Swagger UI assets are embedded in the API binary |

## Feature names and NestJS references

The Go directories below correspond to business-capability areas present in `backend/src/modules`. This is a naming cross-reference only; it does not prescribe matching file layouts or prove that each boundary is final.

| Go feature | NestJS reference |
| --- | --- |
| `actions` | `modules/actions` |
| `agentidentity` | `modules/agent_identities` |
| `agents` | `modules/agents` |
| `analysislogs` | `modules/analysis_logs` |
| `auth` | `modules/auth` session tokens and guards; wallet nonce/signature login is excluded |
| `campaign` | `modules/campaign` |
| `earlyaccess` | `modules/early-access` |
| `embeddedwallet` | `modules/embedded_wallet` |
| `jobs` | `modules/jobs` |
| `lendingpool` | `modules/lending_pool` |
| `metrics` | `modules/metrics` |
| `notifications` | `modules/notifications` |
| `oauth` | `modules/oauth` |
| `policies` | `modules/policies` |
| `portfolio` | `modules/portfolio` |
| `protocolhistory` | `modules/protocol_history` |
| `riskengine` | `modules/risk-engine` |
| `signin` | `modules/sign_in` Google Sign-In; email OTP is excluded |
| `simulations` | `modules/simulations` |
| `stats` | `modules/stats` |
| `system` | `modules/system` |
| `trading` | `modules/trading` |
| `transactions` | `modules/transactions` |
| `usage` | `modules/usage` |
| `userwallet` | `modules/users_wallet` |
| `weeklyruns` | `modules/weekly-runs` |

NestJS's `common`, `infra`, and `config` areas are cross-cutting concerns, not business features. In Go, use `internal/transport`, `internal/infrastructure`, and `config` for those roles. Keep a capability inside a feature when it serves only that feature; promote it to a shared package only after there is a real second consumer and a clear owner.

Feature interfaces live at the feature root (`usecase.go`, `mongo_repository.go`, `delivery.go`, and provider-specific contracts where needed). Implementations live in subfolders. Each contract file declares one interface. HTTP endpoint registration stays in `delivery/http/routes.go`; handlers stay in `delivery/http/handlers.go`.

## Dependency snapshot

The Go module path is `github.com/d0hx4n2503/bento-backend-golang` and `go.mod` declares Go 1.26. Direct dependencies include Echo v5, Echo OpenTelemetry middleware, Viper, Zap, Prometheus, Redis, OpenTelemetry, validation, UUID, Bluemonday, and the MongoDB Go driver. MongoDB is used by Google sign-in. PostgreSQL remains configured for compatibility but the API does not initialize it.

The API has local and Docker config examples. Provide four RS256 key files, `APP_AUTH_GOOGLE_CLIENT_ID`, and reachable MongoDB and Redis instances before startup. Google sign-in introspects the frontend's OAuth access token with Google tokeninfo, binds it to the configured client, and requires a verified email. Accounts are matched by provider subject; a matching email alone does not link an existing account. The API ensures user identity indexes during startup. Other feature directories and worker/migration processes remain unfinished.
