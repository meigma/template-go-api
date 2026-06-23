---
title: template-go-api Docs
slug: /
description: Meigma starter for Go web (HTTP) API services.
---

# template-go-api

`template-go-api` is the Meigma starter for building Go web (HTTP) API services.
It ships a runnable, hexagonal API server (chi + Huma) with a `todo` example
resource, alongside the shared Meigma repository baseline (Moon tasks, pinned CI,
Dependabot, and an enabled release layer). Two persistence adapters sit behind one
port: an in-memory store (the zero-infrastructure default) and a PostgreSQL
adapter (pgx + sqlc + goose), selected at runtime with `--store=memory|postgres`.

## Quick start

```sh
moon run root:build
./bin/template-go-api serve   # listens on :8080
curl -sS -X POST localhost:8080/todos -H 'content-type: application/json' -d '{"title":"buy milk"}'
```

To run against PostgreSQL instead, apply the embedded migrations and switch the
store:

```sh
# start a throwaway PostgreSQL (or point at your own)
docker run --rm -d -p 5432:5432 \
  -e POSTGRES_USER=app -e POSTGRES_PASSWORD=app -e POSTGRES_DB=app postgres:17-alpine
export TEMPLATE_GO_API_DATABASE_URL='postgres://app:app@localhost:5432/app?sslmode=disable'
./bin/template-go-api migrate up
./bin/template-go-api serve --store=postgres
```

See the [README](https://github.com/meigma/template-go-api#readme) for the full
quickstart, configuration reference, the
[Persistence](https://github.com/meigma/template-go-api#persistence) workflow
(migrations, sqlc regeneration, integration tests, dynamic queries), and guidance
on replacing the example resource.

## API reference

The [API Reference](api.md) is generated from the OpenAPI specification. A
running server also serves interactive docs at `/docs` and the live spec at
`/openapi.yaml`.

## Operating notes

- Liveness: `GET /healthz`
- Readiness: `GET /readyz` (reports named per-check results; `--store=postgres` adds a `postgres` connectivity check)
- Metrics: `GET /metrics` on a dedicated listener (`--metrics-addr`, default `:9090`)
- Migrations are explicit: `serve` never runs them; use the `migrate up|down|status` subcommand.

## Support and security

- Issues and contributions: see [CONTRIBUTING.md](https://github.com/meigma/template-go-api/blob/master/CONTRIBUTING.md).
- Security reports: see [SECURITY.md](https://github.com/meigma/template-go-api/blob/master/SECURITY.md).
