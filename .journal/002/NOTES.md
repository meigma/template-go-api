---
id: 002
title: Implement API-server template per TARGET_SHAPE.md
started: 2026-06-22
---

## 2026-06-22 13:24 — Kickoff
Goal for the session: implement the Go web API server template per the agreed
design in `.journal/TARGET_SHAPE.md` (written and approved in session 001).

Current state of the world:
- Repo is a verbatim copy of the generic `template-go` Cobra/Viper CLI scaffold,
  re-referenced to `template-go-api` (PR #3, merged, commit `3d1edae` on `master`).
  No API code exists yet.
- `.journal/TARGET_SHAPE.md` is the approved v1 plan. Headline decisions: chi v5
  on net/http; Huma v2 code-first OpenAPI scoped to the transport layer; pragmatic
  ports & adapters; slog + OpenMetrics `/metrics`; full in-memory reference slice
  with `/healthz` + `/readyz`; Cobra root exposing `serve`/`version`/`openapi`;
  OpenAPI spec exported server-less → MkDocs neoteroi OAD.
- Guiding constraints (TECH_NOTES.md): hexagonal, functional-test-first, agile.

Open threads from session 001 (the implementation work):
- Entrypoint transformation (Cobra `serve`/`version`/`openapi`).
- domain/adapter/app/observability packages.
- Huma + chi wiring, in-memory reference slice.
- OpenAPI → MkDocs pipeline.
- Dependency additions (chi, huma, prometheus client, neoteroi-mkdocs).

Plan: awaiting the user's specific direction on where to start this session.
Will scope mechanical work into a plan and collaborate on design separately
(per memory `separate-mechanical-from-design-work`).

## 2026-06-22 14:20 — Slice 1 implemented, PR #4 opened

Direction: implement an initial vertical slice; be explicit about in/out scope.
Effort set to ultracode (xhigh + workflow orchestration) for this session.

Planning (plan mode): 3 Explore agents mapped the scaffold, 2 Plan agents designed
the slice (comprehensive vs thin). Key correction: both Plan agents read the stale
`journal/jmgilman` worktree and wrongly thought the rename was pending — verified
directly that `master` is already renamed, so NO rename step. Also confirmed
`exhaustruct` and `wrapcheck` are DISABLED in `.golangci.yml` (earlier explorers
misread); real constraints are `sloglint` (inject logger), `gochecknoglobals`
(metrics registry in a struct), `funlen`/`cyclop`, `godoclint`, `mnd`, `promlinter`.

Scope toggles confirmed with user (all "recommended"): metrics IN slice 1; openapi
export command writes the spec file but the neoteroi render pipeline is deferred;
core middleware stack (request-id → recover → access-log → timeout).

Implemented (worktree `feat/api-vertical-slice` off master):
- `internal/todo` domain (entity, Status enum, validation, idempotent Complete,
  consumer-defined Repository port, Service with now/newID seams).
- `internal/adapter/memory` (RWMutex map repo) and `internal/adapter/http`
  (chi + Huma v2 via humachi, 4 typed ops, DTO↔domain mapping, RFC 9457 errors,
  raw-chi /healthz /readyz /metrics).
- `internal/observability` (NewLogger, RequestLogger+request_id child + access log,
  custom slog Recoverer, Metrics struct owning its prometheus registry).
- `internal/config` reshaped to server config (Viper TEMPLATE_GO_API_*, Validate,
  RegisterFlags). `internal/app` composition root + graceful shutdown.
- `internal/cli` serve(default)/version/openapi; deleted `internal/templateinfo`.
- Functional-first tests through the real in-memory adapter; `docs/docs/openapi.yaml`
  committed (3.0.3 via DowngradeYAML).

Verification: `moon run check` green (format, strict golangci lint, build, test,
mkdocs --strict). `go test -race ./...` green. Live server curl-verified end to end
(201/404/422 problem+json, complete, list, /metrics labels, JSON access logs with
request_id, SIGTERM graceful shutdown). Deps: chi/v5, huma/v2, prometheus client,
google/uuid; testify promoted to direct. No cbor, no cors (deferred).

PR: https://github.com/meigma/template-go-api/pull/4 (open, CI running).

Deferred to later slices (seams left in place): docs render pipeline + docs:openapi
Moon wiring + CI drift-guard; CORS/client-IP; authn/authz; Postgres+testcontainers;
OTel; rate limiting; pagination; API versioning; mockery. Also pending: README/
DELETE_ME refresh for concrete API-server usage now that code has landed.

Possible refinement noted: service-level logs ("todo created") use the base logger,
not the request-scoped one, so they lack request_id; a later pass could thread the
context logger (LoggerFrom) into the service.
