---
id: 011
title: Finalize the repo
started: 2026-06-24
---

## 2026-06-24 09:51 — Kickoff
Goal for the session: finalize this repository today. The specific scope of
"finalize" is not yet stated — awaiting the user's concrete request before
starting substantive work.

Current state of the world:
- The API-server template is feature-complete and on `master` at `f2c5210`
  (PR #15). Sessions 001–010 are all closed/complete in `INDEX.md`.
- Architecture (per `TECH_NOTES.md`): chi v5 + Huma v2 (code-first OpenAPI),
  per-domain ports & adapters under `internal/todo/{httpapi,postgres,authz}`
  with shared infra under `internal/adapter/{http,postgres}`; PostgreSQL-only
  persistence (pgx + sqlc + goose); Cedar authz with deferred API-key authn
  (now stored as SHA-256 hashes); Docker Compose day-one stack; keyset
  pagination on `GET /todos`; CI runs the container-backed integration suite on
  `ubuntu-latest`; sqlc binary integrity-pinned.
- Recent work (session 010) remediated all three findings from an independent
  Codex security review as separate squash PRs (#13/#14/#15).

Known open threads carried from prior sessions:
- Future-slice seams left as documented extension points (not built): OTel
  tracing, rate limiting, API versioning.
- Session **005** still shows `in-progress`/empty in `INDEX.md` (pre-existing).
- The main checkout has untracked local tooling dirs (`.agents/skills/
  codex-security-scan/`, `.codex-security/`, `.claude/`) — local artifacts vs.
  template content; disposition still open.
- The other three Proto plugins verify against unsigned upstream checksum files
  (defense-in-depth pinning was scoped out of #1).

Plan: wait for the user's concrete definition of "finalize," then scope and
sequence the work (worktree → PR → squash-merge per the session protocol).

## 2026-06-24 10:14 — API versioning: decision + plan

Scope for the session is "tie up the loose threads." Starting with **API
versioning** (a documented-but-unbuilt seam).

**Design decisions (user-confirmed via AskUserQuestion):**
- **Strategy: URI path versioning under `/v1`** (over header / media-type /
  date-based). The pragmatic, discoverable, cache-friendly default a reference
  template should teach.
- **Depth: establish the `/v1` seam only** — mount all current operations under
  `/v1`, keep infra routes unversioned, document how a future `/v2` group is
  added. Do NOT build a second version that doesn't exist yet.

**Grounding (verified against Huma v2.38.0 source in the module cache, not
memory):**
- Use Huma's first-class `huma.NewGroup(api, "/v1")` (group.go). A `Group`
  implements `huma.API`, so it drops into the existing `Registrar
  func(huma.API)` seam with no signature changes.
- `Group.Middlewares()` = parent API middlewares + group's own (group.go:180).
  `huma.Register` bakes `api.Middlewares()` into each op (huma.go:777). Since
  `InstallAuthz(api)` runs `api.UseMiddleware` BEFORE `registerResources`
  creates the group, the authz middleware is inherited by every `/v1` route —
  **no auth bypass**. This was the key risk and it's cleared.
- The group writes into the parent's shared `OpenAPI()` doc (group.go:109), so
  `FinalizeAuthz`/`DocumentSecurity` and the server-less `SpecYAML` export both
  pick up the `/v1` paths. Single prefix ⇒ OperationIDs/tags untouched
  (group.go:24 guard); `{id}` param preserved so `ctx.Param("id")` still works.

**Implementation (single composition point):** wrap resource registration in
`app.go`'s `registerResources` with `huma.NewGroup(api, apiVersionV1)`. Both the
running server (`NewRouter`) and the spec export (`OpenAPIYAML`→`SpecYAML`) flow
through `registerResources`, so the prefix propagates consistently. The
`httpapi` adapter stays version-agnostic (its own paths remain `/todos`).

**Blast radius:**
- UNCHANGED (isolation unit tests on a bare API): `httpapi/handler.go`,
  `httpapi/api_test.go`, `internal/authz/middleware_test.go`,
  `internal/todo/authz/{policy,facts}_test.go`, `adapter/http/router_test.go`
  (Register nil; its `/todos` is an arbitrary unrouted path).
- UPDATE to `/v1/todos` (composed/full-app): `internal/app/app_test.go`,
  `internal/integration/authz_e2e_test.go`; tighten `internal/cli/openapi_test.go`
  to assert `/v1/todos`.
- REGENERATE: `docs/docs/openapi.yaml` (via `root:openapi`).
- DOCS: `README.md`, `docs/docs/index.md`, `compose.yaml` (comment),
  `DELETE_ME.md` ("add a resource" → register on `/v1`).

Next: implementation worktree off `master`, implement, `root:check` + container
integration suite, PR → squash-merge.

## 2026-06-24 10:23 — API versioning: shipped (PR #16, `a485f7e`)

**Done.** `feat(api): serve resource routes under a /v1 version prefix` —
**PR #16 squash-merged to `master` `a485f7e`**. Worktree removed, local `master`
fast-forwarded, remote branch deleted, `.journal` still untracked on `master`.

**The change (9 files):**
- Core: `internal/app/app.go` — `apiVersionV1 = "/v1"` const + `registerResources`
  now does `v1 := huma.NewGroup(api, apiVersionV1); httpapi.Register(v1, service)`.
  Single composition point feeds both the running router and the server-less
  spec export, so `/v1` propagates consistently. The `httpapi` adapter is
  unchanged (version-agnostic, still declares `Path: "/todos"`).
- Composed tests → `/v1/todos`: `app_test.go`, `integration/authz_e2e_test.go`;
  tightened `cli/openapi_test.go` to assert `/v1/todos`. Adapter/authz UNIT tests
  (bare-API isolation) left at `/todos` on purpose.
- Regenerated `docs/docs/openapi.yaml` (paths → `/v1/...`; IDs/tags unchanged).
- Docs: README (new **API versioning** section + curl + add-a-resource step),
  `docs/docs/index.md`, `compose.yaml`, `DELETE_ME.md`.

**Verification:** `root:check` green; `root:test-integration` green locally
(`postgres:17-alpine`, 10.959s); CI green (`ci` 1m11s) and confirmed it ran
`root:test-integration` on the runner (`internal/integration ok 14.352s`) — so
the `/v1` routing was proven end-to-end with authz enforced on `ubuntu-latest`.

**Key proof (cleared the one real risk):** Huma v2.38.0 `Group.Middlewares()`
= parent + own, and `huma.Register` bakes `api.Middlewares()` per op (huma.go:777);
`InstallAuthz(api)` runs before the group is created, so the deny-by-default
authz middleware is inherited by every `/v1` route — no auth bypass. The
integration suite's 401/403/200 matrix confirms it at runtime.

Remaining loose threads (for the rest of the session): session 005 stuck
`in-progress` in INDEX; untracked local tooling dirs in the main checkout
(`.agents/skills/codex-security-scan/`, `.codex-security/`, `.claude/`);
defense-in-depth checksum pinning for the other 3 Proto plugins; other
future-slice seams (OTel tracing, rate limiting).

## 2026-06-24 11:51 — Rate limiting: decision + plan

Next thread (user-chosen): **build rate limiting**.

**Design decisions (user-confirmed via AskUserQuestion):**
- **Scope: per-IP, pre-auth** (over per-principal / layered). Protects the auth
  path + DB (every request hits `api_keys`) from anonymous floods; key func is
  the seam to evolve to per-principal.
- **Backend: in-process token bucket + documented Redis seam, behind a
  `ratelimit.Limiter` port** (over shipping Redis now / no-port). Matches the
  repo's port+adapter+seam idiom (`todo.Repository`, `Authenticator`).

**Grounding (verified against Huma v2.38.0 + chi v5.3.0 + module graph):**
- `golang.org/x/time v0.11.0` already in the graph → use `x/time/rate` (token
  bucket), no new direct-dep risk.
- Implement as a **Huma middleware** (like authz): auto-exempts infra routes
  (they bypass Huma), native RFC 9457 via `huma.WriteErr`, `ctx.SetHeader` for
  `Retry-After`. "Pre-auth" = install order: rate-limit middleware installed
  BEFORE authz's `authenticate`, so a limited request never touches the DB.
- Client IP at the Huma layer: `humachi.Unwrap(ctx)` → `*http.Request` →
  `chimiddleware.GetClientIP(r.Context())` (the existing ClientIP middleware,
  spoof-safe via `--trusted-proxy-header`, already populated it).
- **Headers decision:** ship `Retry-After` (RFC 9110, unambiguous) on 429. The
  IETF `RateLimit`/`RateLimit-Policy` structured-field headers are still a DRAFT
  (draft-ietf-httpapi-ratelimit-headers-11, May 2026) and the token-bucket→
  window mapping is approximate, so DON'T ship an approximate impl in a template
  others copy verbatim — document the draft as a noted enhancement instead.
- **OpenAPI:** 429 is cross-cutting middleware, not per-op (authz didn't add
  401/403 per-op either) → do NOT add 429 to each operation. Spec UNCHANGED, no
  regen, openapi-check stays green.

**Layering (hexagonal):**
- New `internal/ratelimit`: `Limiter` port + `Decision` (limiter.go); in-process
  token-bucket adapter w/ per-key `*rate.Limiter` registry + idle-eviction
  janitor + `Stop()` (memory.go); `Middleware` taking a router-agnostic
  `KeyFunc func(huma.Context)(string,error)`, `Install()` via `UseMiddleware`,
  inert when disabled (middleware.go). Imports huma only (like authz), NOT chi.
- `internal/adapter/http`: a `ClientIPKeyFunc` helper (humachi+GetClientIP) —
  the chi-specific key extraction stays in the transport adapter; `RouterDeps`
  gains `InstallRateLimit func(huma.API)` called BEFORE `InstallAuthz`.
- `internal/app/app.go`: build the limiter (if enabled) + install hook; store
  the limiter on `App` and `Stop()` it on shutdown (mirror `closePool`).
- `internal/config`: `--rate-limit-enabled` (default true, like authz),
  `--rate-limit-rps` (default 10), `--rate-limit-burst` (default 20) + env +
  Load + setDefaults + config_test.

**Test strategy:** unit `memory_test.go` (burst→deny→refill) + `middleware_test.go`
(humatest: allow/deny, 429 RFC9457 shape + Retry-After, per-key isolation,
disabled passthrough). Keep the authz e2e deterministic by setting
`rate-limit-enabled=false` in `e2eServer` (orthogonal concern). app_test (1-2
reqs) passes under default-on. Docs: README (Rate limiting section + flags),
DELETE_ME (inventory + seam), router.go stale "rate limiting" seam comment.

Next: worktree off `master`, implement, `root:check` + integration suite, PR.

## 2026-06-24 12:06 — Rate limiting: shipped (PR #17, `867662f`)

**Done.** `feat(api): add per-client IP rate limiting` — **PR #17 squash-merged
to `master` `867662f`**. Worktree removed, local `master` fast-forwarded, remote
branch deleted, `.journal` still untracked on `master`.

**The change (new package + wiring, 11 files + 5 new):**
- New `internal/ratelimit`: `Limiter` port + `Decision` (limiter.go); in-process
  per-key token-bucket adapter w/ idle-eviction janitor + `Stop()` (memory.go);
  Huma `Middleware` taking a router-agnostic `KeyFunc`, inert when disabled
  (middleware.go); unit tests (memory_test.go, middleware_test.go).
- `internal/adapter/http/ratelimit.go`: `ClientIPKeyFunc` (humachi.Unwrap +
  chi GetClientIP) — chi-specifics stay in the transport adapter.
- `router.go`: `RouterDeps.InstallRateLimit` called BEFORE `InstallAuthz`;
  refreshed the stale "deferred seams" comment.
- `app.go`: `buildRateLimiter` (nil when disabled) + wired install hook + stored
  the limiter on `App`; `serve.go`: `stopRateLimiter` deferred in Run (mirrors
  `closePool`).
- `config.go` + `config_test.go`: `--rate-limit-enabled` (default true),
  `--rate-limit-rps` (10), `--rate-limit-burst` (20) + validation + coverage.
- `app_test.go`: `TestAppWiringRateLimits` — functional proof on the composed
  handler (burst 1 → 2nd req 429 BEFORE auth, i.e. not 403; /healthz exempt).
- `authz_e2e_test.go`: disabled rate limiting in `e2eServer` (orthogonal).
- Docs: README **Rate limiting** section + 3 flags; DELETE_ME inventory + seam.
- `go.mod`: promoted `golang.org/x/time` to a direct require.

**Verification:** `root:check` green (lint/format/test/openapi-check/sqlc — spec
UNCHANGED, no per-op 429, as decided); `root:test-integration` green locally
(`postgres:17-alpine`); CI green (`ci` 1m1s) and confirmed it ran both the
`ratelimit` unit suite (`ok ... 0.029s`) and `root:test-integration`
(`internal/integration ok 17.868s`) on `ubuntu-latest`.

**Notes / gotchas:**
- Huma serializes an empty output struct as **204**, not 200 — the middleware
  tests assert `StatusNoContent` for allowed requests (initial 200 assert failed).
- testifylint: float field asserts need `InDelta`/`InEpsilon`, not `Equal`
  (`RateLimitRPS`). godoclint: stdlib refs in doc comments need `[time.NewTicker]`.
- With rate-limit default-on, `app.New`-without-`Run` tests leak the janitor
  goroutine (10-min ticker, harmless; repo has no goleak). Acceptable; documented.

Remaining loose threads (housekeeping, if pursued): session 005 stuck
`in-progress` in INDEX; untracked local tooling dirs in the main checkout;
defense-in-depth checksum pinning for the other 3 Proto plugins; OTel tracing
seam still unbuilt (deferred — user chose rate limiting only).
