---
title: template-go-api Docs
slug: /
description: Meigma starter for Go web (HTTP) API services.
---

# template-go-api

`template-go-api` is the Meigma starter for building Go web (HTTP) API services.
It ships a runnable, hexagonal API server (chi + Huma) with a `todo` example
resource backed by an in-memory store, alongside the shared Meigma repository
baseline (Moon tasks, pinned CI, Dependabot, and an enabled release layer).

## Quick start

```sh
moon run root:build
./bin/template-go-api serve   # listens on :8080
curl -sS -X POST localhost:8080/todos -H 'content-type: application/json' -d '{"title":"buy milk"}'
```

See the [README](https://github.com/meigma/template-go-api#readme) for the full
quickstart, configuration reference, and guidance on replacing the example
resource.

## API reference

The [API Reference](api.md) is generated from the OpenAPI specification. A
running server also serves interactive docs at `/docs` and the live spec at
`/openapi.yaml`.

## Operating notes

- Liveness: `GET /healthz`
- Readiness: `GET /readyz` (reports named per-check results)
- Metrics: `GET /metrics` (Prometheus exposition)

## Support and security

- Issues and contributions: see [CONTRIBUTING.md](https://github.com/meigma/template-go-api/blob/master/CONTRIBUTING.md).
- Security reports: see [SECURITY.md](https://github.com/meigma/template-go-api/blob/master/SECURITY.md).
