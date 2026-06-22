# Target Shape — template-go-api (TEMPORARY)

> **Status: temporary design guide.** Records the agreed target shape for turning
> `template-go-api` into a reusable Go web API server template, so implementation
> passes (likely across multiple sessions) stay consistent. Delete or supersede
> once the template is built. Created 2026-06-22 (session 001); v1 for review.

## Purpose

`template-go-api` is currently a verbatim copy of the generic `template-go`
scaffold (a small Cobra/Viper CLI) that has only been re-referenced to its new
name. The goal is to specialize it into the standard **Meigma starter for Go
web/HTTP API services** that downstream repos inherit and customize.

This doc is the plan, not the implementation. No API code exists yet.

## Guiding constraints (from `.journal/TECH_NOTES.md`)

- **Hexagonal**: business logic isolated from transport, storage, and other
  external adapters.
- **Functional-test-first**: prove behavior end-to-end before calling a feature
  done; unit tests support but don't replace that.
- **Agile**: prototype early, refine from working behavior; underspecify where
  useful.

## Decisions (session 001)

| Area | Choice | Why (short) |
|------|--------|-------------|
| HTTP layer | **chi v5 on net/http** (stdlib `ServeMux` fallback) | 100% `http.Handler` types → thin, portable adapters; tiny stable core; low inheritance risk; near-zero churn to/from stdlib. |
| OpenAPI | **Huma v2 (code-first), transport-scoped** | Typed operations generate OpenAPI 3.1 + validation; spec can't drift from code. Used *only* for the OpenAPI problem. |
| Docs | Build-time server-less spec export → **MkDocs via neoteroi OAD** | Spec is a build artifact wired into Moon `docs:build`; static, themed, searchable reference. |
| Layering | **Pragmatic ports & adapters** | Consumer-defined ports, inward dependency arrows, composition-root wiring; interface only where substitution is needed. |
| Cross-cutting | **slog** + **OpenMetrics `/metrics`** | Structured logging always on; lightweight ubiquitous metrics; tracing left as opt-in OTel seam. |
| Reference slice | **Full vertical slice, in-memory store** + `/healthz` + `/readyz` | Demonstrates the whole hexagonal flow with zero infra; runnable + functional-testable out of the box; teams swap the store. |
| Entrypoint | **Cobra root** with `serve` (default), `version`, `openapi` | Keeps existing Cobra/Viper; server launcher replaces the demo "message" CLI; `openapi` dumps the spec without starting the server. |
| Testing | **Functional-first** via `httptest` through the in-memory adapter | Table-driven handler/API tests, no infra; `testcontainers-go` reserved for when a real DB adapter is added. |

### Scoping notes for Huma (OpenAPI only)

- **Take**: `humachi` adapter, `huma.Register` typed operations, input/output
  structs with tags, schema validation, RFC 9457 errors, OpenAPI 3.1 generation.
- **Leave**: `humacli` (keep Cobra/Viper), and other out-of-scope extras.
- **Middleware stays at the chi level** (`func(http.Handler) http.Handler`); only
  register at the Huma level when it must appear in the spec (e.g. security schemes).
- **Spec-completeness convention**: API endpoints go through Huma (and thus the
  spec); only infra routes (`/healthz`, `/readyz`, `/metrics`) may be raw chi.
- **Tagged structs are transport DTOs** in the HTTP adapter, mapped to/from domain
  types. Tags do *structural* validation; *business-rule* validation lives in the
  service.
- **Accepted risk**: Huma's bus factor (~1 maintainer, ~4.2k★) is contained to the
  transport adapter; if it stalled, migration = rewrite handlers to plain chi with
  a different OpenAPI approach, domain untouched.

## Proposed package layout

