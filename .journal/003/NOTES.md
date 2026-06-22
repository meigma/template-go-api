---
id: 003
title: Continue API-server template — slice 2 / deferred follow-ups
started: 2026-06-22
---

## 2026-06-22 16:02 — Kickoff
Goal for the session: continue the API-server template work begun in session 002,
picking up the deferred follow-ups from `.journal/TARGET_SHAPE.md`. First task
requested by the user: review session 002's notes and summarize what was / wasn't
implemented, so we have a clear starting foundation before choosing the next slice.

Current state of the world:
- `master` is a runnable hexagonal Go API server template. Slice 1 of
  TARGET_SHAPE.md is **complete and merged** (PR #4, squash → `master 745a9ed`).
- Shipped in slice 1: `internal/todo` domain (entity + Status enum + validation +
  idempotent Complete + consumer-defined Repository port + Service with now/newID
  seams); `internal/adapter/memory` (RWMutex map repo); `internal/adapter/http`
  (resource-agnostic chi+Huma assembly, problem+json on every surface) with
  `internal/adapter/http/todoapi` (4 typed ops, DTO↔domain, mounted via a
  `Registrar func(huma.API)` seam); `internal/observability` (logger, request-id
  child logger + access log, slog Recoverer, Metrics struct owning its prometheus
  registry with bounded label cardinality); `internal/config` (Viper
  TEMPLATE_GO_API_*, Validate, RegisterFlags); `internal/app` composition root +
  graceful shutdown; `internal/cli` serve(default)/version/openapi;
  `docs/docs/openapi.yaml` committed (3.0.3). Functional-first tests through the
  real in-memory adapter; `moon run check` + `go test -race ./...` green; live
  server curl-verified end to end.
- Session 002 is still marked **in-progress** in INDEX.md even though its PR merged
  and slice-1 scope is complete. Flagged to the user — likely wants to close 002.

Deferred follow-ups still open (seams already in place, from 002 + TARGET_SHAPE.md):
- Docs render pipeline: neoteroi OAD render + `docs:openapi` Moon wiring + CI
  drift-guard (regeneration produces no diff).
- CORS / client-IP middleware.
- README / DELETE_ME refresh for concrete API-server usage now that code landed.
- Thread the request-scoped logger (LoggerFrom) into the service so service-level
  logs carry request_id.
- Optionally richer (named) readiness reporting.
- Larger seams: authn/authz; Postgres adapter + testcontainers; OTel tracing
  exporter; rate limiting; pagination conventions; API versioning; mockery.

Plan: deliver the implemented/not-implemented summary first (this kickoff), then
await the user's direction on which follow-up to tackle. Scope mechanical work into
a plan and collaborate on design separately (per memory
`separate-mechanical-from-design-work`).
