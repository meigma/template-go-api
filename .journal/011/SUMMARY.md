---
id: 011
title: Finalize the repo — API versioning, rate limiting, OTel tracing, housekeeping
date: 2026-06-24
status: complete
repos_touched: [template-go-api]
related_sessions: ["005", "009", "010"]
---

## Goal
Finalize the template by tying up the loose threads carried from prior sessions:
build the three documented-but-unbuilt feature seams (API versioning, rate
limiting, OTel tracing) and clear the housekeeping items (a stuck journal row,
untracked local tooling dirs, defense-in-depth Proto checksum pinning).

## Outcome
Met. Five squash PRs merged to `master` (now `5d120e2`), each CI-green on
`ubuntu-latest`, worktrees removed, local `master` fast-forwarded, working tree
clean:

- **PR #16 `a485f7e`** — `feat(api): serve resource routes under a /v1 version prefix`.
- **PR #17 `867662f`** — `feat(api): add per-client IP rate limiting`.
- **PR #18 `6625ab1`** — `feat(api): add OpenTelemetry tracing (HTTP + DB spans)`.
- **PR #19 `95ecd8a`** — `chore: ignore local tooling artifacts and document CLI checksum verification`.
- **PR #20 `5d120e2`** — `chore: ignore .claude as a symlink, not only a directory`.

Each feature was designed collaboratively (grounded in current Huma/OTel docs +
the actual code, then 1–2 AskUserQuestion decision points) before implementing.
All three feature seams documented in `TARGET_SHAPE`/`TECH_NOTES` as "future"
are now built. The housekeeping cluster is fully resolved — see Key Decisions.

## Key Decisions
- **Versioning: URI path `/v1` via `huma.NewGroup`, seam only** (user choice).
  Applied in ONE spot — `registerResources` in `app.go` — which is the single
  `Registrar` feeding both the running router and the server-less spec export, so
  `/v1` propagates consistently and `openapi-check` stays in sync. A Huma group
  IS a `huma.API`, so the adapter is unchanged (still declares `Path: "/todos"`)
  and the root-API authz middleware is inherited by grouped routes — verified
  against Huma v2.38.0 source (`Group.Middlewares()` = parent + own;
  `huma.Register` bakes `api.Middlewares()` per op). Built the seam + documented
  `/v2`; did not build a second version.
- **Rate limiting: per-IP, pre-auth, in-process token bucket + port seam** (user
  choices). A Huma middleware installed BEFORE authn (rejects over-limit requests
  with RFC 9457 429 + `Retry-After` before the credential store is touched);
  infra routes bypass Huma so they are exempt. `golang.org/x/time/rate` behind a
  `ratelimit.Limiter` port (the Redis seam); pluggable `KeyFunc` (default client
  IP). On by default, generous (10 rps / 20 burst). Shipped `Retry-After` only —
  the IETF `RateLimit` structured-field headers are still a draft and map loosely
  onto a token bucket, so they are a documented enhancement, not an approximate impl.
- **Tracing: opt-in, standard `OTEL_*` env config, HTTP + DB spans** (user
  choices). Default OFF (needs a collector — unlike the self-contained authz/
  rate-limit tiers). otelhttp server spans (named by operation via a span-namer
  Huma middleware, infra routes filtered) + otelpgx DB spans; OTLP/HTTP exporter
  configured entirely via `OTEL_*` env (no bespoke flags), one `--tracing-enabled`
  switch. Flushed on shutdown.
- **Housekeeping item 1 (session 005): no-op.** Verified 005 is `complete` with a
  full `SUMMARY.md`; the "stuck in-progress" note was stale and wrongly copied
  forward through the 008/010 summaries. Marking it abandoned would have been wrong.
- **Housekeeping item 3 (Proto checksum pinning): skipped + documented** (user
  choice). golangci-lint/goose/mockery already verify against upstream
  `checksum-url`; sqlc is repo-pinned only because it publishes none. Repo-pinning
  the other three would add ~12 cross-platform digests to maintain per bump,
  duplicating an existing control. Rationale documented in README "CI and Security".

