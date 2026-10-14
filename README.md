# Bento Go backend

Go implementation of the Bento backend. The repository is currently at the architecture and scaffold stage: most feature packages, the API entry point, database connections, and operational files are placeholders.

Read the [documentation index](docs/index.md) before adding a feature. It describes the intended dependency boundaries, the current package inventory, MongoDB direction, and rules for sharing code across features.

The `backend/` NestJS application is a business-behavior reference. Its module layout is not a Go package template; Go code should follow the boundaries and conventions documented here.
