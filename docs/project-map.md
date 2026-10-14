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
| `cmd/api` | HTTP API process entry point | `main.go` is empty |
| `cmd/worker` | Background worker process, if required by a feature | Placeholder only |
| `cmd/migrate` | Explicit database migration process, if adopted | Placeholder only |
| `config` | Typed configuration and Viper loading | `config.go` exists; YAML examples are empty |
| `internal/app` | Composition root and application lifecycle | Placeholder only |
| `internal/features` | Business capabilities and their domain/use-case boundaries | Directories exist; most are empty scaffolds |
| `internal/infrastructure` | Shared vendor clients and technical adapters | Observability and Redis have initial code; most adapters are placeholders |
| `internal/transport/http` | HTTP server, handlers, shared middleware, and response mapping | Tracing middleware exists; other files are placeholders |
| `migrations` | Versioned database schema/index changes | MongoDB and PostgreSQL directories are placeholders |
| `pkg` | Reusable packages that are intentionally exported from this module | A few utilities exist; see [Reusable packages](reusable-packages.md) |
| `docs` | Architecture and implementation documentation | This documentation set |

## Feature names and NestJS references

The Go directories below correspond to business-capability areas present in `backend/src/modules`. This is a naming cross-reference only; it does not prescribe matching file layouts or prove that each boundary is final.

| Go feature | NestJS reference |
| --- | --- |
| `actions` | `modules/actions` |
| `agentidentity` | `modules/agent_identities` |
| `agents` | `modules/agents` |
| `analysislogs` | `modules/analysis_logs` |
| `auth` | `modules/auth` |
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
| `signin` | `modules/sign_in` |
| `simulations` | `modules/simulations` |
| `stats` | `modules/stats` |
| `system` | `modules/system` |
| `trading` | `modules/trading` |
| `transactions` | `modules/transactions` |
| `usage` | `modules/usage` |
| `userwallet` | `modules/users_wallet` |
| `weeklyruns` | `modules/weekly-runs` |

NestJS's `common`, `infra`, and `config` areas are cross-cutting concerns, not business features. In Go, use `internal/transport`, `internal/infrastructure`, and `config` for those roles. Keep a capability inside a feature when it serves only that feature; promote it to a shared package only after there is a real second consumer and a clear owner.

## Dependency snapshot

The Go module path is `github.com/d0hx4n2503/bento-backend-golang` and `go.mod` declares Go 1.26. The direct dependencies currently include Echo v5, Echo OpenTelemetry middleware, Viper, Zap, Prometheus, Redis, OpenTelemetry, validation, UUID, and Bluemonday. MongoDB and PostgreSQL drivers are not yet declared, so database repository code is not ready to implement against an installed driver.

The root `README.md`, Makefile, CI workflow, lint configuration, environment example, Docker files, and local YAML configuration are currently empty. Do not assume those files define working run, build, or deployment commands yet.