```
cmd/template-go-api/        # main: build Cobra root, execute
internal/
  app/                      # composition root: construct deps, wire, run serve
  config/                   # Viper config (server addr, timeouts, log, CORS, ...)
  <resource>/               # domain core for the example resource (e.g. "todo")
    <resource>.go           #   entities + business rules
    service.go              #   use-case logic (the service)
    ports.go                #   outbound port interfaces it consumes (Repository)
  adapter/
    http/                   # inbound: Huma handlers, DTOs, domain<->DTO mapping,
                            #   router assembly, middleware, health/metrics wiring
    memory/                 # outbound: in-memory impl of <resource>.Repository
  observability/ (or obs/)  # slog setup + OpenMetrics registry/handler
docs/docs/openapi.yaml      # generated spec (build artifact; see pipeline)
```

Dependency arrows point **inward**: `adapter/*` and `app` depend on the domain
package; the domain package depends on nothing in `adapter`. Interfaces are
declared by their consumer (the domain/service), implemented by outbound adapters.

## Request flow (inbound)

```
HTTP request
  → chi router + middleware (request id, recovery, access log, timeout, clientIP, CORS)
  → Huma operation (decode + validate Input DTO)
  → adapter/http handler: map DTO → domain call
  → <resource>.Service (business logic)
  → <resource>.Repository port → adapter/memory (in-memory store)
  ← domain result → map → Output DTO → Huma (encode, content negotiation)
```

## Cross-cutting detail

- **Config** (Viper, `TEMPLATE_GO_API_*` env): listen address/port, read/write/
  idle/header timeouts, shutdown grace period, log level + format, CORS origins.
- **Logging**: `log/slog`, injected `*slog.Logger` (no global), JSON handler by
  default; request-scoped child logger carrying the request id.
- **Middleware order** (chi): request id → recovery → access log → timeout →
  `ClientIP` (chi's security-fixed one) → CORS (configurable).
- **Metrics**: OpenMetrics `/metrics` (Prometheus client) — HTTP server metrics
  (request count/duration/in-flight) + Go runtime metrics.
- **Tracing**: not baked in; leave an opt-in OTel seam for a later pass.
- **Graceful shutdown**: `http.Server` with the configured timeouts +
  `signal.NotifyContext` (SIGINT/SIGTERM) + `server.Shutdown(ctx)` on a drain deadline.

## OpenAPI & docs pipeline

1. **Server-less export**: an `openapi` command (on our Cobra root) or a small
   `go run ./tools/openapi` builds the `api` and writes `docs/docs/openapi.yaml`
   via `api.OpenAPI().DowngradeYAML()` (3.0.3 for the renderer; `.YAML()` = 3.1).
2. **Moon**: a new `docs:openapi` (Go) task produces the spec; the existing
   `docs:build` (`mkdocs build --strict`) depends on it → spec can't drift.
3. **Render**: neoteroi OAD (`[OAD(./openapi.yaml)]`), static + themed + searchable.
4. Optional **drift-guard** CI check: regeneration produces no diff.
5. Huma's runtime `/docs` (Stoplight Elements) remains a separate, free surface
   from the live server.

## Dependencies to add

- `github.com/go-chi/chi/v5`
- `github.com/danielgtaylor/huma/v2` (+ `adapters/humachi`)
- Prometheus Go client (`github.com/prometheus/client_golang`) for `/metrics`
- Docs (uv project): `neoteroi-mkdocs`
- CBOR and other Huma extras: opt out unless needed.

## Out of scope (future passes / seams left open)

- Authn/authz (leave a middleware + Huma security-scheme seam).
- Real persistence (Postgres outbound adapter + `testcontainers-go` tests).
- OTel tracing exporter (seam only for now).
- Rate limiting, pagination conventions, API versioning policy.

## Delta from the current scaffold

- `internal/cli/root.go`: replace the demo `--message` command with a Cobra root
  exposing `serve` (default), `version`, `openapi`.
- `internal/templateinfo` + `internal/config`: reshape toward server/runtime
  config; drop the message-demo bits.
- Add the domain/adapter/app/observability packages above.
- README/DELETE_ME: update the now-accurate "API server" specifics once code lands
  (this pass only reframed identity; concrete usage docs come with implementation).
```
