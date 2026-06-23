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

## 2026-06-22 17:09 — Slice 2 implemented, PR #5 opened

Direction: implement ONE slice covering all five near-term deferred items. Effort
set to ultracode (xhigh + workflow orchestration).

Planning (plan mode): 3 Explore agents mapped the docs pipeline, the
middleware/observability/config code, and the stale human docs; 3 Plan agents
designed each chunk. Two key discoveries shaped the design: (1) chi v5.3.0
**deprecated `middleware.RealIP`** (IP-spoofing CVEs) in favor of a new safe
`ClientIPFrom*` family; (2) neoteroi OAD's **CSS is not in the pip wheel** and must
be vendored from the GitHub release. Four design decisions confirmed with the user
(all recommended): CORS via `go-chi/cors`; client-IP header-opt-in only; include
named readiness; one PR with grouped commits.

Implemented (worktree `feat/api-template-finish` off master, 5 commits):
- `internal/logctx` (stdlib-only leaf) + service resolves request-scoped logger →
  service logs carry `request_id`; domain stays prometheus-free (verified via
  `go list -deps`).
- CORS (`go-chi/cors`, disabled until `--cors-allowed-origins` set) + client-IP
  (chi `ClientIPFrom*`; default trusts TCP peer, `--trusted-proxy-header` opts in);
  access log gains `client_ip`. New config fields + flags/env.
- Named readiness: `ReadinessCheck` → `{Name, Check}` struct; `/readyz` runs all
  checks, reports `{"status":...,"checks":{name:ok|unavailable}}`.
- Docs render pipeline: neoteroi OAD API Reference page (vendored CSS), root
  `openapi` + `openapi-check` Moon tasks; drift-guard wired into `root:check`.
- Human docs: README/DELETE_ME/index/CONTRIBUTING rewritten for the API server.

Design refinement vs the approved plan: kept `docs:build` **decoupled from Go**
(renders the committed spec) and enforced freshness via the `openapi-check`
drift-guard in `root:check`, rather than wiring `docs:build` → a Go task. Avoids
Go in the Pages workflow and a spec read/write race during `check`; same no-drift
guarantee. Flagged in the PR body.

Verification: `moon run root:check` green (format, lint, build, test, openapi-check,
docs:build). `go test -race ./...` green. neoteroi renders styled endpoint tables
under `mkdocs build --strict`; drift-guard verified to fail on a tampered spec and
pass clean. Live server end-to-end: CORS preflight allow-origin/Vary; `client_ip`
resolved to trusted `X-Real-IP` (203.0.113.7) while forged `X-Forwarded-For`
(9.9.9.9) was ignored; `todo created` service log shared the access line's
`request_id`; `/readyz` → `{"status":"ready","checks":{}}`; runtime `/docs`,
`/openapi.yaml`, `/openapi.json` all 200; graceful shutdown on SIGTERM. Deps added:
`go-chi/cors` v1.2.2, `neoteroi-mkdocs==1.2.0` (docs uv).

PR: https://github.com/meigma/template-go-api/pull/5 (open, CI running).

Still deferred (future slices): authn/authz; Postgres adapter + testcontainers;
OTel tracing; rate limiting; pagination; API versioning; mockery. Also: session 002
remains formally `in-progress` in INDEX despite its merge (flagged to the user).
