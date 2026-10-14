---
type: Engineering Guide
title: Feature implementation guide
description: Gives one consistent implementation pattern for Go features and their adapters.
tags: [bento-backend, golang, implementation, clean-architecture]
sources:
  - resource: ../internal/features/auth/usecase/ports.go
    title: Auth use case port scaffold
  - resource: ../internal/features/auth/delivery/interface.go
    title: Auth delivery scaffold
  - resource: ../internal/app/.gitkeep
    title: Application composition root placeholder
---
# Feature implementation guide

Use this sequence when implementing a feature. Start with the behavior and its owner, then add only the packages needed to support that behavior.

## 1. Define the feature boundary

Before writing code, identify the feature that owns the invariant and data. Read the corresponding NestJS service, controller, DTOs, schemas, processors, and any shared helpers that affect the behavior. Record the existing API contract and behavior that must remain compatible. Do not turn a NestJS controller/service/schema file into a one-to-one Go file translation.

If two Go features need the same behavior, decide whether it is a real shared capability or whether one feature owns a use case that the other should call. Avoid importing another feature's MongoDB repository or domain storage model.

## 2. Put business rules in the domain

Domain packages define feature-owned entities, value types, invariants, and errors. Keep them independent of Echo, MongoDB BSON types, Redis, HTTP status codes, and external SDKs. Prefer unexported implementation details; export only names another layer needs.

## 3. Put orchestration and ports in the use case

The use case owns the application workflow and the interfaces it needs. Define an interface in the package that consumes it. Keep request/response DTOs at the delivery boundary unless they represent application-level commands/results shared by multiple adapters.

```go
package usecase

type UserReader interface {
	FindByID(ctx context.Context, id string) (domain.User, error)
}

type GetUser struct {
	users UserReader
}

func NewGetUser(users UserReader) *GetUser {
	return &GetUser{users: users}
}
```

This is an illustrative pattern, not a compiled feature implementation. The real use case should define its own context, input, output, and error behavior.

## 4. Implement adapters at the edge

An HTTP delivery adapter parses transport input, calls a use case, and maps its result to the API contract. A MongoDB repository implements the use-case port and maps between BSON documents and domain types. It should not return `bson.M`, `*mongo.Collection`, or driver errors as part of a public feature API.

```go
package mongodb

var _ usecase.UserReader = (*UserRepository)(nil)
```

Use compile-time interface assertions for adapters where they clarify the contract. Translate vendor errors at the adapter boundary into stable feature/application errors so use cases and handlers do not inspect raw driver strings.

## 5. Wire dependencies in the composition root

`cmd/api` should delegate construction to `internal/app`. The composition root loads configuration, creates shared infrastructure clients, constructs feature repositories and use cases, registers transport handlers, and owns graceful shutdown. Pass dependencies through constructors. Do not create clients in handlers, feature package `init` functions, or hidden globals.

## 6. Keep each feature consistent

For every feature, use the same conceptual flow:

```text
delivery -> usecase -> domain
                 ^
                 |
       repository adapter implements usecase port
```

Naming and folder presence can vary with feature complexity. The dependency direction and responsibility boundaries should not. A feature with no persistence does not need a `repository/mongodb` directory; a feature with no HTTP endpoint does not need an HTTP handler.

## Go package conventions

- Keep package names short, lowercase, and tied to one responsibility. Avoid mixed-case package names such as `httpErrors` and broad packages such as `utils` for new code.
- Pass `context.Context` as the first parameter for I/O and cancellable operations. Do not store request contexts in long-lived structs.
- Wrap errors with `%w` when callers may need `errors.Is` or `errors.As`; use stable sentinel or typed errors for expected application outcomes.
- Return errors instead of panicking for normal failures. Add context at boundaries without repeatedly logging the same error at every layer.
- Keep HTTP and storage models separate from domain types when their shape or lifecycle differs.
- Avoid cross-feature cycles. If two packages need a type, place it with its owner or introduce a small interface/type package with an explicit reason.
- Apply `gofmt` to changed Go files. Add or update tests as part of normal feature implementation; this scaffold repository does not yet provide a complete verified test/build pipeline.

## Change checklist

Before a feature change is ready for review, confirm:

1. The feature owner and externally visible behavior are clear.
2. Domain and use-case packages do not import transport or vendor drivers.
3. Interfaces are defined by the consuming use case, not by an infrastructure package “just in case.”
4. Shared behavior was searched for in the [reusable package catalog](reusable-packages.md).
5. MongoDB mappings and query behavior stay inside the feature adapter.
6. `internal/app` is the only place that wires concrete adapters into use cases.
7. The feature follows the same dependency flow as existing features, or documents why a different boundary is needed.