## Changes
- **PR #16:** `internal/app/app.go` (`registerResources` → `huma.NewGroup(api,
  "/v1")` + `apiVersionV1`); composed tests `app_test.go`,
  `integration/authz_e2e_test.go` → `/v1/todos`; `cli/openapi_test.go` tightened;
  regenerated `docs/docs/openapi.yaml`; README (new **API versioning** section),
  `docs/docs/index.md`, `compose.yaml`, `DELETE_ME.md`.
- **PR #17:** new `internal/ratelimit/{limiter,memory,middleware}.go` + tests;
  `internal/adapter/http/ratelimit.go` (`ClientIPKeyFunc`); `router.go`
  (`InstallRateLimit` before `InstallAuthz`); `app.go`/`serve.go` (build + Stop on
  shutdown); `config.go`/`config_test.go` (`--rate-limit-{enabled,rps,burst}`);
  `app_test.go` (`TestAppWiringRateLimits`); README **Rate limiting** + DELETE_ME;
  `go.mod` promoted `golang.org/x/time` to direct.
- **PR #18:** new `internal/observability/tracing.go` (`NewTracerProvider` +
  `TraceSpanNamer`) + test; `router.go` (`Tracing` → otelhttp wrap +
  `traceableRequest` filter + infra-path consts) + test; `adapter/postgres/
  postgres.go` (`Config.Tracing` → otelpgx); `app.go`/`serve.go` (provider +
  flush); `config.go`/`config_test.go` (`--tracing-enabled`, default false);
  README **Tracing** + DELETE_ME; `go.mod` (otelpgx, otlptracehttp; otel 1.43→1.44).
- **PR #19:** `.gitignore` (`.codex-security/`, `.agents/skills/codex-security-scan/`);
  README "CI and Security" checksum-verification note.
- **PR #20:** `.gitignore` `.claude/` → bare `.claude`.

## Open Threads
- Tracing default OFF: enabling it in a composed-app test can't assert spans
  easily because `otelhttp.NewHandler` captures the global tracer at construction
  (can't inject an in-memory exporter post-`app.New`); the span path is covered by
  the `TraceSpanNamer` unit test. otelpgx DB spans are not span-asserted (the
  integration suite runs with tracing off). A future enhancement: a demo OTel
  Collector in Compose + asserting DB spans.
- Rate limiting ships `Retry-After` only; the IETF `RateLimit`/`RateLimit-Policy`
  headers remain a documented future enhancement (still an IETF draft).
- Versioning is seam-only — no live `/v2` exists; the migration path is documented.
- No remaining "future seam" backlog: versioning, rate limiting, and tracing were
  the last three. (Service-level manual tracing spans intentionally out of scope.)

## References
- PRs: #16 https://github.com/meigma/template-go-api/pull/16 (`a485f7e`),
  #17 /pull/17 (`867662f`), #18 /pull/18 (`6625ab1`), #19 /pull/19 (`95ecd8a`),
  #20 /pull/20 (`5d120e2`).
- Session log: `.journal/011/NOTES.md`.
- Builds on: `.journal/005/SUMMARY.md` (authz tier the rate-limit/tracing
  middleware sit beside), `.journal/009/SUMMARY.md` (CI affected-gating, the
  `.agents/` gitignore thread), `.journal/010/SUMMARY.md` (sqlc checksum pinning
  the item-3 decision references).

## Lessons
- **A trailing-slash gitignore pattern matches only a real directory, never a
  symlink.** `.claude` was a symlink → `.agents` (local convenience), so `.claude/`
  could never ignore it while `.codex-security/` (real dir) matched — it sat
  untracked indefinitely. Bare `.claude` matches dir OR symlink. The diagnostic
  tell: `ls -la` not listing `.claude` while `ls -la .claude` shows the target's
  contents. (Burned real time on check-ignore/hexdump/global-excludes forensics
  before spotting the symlink — check `readlink` early when a valid ignore pattern
  inexplicably doesn't match.)
- **Huma `NewGroup` is the clean versioning primitive on v2.38+:** a group is a
  `huma.API`, shares the parent's OpenAPI doc, and inherits parent middleware
  (so authz installed pre-group still applies). Verified in source before relying
  on it — the one real risk (losing authz on `/v1`) was cleared this way and by
  the live integration 401/403/200 matrix.
- **Huma serializes an empty output struct as 204, not 200** — rate-limit
  middleware tests assert `StatusNoContent` for allowed requests.
- **OTel resource schema-URL conflicts** are avoided by keeping only
  `WithTelemetrySDK` schema-bearing while `WithAttributes`/`WithFromEnv` stay
  schemaless; `WithFromEnv` last lets `OTEL_SERVICE_NAME` override the default.
- **`moon ci` affected-gating** still means docs/config-only PRs (#19, #20) run no
  Go tasks; the code PRs (#16–#18) exercised `test-integration` on the runner.
