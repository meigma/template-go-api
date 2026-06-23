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

## 2026-06-22 15:45 — PR #4 merged

Interactive review on the PR drove four follow-up commits before merge:
1. `fix`: bounded metrics label cardinality — `route` was already safe (chi pattern,
   unmatched → "unmatched"), but `method` echoed raw `r.Method`; arbitrary method
   tokens collapse to "other" now.
2. `refactor`: split the todo transport into `internal/adapter/http/todoapi`; the
   generic `internal/adapter/http` is now resource-agnostic (zero `todo` imports)
   and resources mount via a `Registrar func(huma.API)` seam composed in `app`.
3. `refactor`: `Readiness []func() error` → named context-aware `ReadinessCheck`
   (`func(context.Context) error`); `/readyz` threads the request context.
4. `fix`: RFC 9457 standardization. A verification workflow (5 agents: code audit
   + live-server probe + synthesis) found Huma errors conformed but every non-Huma
   surface did not (chi default 404 text/plain, 405 empty body, panic bare 500,
   timeout bare 504). Added a single problem+json writer; wired mux.NotFound/
   MethodNotAllowed (405 rebuilds the Allow header by probing routes); moved the
   panic Recoverer into transport; replaced chi's Timeout with a problem+json 504.
   /healthz, /readyz, /metrics intentionally excluded (documented in code).

Merged via squash → master `745a9ed`. Remote branch + `feat/api-vertical-slice`
worktree removed; master updated locally. CI green (ci, Pages, Kusari).

State of the world: master is now a runnable hexagonal Go API server template
(todo reference slice). TARGET_SHAPE.md slice-1 scope is complete.

Open follow-ups for future slices (all seams already in place): docs render pipeline
(neoteroi OAD + docs:openapi Moon wiring + drift-guard); CORS/client-IP; authn/authz;
Postgres adapter + testcontainers; OTel; rate limiting; pagination; API versioning;
README/DELETE_ME refresh for concrete API usage; thread request-scoped logger into
the service; optionally richer (named) readiness reporting.

## 2026-06-22 18:12 — Close

Closed retroactively during session 003's close-out (the session was never formally
closed after PR #4 merged). Work was complete and merged: slice 1 →
`master 745a9ed` (PR #4). All "open follow-ups" above were picked up and completed in
session 003 (PR #5, `05f5446`). See `.journal/002/SUMMARY.md` for the postmortem.
Status flipped to `complete` in INDEX.
