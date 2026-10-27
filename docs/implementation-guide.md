---
type: Engineering Guide
title: Feature implementation guide
description: Gives one consistent implementation pattern for Go features and their adapters.
tags: [bento-backend, golang, implementation, clean-architecture]
sources:
  - resource: ../internal/features/auth/usecase.go
    title: Auth use case contract
  - resource: ../internal/features/auth/delivery.go
    title: Auth delivery contract
  - resource: ../internal/transport/http/handlers.go
    title: Feature wiring and route registration
---
# Feature implementation guide

Use this sequence when implementing a feature. Start with the behavior and its owner, then add only the packages needed to support that behavior.

## 1. Define the feature boundary

Before writing code, identify the feature that owns the invariant and data. Read the corresponding NestJS service, controller, DTOs, schemas, processors, and any shared helpers that affect the behavior. Record the existing API contract and behavior that must remain compatible. Do not turn a NestJS controller/service/schema file into a one-to-one Go file translation.

If two Go features need the same behavior, decide whether it is a real shared capability or whether one feature owns a use case that the other should call. Avoid importing another feature's MongoDB repository or internal storage model.

## 2. Put shared data shapes in models

Place persisted and cross-layer data shapes in `internal/models/<feature>_model.go`, following the repository convention. Keep the feature root for interfaces only. Put feature-owned errors and business rules/normalization in `internal/features/<feature>/domain/` when needed. Here, `domain` is a small folder for rules and errors; it does not mean every feature needs a separate entity/model layer. Models describe data; use cases coordinate workflows, and domain helpers enforce feature invariants.

## 3. Declare one feature contract per root file

Declare use-case, repository, provider, and handler interfaces at the feature root, with one interface per file. Implement each interface in its responsibility subfolder and return that interface from its constructor. Keep request parsing and HTTP response details in delivery handlers.

```go
package users

import (
    "context"

    "github.com/d0hx4n2503/bento-backend-golang/internal/models"
)

type Usecase interface {
    GetUser(ctx context.Context, id string) (models.User, error)
}
```

The interface is declared once at the feature root; its implementation lives in `usecase/`.

## 4. Implement adapters at the edge

An HTTP delivery adapter parses transport input, calls the use case, and maps its result to the API contract. A MongoDB repository implements the feature’s repository interface and maps BSON documents to models. It should not return `bson.M`, `*mongo.Collection`, or raw driver errors as part of a public feature API.

```go
package mongodb

var _ users.MongoRepository = (*userRepository)(nil)
```

Use compile-time interface assertions for adapters where they clarify the contract. Translate vendor errors at the adapter boundary into stable feature/application errors so use cases and handlers do not inspect raw driver strings.

## 5. Wire dependencies in the composition root

`cmd/api` delegates process startup to `internal/app`. The app initializes shared infrastructure; `internal/transport/http.Server` constructs repositories, use cases, and handlers, creates route groups, registers feature routes, and owns HTTP serving and graceful shutdown. Pass dependencies through constructors. Do not create clients in handlers, feature package `init` functions, or hidden globals.

## 6. Keep each feature consistent

Build and connect feature code in this order:

```text
models -> repository -> usecase -> handler -> route
```

At runtime, calls move in the opposite direction: route to handler to use case to repository. A feature with no persistence does not need a `repository/mongodb` directory; a feature with no HTTP endpoint does not need an HTTP handler.

## Go package conventions

- Keep package names short, lowercase, and tied to one responsibility. Avoid mixed-case package names such as `httpErrors` and broad packages such as `utils` for new code.
- Pass `context.Context` as the first parameter for I/O and cancellable operations. Do not store request contexts in long-lived structs.
- Wrap errors with `%w` when callers may need `errors.Is` or `errors.As`; use stable sentinel or typed errors for expected application outcomes.
- Return errors instead of panicking for normal failures. Add context at boundaries without repeatedly logging the same error at every layer.
- Keep provider HTTP response models and persisted user models separate when their shape or lifecycle differs; place each in the matching `internal/models/<feature>_model.go` file.
- Avoid cross-feature cycles. If two packages need a type, place it with its owner or introduce a small interface/type package with an explicit reason.
- Apply `gofmt` to changed Go files and run `go test ./...`. Add focused tests for non-trivial behavior and edge cases.

## Change checklist

Before a feature change is ready for review, confirm:

1. The feature owner and externally visible behavior are clear.
2. Use-case implementations do not import transport packages or vendor drivers.
3. Interfaces are defined by the consuming use case, not by an infrastructure package “just in case.”
4. Shared behavior was searched for in the [reusable package catalog](reusable-packages.md).
5. MongoDB mappings and query behavior stay inside the feature adapter.
6. `internal/transport/http.Server` wires concrete adapters into use cases; `internal/app` owns shared process dependencies.
7. `gofmt` and `go test ./...` pass; the feature follows the same dependency flow as existing features, or documents why a different boundary is needed.
