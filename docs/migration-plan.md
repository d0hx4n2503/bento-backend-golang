---
type: Migration Plan
title: NestJS to Go function migration plan
description: Ordered function scope, priorities, supporting work, and completion checklist for the Go backend migration.
tags: [bento-backend, golang, migration, checklist]
sources:
  - resource: ../../backend/src/app.module.ts
    title: NestJS application module
  - resource: project-map.md
    title: Go project map
  - resource: architecture.md
    title: Go architecture
---
# NestJS to Go function migration plan

This plan migrates externally observable behavior and business functions from `backend/` into Go. It does not copy NestJS module, controller, or service layouts. Organize Go code by feature and responsibility: `internal/models/<feature>_model.go` for data shapes, repository adapters for storage, use cases for workflows, handlers for HTTP behavior, and separate route files for endpoints. Feature interfaces live at the feature root.

## Scope decisions

- Google is the only user sign-in provider in the current migration scope. Keep the provider boundary open for Facebook and Twitter later.
- Do not migrate wallet nonce/signature authentication or email OTP sign-in.
- Embedded wallets remain in scope for asset custody, portfolio, agent, and transaction functions. They are not user login credentials.
- NestJS currently starts required embedded-wallet provisioning as a best-effort side effect after Google sign-in. Wire that hook with `embeddedwallet` (STT 11), where the wallet provisioning behavior is migrated.
- OAuth server authorization for clients/agents and Twitter OAuth used by campaign are separate product functions from social sign-in.
- Authenticate ownership and user access through the internal user ID created or found through Google Sign-In. Revisit NestJS endpoints that still assume `req.user.address` before migrating them.
- The first three entries below are the currently authorized implementation scope. Later entries remain planned and unchecked.
- The current user collection keeps the legacy `google_id` field while writing the provider identity map. Adding Facebook or Twitter later needs its verifier route and a unique index for that provider subject.

## Ordered scope and checklist

Priority: **P0** foundation and identity; **P1** core product flows; **P2** supporting product and reporting; **P3** campaign and secondary features. STT is the suggested implementation order.

| STT | Priority | Feature | Function scope | Supporting / non-function work | Check |
|---:|:---:|---|---|---|:---:|
| 1 | P0 | API foundation | Health endpoint; start and stop API process; standard response and error handling | Config loading/validation, Echo routing/versioning, request IDs, panic recovery, CORS/security/body-limit middleware, structured logs, metrics, tracing, MongoDB/Redis clients, shared rate limits, graceful shutdown | [x] |
| 2 | P0 | `signin` | Introspect the Google OAuth access token against the configured OAuth client; require a verified email; find or create a user by provider subject; sign in and set auth cookies | User/provider identity model and MongoDB repository/indexes; no silent account linking by email; provider interface ready for later Google/Facebook/Twitter adapters; no email OTP or wallet signature login | [x] |
| 3 | P0 | `auth` | Issue/refresh access and refresh tokens after Google sign-in; logout; check authenticated user; admin login if configured | RS256 key loading/validation, access guard for Bearer or cookie, HttpOnly/Secure/SameSite cookies, CSRF, expiry and role claims; no nonce/signature authentication for wallets | [x] |
| 4 | P0 | `agentidentity` | Register agent API key; claim status/regenerate/verify; link agent to user | Key hashing, one-time claim token and expiry, ownership by internal user ID | [ ] |
| 5 | P0 | `agents` | Create/read/update/delete agents; status/settings/icon/allowed target; policy links; required onchain registration and confirmation | Agent schema/indexes, ownership checks by user ID, asset storage, chain adapters | [ ] |
| 6 | P0 | `policies` + `tools` | Policy CRUD/library/evaluation; USD token valuation and spend limit checks | Policy schema/versioning, price source, exact amount handling, audit of evaluations | [ ] |
| 7 | P0 | `system` | Read/update runtime settings, verifier, active LLM, onchain mode/config, relayer/public key | Admin authorization, feature flag sync, secret/key management | [ ] |
| 8 | P1 | `actions` | Action log, filters, stats/trends, pending approvals, verification, decisions, SSE updates | State transitions, locking, idempotency, ownership and audit history | [ ] |
| 9 | P1 | `riskengine` | Analyze actions; prepare transfer/swap drafts; prepare execution, sign, execute, and read status | LLM/evidence, integrity and valuation, spend limits, race prevention | [ ] |
| 10 | P1 | `transactions` | Create/read/list transactions and process analysis | MongoDB schema/indexes, queue workers, retry/idempotency | [ ] |
| 11 | P1 | `embeddedwallet` | Provision/list/rename wallets; balances/portfolio/whitelist; create/approve/execute user and agent transfers | Crossmint and chain adapters, signer/secret handling, transaction state; wallet is never an authentication method | [ ] |
| 12 | P1 | `oauth` | OAuth discovery/client registration, authorize/consent, token/introspect/revoke, connection management | Scope and redirect validation, token storage/expiry, indexes; distinct from social sign-in | [ ] |
| 13 | P1 | `trading` | Token search, swap quote/prepare/sign/submit, SOL wrapping, agent trading | DEX/Titan adapters, slippage, draft expiry and execution state | [ ] |
| 14 | P1 | `charge` | Configure swap fee, calculate fee, build fee instruction, record fee ledger | Exact arithmetic, admin access, ledger reconciliation | [ ] |
| 15 | P1 | `simulations` + `analysislogs` | Create/read simulations; create/query analysis logs and incidents | Schema/indexes, transaction/action linkage, retention | [ ] |
| 16 | P2 | `protocolhistory` | Create/list history and update transaction status | Schema/indexes, ownership, status synchronization | [ ] |
| 17 | P2 | `portfolio` | User/OAuth PnL and portfolio event recording | Valuation/cost basis, price source, event history | [ ] |
| 18 | P2 | `lendingpool` | Pool/reserve/position reads; deposit, borrow, repay, withdraw, submit for user/agent | Stellar Blend adapter, transaction builder and worker | [ ] |
| 19 | P2 | `notifications` | Create/list/read notifications | Persistence, queue/realtime delivery, duplicate prevention | [ ] |
| 20 | P2 | `jobs` + `usage` | Job status; usage create/update/list by transaction, agent, or user; relayer fee | Queue status adapter, TTL, idempotent accounting | [ ] |
| 21 | P2 | `stats` + `metrics` | Dashboard, user/agent overview, model, monitoring, and trend reports | Aggregation, scheduled refresh and cache | [ ] |
| 22 | P3 | `weeklyruns` | Leaderboard and share-run summary | Ranking and period rules, cache | [ ] |
| 23 | P3 | `campaign` | Campaign analysis/card generation, claim, result/download, leaderboard/export | Twitter campaign OAuth, workers, images, SSE, rate limiting; distinct from user sign-in | [ ] |
| 24 | P3 | `earlyaccess` | Register and list early access | Uniqueness, anti-spam, admin access | [ ] |

## Completion rule

Tick an entry only when its function behavior and supporting work are implemented in Go, relevant API contracts and access rules have been reviewed against NestJS, and implementation notes are recorded. Keep this table current as migration work proceeds.
