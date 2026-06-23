---
id: 002
title: Implement API-server template per TARGET_SHAPE.md (slice 1)
date: 2026-06-22
status: complete
repos_touched: [template-go-api]
related_sessions: ["001", "003"]
---

## Goal
Implement the first vertical slice of the Go web API server template per the
approved design in `.journal/TARGET_SHAPE.md` (chi + Huma, pragmatic ports &
adapters, in-memory reference resource, observability, server-less OpenAPI export).

## Outcome
Met. Slice 1 shipped as **PR #4, squash-merged to `master 745a9ed`** — a runnable
hexagonal Go API-server template with a `todo` reference resource. `moon run check`,
`go test -race ./...`, and live curl probes (201/404/422 problem+json, complete,
list, `/metrics`, JSON access logs with `request_id`, SIGTERM graceful shutdown) all
green at merge. The docs render pipeline and several cross-cutting middleware were
intentionally deferred to a later slice (became session 003).

## Key Decisions
- Scope toggles (all confirmed with user): metrics IN slice 1; the `openapi` command
  writes the spec file but the neoteroi render pipeline is deferred; core middleware
  stack only (request-id → recover → access-log → timeout).
- Verified real lint constraints directly (two Plan agents had read the stale journal
  worktree and misreported): `exhaustruct`/`wrapcheck` DISABLED; real constraints are
  `sloglint`, `gochecknoglobals`, `funlen`/`cyclop`, `godoclint`, `mnd`, `promlinter`.
  No rename step — `master` was already renamed in session 001. → memory
  `subagents-may-read-divergent-worktree`.
- Four interactive-review follow-ups before merge: bounded metrics label cardinality
  (raw `method` → "other"); split transport into resource-agnostic `internal/adapter/http`
  + `internal/adapter/http/todoapi` via a `Registrar func(huma.API)` seam; named
  context-aware `ReadinessCheck`; RFC 9457 standardization on every non-Huma surface
  (chi 404/405, panic 500, timeout 504) via a single problem+json writer.

## Changes
- `internal/todo` — domain (entity, Status enum, validation, idempotent Complete,
  consumer-defined `Repository` port, `Service` with now/newID seams).
- `internal/adapter/memory` — RWMutex map repo; `internal/adapter/http` (+ `todoapi`)
  — chi + Huma v2 (humachi), 4 typed ops, DTO↔domain, RFC 9457 errors, raw `/healthz`
  `/readyz` `/metrics`.
- `internal/observability` — logger, request-id child logger + access log, slog
  Recoverer, Metrics struct owning its prometheus registry.
- `internal/config` (Viper `TEMPLATE_GO_API_*`), `internal/app` composition root +
  graceful shutdown, `internal/cli` serve/version/openapi; deleted `internal/templateinfo`.
- `docs/docs/openapi.yaml` committed (3.0.3 via DowngradeYAML). Deps: chi/v5, huma/v2,
  prometheus client, google/uuid; testify promoted to direct.

## Open Threads
All picked up and completed in session 003 (PR #5): docs render pipeline + drift-guard;
CORS / client-IP; request-scoped logger into the service; named readiness reporting;
README/DELETE_ME refresh. Still deferred to future slices: authn/authz; Postgres adapter
+ testcontainers; OTel tracing; rate limiting; pagination; API versioning; mockery.

## References
- PR #4: https://github.com/meigma/template-go-api/pull/4 (merged, `745a9ed`)
- Design doc: `.journal/TARGET_SHAPE.md`; prior session: `.journal/001/SUMMARY.md`
- Follow-on: `.journal/003/SUMMARY.md`
- Memory: `subagents-may-read-divergent-worktree`
