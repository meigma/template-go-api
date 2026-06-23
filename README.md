# template-go-api

`template-go-api` is the Meigma starter for building Go web (HTTP) API services.
It ships a runnable, hexagonal API server — built on [chi](https://github.com/go-chi/chi)
and [Huma](https://huma.rocks) with a `todo` example resource — plus the shared
Meigma repository baseline: Moon tasks, pinned CI, Dependabot, baseline security
settings, and an enabled Release Please and GoReleaser release layer.

Persistence is a PostgreSQL adapter behind the domain's `todo.Repository` port
([pgx](https://github.com/jackc/pgx) + [sqlc](https://sqlc.dev) typed queries,
[goose](https://github.com/pressly/goose) migrations). The port stays the seam:
implement it to back the template with a different datastore without touching the
domain or transport.

The example resource is a reference slice, not a product feature: swap it for
your own resource and keep the persistence wiring.

## Prerequisites

- Go 1.26.4
- Moon 2.x
- Python 3.14.3 and uv 0.11.0 (only for the MkDocs documentation project)
- Docker (to run a local PostgreSQL for the server — see [Persistence](#persistence)
  and [Local stack](#local-stack-docker-compose) — and for the container-backed
  integration tests)

The `sqlc`, `goose`, and `mockery` CLIs are pinned in `.prototools` and run
through Proto, so they are fetched on demand by Moon — there is nothing to install
by hand.

> **New repository from this template?** Work through [DELETE_ME.md](DELETE_ME.md)
> first — it covers renaming the module, binary, image, and env prefix, and
> replacing the example resource.

## Quickstart

The server persists to PostgreSQL, so running it needs a database. The fastest
way to bring up the whole stack — database, migrations, seed data, and the API —
is Docker Compose:

```sh
docker compose up --build     # API on :8080, /metrics on :9090 (see "Local stack" below)
```

To build the binary and run it against your own PostgreSQL instead, see
[Persistence](#persistence) (`serve` is the default subcommand and needs
`--database-url`):

```sh
moon run root:build          # or: go build -o bin/template-go-api ./cmd/template-go-api
```

With the stack up, exercise the example `todo` API:

```sh
# Create a todo
curl -sS -X POST localhost:8080/todos \
  -H 'content-type: application/json' \
  -d '{"title":"buy milk"}'
# => 201 {"id":"...","title":"buy milk","status":"open","createdAt":"...","completedAt":null}

curl -sS localhost:8080/todos                 # list
curl -sS localhost:8080/todos/<id>            # fetch one (404 if unknown)
curl -sS -X POST localhost:8080/todos/<id>/complete   # mark complete

# Validation and not-found errors use RFC 9457 problem+json:
curl -sS -i -X POST localhost:8080/todos -H 'content-type: application/json' -d '{"title":""}'
# => 422 application/problem+json
```

Operational endpoints:

```sh
curl -sS localhost:8080/healthz   # liveness  => {"status":"ok"}
curl -sS localhost:8080/readyz    # readiness => {"status":"ready","checks":{}}
curl -sS localhost:9090/metrics   # Prometheus exposition (separate listener)
```

`/metrics` is served on a dedicated listener (`--metrics-addr`, default `:9090`)
so it stays off the public API surface and outside the API middleware chain; set
`--metrics-addr ""` to co-locate it on the API port instead.

The running server also serves interactive API docs at `/docs` (Stoplight
Elements) and the live spec at `/openapi.yaml` and `/openapi.json`.

## Local stack (Docker Compose)

`docker compose up --build` brings up the **full** template against PostgreSQL —
no local Go toolchain or database setup required:

```sh
docker compose up --build
curl -sS localhost:8080/todos    # => the seeded todos
curl -sS localhost:8080/readyz   # => {"status":"ready","checks":{"postgres":"ok"}}
```

Startup is an ordered DAG, because migrations are explicit (the server never runs
them) and the seed data needs the schema to exist first:

| Step | Service    | What it does                                                          |
|------|------------|----------------------------------------------------------------------|
| 1    | `postgres` | PostgreSQL 17 with a known config; the stack waits for `pg_isready`.  |
| 2    | `migrate`  | One-shot `migrate up` — applies the embedded goose migrations.        |
| 3    | `seed`     | One-shot — applies every `hack/sql/*.sql` (sorted) with `psql`.       |
| 4    | `api`      | Serves the API against the prebaked connection string.               |

The database is **ephemeral and reproducible**: no volume is persisted, so every
`up` rebuilds a clean database, re-runs migrations, and re-applies the seeds;
`docker compose down` discards it.

Prepopulate local data by dropping SQL files in [`hack/sql/`](hack/sql/) — they
run after the schema exists, so you can `INSERT` straight into tables like `todos`
without touching migrations or adding setup code to the server. The bundled
`hack/sql/0001_seed_todos.sql` seeds a few todos so the API returns data on the
first request.

## Commands

| Command | Description |
| --- | --- |
| `serve` (default) | Run the HTTP API server. |
| `version` | Print version, commit, and build date. |
| `openapi` | Write the OpenAPI 3.0.3 spec to stdout or a file (`--output/-o`). |
| `migrate up\|down\|status` | Apply, roll back, or report the embedded PostgreSQL migrations against `--database-url`. |

```sh
./bin/template-go-api openapi -o docs/docs/openapi.yaml
./bin/template-go-api version
./bin/template-go-api migrate status --database-url postgres://app:app@localhost:5432/app?sslmode=disable
```

## Configuration

Flags bind to Viper, so every setting is also a `TEMPLATE_GO_API_*` environment
variable (uppercase, dashes become underscores). Precedence is flag > env >
default.

| Flag | Env var | Default | Description |
| --- | --- | --- | --- |
| `--addr` | `TEMPLATE_GO_API_ADDR` | `:8080` | host:port the API listens on |
| `--metrics-addr` | `TEMPLATE_GO_API_METRICS_ADDR` | `:9090` | dedicated `/metrics` listener; empty serves `/metrics` on `--addr` |
| `--log-level` | `TEMPLATE_GO_API_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` |
| `--log-format` | `TEMPLATE_GO_API_LOG_FORMAT` | `json` | `json` or `text` |
| `--read-timeout` | `TEMPLATE_GO_API_READ_TIMEOUT` | `5s` | reading an entire request |
| `--read-header-timeout` | `TEMPLATE_GO_API_READ_HEADER_TIMEOUT` | `5s` | reading request headers |
| `--write-timeout` | `TEMPLATE_GO_API_WRITE_TIMEOUT` | `10s` | writing the response |
| `--idle-timeout` | `TEMPLATE_GO_API_IDLE_TIMEOUT` | `120s` | idle keep-alive connections |
| `--request-timeout` | `TEMPLATE_GO_API_REQUEST_TIMEOUT` | `15s` | per-request processing |
| `--shutdown-grace` | `TEMPLATE_GO_API_SHUTDOWN_GRACE` | `15s` | graceful shutdown window |
| `--cors-allowed-origins` | `TEMPLATE_GO_API_CORS_ALLOWED_ORIGINS` | _(none)_ | allowed CORS origins (comma-separated); empty disables CORS |
| `--trusted-proxy-header` | `TEMPLATE_GO_API_TRUSTED_PROXY_HEADER` | _(none)_ | proxy header to read the client IP from (e.g. `X-Real-IP`); empty trusts the TCP peer |
| `--database-url` | `TEMPLATE_GO_API_DATABASE_URL` | _(none)_ | PostgreSQL connection URL (**required**) |
| `--db-max-conns` | `TEMPLATE_GO_API_DB_MAX_CONNS` | `0` | maximum PostgreSQL pool connections; `0` uses the driver default |

CORS is off until you set origins. Client IP is read from the direct TCP peer
unless you opt into a trusted proxy header — never from `X-Forwarded-For`
implicitly — so the default is not spoofable.

`--database-url` is **required**; the server rejects a missing URL at startup.

## Persistence

The `todo.Repository` port is implemented by a PostgreSQL adapter under
`internal/todo/postgres`: [sqlc](https://sqlc.dev) type-safe queries over a
[pgx](https://github.com/jackc/pgx) connection pool, with
[goose](https://github.com/pressly/goose) migrations. The shared connection pool
and migration machinery stay under `internal/adapter/postgres` as database-level
concerns. The composition root in `internal/app/app.go` wires the adapter and
registers a `postgres` readiness check so `/readyz` reflects database
connectivity.

The port is the extension seam: to back the template with a different datastore,
implement `todo.Repository` in a new adapter and wire it in `app.go` (or inject it
via `app.WithRepository`) — the domain and transport layers stay untouched.

### Running with PostgreSQL

Start a database (any reachable PostgreSQL works; this is just an example):

```sh
docker run --rm -d --name template-pg \
  -e POSTGRES_PASSWORD=app -e POSTGRES_USER=app -e POSTGRES_DB=app \
  -p 5432:5432 postgres:17-alpine
export TEMPLATE_GO_API_DATABASE_URL='postgres://app:app@localhost:5432/app?sslmode=disable'
```

Apply migrations, then serve against the database:

```sh
./bin/template-go-api migrate up           # create the schema
./bin/template-go-api serve                # reads TEMPLATE_GO_API_DATABASE_URL
curl -sS localhost:8080/readyz             # => {"status":"ready","checks":{"postgres":"ok"}}
```

`--database-url` (env `TEMPLATE_GO_API_DATABASE_URL`) is shared by `serve` and
`migrate`. The connection URL can also be passed as a flag instead of an
environment variable.

### Migrations

Migrations live in `internal/adapter/postgres/migrations/*.sql` (goose format) and
are embedded in the binary. They are **explicit** — `serve` never runs them, which
avoids multi-replica races. The `migrate` subcommand drives goose as a library:

```sh
./bin/template-go-api migrate up       # apply all pending migrations
./bin/template-go-api migrate status   # show applied/pending versions
./bin/template-go-api migrate down     # roll back the most recent migration
```

Moon wraps the same command for local dev (arguments after `--` pass through):

```sh
moon run root:migrate -- up --database-url "$TEMPLATE_GO_API_DATABASE_URL"
```

Scaffold a new migration file with the Proto-managed goose CLI, then edit its
`-- +goose Up` / `-- +goose Down` sections:

```sh
proto run goose -- -dir internal/adapter/postgres/migrations create add_something sql
```

Because sqlc reads the migrations directory as its schema, a schema change means
regenerating the typed query layer (below).

### Type-safe queries (sqlc)

Hand-written queries live in `internal/todo/postgres/queries/todos.sql`; sqlc
generates the typed Go in `internal/todo/postgres/sqlc/` from those queries and
the migration schema. The generated package is **committed and drift-guarded**
(mirroring the `openapi` / `openapi-check` pattern). After changing a migration or
a query, regenerate and commit:

```sh
moon run root:sqlc       # regenerate internal/todo/postgres/sqlc/
```

`moon run root:sqlc-check` (part of `root:check`) regenerates into a throwaway
directory and fails if the committed code is stale, so the generated layer can
never drift from the schema and queries. The `sqlc.yaml` config maps the `uuid`
column to `github.com/google/uuid.UUID` and `timestamptz` to `time.Time` /
`*time.Time`; the adapter converts to and from the domain's `string` ID and
`Status` at the mapping boundary.

### Integration tests

Integration tests live in their own package, `internal/integration` (package
`integration`, `//go:build integration`), separate from the unit tests that sit
beside the code, and drive the adapters through their public APIs. The suite is
container-backed and behind the build tag, so the default `go test ./...` and
`moon run root:check` stay hermetic (no Docker). It uses
[testcontainers](https://golang.testcontainers.org/) to spin a throwaway
`postgres:17-alpine`, applies the embedded migrations, and snapshots the clean
schema for fast per-test isolation. It requires a running Docker daemon:

```sh
moon run root:test-integration   # or: go test -tags integration ./internal/integration/...
```

Wiring `test-integration` into CI is a follow-up: the GitHub workflows are
currently `.disabled` and need a Docker-capable runner.

### Dynamic queries

This template ships **no query-builder dependency**. The three port methods are
fixed queries, and `List` is parameter-free. For optional filtering that sqlc can
still type-check, use `sqlc.narg()` (nullable named arguments) — a commented
example lives in `queries/todos.sql`: a `NULL` argument disables the filter, a
non-`NULL` value applies it, all in one prepared statement.

When you genuinely need queries assembled at runtime (variable column sets,
arbitrary `AND`/`OR` trees), keep that complexity **inside the adapter, behind the
port**. Define a criteria struct on the port and translate it to SQL in the
PostgreSQL adapter, reaching for [Squirrel](https://github.com/Masterminds/squirrel)
or [Bob](https://github.com/stephenafamo/bob) *there*:

```go
// In the domain port (internal/todo) — no builder types appear here.
type TodoFilter struct {
    Status *Status // nil means "any status"
}

// List(ctx context.Context, f TodoFilter) ([]Todo, error)
```

The rule: query-builder types must never appear in a port signature. The domain
speaks in domain criteria; only the adapter knows SQL. Swapping Squirrel for Bob,
or back to plain sqlc, then stays a change inside one package.

## Testing

Unit tests sit beside the code and use [Testify](https://github.com/stretchr/testify)
(`assert` / `require`). Repository doubles are **mockery-generated** testify mocks,
committed under `internal/todo/mocks` and drift-guarded like the sqlc layer:

```sh
moon run root:mockery        # regenerate internal/todo/mocks from the ports
```

`moon run root:mockery-check` (part of `root:check`) regenerates into a throwaway
directory and fails if the committed mocks are stale, so they can never drift from
the interfaces. Add a port to `.mockery.yaml`, then regenerate and commit. Use the
generated mock for interaction and error-injection assertions — see
`internal/todo/service_test.go`.

For tests that need a real, stateful store end to end — such as the HTTP
functional tests that create a todo and read it back — a small in-memory fake
lives in `internal/todo/todotest`, kept deliberately separate from the generated
mocks. The container-backed [integration tests](#integration-tests) cover the
PostgreSQL adapter against a real database.

## Project layout

The server follows pragmatic hexagonal (ports & adapters) layering: the domain
core depends on nothing in the adapters, and dependencies point inward.

```
cmd/template-go-api/        thin main; builds the Cobra root and executes
internal/
  cli/                      serve / version / openapi / migrate commands, Viper wiring
  config/                   server runtime config (flags + TEMPLATE_GO_API_* env)
  todo/                     domain: entity, Repository port, Service (the example)
    httpapi/                inbound transport: the todo resource's DTOs, mapping, handlers
    postgres/               outbound adapter: PostgreSQL Repository (pgx + sqlc)
      queries/              hand-written sqlc queries
      sqlc/                 generated, committed, drift-guarded query layer
    mocks/                  generated testify mock of the Repository port (mockery)
    todotest/               in-memory Repository fake for tests
  adapter/                  shared, cross-domain infrastructure (not domain-specific)
    http/                   generic transport: chi router, middleware, RFC 9457 errors,
                            /healthz /readyz /metrics, OpenAPI export, Registrar seam
    postgres/               connection pool (Connect) + goose migrate library
      migrations/           embedded goose migrations (also sqlc's schema source)
  observability/            slog logger, request logging, Prometheus metrics
  logctx/                   carries the request-scoped logger on the context
  app/                      composition root: wires everything and runs the server
  integration/              container-backed integration tests (build tag: integration)
compose.yaml                day-one local stack: postgres + migrate + seed + api
hack/sql/                   *.sql seeds applied to the Compose database (local dev)
docs/                       MkDocs site; docs/docs/openapi.yaml is the exported spec
sqlc.yaml                   sqlc generation config (repo root)
.mockery.yaml               mockery generation config (repo root)
```

## Adding a resource

Replace or extend the `todo` example by following the same seams:

Each resource owns its code under `internal/<resource>/`: the domain core at the
package root, with its adapters nested beneath it.

1. Add a domain package `internal/<resource>` (entity + `Repository` port + `Service`), mirroring `internal/todo`.
2. Implement the port in a nested adapter — mirror `internal/todo/postgres` for a PostgreSQL-backed datastore (see [Persistence](#persistence) for the sqlc/goose workflow).
3. Add a transport adapter `internal/<resource>/httpapi` (DTOs, domain mapping, error translation, and a `Register` function), mirroring `internal/todo/httpapi`.
4. Add one `Register` call in `registerResources` in `internal/app/app.go`.

Shared, cross-domain infrastructure needs no changes: the generic transport in
`internal/adapter/http` and the connection pool / migrations in
`internal/adapter/postgres`. Because each resource's `postgres` package shares its
name with that shared infra (and, across resources, with each other), import the
per-resource adapters with aliases in `app.go` — as the `todopostgres` import
already shows. After changing the API, run `moon run openapi` to refresh the
committed spec (CI fails if it drifts). If you back the resource with PostgreSQL,
also add its readiness check to the `Readiness` slice in `internal/app/app.go` so
`/readyz` reflects it.

## Documentation

The MkDocs site publishes to GitHub Pages at
<https://meigma.github.io/template-go-api/>, including a generated
[API Reference](https://meigma.github.io/template-go-api/api/) rendered from the
OpenAPI spec. Build it locally with `moon run docs:build` or preview with
`moon run docs:serve`.

## Common tasks

Moon is the standard task front door:

```sh
moon run root:format
moon run root:lint
moon run root:build
moon run root:test
moon run root:check    # the aggregate gate CI runs via `moon ci --summary minimal`
```

Persistence- and testing-related tasks (see [Persistence](#persistence) and
[Testing](#testing)):

```sh
moon run root:sqlc              # regenerate the committed sqlc query layer
moon run root:mockery           # regenerate the committed testify mocks
moon run root:migrate -- up     # apply migrations (pass --database-url after --)
moon run root:test-integration  # container-backed adapter tests (needs Docker)
```

`root:check` already runs `sqlc-check` and `mockery-check` (drift guards for the
generated query layer and mocks) alongside the formatter, linter, build, tests,
and OpenAPI drift guard.

## Container Image

The included Dockerfile builds a static Linux binary and copies it into a
non-root distroless runtime image. The default entrypoint runs the server:

```sh
docker build --target test .
docker build -t template-go-api:dev .
docker run --rm -p 8080:8080 template-go-api:dev
```

The Dockerfile pins the builder and runtime images by digest and verifies that
the selected Go builder image matches `.go-version`. When bumping Go, update
`.go-version` and the builder `FROM` tag/digest together.

Release builds can pass the same binary metadata injected by GoReleaser:

```sh
docker build \
  --build-arg VERSION="$(git describe --tags --always --dirty)" \
  --build-arg COMMIT="$(git rev-parse HEAD)" \
  --build-arg DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -t template-go-api:dev .
```

## CI and Security

The default CI workflow keeps permissions minimal, pins external actions, disables checkout credential persistence, and delegates checks to Moon.
It uses GitHub-hosted dependency caches for Go, golangci-lint, and uv download artifacts while leaving Moon remote caching as an optional follow-up for repositories that need a shared task-output cache.
The docs workflow builds the MkDocs site on pull requests and deploys `docs/build` to GitHub Pages from the default branch.
The scheduled security scan workflow builds the local container image weekly, scans it for high/critical fixed vulnerabilities, and uploads SARIF results to GitHub code scanning.
Dependabot covers GitHub Actions, Docker base images, the root Go module, and the docs uv project.

Repository settings live in `.github/repository-settings.toml`.
They default to immutable releases, private vulnerability reporting, signed commits, squash-only merges, GitHub Pages workflow publishing, and protected tags.

## Release Layer

Release automation is enabled for the template application so this repository proves the full binary and container release lifecycle before generated projects inherit it.
Repositories generated from the template should update the release app credentials, package names, asset patterns, container image name, and `ghd.toml` signer workflow before cutting their first release.

The release path is:

- Release Please opens and maintains the release PR.
- Release Please creates a draft GitHub release and tag after merge.
- Release Dry Run rehearses the GoReleaser binary path and native-runner Docker container build path on pull requests.
- GoReleaser builds binaries, checksums, and SBOMs without publishing directly.
- The release workflow uploads assets to the draft release and creates a GitHub-hosted attestation for `checksums.txt`.
- The release workflow builds amd64 and arm64 container images on native GitHub-hosted runners, publishes `ghcr.io/meigma/template-go-api:vX.Y.Z` as a multi-platform manifest, attaches BuildKit provenance and SBOM metadata, and creates a GitHub-native attestation for the manifest digest.
- A human inspects the draft release before publication.

The root `ghd.toml` matches the default GoReleaser output so generated projects can be installed with `ghd` once the release workflow runs.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines, local setup expectations, and pull request workflow.

## Security

See [SECURITY.md](SECURITY.md) for supported versions and the private vulnerability reporting path.

## License

Add the repository license before publishing a project generated from this template.
