---
okf_version: "0.2"
---

# Bento Go backend documentation

This index links to the maintained documentation for the Go backend. Pages distinguish the current repository state from conventions for new implementation.

## Start here

- [Architecture](architecture.md): dependency direction, feature boundaries, and the service composition model.
- [Project map](project-map.md): how the Go feature packages relate to the existing NestJS modules.
- [Implementation guide](implementation-guide.md): the expected workflow and code shape for a feature.
- [Reusable packages](reusable-packages.md): current shared code, ownership rules, and how to avoid duplicate helpers.
- [Data and storage](data-and-storage.md): MongoDB direction, repository boundaries, and migration expectations.
- [Configuration and observability](configuration-and-observability.md): configuration loading, logging, metrics, and tracing.
- [OpenAPI reference](openapi/openapi.json): machine-readable documentation for currently registered HTTP routes; served with Swagger UI at `/swagger/`.

## Current maturity

The folder structure is established, but most files are empty scaffolds. Treat package inventory as a code discovery guide, not a guarantee that every function is complete, covered, or safe to reuse. Check the implementation and its callers before building on it.
