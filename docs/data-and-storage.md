---
type: Data Architecture
title: Data and storage
description: Defines the intended MongoDB repository boundary and records the current persistence scaffold state.
tags: [bento-backend, golang, mongodb, data]
sources:
  - resource: ../internal/infrastructure/mongodb/conn.go
    title: MongoDB client connection
  - resource: ../migrations/mongodb/.gitkeep
    title: MongoDB migrations placeholder
  - resource: ../config/config.go
    title: Database configuration schema
  - resource: ../go.mod
    title: Current Go dependencies
---
# Data and storage

## Direction

MongoDB is the target store for new feature persistence. The MongoDB client belongs to `internal/infrastructure/mongodb`; feature-specific query and mapping code belongs to the feature that owns the data.

The API uses the MongoDB v2 driver through `internal/infrastructure/mongodb`. Google sign-in stores users in the existing `users` collection and creates unique email and Google subject indexes at startup. The migration directory still has only a placeholder; the sign-in repository currently ensures its indexes during startup. Broader schema migration tooling remains to be implemented.

## Repository boundary

Use a feature-owned adapter to implement a port declared by that feature's use case:

```text
internal/features/<feature>/mongo_repository.go       repository interface
internal/features/<feature>/repository/mongodb/       MongoDB implementation
internal/infrastructure/mongodb/                      shared client and lifecycle
```

Repositories should accept `context.Context`, take feature models as inputs, and return models or application results. Keep Mongo filters, BSON documents, collection names, index names, and driver errors inside the adapter. Do not expose `*mongo.Client`, `*mongo.Collection`, `bson.M`, or database-specific identifiers through feature contracts.

The feature owns its data access. A different feature should call an explicit use case or consume a stable event/API; it should not query another feature's collection directly. If two features need the same underlying record, first determine its business owner and define the read contract there.

## Schema and index changes

Record schema and index changes in source-controlled, ordered migrations or a documented migration mechanism before deployment depends on them. A migration must be safe to run in each environment and have a defined rollout/rollback strategy where rollback is possible. Indexes should be created deliberately and reviewed with the query patterns they support.

The repository already has `migrations/mongodb` and `migrations/postgres`, but neither currently defines a usable migration process. Do not claim automatic migration behavior until a migration runner and its operational command are implemented and documented.

## PostgreSQL configuration

`config.Config` currently includes PostgreSQL fields and `migrations/postgres/schemas.sql` exists as an empty placeholder. Keep PostgreSQL use explicit: it may be needed for compatibility or a staged migration, but it is not the default for new feature storage under the current MongoDB direction. Avoid initializing both databases in every process unless the process actually needs both.

## Client lifecycle

The composition root should create the shared client once, configure connection/pool behavior from validated configuration, perform a bounded startup connectivity check where appropriate, inject it into repositories, and close it during graceful shutdown. Repositories should not create clients per request or own process-wide shutdown.
