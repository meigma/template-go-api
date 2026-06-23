---
id: 003
title: Finish the API-server template (slice 2 — deferred follow-ups)
date: 2026-06-22
status: complete
repos_touched: [template-go-api]
related_sessions: ["002"]
---

## Goal
Close the near-term items deferred from session 002's slice 1 so the template is
complete for a first consumer: docs render pipeline, CORS, client-IP, request-scoped
service logging, named readiness reporting, and a human-docs refresh.

## Outcome
Met. Shipped as **PR #5, squash-merged to `master 05f5446`** (8 commits: the 5-part
slice plus three interactive-review follow-ups — test consolidation, a middleware
package refactor, and a dedicated metrics listener). `moon run root:check`, `go test
-race ./...`, and live-server probes all green; CI (ci / Pages / Kusari) green on
every push. `master` is now the finished hexagonal Go API-server template.

## Key Decisions
- **Client-IP avoids chi's `middleware.RealIP`** — it is deprecated in chi v5.3.0 for
  IP-spoofing CVEs. Used the new `ClientIPFrom*` family: default trusts the TCP peer
  (unspoofable), `--trusted-proxy-header` opts into a single proxy header.
- **CORS via `go-chi/cors`**, disabled until `--cors-allowed-origins` is set (safe
  template default); correct preflight/Vary handling is fiddly to hand-roll.
- **`internal/logctx` leaf** (stdlib-only) carries the request-scoped logger so the
  domain inherits `request_id` without importing the prometheus-laden `observability`.
- **Docs decoupled from Go (deviation from the approved plan):** instead of wiring
  `docs:build` → a Go `openapi` task, the docs build renders the committed spec and a
  temp-file `openapi-check` drift-guard in `root:check` enforces freshness. Avoids Go
  in the Pages workflow and a spec read/write race during `check`; same no-drift guarantee.
- **neoteroi OAD CSS vendored** from the v1.2.0 release — the pip wheel omits it.
- **Middleware grouped** into `internal/adapter/http/middleware`; the shared RFC 9457
  writer extracted to `internal/adapter/http/problem` to avoid an import cycle (router
  fallbacks + middleware both use it).
- **Dedicated metrics listener** (`--metrics-addr`, default `:9090`): `/metrics` moves
  off the API listener and outside its middleware chain (no self-counting, no access-log
  spam, off the public surface); the metrics middleware still records API requests; empty
  co-locates it on the API port. Health/readiness stay on the API listener (name is
  explicitly "metrics", not "admin").

## Changes
- New packages: `internal/logctx`, `internal/adapter/http/middleware` (CORS/ClientIP/
  Recoverer/Timeout), `internal/adapter/http/problem`. New `adapter/http/metrics.go`
  (dedicated metrics handler).
- `internal/config` + `app` + `adapter/http/router` — `--cors-allowed-origins`,
  `--trusted-proxy-header`, `--metrics-addr`; named `ReadinessCheck` struct + per-check
  `/readyz` body; `client_ip` in the access log; two-listener `app`/`serve`.
- Docs pipeline: `docs/pyproject.toml` + `uv.lock` (neoteroi-mkdocs==1.2.0), vendored
  `docs/docs/stylesheets/mkdocsoad.css`, `mkdocs.yml` (plugin/nav/extra_css), new
  `docs/docs/api.md`, root `moon.yml` `openapi` + `openapi-check` tasks.
- Human docs: README/DELETE_ME/index/CONTRIBUTING rewritten for the API server.

## Open Threads
Future-slice seams (all left intentionally): authn/authz; Postgres adapter +
testcontainers; OTel tracing exporter; rate limiting; pagination conventions; API
versioning; mockery. Minor: the metrics default `:9090` collides with a locally-run
Prometheus server (overridable); health/readiness could optionally move to the metrics
listener if a broader "ops" port is wanted later.

## References
- PR #5: https://github.com/meigma/template-go-api/pull/5 (merged, `05f5446`)
- Prior slice: `.journal/002/SUMMARY.md`; design doc: `.journal/TARGET_SHAPE.md`
- Memory: `separate-mechanical-from-design-work`, `subagents-may-read-divergent-worktree`

## Lessons
- A planned build-graph edge (docs → Go task) was correct on paper but worse in
  practice (Go in the Pages workflow + a spec read/write race during `check`).
  Refining it to a decoupled drift-guard during implementation — and flagging the
  deviation in the PR — beat following the plan literally. Agile over waterfall.
