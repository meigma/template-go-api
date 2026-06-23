# template-go-api

`template-go-api` is the Meigma starter for building Go web (HTTP) API services.
It ships a runnable, hexagonal API server — built on [chi](https://github.com/go-chi/chi)
and [Huma](https://huma.rocks) with a `todo` example resource backed by an
in-memory store — plus the shared Meigma repository baseline: Moon tasks, pinned
CI, Dependabot, baseline security settings, and an enabled Release Please and
GoReleaser release layer.

The example resource is a reference slice, not a product feature: swap it for
your own resource and replace the in-memory store with a real datastore.

## Prerequisites

- Go 1.26.4
- Moon 2.x
- Python 3.14.3 and uv 0.11.0 (only for the MkDocs documentation project)

> **New repository from this template?** Work through [DELETE_ME.md](DELETE_ME.md)
> first — it covers renaming the module, binary, image, and env prefix, and
> replacing the example resource.

## Quickstart

Build and run the server:

```sh
moon run root:build          # or: go build -o bin/template-go-api ./cmd/template-go-api
./bin/template-go-api serve   # listens on :8080; `serve` is the default subcommand
```

Exercise the example `todo` API:

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
curl -sS localhost:8080/metrics   # Prometheus exposition
```

The running server also serves interactive API docs at `/docs` (Stoplight
Elements) and the live spec at `/openapi.yaml` and `/openapi.json`.

## Commands

| Command | Description |
| --- | --- |
| `serve` (default) | Run the HTTP API server. |
| `version` | Print version, commit, and build date. |
| `openapi` | Write the OpenAPI 3.0.3 spec to stdout or a file (`--output/-o`). |

```sh
./bin/template-go-api openapi -o docs/docs/openapi.yaml
./bin/template-go-api version
```

## Configuration

Flags bind to Viper, so every setting is also a `TEMPLATE_GO_API_*` environment
variable (uppercase, dashes become underscores). Precedence is flag > env >
default.

| Flag | Env var | Default | Description |
| --- | --- | --- | --- |
| `--addr` | `TEMPLATE_GO_API_ADDR` | `:8080` | host:port to listen on |
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

CORS is off until you set origins. Client IP is read from the direct TCP peer
unless you opt into a trusted proxy header — never from `X-Forwarded-For`
implicitly — so the default is not spoofable.

## Project layout

The server follows pragmatic hexagonal (ports & adapters) layering: the domain
core depends on nothing in the adapters, and dependencies point inward.

```
cmd/template-go-api/        thin main; builds the Cobra root and executes
internal/
  cli/                      serve / version / openapi commands, Viper wiring
  config/                   server runtime config (flags + TEMPLATE_GO_API_* env)
  todo/                     domain: entity, Repository port, Service (the example)
  adapter/
    memory/                 outbound adapter: in-memory Repository implementation
    http/                   inbound transport: chi router, middleware, RFC 9457
                            errors, /healthz /readyz /metrics, OpenAPI export
      todoapi/              the todo resource's transport (DTOs, mapping, handlers)
  observability/            slog logger, request logging, Prometheus metrics
  logctx/                   carries the request-scoped logger on the context
  app/                      composition root: wires everything and runs the server
docs/                       MkDocs site; docs/docs/openapi.yaml is the exported spec
```

## Adding a resource

Replace or extend the `todo` example by following the same seams:

1. Add a domain package under `internal/<resource>` (entity + `Repository` port + `Service`), mirroring `internal/todo`.
2. Implement the port — start from `internal/adapter/memory`, swap for a real datastore later.
3. Add a transport adapter under `internal/adapter/http/<resource>api` (DTOs, domain mapping, error translation, and a `Register` function), mirroring `todoapi`.
4. Add one `Register` call in `registerResources` in `internal/app/app.go`.

The generic transport in `internal/adapter/http` needs no changes. After changing
the API, run `moon run openapi` to refresh the committed spec (CI fails if it drifts).

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
